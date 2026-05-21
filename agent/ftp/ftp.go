package ftp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zenspanel/zenspanel/agent/safe"
)

const (
	virtualUsersFile = "/etc/vsftpd/virtual_users.txt"
	virtualUsersDB   = "/etc/vsftpd/virtual_users.db"
	userConfigDir    = "/etc/vsftpd/users"
)

// ftpUsernameRe: FTP usernames are panel_username + "_ftp" suffix or
// panel_username + "_ftp_<label>". We accept alphanumeric + underscore,
// 3-64 chars. Validated here because the username ends up in a filename
// and in the PAM DB — no shell metacharacters allowed (V59).
var ftpUsernameRe = regexp.MustCompile(`^[a-z][a-z0-9_]{2,63}$`)

func FTPUsername(u string) error {
	if !ftpUsernameRe.MatchString(u) {
		return fmt.Errorf("agent: invalid ftp username %q (must be lowercase alphanumeric/underscore, 3-64 chars)", u)
	}
	return nil
}

// CreateAccount adds a vsftpd virtual user with the given password and
// restricts them to homeDir. Steps:
//  1. Validate inputs (V59, V60)
//  2. Ensure homeDir exists and is owned by the panel user
//  3. Write per-user vsftpd config (local_root, chroot)
//  4. Add/update entry in virtual_users.txt
//  5. Recompile the PAM DB with db_load
//  6. Reload vsftpd (SIGHUP — no connection drop)
func CreateAccount(ftpUser, password, homeDir, panelUsername string) error {
	if err := FTPUsername(ftpUser); err != nil {
		return err
	}
	if err := safe.Username(panelUsername); err != nil {
		return err
	}
	if len(password) < 8 || len(password) > 128 {
		return fmt.Errorf("ftp password must be 8-128 characters")
	}
	// Validate homeDir is an absolute path with no traversal
	clean := filepath.Clean(homeDir)
	if !filepath.IsAbs(clean) || strings.Contains(clean, "..") {
		return fmt.Errorf("agent: invalid ftp home_dir %q", homeDir)
	}

	// Ensure home dir exists
	if err := os.MkdirAll(clean, 0755); err != nil {
		return fmt.Errorf("mkdir home_dir: %w", err)
	}

	// Write per-user vsftpd config
	if err := os.MkdirAll(userConfigDir, 0755); err != nil {
		return fmt.Errorf("mkdir user config dir: %w", err)
	}
	userConf := fmt.Sprintf("local_root=%s\nwrite_enable=YES\n", clean)
	confPath := filepath.Join(userConfigDir, ftpUser)
	if err := os.WriteFile(confPath, []byte(userConf), 0644); err != nil {
		return fmt.Errorf("write user config: %w", err)
	}

	// Update virtual_users.txt — read existing, remove old entry for this
	// user if present, append new entry.
	if err := upsertVirtualUser(ftpUser, password); err != nil {
		return err
	}

	// Recompile PAM DB
	if err := recompileDB(); err != nil {
		return err
	}

	return reloadVsftpd()
}

// DeleteAccount removes a vsftpd virtual user. Steps:
//  1. Remove from virtual_users.txt
//  2. Recompile PAM DB
//  3. Remove per-user config file
//  4. Reload vsftpd
func DeleteAccount(ftpUser string) error {
	if err := FTPUsername(ftpUser); err != nil {
		return err
	}

	if err := removeVirtualUser(ftpUser); err != nil {
		return err
	}

	if err := recompileDB(); err != nil {
		return err
	}

	// Best-effort config removal
	_ = os.Remove(filepath.Join(userConfigDir, ftpUser))

	return reloadVsftpd()
}

// upsertVirtualUser adds or replaces the entry for ftpUser in the flat
// virtual_users.txt file. Format: alternating lines of username / password.
func upsertVirtualUser(ftpUser, password string) error {
	lines, err := readLines(virtualUsersFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read virtual users: %w", err)
	}

	// Remove existing entry (two consecutive lines: username + password)
	var filtered []string
	for i := 0; i < len(lines); i++ {
		if lines[i] == ftpUser {
			i++ // skip password line too
			continue
		}
		filtered = append(filtered, lines[i])
	}

	// Append new entry
	filtered = append(filtered, ftpUser, password)

	return writeLines(virtualUsersFile, filtered)
}

// removeVirtualUser removes the entry for ftpUser from virtual_users.txt.
func removeVirtualUser(ftpUser string) error {
	lines, err := readLines(virtualUsersFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read virtual users: %w", err)
	}

	var filtered []string
	for i := 0; i < len(lines); i++ {
		if lines[i] == ftpUser {
			i++ // skip password line
			continue
		}
		filtered = append(filtered, lines[i])
	}

	return writeLines(virtualUsersFile, filtered)
}

// recompileDB runs db_load to rebuild the Berkeley DB from the flat text file.
func recompileDB() error {
	// db_load -T -t hash -f <input> <output>
	// -T: allow plain text input, -t hash: hash table format
	cmd := exec.Command("db_load", "-T", "-t", "hash", "-f", virtualUsersFile, virtualUsersDB)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("db_load: %w: %s", err, out)
	}
	// Restrict DB permissions — PAM reads it as root, no need for world-read
	return os.Chmod(virtualUsersDB, 0600)
}

// reloadVsftpd sends SIGHUP to vsftpd so it re-reads config without
// dropping existing connections.
func reloadVsftpd() error {
	cmd := exec.Command("systemctl", "reload-or-restart", "vsftpd")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("reload vsftpd: %w: %s", err, out)
	}
	return nil
}

func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, l := range strings.Split(string(data), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines, nil
}

func writeLines(path string, lines []string) error {
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0600)
}
