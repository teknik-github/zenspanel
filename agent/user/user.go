package user

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/zenspanel/zenspanel/agent/safe"
)

const (
	minPanelUID = 10000
	maxPanelUID = 60000
)

// nextFreeUID scans /etc/passwd for the lowest unused UID in the panel
// range. We do this in the agent rather than the API because the kernel +
// /etc/passwd are the authoritative source — the panel DB can drift if a
// user is added or removed via useradd/userdel directly, or if a previous
// panel row was deleted but the Linux account survived.
func nextFreeUID(preferred int) (int, error) {
	taken := map[int]bool{}
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return 0, fmt.Errorf("read /etc/passwd: %w", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Split(sc.Text(), ":")
		if len(fields) < 3 {
			continue
		}
		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		taken[uid] = true
	}
	if err := sc.Err(); err != nil {
		return 0, fmt.Errorf("scan /etc/passwd: %w", err)
	}

	// Honor the caller's preferred UID when it's free and inside the panel
	// range — keeps the panel DB row aligned with reality.
	if preferred >= minPanelUID && preferred <= maxPanelUID && !taken[preferred] {
		return preferred, nil
	}
	for uid := minPanelUID; uid <= maxPanelUID; uid++ {
		if !taken[uid] {
			return uid, nil
		}
	}
	return 0, fmt.Errorf("no free UID in range %d-%d", minPanelUID, maxPanelUID)
}

// Create provisions a Linux user. The caller proposes a UID; we accept it
// if it's free, otherwise we pick the next free one in the panel range and
// return it via the result tuple so the API can update its DB row.
func Create(username string, uid int, homeBase string) (int, error) {
	if err := safe.Username(username); err != nil {
		return 0, err
	}
	chosen, err := nextFreeUID(uid)
	if err != nil {
		return 0, err
	}
	homeDir := homeBase + "/" + username
	cmd := exec.Command("useradd",
		"-u", strconv.Itoa(chosen),
		"-m",
		"-d", homeDir,
		"-s", "/bin/rbash",
		username,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return 0, fmt.Errorf("useradd failed: %w: %s", err, out)
	}
	// Make the home dir traversable by nginx (running as www-data) and
	// PHP-FPM (running as the user). Default umask gives 0700, which
	// blocks www-data from stat'ing files inside — every static-file
	// request would fall through to PHP-FPM and 404 with "File not
	// found." Mode 0711 lets others traverse without listing, so
	// www-data can stat known paths but not enumerate $HOME.
	if err := os.Chmod(homeDir, 0711); err != nil {
		return 0, fmt.Errorf("chmod home: %w", err)
	}
	return chosen, nil
}

func Delete(username string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	cmd := exec.Command("userdel", "-r", username)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("userdel failed: %w: %s", err, out)
	}
	return nil
}
