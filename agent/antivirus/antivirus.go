package antivirus

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zenspanel/zenspanel/agent/safe"
)

// ScanJob tracks the state of an async scan.
type ScanJob struct {
	Phase     string    `json:"phase"`
	Infected  []string  `json:"infected"`
	Scanned   int       `json:"scanned"`
	Done      bool      `json:"done"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

var (
	jobsMu sync.Mutex
	jobs   = map[string]*ScanJob{}
)

// Status returns the current state of a scan job.
func Status(jobID string) (*ScanJob, error) {
	jobsMu.Lock()
	defer jobsMu.Unlock()
	j, ok := jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("scan job %q not found", jobID)
	}
	return j, nil
}

// Scan starts an async ClamAV scan of scanPath under the user's home jail.
// Returns immediately; poll Status(jobID) for progress (V40).
func Scan(jobID, username, homeBase, scanPath string) error {
	if err := safe.Username(username); err != nil {
		return err
	}

	// Path jail (V40): resolve and verify scanPath is under user home.
	userHome := filepath.Clean(filepath.Join(homeBase, username))
	var target string
	if scanPath == "" || scanPath == "/" {
		target = userHome
	} else {
		// Allow relative paths — resolve against home.
		if !filepath.IsAbs(scanPath) {
			scanPath = filepath.Join(userHome, scanPath)
		}
		cleaned := filepath.Clean(scanPath)
		if !strings.HasPrefix(cleaned+"/", userHome+"/") {
			return fmt.Errorf("scan path %q is outside user home directory", scanPath)
		}
		target = cleaned
	}

	// Verify target exists.
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("scan path not found: %w", err)
	}

	j := &ScanJob{Phase: "starting", StartedAt: time.Now()}
	jobsMu.Lock()
	jobs[jobID] = j
	jobsMu.Unlock()

	go func() {
		runScan(j, username, target)
	}()
	return nil
}

func runScan(j *ScanJob, username, target string) {
	setPhase := func(p string) {
		jobsMu.Lock()
		j.Phase = p
		jobsMu.Unlock()
	}
	fail := func(err error) {
		jobsMu.Lock()
		j.Phase = "failed"
		j.Error = err.Error()
		j.Done = true
		jobsMu.Unlock()
	}

	setPhase("scanning")

	// Run clamscan as the panel user (V40 — not root).
	// --infected: only print infected files
	// --recursive: scan subdirectories
	// --no-summary: suppress the summary line (we parse output ourselves)
	cmdStr := fmt.Sprintf("clamscan --infected --recursive --no-summary %q 2>&1", target)
	cmd := exec.Command("su", "-s", "/bin/sh", "-c", cmdStr, username)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fail(fmt.Errorf("pipe: %w", err))
		return
	}
	if err := cmd.Start(); err != nil {
		// clamscan not installed — return graceful error.
		fail(fmt.Errorf("clamscan not found — install clamav: %w", err))
		return
	}

	var infected []string
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Text()
		// clamscan output: "/path/to/file: Virus.Name FOUND"
		if strings.HasSuffix(line, "FOUND") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 1 {
				infected = append(infected, strings.TrimSpace(parts[0]))
			}
		}
		jobsMu.Lock()
		j.Scanned++
		jobsMu.Unlock()
	}

	// clamscan exits 0 = clean, 1 = infected, 2 = error.
	// We treat 0 and 1 as success (scan completed); 2 as error.
	if err := cmd.Wait(); err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			// Exit 1 = infected files found — not an error.
		} else if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 2 {
			fail(fmt.Errorf("clamscan error: %w", err))
			return
		}
	}

	jobsMu.Lock()
	j.Infected = infected
	if j.Infected == nil {
		j.Infected = []string{}
	}
	j.Phase = "done"
	j.Done = true
	jobsMu.Unlock()
}

// ClamAVRunning returns true if the clamd daemon is active.
func ClamAVRunning() bool {
	out, err := exec.Command("systemctl", "is-active", "clamav-daemon").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}

// Alert is emitted by the realtime watcher when a threat is detected.
type Alert struct {
	Path       string    `json:"path"`
	Threat     string    `json:"threat"`
	DetectedAt time.Time `json:"detected_at"`
}

// AlertCallback is called by WatchStart for each detected threat.
type AlertCallback func(alert Alert)

// watchSessions tracks active inotifywait sessions by watchID.
var (
	watchMu       sync.Mutex
	watchSessions = map[string]*exec.Cmd{}
)

// WatchStart launches an inotifywait watcher on the user's home directory.
// For every new or modified file, it runs clamscan on that file. If a
// threat is found, cb is called with the alert details (V40, V46).
// Returns a watchID that can be passed to WatchStop.
func WatchStart(username, homeBase string, cb AlertCallback) (string, error) {
	if err := safe.Username(username); err != nil {
		return "", err
	}
	userHome := filepath.Join(homeBase, username)
	if _, err := os.Stat(userHome); err != nil {
		return "", fmt.Errorf("user home not found: %w", err)
	}

	watchID := fmt.Sprintf("%s-%d", username, time.Now().UnixNano())

	// inotifywait monitors CREATE and CLOSE_WRITE events recursively.
	// We use -m (monitor) so it runs indefinitely until killed.
	cmd := exec.Command("inotifywait",
		"-m", "-r", "-e", "create,close_write",
		"--format", "%w%f",
		userHome,
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("inotifywait pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("inotifywait start: %w", err)
	}

	watchMu.Lock()
	watchSessions[watchID] = cmd
	watchMu.Unlock()

	go func() {
		defer func() {
			watchMu.Lock()
			delete(watchSessions, watchID)
			watchMu.Unlock()
		}()
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			filePath := strings.TrimSpace(sc.Text())
			if filePath == "" {
				continue
			}
			// Path jail (V40): skip files outside user home.
			if !strings.HasPrefix(filePath, userHome+"/") && filePath != userHome {
				continue
			}
			// Scan the single file with clamscan.
			out, err := exec.Command("clamscan", "--no-summary", filePath).Output()
			if err == nil {
				continue // exit 0 = clean
			}
			// exit 1 = infected; parse the output for threat name.
			for _, line := range strings.Split(string(out), "\n") {
				if strings.HasSuffix(strings.TrimSpace(line), "FOUND") {
					parts := strings.SplitN(line, ":", 2)
					threat := "unknown"
					if len(parts) == 2 {
						threat = strings.TrimSuffix(strings.TrimSpace(parts[1]), " FOUND")
					}
					cb(Alert{
						Path:       filePath,
						Threat:     threat,
						DetectedAt: time.Now(),
					})
					break
				}
			}
		}
	}()

	return watchID, nil
}

// WatchStop terminates the inotifywait watcher for the given watchID.
func WatchStop(watchID string) error {
	watchMu.Lock()
	cmd, ok := watchSessions[watchID]
	if ok {
		delete(watchSessions, watchID)
	}
	watchMu.Unlock()
	if !ok {
		return nil // already stopped — idempotent
	}
	if cmd.Process != nil {
		return cmd.Process.Kill()
	}
	return nil
}

// pendingAlerts stores alerts per username until the API polls them.
var (
	alertsMu      sync.Mutex
	pendingAlerts = map[string][]Alert{}
)

// StoreAlert queues an alert for the given username.
func StoreAlert(username string, a Alert) {
	alertsMu.Lock()
	pendingAlerts[username] = append(pendingAlerts[username], a)
	alertsMu.Unlock()
}

// DrainAlerts returns and clears all pending alerts for the username.
func DrainAlerts(username string) []Alert {
	alertsMu.Lock()
	alerts := pendingAlerts[username]
	delete(pendingAlerts, username)
	alertsMu.Unlock()
	if alerts == nil {
		return []Alert{}
	}
	return alerts
}
