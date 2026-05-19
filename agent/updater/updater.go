package updater

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// UpdateInfo is what update.check returns. behind_by counts the number of
// commits between the working tree and origin/main; 0 means up to date.
// changelog is a small excerpt of CHANGELOG.md from the remote so the
// admin can see what's about to ship before clicking Apply.
type UpdateInfo struct {
	CurrentSHA string `json:"current_sha"`
	LatestSHA  string `json:"latest_sha"`
	BehindBy   int    `json:"behind_by"`
	Changelog  string `json:"changelog"`
}

// UpdateStatus is what update.status returns while a Run is in progress
// or after one finishes. Phase walks through "idle" → "pulling" → ... →
// "done" or "failed". Log is a rolling tail (last 100 lines) of the
// underlying commands' combined stdout/stderr so the UI can show real
// progress without holding a streaming connection.
type UpdateStatus struct {
	Phase string   `json:"phase"`
	Log   []string `json:"log"`
	Done  bool     `json:"done"`
	Error string   `json:"error,omitempty"`
}

const maxLogLines = 100

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

// reset clears state for a fresh run. Caller holds stateMu.
func resetLocked() {
	state.Phase = "starting"
	state.Log = []string{}
	state.Done = false
	state.Error = ""
}

// Check performs a `git fetch` and returns commit information about how
// far the working tree is behind origin/main. It is read-only as far as
// the working tree is concerned — only refs are updated.
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

	// Pull a small slice of CHANGELOG.md from the remote — show the last
	// 60 lines so the admin sees roughly the last few entries.
	changelog := ""
	if out, err := runIn(srcDir, "git", "show", "origin/main:CHANGELOG.md"); err == nil {
		changelog = clipLines(out, 60)
	}
	return UpdateInfo{
		CurrentSHA: strings.TrimSpace(cur),
		LatestSHA:  strings.TrimSpace(latest),
		BehindBy:   behind,
		Changelog:  changelog,
	}, nil
}

// Run kicks off the update in a goroutine and returns immediately. If a
// run is already in progress it returns a sentinel error — callers
// should poll Status to drive the UI rather than treating the second
// call as an error.
func Run(srcDir, binDir, frontendDir string) error {
	stateMu.Lock()
	if running {
		stateMu.Unlock()
		return fmt.Errorf("update already running")
	}
	running = true
	resetLocked()
	stateMu.Unlock()

	go runUpdate(srcDir, binDir, frontendDir)
	return nil
}

func runUpdate(srcDir, binDir, frontendDir string) {
	steps := []struct {
		phase string
		fn    func() error
	}{
		{"pulling", func() error {
			return runStreaming(srcDir, "git", "pull", "origin", "main")
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
		{"restarting", func() error {
			// We deliberately do NOT restart zenspanel-agent here —
			// systemctl restart on the agent kills this goroutine
			// mid-step, leaving the panel UI stuck on "restarting"
			// forever. The new agent binary is on disk and will be
			// picked up on the next manual restart. The API is the
			// service the user-visible feature changes live in, so
			// restarting just the API is enough for 99% of updates.
			//
			// If a release ships agent-only changes that need a
			// restart, the operator can run `systemctl restart
			// zenspanel-agent` after the update completes (or use
			// zenspanel-cli's Restart Services screen).
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

// runIn runs a command in dir and returns its combined output. Used for
// short read-only git commands during Check.
func runIn(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// runStreaming runs a long command and pumps each line of stdout/stderr
// into the shared log so the UI can show progress as it happens.
// Empty dir means the current working directory; we use that for the
// system commands (cp, systemctl) where the source tree isn't relevant.
func runStreaming(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
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
