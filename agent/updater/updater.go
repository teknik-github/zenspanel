package updater

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// UpdateInfo is what update.check returns. behind_by counts the number of
// commits between the working tree and origin/main; 0 means up to date.
// changelog is a small excerpt of CHANGELOG.md from the remote so the
// admin can see what's about to ship before clicking Apply.
//
// DownloadURL/ReleaseTag are populated when GitHub Releases has a tagged
// build matching origin/main. When set, the UI should prefer the
// download path over the build-from-source path because the latter
// peaks at ~1.5 GB RAM and OOMs on 1-2 GB VPS hosts.
type UpdateInfo struct {
	CurrentSHA    string `json:"current_sha"`
	LatestSHA     string `json:"latest_sha"`
	BehindBy      int    `json:"behind_by"`
	Changelog     string `json:"changelog"`
	CurrentBranch string `json:"current_branch"`
	DownloadURL   string `json:"download_url"`
	ReleaseTag    string `json:"release_tag"`
}

// UpdateStatus is what update.status returns while a Run is in progress
// or after one finishes. Phase walks through "idle" → various build/
// download steps → "done" or "failed". Log is a rolling tail (last 100
// lines) of the underlying commands' combined stdout/stderr so the UI
// can show real progress without holding a streaming connection.
type UpdateStatus struct {
	Phase string   `json:"phase"`
	Log   []string `json:"log"`
	Done  bool     `json:"done"`
	Error string   `json:"error,omitempty"`
}

const (
	maxLogLines = 100

	// githubReleasesAPI returns the latest published release for the
	// upstream repo. The agent talks to api.github.com directly with
	// no auth; only public release metadata is read so anonymous
	// rate limits (60/hour/IP) are plenty for a Check button.
	githubReleasesAPI = "https://api.github.com/repos/teknik-github/zenspanel/releases/latest"
)

var (
	stateMu sync.Mutex
	state   = UpdateStatus{Phase: "idle", Log: []string{}}
	running bool
)

func getStatus() UpdateStatus {
	stateMu.Lock()
	defer stateMu.Unlock()
	logCopy := append([]string(nil), state.Log...)
	return UpdateStatus{Phase: state.Phase, Log: logCopy, Done: state.Done, Error: state.Error}
}

// Status is the public read-only view of the current update state.
func Status() UpdateStatus { return getStatus() }

func setPhase(phase string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	state.Phase = phase
	state.Log = append(state.Log, "==> "+phase)
	if len(state.Log) > maxLogLines {
		state.Log = state.Log[len(state.Log)-maxLogLines:]
	}
}

func appendLog(line string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	state.Log = append(state.Log, line)
	if len(state.Log) > maxLogLines {
		state.Log = state.Log[len(state.Log)-maxLogLines:]
	}
}

func finish(err error) {
	stateMu.Lock()
	defer stateMu.Unlock()
	state.Done = true
	if err != nil {
		state.Phase = "failed"
		state.Error = err.Error()
		state.Log = append(state.Log, "ERROR: "+err.Error())
	} else {
		state.Phase = "done"
		state.Error = ""
	}
	if len(state.Log) > maxLogLines {
		state.Log = state.Log[len(state.Log)-maxLogLines:]
	}
	running = false
}

func resetLocked() {
	state.Phase = "starting"
	state.Log = []string{}
	state.Done = false
	state.Error = ""
}

// ghAsset is the slim subset of GitHub's release JSON we care about.
// We only read the tarball download URL and the tag name; everything
// else is ignored.
type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

// fetchLatestRelease asks api.github.com for the most recent published
// release of the upstream repo. We pick the .tar.gz asset whose name
// starts with "zenspanel-" so renames or extra assets in the same
// release won't trip us up. Errors here are non-fatal — the UI just
// won't show a download link and falls back to build-from-source.
func fetchLatestRelease(ctx context.Context) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", githubReleasesAPI, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "zenspanel-updater")

	cli := &http.Client{Timeout: 10 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("github releases: HTTP %d", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", "", err
	}
	for _, a := range rel.Assets {
		if strings.HasPrefix(a.Name, "zenspanel-") && strings.HasSuffix(a.Name, ".tar.gz") {
			return rel.TagName, a.BrowserDownloadURL, nil
		}
	}
	return rel.TagName, "", fmt.Errorf("no zenspanel-*.tar.gz asset in release")
}

// Check performs a `git fetch` and returns commit information about how
// far the working tree is behind origin/main. Read-only — only refs are
// updated. Also asks GitHub for the latest release so the UI can offer
// the download path when available.
func Check(srcDir string) (UpdateInfo, error) {
	if _, err := os.Stat(filepath.Join(srcDir, ".git")); err != nil {
		return UpdateInfo{}, fmt.Errorf("not a git repo: %s", srcDir)
	}
	if out, err := runIn(srcDir, "git", "fetch", "origin", "main"); err != nil {
		return UpdateInfo{}, fmt.Errorf("git fetch: %w: %s", err, out)
	}
	cur, err := runIn(srcDir, "git", "rev-parse", "HEAD")
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("rev-parse HEAD: %w", err)
	}
	latest, err := runIn(srcDir, "git", "rev-parse", "origin/main")
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("rev-parse origin/main: %w", err)
	}
	count, err := runIn(srcDir, "git", "rev-list", "--count", "HEAD..origin/main")
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("rev-list count: %w", err)
	}
	behind, _ := strconv.Atoi(strings.TrimSpace(count))

	changelog := ""
	if out, err := runIn(srcDir, "git", "show", "origin/main:CHANGELOG.md"); err == nil {
		changelog = clipLines(out, 60)
	}

	branch := "main"
	if out, err := runIn(srcDir, "git", "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		b := strings.TrimSpace(out)
		if b != "" && b != "HEAD" {
			branch = b
		}
	}

	// Best-effort fetch of the latest release. Failure to reach GitHub
	// (no internet, rate limited, repo private) is non-fatal — we just
	// leave DownloadURL empty and the UI falls back to build-from-source.
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	tag, dlURL, _ := fetchLatestRelease(ctx)

	return UpdateInfo{
		CurrentSHA:    strings.TrimSpace(cur),
		LatestSHA:     strings.TrimSpace(latest),
		BehindBy:      behind,
		Changelog:     changelog,
		CurrentBranch: branch,
		DownloadURL:   dlURL,
		ReleaseTag:    tag,
	}, nil
}

// Run kicks off the update in a goroutine and returns immediately. If a
// run is already in progress it returns a sentinel error — callers
// should poll Status to drive the UI rather than treating the second
// call as an error.
//
// downloadURL is the optional pre-built tarball URL (from update.check).
// When non-empty we take the lightweight download path; when empty we
// fall back to build-from-source which works but OOMs small VPS hosts.
func Run(srcDir, binDir, frontendDir, downloadURL string) error {
	stateMu.Lock()
	if running {
		stateMu.Unlock()
		return fmt.Errorf("update already running")
	}
	running = true
	resetLocked()
	stateMu.Unlock()

	if downloadURL != "" {
		go runDownloadUpdate(srcDir, binDir, frontendDir, downloadURL)
	} else {
		go runBuildUpdate(srcDir, binDir, frontendDir)
	}
	return nil
}

// runDownloadUpdate is the lightweight path: pull the pre-built tarball
// from GitHub Releases, extract, deploy, restart. Memory footprint stays
// under 100 MB even on the smallest VPS.
func runDownloadUpdate(srcDir, binDir, frontendDir, downloadURL string) {
	tmpDir, err := os.MkdirTemp("", "zp-update-*")
	if err != nil {
		finish(fmt.Errorf("mktemp: %w", err))
		return
	}
	defer os.RemoveAll(tmpDir)
	tarPath := filepath.Join(tmpDir, "zenspanel.tar.gz")

	steps := []struct {
		phase string
		fn    func() error
	}{
		{"downloading", func() error {
			return runStreaming("", "curl", "-fsSL", "-o", tarPath, downloadURL)
		}},
		{"extracting", func() error {
			return runStreaming("", "tar", "-xzf", tarPath, "-C", tmpDir)
		}},
		{"stopping_services", func() error {
			// Stop services before replacing binaries — Linux refuses to
			// overwrite a running executable ("Text file busy"). We stop
			// both here; the restart step below brings them back up.
			_ = runStreaming("", "systemctl", "stop", "zenspanel-api")
			_ = runStreaming("", "systemctl", "stop", "zenspanel-agent")
			return nil
		}},
		{"deploying_binaries", func() error {
			extracted := filepath.Join(tmpDir, "zenspanel")
			for _, name := range []string{"zenspanel-api", "zenspanel-agent", "zenspanel-cli"} {
				src := filepath.Join(extracted, "bin", name)
				dst := filepath.Join(binDir, name)
				if _, err := os.Stat(src); err != nil {
					continue
				}
				if err := runStreaming("", "install", "-m", "0755", src, dst); err != nil {
					return err
				}
			}
			return nil
		}},
		{"deploying_frontend", func() error {
			extracted := filepath.Join(tmpDir, "zenspanel", "frontend")
			for _, app := range []string{"admin", "user"} {
				src := filepath.Join(extracted, app)
				dst := filepath.Join(frontendDir, app)
				if _, err := os.Stat(src); err != nil {
					continue
				}
				if err := os.RemoveAll(dst); err != nil {
					return fmt.Errorf("remove old %s: %w", app, err)
				}
				if err := runStreaming("", "cp", "-r", src, dst); err != nil {
					return err
				}
				_ = runStreaming("", "chmod", "-R", "a+rX", dst)
			}
			return runStreaming("", "systemctl", "reload", "nginx")
		}},
		{"setup_dependencies", func() error {
			setupScript := filepath.Join(srcDir, "scripts", "setup.sh")
			if _, err := os.Stat(setupScript); err != nil {
				appendLog("WARN: setup.sh not found, skipping dependency setup")
				return nil
			}
			if err := runStreaming("", "bash", setupScript); err != nil {
				// Non-fatal — binaries are already deployed; a missing
				// optional dependency shouldn't block the restart.
				appendLog("WARN: setup.sh had errors (non-fatal): " + err.Error())
			}
			return nil
		}},
		{"pulling_source", func() error {
			// Keep the source tree current too so the next Check sees
			// the right HEAD. Failure here is non-fatal — the binaries
			// already match the new release; only the in-tree CHANGELOG
			// and source files would be stale.
			if err := runStreaming(srcDir, "git", "pull", "origin", "main"); err != nil {
				appendLog("WARN: git pull failed (non-fatal): " + err.Error())
			}
			return nil
		}},
		{"restarting", func() error {
			// API only — restarting the agent kills this goroutine.
			// Agent updates take effect on the next manual restart.
			return runStreaming("", "systemctl", "restart", "zenspanel-api")
		}},
	}

	for _, s := range steps {
		setPhase(s.phase)
		if err := s.fn(); err != nil {
			finish(err)
			return
		}
	}
	finish(nil)
}

// runBuildUpdate is the legacy build-from-source path. Kept as a
// fallback for environments without internet access to GitHub Releases
// (air-gapped installs, GitHub down) or for builds from untagged
// commits where no release artifact exists yet.
func runBuildUpdate(srcDir, binDir, frontendDir string) {
	steps := []struct {
		phase string
		fn    func() error
	}{
		{"pulling", func() error {
			return runStreaming(srcDir, "git", "pull", "origin", "main")
		}},
		{"stopping_services", func() error {
			_ = runStreaming("", "systemctl", "stop", "zenspanel-api")
			_ = runStreaming("", "systemctl", "stop", "zenspanel-agent")
			return nil
		}},
		{"building_api", func() error {
			return runStreaming(srcDir, "go", "build", "-o", filepath.Join(binDir, "zenspanel-api"), "./cmd/api")
		}},
		{"building_agent", func() error {
			return runStreaming(srcDir, "go", "build", "-o", filepath.Join(binDir, "zenspanel-agent"), "./cmd/agent")
		}},
		{"building_cli", func() error {
			return runStreaming(srcDir, "go", "build", "-o", filepath.Join(binDir, "zenspanel-cli"), "./cmd/cli")
		}},
		{"building_frontend", func() error {
			return runStreaming(filepath.Join(srcDir, "frontend"), "pnpm", "-r", "build")
		}},
		{"deploying_frontend", func() error {
			for _, app := range []string{"admin", "user"} {
				src := filepath.Join(srcDir, "frontend", "apps", app, "dist")
				dst := filepath.Join(frontendDir, app)
				if err := os.RemoveAll(dst); err != nil {
					return fmt.Errorf("remove old %s: %w", app, err)
				}
				if err := runStreaming("", "cp", "-r", src, dst); err != nil {
					return err
				}
				_ = runStreaming("", "chmod", "-R", "a+rX", dst)
			}
			return runStreaming("", "systemctl", "reload", "nginx")
		}},
		{"setup_dependencies", func() error {
			setupScript := filepath.Join(srcDir, "scripts", "setup.sh")
			if _, err := os.Stat(setupScript); err != nil {
				appendLog("WARN: setup.sh not found, skipping dependency setup")
				return nil
			}
			if err := runStreaming("", "bash", setupScript); err != nil {
				appendLog("WARN: setup.sh had errors (non-fatal): " + err.Error())
			}
			return nil
		}},
		{"restarting", func() error {
			return runStreaming("", "systemctl", "restart", "zenspanel-api")
		}},
	}

	for _, s := range steps {
		setPhase(s.phase)
		if err := s.fn(); err != nil {
			finish(err)
			return
		}
	}
	finish(nil)
}

const extendedPATH = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/snap/bin:/root/.local/share/pnpm:/root/.npm-global/bin"

func withPathEnv() []string {
	env := os.Environ()
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, "PATH="+extendedPATH)
	return out
}

func runIn(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = withPathEnv()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runStreaming(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = withPathEnv()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go scanInto(stdout, &wg)
	go scanInto(stderr, &wg)
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func scanInto(r io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		appendLog(sc.Text())
	}
}

func clipLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n") + "\n…"
}
