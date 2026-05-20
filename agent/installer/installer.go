package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zenspanel/zenspanel/agent/safe"
)

// App describes an installable web application.
type App struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// Catalog is the list of supported apps. Versions are pinned so installs
// are reproducible; bump them when a new stable release ships.
var Catalog = []App{
	{
		ID:          "wordpress",
		Name:        "WordPress",
		Version:     "6.5.3",
		Description: "The world's most popular CMS. Requires a MySQL database.",
	},
	{
		ID:          "laravel",
		Name:        "Laravel",
		Version:     "11",
		Description: "PHP web application framework. Installs via Composer.",
	},
	{
		ID:          "html",
		Name:        "Plain HTML",
		Version:     "—",
		Description: "A simple index.html starter page. No database required.",
	},
}

// JobStatus tracks the progress of an async install.
type JobStatus struct {
	Phase  string    `json:"phase"`
	Log    []string  `json:"log"`
	Done   bool      `json:"done"`
	Error  string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

var (
	jobsMu sync.Mutex
	jobs   = map[string]*JobStatus{}
)

func getJob(jobID string) (*JobStatus, bool) {
	jobsMu.Lock()
	defer jobsMu.Unlock()
	j, ok := jobs[jobID]
	return j, ok
}

func newJob(jobID string) *JobStatus {
	j := &JobStatus{Phase: "starting", StartedAt: time.Now()}
	jobsMu.Lock()
	jobs[jobID] = j
	jobsMu.Unlock()
	return j
}

func (j *JobStatus) log(msg string) {
	jobsMu.Lock()
	j.Log = append(j.Log, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg))
	jobsMu.Unlock()
}

func (j *JobStatus) setPhase(phase string) {
	jobsMu.Lock()
	j.Phase = phase
	jobsMu.Unlock()
	j.log("Phase: " + phase)
}

func (j *JobStatus) fail(err error) {
	jobsMu.Lock()
	j.Phase = "failed"
	j.Error = err.Error()
	j.Done = true
	jobsMu.Unlock()
	j.log("ERROR: " + err.Error())
}

func (j *JobStatus) done() {
	jobsMu.Lock()
	j.Phase = "done"
	j.Done = true
	jobsMu.Unlock()
	j.log("Installation complete.")
}

// Status returns the current status of a job.
func Status(jobID string) (*JobStatus, error) {
	j, ok := getJob(jobID)
	if !ok {
		return nil, fmt.Errorf("job %q not found", jobID)
	}
	return j, nil
}

// RunParams holds all parameters for an install job.
type RunParams struct {
	JobID      string
	AppID      string
	Username   string
	HomeBase   string
	DocRoot    string
	DBName     string
	DBUser     string
	DBPass     string
	DBHost     string
	SiteURL    string
	Overwrite  bool
}

// Run starts an async installation. Returns immediately; poll Status(jobID).
func Run(p RunParams) error {
	if err := safe.Username(p.Username); err != nil {
		return err
	}
	// Validate app ID against catalog.
	var found bool
	for _, a := range Catalog {
		if a.ID == p.AppID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("unknown app %q", p.AppID)
	}

	j := newJob(p.JobID)
	go func() {
		if err := runInstall(j, p); err != nil {
			j.fail(err)
		} else {
			j.done()
		}
	}()
	return nil
}

func runInstall(j *JobStatus, p RunParams) error {
	// Overwrite guard (V32): refuse to install into a non-empty docroot
	// unless the caller explicitly set overwrite=true.
	if !p.Overwrite {
		entries, err := os.ReadDir(p.DocRoot)
		if err == nil && len(entries) > 0 {
			// Allow if the only file is the placeholder index.html we created.
			if !(len(entries) == 1 && entries[0].Name() == "index.html") {
				return fmt.Errorf("docroot %q is not empty; set overwrite=true to proceed", p.DocRoot)
			}
		}
	}

	switch p.AppID {
	case "wordpress":
		return installWordPress(j, p)
	case "laravel":
		return installLaravel(j, p)
	case "html":
		return installHTML(j, p)
	default:
		return fmt.Errorf("unknown app %q", p.AppID)
	}
}

// runAs runs a shell command as the panel user (V33 — installer must not
// leave root-owned files in the user's home directory).
func runAs(j *JobStatus, username string, args ...string) error {
	cmd := exec.Command("su", append([]string{"-s", "/bin/sh", "-c",
		strings.Join(args, " "), username}, []string{}...)...)
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		j.log(string(out))
	}
	return err
}

// chownR recursively chowns a path to the panel user (V33).
func chownR(path, username string) error {
	return exec.Command("chown", "-R", username+":"+username, path).Run()
}

func installWordPress(j *JobStatus, p RunParams) error {
	j.setPhase("downloading")
	wpURL := fmt.Sprintf("https://wordpress.org/wordpress-%s.tar.gz", "6.5.3")
	tmpTar := filepath.Join(os.TempDir(), fmt.Sprintf("wp-%s.tar.gz", p.JobID))
	defer os.Remove(tmpTar)

	if out, err := exec.Command("curl", "-fsSL", "-o", tmpTar, wpURL).CombinedOutput(); err != nil {
		return fmt.Errorf("download wordpress: %w: %s", err, out)
	}

	j.setPhase("extracting")
	tmpDir := filepath.Join(os.TempDir(), "wp-extract-"+p.JobID)
	defer os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("mkdir tmp: %w", err)
	}
	if out, err := exec.Command("tar", "-xzf", tmpTar, "-C", tmpDir).CombinedOutput(); err != nil {
		return fmt.Errorf("extract: %w: %s", err, out)
	}

	j.setPhase("deploying")
	// WordPress extracts to tmpDir/wordpress/ — copy contents to docroot.
	wpSrc := filepath.Join(tmpDir, "wordpress")
	if out, err := exec.Command("rsync", "-a", "--delete", wpSrc+"/", p.DocRoot+"/").CombinedOutput(); err != nil {
		// Fallback to cp if rsync not available.
		if out2, err2 := exec.Command("cp", "-r", wpSrc+"/.", p.DocRoot+"/").CombinedOutput(); err2 != nil {
			return fmt.Errorf("deploy: %w: %s %s", err, out, out2)
		}
	}

	j.setPhase("configuring")
	// Write wp-config.php from the sample.
	sampleConf := filepath.Join(p.DocRoot, "wp-config-sample.php")
	conf := filepath.Join(p.DocRoot, "wp-config.php")
	data, err := os.ReadFile(sampleConf)
	if err != nil {
		return fmt.Errorf("read wp-config-sample: %w", err)
	}
	s := string(data)
	s = strings.ReplaceAll(s, "database_name_here", p.DBName)
	s = strings.ReplaceAll(s, "username_here", p.DBUser)
	s = strings.ReplaceAll(s, "password_here", p.DBPass)
	s = strings.ReplaceAll(s, "localhost", p.DBHost)
	if err := os.WriteFile(conf, []byte(s), 0644); err != nil {
		return fmt.Errorf("write wp-config: %w", err)
	}

	j.setPhase("setting_permissions")
	// Chown everything to the panel user (V33).
	if err := chownR(p.DocRoot, p.Username); err != nil {
		j.log("WARN: chown failed: " + err.Error())
	}
	return nil
}

func installLaravel(j *JobStatus, p RunParams) error {
	j.setPhase("creating_project")
	// composer create-project runs as the panel user (V33).
	// We install into a temp dir then move to avoid partial-install in docroot.
	tmpDir := filepath.Join(os.TempDir(), "laravel-"+p.JobID)
	defer os.RemoveAll(tmpDir)

	j.log("Running: composer create-project laravel/laravel " + tmpDir)
	cmd := exec.Command("su", "-s", "/bin/sh", "-c",
		fmt.Sprintf("composer create-project laravel/laravel %q --prefer-dist --no-interaction 2>&1", tmpDir),
		p.Username)
	out, err := cmd.CombinedOutput()
	j.log(string(out))
	if err != nil {
		return fmt.Errorf("composer create-project: %w", err)
	}

	j.setPhase("deploying")
	if out, err := exec.Command("rsync", "-a", "--delete", tmpDir+"/", p.DocRoot+"/").CombinedOutput(); err != nil {
		if out2, err2 := exec.Command("cp", "-r", tmpDir+"/.", p.DocRoot+"/").CombinedOutput(); err2 != nil {
			return fmt.Errorf("deploy: %w: %s %s", err, out, out2)
		}
	}

	j.setPhase("configuring")
	// Copy .env.example → .env and set APP_URL + DB credentials.
	envExample := filepath.Join(p.DocRoot, ".env.example")
	envFile := filepath.Join(p.DocRoot, ".env")
	data, err := os.ReadFile(envExample)
	if err == nil {
		s := string(data)
		if p.SiteURL != "" {
			s = strings.ReplaceAll(s, "APP_URL=http://localhost", "APP_URL="+p.SiteURL)
		}
		s = strings.ReplaceAll(s, "DB_DATABASE=laravel", "DB_DATABASE="+p.DBName)
		s = strings.ReplaceAll(s, "DB_USERNAME=root", "DB_USERNAME="+p.DBUser)
		s = strings.ReplaceAll(s, "DB_PASSWORD=", "DB_PASSWORD="+p.DBPass)
		s = strings.ReplaceAll(s, "DB_HOST=127.0.0.1", "DB_HOST="+p.DBHost)
		_ = os.WriteFile(envFile, []byte(s), 0644)
	}

	// Generate app key as the panel user.
	_ = runAs(j, p.Username, fmt.Sprintf("cd %q && php artisan key:generate --ansi 2>&1", p.DocRoot))

	j.setPhase("setting_permissions")
	if err := chownR(p.DocRoot, p.Username); err != nil {
		j.log("WARN: chown failed: " + err.Error())
	}
	// storage + bootstrap/cache must be writable.
	_ = exec.Command("chmod", "-R", "775",
		filepath.Join(p.DocRoot, "storage"),
		filepath.Join(p.DocRoot, "bootstrap", "cache")).Run()
	return nil
}

func installHTML(j *JobStatus, p RunParams) error {
	j.setPhase("creating")
	html := `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Welcome</title>
<style>body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f9fafb}
.card{text-align:center;padding:2rem;background:#fff;border-radius:1rem;box-shadow:0 1px 3px rgba(0,0,0,.1)}
h1{color:#4f46e5;margin-bottom:.5rem}p{color:#6b7280}</style>
</head>
<body><div class="card"><h1>Hello World</h1><p>Your website is ready. Upload your files to get started.</p></div></body>
</html>`
	indexPath := filepath.Join(p.DocRoot, "index.html")
	if err := os.WriteFile(indexPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("write index.html: %w", err)
	}
	if err := chownR(p.DocRoot, p.Username); err != nil {
		j.log("WARN: chown failed: " + err.Error())
	}
	return nil
}
