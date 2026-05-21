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
	RequiresDB  bool   `json:"requires_db"`
	DownloadURL string `json:"download_url"`
}

// Catalog is the list of supported apps. Versions are pinned so installs
// are reproducible; bump them when a new stable release ships (V52).
var Catalog = []App{
	{
		ID:          "wordpress",
		Name:        "WordPress",
		Version:     "6.5.3",
		Description: "The world's most popular CMS. Requires a MySQL database.",
		RequiresDB:  true,
		DownloadURL: "https://wordpress.org/wordpress-6.5.3.tar.gz",
	},
	{
		ID:          "joomla",
		Name:        "Joomla",
		Version:     "5.1.2",
		Description: "Flexible CMS for websites and web applications. Requires MySQL.",
		RequiresDB:  true,
		DownloadURL: "https://downloads.joomla.org/cms/joomla5/5-1-2/Joomla_5.1.2-Stable-Full_Package.tar.gz",
	},
	{
		ID:          "drupal",
		Name:        "Drupal",
		Version:     "10.3.0",
		Description: "Enterprise-grade CMS. Requires MySQL.",
		RequiresDB:  true,
		DownloadURL: "https://ftp.drupal.org/files/projects/drupal-10.3.0.tar.gz",
	},
	{
		ID:          "prestashop",
		Name:        "PrestaShop",
		Version:     "8.1.7",
		Description: "Open-source e-commerce platform. Requires MySQL.",
		RequiresDB:  true,
		DownloadURL: "https://github.com/PrestaShop/PrestaShop/releases/download/8.1.7/prestashop_8.1.7.zip",
	},
	{
		ID:          "codeigniter",
		Name:        "CodeIgniter",
		Version:     "4.5.1",
		Description: "Lightweight PHP framework. No database required.",
		RequiresDB:  false,
		DownloadURL: "https://github.com/CodeIgniter/CodeIgniter4/releases/download/v4.5.1/framework-4.5.1.zip",
	},
	{
		ID:          "laravel",
		Name:        "Laravel",
		Version:     "11",
		Description: "PHP web application framework. Installs via Composer.",
		RequiresDB:  true,
		DownloadURL: "",
	},
	{
		ID:          "html",
		Name:        "Plain HTML",
		Version:     "—",
		Description: "A simple index.html starter page. No database required.",
		RequiresDB:  false,
		DownloadURL: "",
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
	case "joomla":
		return installJoomla(j, p)
	case "drupal":
		return installDrupal(j, p)
	case "prestashop":
		return installPrestaShop(j, p)
	case "codeigniter":
		return installCodeIgniter(j, p)
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
	// Use exec.Command arg array — no shell string interpolation (V43).
	cmd := exec.Command("su", "-s", "/bin/sh", "-c",
		"composer create-project laravel/laravel \"$1\" --prefer-dist --no-interaction 2>&1",
		"--", tmpDir, p.Username)
	// Note: su syntax: su -s /bin/sh -c '<script>' -- <arg0> <username>
	// The shell receives $1=tmpDir; username is the su target, not a shell arg.
	// Rewrite to pass tmpDir via env to avoid any interpolation risk.
	cmd = exec.Command("su", "-s", "/bin/sh", p.Username, "-c",
		"composer create-project laravel/laravel \"${LARAVEL_DEST}\" --prefer-dist --no-interaction 2>&1")
	cmd.Env = append(os.Environ(), "LARAVEL_DEST="+tmpDir)
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

	// Generate app key as the panel user — use env var to pass docroot
	// so no shell metacharacters in p.DocRoot can escape (V43).
	artisanCmd := exec.Command("su", "-s", "/bin/sh", p.Username, "-c",
		"cd \"${APP_DIR}\" && php artisan key:generate --ansi 2>&1")
	artisanCmd.Env = append(os.Environ(), "APP_DIR="+p.DocRoot)
	if out, err := artisanCmd.CombinedOutput(); err != nil {
		j.log("WARN: artisan key:generate failed: " + string(out))
	} else {
		j.log(string(out))
	}

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

// downloadAndExtract downloads a tarball or zip to a temp dir and returns
// the path. Caller is responsible for cleanup via defer os.RemoveAll.
func downloadAndExtract(j *JobStatus, jobID, url, ext string) (string, error) {
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("zp-install-%s%s", jobID, ext))
	defer os.Remove(tmpFile)

	j.log("Downloading " + url)
	if out, err := exec.Command("curl", "-fsSL", "-o", tmpFile, url).CombinedOutput(); err != nil {
		return "", fmt.Errorf("download: %w: %s", err, out)
	}

	tmpDir := filepath.Join(os.TempDir(), "zp-extract-"+jobID)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	j.log("Extracting...")
	var cmd *exec.Cmd
	if ext == ".zip" {
		cmd = exec.Command("unzip", "-q", tmpFile, "-d", tmpDir)
	} else {
		cmd = exec.Command("tar", "-xzf", tmpFile, "-C", tmpDir)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("extract: %w: %s", err, out)
	}
	return tmpDir, nil
}

// deployDir copies src/* to docroot and chowns to username (V33).
func deployDir(j *JobStatus, src, docroot, username string) error {
	j.setPhase("deploying")
	if out, err := exec.Command("rsync", "-a", "--delete", src+"/", docroot+"/").CombinedOutput(); err != nil {
		if out2, err2 := exec.Command("cp", "-r", src+"/.", docroot+"/").CombinedOutput(); err2 != nil {
			return fmt.Errorf("deploy: %w: %s %s", err, out, out2)
		}
	}
	j.setPhase("setting_permissions")
	if err := chownR(docroot, username); err != nil {
		j.log("WARN: chown failed: " + err.Error())
	}
	return nil
}

func installJoomla(j *JobStatus, p RunParams) error {
	j.setPhase("downloading")
	tmpDir, err := downloadAndExtract(j, p.JobID, "https://downloads.joomla.org/cms/joomla5/5-1-2/Joomla_5.1.2-Stable-Full_Package.tar.gz", ".tar.gz")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Joomla extracts directly into tmpDir (no subdirectory).
	if err := deployDir(j, tmpDir, p.DocRoot, p.Username); err != nil {
		return err
	}

	// Write configuration.php with DB settings.
	j.setPhase("configuring")
	configPath := filepath.Join(p.DocRoot, "configuration.php")
	config := fmt.Sprintf(`<?php
class JConfig {
	public $dbtype = 'mysqli';
	public $host = '%s';
	public $user = '%s';
	public $password = '%s';
	public $db = '%s';
	public $dbprefix = 'jos_';
	public $live_site = '';
	public $secret = 'changeme';
	public $sef = '1';
	public $sef_rewrite = '1';
}`, p.DBHost, p.DBUser, p.DBPass, p.DBName)
	_ = os.WriteFile(configPath, []byte(config), 0644)
	_ = chownR(p.DocRoot, p.Username)
	return nil
}

func installDrupal(j *JobStatus, p RunParams) error {
	j.setPhase("downloading")
	tmpDir, err := downloadAndExtract(j, p.JobID, "https://ftp.drupal.org/files/projects/drupal-10.3.0.tar.gz", ".tar.gz")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Drupal extracts to tmpDir/drupal-10.3.0/
	entries, _ := os.ReadDir(tmpDir)
	src := tmpDir
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "drupal") {
			src = filepath.Join(tmpDir, e.Name())
			break
		}
	}

	if err := deployDir(j, src, p.DocRoot, p.Username); err != nil {
		return err
	}

	// Create sites/default/settings.php from default.settings.php.
	j.setPhase("configuring")
	defaultSettings := filepath.Join(p.DocRoot, "sites", "default", "default.settings.php")
	settings := filepath.Join(p.DocRoot, "sites", "default", "settings.php")
	if data, err := os.ReadFile(defaultSettings); err == nil {
		s := string(data)
		dbURL := fmt.Sprintf("mysql://%s:%s@%s/%s", p.DBUser, p.DBPass, p.DBHost, p.DBName)
		s += fmt.Sprintf("\n$databases['default']['default'] = \\Drupal\\Core\\Database\\Database::convertDbUrlToConnectionInfo('%s', DRUPAL_ROOT);\n", dbURL)
		_ = os.WriteFile(settings, []byte(s), 0644)
	}
	_ = os.MkdirAll(filepath.Join(p.DocRoot, "sites", "default", "files"), 0755)
	_ = chownR(p.DocRoot, p.Username)
	return nil
}

func installPrestaShop(j *JobStatus, p RunParams) error {
	j.setPhase("downloading")
	tmpDir, err := downloadAndExtract(j, p.JobID, "https://github.com/PrestaShop/PrestaShop/releases/download/8.1.7/prestashop_8.1.7.zip", ".zip")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// PrestaShop zip contains prestashop.zip inside — extract that too.
	innerZip := filepath.Join(tmpDir, "prestashop.zip")
	innerDir := filepath.Join(tmpDir, "prestashop-inner")
	if _, err := os.Stat(innerZip); err == nil {
		_ = os.MkdirAll(innerDir, 0755)
		if out, err := exec.Command("unzip", "-q", innerZip, "-d", innerDir).CombinedOutput(); err != nil {
			j.log("WARN: inner unzip: " + string(out))
		}
		if err := deployDir(j, innerDir, p.DocRoot, p.Username); err != nil {
			return err
		}
	} else {
		if err := deployDir(j, tmpDir, p.DocRoot, p.Username); err != nil {
			return err
		}
	}
	j.log("PrestaShop deployed. Complete installation via web browser.")
	return nil
}

func installCodeIgniter(j *JobStatus, p RunParams) error {
	j.setPhase("downloading")
	tmpDir, err := downloadAndExtract(j, p.JobID, "https://github.com/CodeIgniter/CodeIgniter4/releases/download/v4.5.1/framework-4.5.1.zip", ".zip")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// CodeIgniter zip extracts to CodeIgniter4-framework-4.5.1/
	entries, _ := os.ReadDir(tmpDir)
	src := tmpDir
	for _, e := range entries {
		if e.IsDir() {
			src = filepath.Join(tmpDir, e.Name())
			break
		}
	}

	if err := deployDir(j, src, p.DocRoot, p.Username); err != nil {
		return err
	}

	// Write .env with DB settings.
	j.setPhase("configuring")
	envPath := filepath.Join(p.DocRoot, ".env")
	envContent := fmt.Sprintf(`CI_ENVIRONMENT = production
database.default.hostname = %s
database.default.database = %s
database.default.username = %s
database.default.password = %s
database.default.DBDriver = MySQLi
`, p.DBHost, p.DBName, p.DBUser, p.DBPass)
	_ = os.WriteFile(envPath, []byte(envContent), 0644)
	_ = chownR(p.DocRoot, p.Username)
	return nil
}
