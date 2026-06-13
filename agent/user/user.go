package user

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	osuser "os/user"
	"path/filepath"
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
// phpVersion seeds ~/bin/php and ~/bin/composer so the terminal shell
// picks up the user's configured PHP — not the system default.
func Create(username string, uid int, homeBase, phpVersion string) (int, error) {
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
	// Lock .bash_profile and .bashrc so the user cannot override PATH
	// and escape the rbash restriction.
	if err := lockShellProfile(homeDir); err != nil {
		return 0, fmt.Errorf("lock shell profile: %w", err)
	}
	if phpVersion != "" {
		if err := SetupBin(username, homeBase, phpVersion); err != nil {
			return 0, fmt.Errorf("setup bin: %w", err)
		}
	}
	return chosen, nil
}

// SetupBin populates ~/bin with a php symlink and a composer wrapper
// pinned to the user's chosen PHP version. The terminal shell's PATH
// starts with ~/bin (set by useradd's default profile + rbash), so
// `php` and `composer` invocations resolve to the version set here
// regardless of which PHP is the system default.
//
// Idempotent — safe to call on every php_version change. Removes any
// existing php symlink before recreating; rewrites the composer
// wrapper unconditionally.
func SetupBin(username, homeBase, phpVersion string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	if err := safe.PHPVersion(phpVersion); err != nil {
		return err
	}
	binDir := filepath.Join(homeBase, username, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("mkdir bin: %w", err)
	}

	// Resolve uid/gid so the dir + wrapper are owned by the panel user.
	// Without this, MkdirAll runs as root and the user can't drop new
	// binaries into their own ~/bin later.
	var uid, gid int = -1, -1
	if u, err := osuser.Lookup(username); err == nil {
		uid, _ = strconv.Atoi(u.Uid)
		gid, _ = strconv.Atoi(u.Gid)
		_ = os.Chown(binDir, uid, gid)
	}

	phpBin := fmt.Sprintf("/usr/bin/php%s", phpVersion)
	phpLink := filepath.Join(binDir, "php")
	_ = os.Remove(phpLink)
	if err := os.Symlink(phpBin, phpLink); err != nil {
		return fmt.Errorf("symlink php: %w", err)
	}
	if uid >= 0 {
		_ = os.Lchown(phpLink, uid, gid)
	}

	composerWrapper := fmt.Sprintf("#!/bin/sh\nexec %s /usr/local/bin/composer.phar \"$@\"\n", phpBin)
	composerPath := filepath.Join(binDir, "composer")
	if err := os.WriteFile(composerPath, []byte(composerWrapper), 0755); err != nil {
		return fmt.Errorf("write composer wrapper: %w", err)
	}
	// WriteFile honors umask — re-chmod to ensure executable bit survives.
	if err := os.Chmod(composerPath, 0755); err != nil {
		return fmt.Errorf("chmod composer: %w", err)
	}
	if uid >= 0 {
		_ = os.Chown(composerPath, uid, gid)
	}
	return nil
}

// lockShellProfile writes root-owned, read-only .bash_profile and .bashrc
// that export a locked PATH pointing only at ~/bin. Without this, a panel
// user can edit their own dotfiles and set PATH=/usr/bin to bypass rbash.
func lockShellProfile(homeDir string) error {
	profileContent := "export PATH=$HOME/bin\nexport HOME=" + homeDir + "\n"
	bashrcContent := "export PATH=$HOME/bin\n"

	files := map[string]string{
		filepath.Join(homeDir, ".bash_profile"): profileContent,
		filepath.Join(homeDir, ".bashrc"):        bashrcContent,
	}
	for path, content := range files {
		// Write as root (agent runs as root), then lock permissions.
		if err := os.WriteFile(path, []byte(content), 0444); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		// Ensure root owns the file regardless of any prior state.
		if err := os.Chown(path, 0, 0); err != nil {
			return fmt.Errorf("chown %s: %w", path, err)
		}
		// 0444 — world-readable, nobody-writable (not even root without chmod).
		if err := os.Chmod(path, 0444); err != nil {
			return fmt.Errorf("chmod %s: %w", path, err)
		}
	}
	return nil
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
