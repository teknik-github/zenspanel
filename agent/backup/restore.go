package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zenspanel/zenspanel/agent/safe"
)

// pathInside returns nil iff target resolves to a path strictly under root.
// Used to prevent the panel asking the agent to restore from a tar outside
// the user's backup directory (e.g. /etc/shadow.tar.gz). The agent runs as
// root, so this gate is the difference between a restore and a privesc.
func pathInside(root, target string) error {
	r, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("abs root: %w", err)
	}
	t, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("abs target: %w", err)
	}
	if t != r && !strings.HasPrefix(t, r+string(filepath.Separator)) {
		return fmt.Errorf("path %q escapes %q", target, root)
	}
	return nil
}

// lookupUID reads /etc/passwd directly (rather than calling user.Lookup)
// because we already pull /etc/passwd in agent/user for nextFreeUID, and
// avoiding the Cgo path means restoring a deleted-then-recreated user
// works the same way on every distro.
func lookupUID(username string) (int, error) {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return 0, fmt.Errorf("read /etc/passwd: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 3 || fields[0] != username {
			continue
		}
		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			return 0, fmt.Errorf("parse uid for %q: %w", username, err)
		}
		return uid, nil
	}
	return 0, fmt.Errorf("user %q not found", username)
}

// RestoreFiles wipes <homeBase>/<username> and replays the tarball over
// it. The archive is expected to contain <username>/ at the root (which is
// what runBackup writes when type is "files" or "full"). Ownership is
// reset via chown -R so the extracted tree is owned by the panel user
// regardless of who created the tarball or what UIDs were stored in it.
//
// Caller invariant: archivePath has already been validated to live under
// <backupBase>, so we only need to check it's a real file.
func RestoreFiles(username, homeBase, backupBase, archivePath string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	if err := pathInside(backupBase, archivePath); err != nil {
		return err
	}
	if _, err := os.Stat(archivePath); err != nil {
		return fmt.Errorf("archive: %w", err)
	}

	homeDir := filepath.Join(homeBase, username)
	// Refuse to wipe the home base itself if username resolves weirdly —
	// safe.Username already prevents this, but it's cheap defence in depth.
	homeAbs, _ := filepath.Abs(homeDir)
	baseAbs, _ := filepath.Abs(homeBase)
	if homeAbs == baseAbs || !strings.HasPrefix(homeAbs, baseAbs+string(filepath.Separator)) {
		return fmt.Errorf("refusing to wipe home base %q", homeBase)
	}

	if err := os.RemoveAll(homeDir); err != nil {
		return fmt.Errorf("wipe home: %w", err)
	}
	if err := os.MkdirAll(homeBase, 0755); err != nil {
		return fmt.Errorf("recreate home base: %w", err)
	}

	// tar restores <username>/ as a child of homeBase
	cmd := exec.Command("tar", "-xzf", archivePath, "-C", homeBase, username)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar extract: %w", err)
	}

	uid, err := lookupUID(username)
	if err != nil {
		return fmt.Errorf("lookup uid: %w", err)
	}
	owner := fmt.Sprintf("%d:%d", uid, uid)
	chown := exec.Command("chown", "-R", owner, homeDir)
	chown.Stderr = os.Stderr
	if err := chown.Run(); err != nil {
		return fmt.Errorf("chown: %w", err)
	}
	return nil
}

// RestoreDB extracts databases.sql from the archive (full or db archives
// have it at the root) and runs `mysql --one-database <dbName>` against
// it using credentials from mysqlAdminDSN. --one-database is critical —
// without it a SQL file containing `USE other_db;` would let one user's
// restore stomp on another user's schema.
func RestoreDB(dbName, archivePath, mysqlAdminDSN, backupBase string) error {
	if err := safe.DBIdent(dbName); err != nil {
		return err
	}
	if err := pathInside(backupBase, archivePath); err != nil {
		return err
	}
	if _, err := os.Stat(archivePath); err != nil {
		return fmt.Errorf("archive: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "zp-restore-*")
	if err != nil {
		return fmt.Errorf("mktemp: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract only databases.sql; ignore the rest of the archive
	cmd := exec.Command("tar", "-xzf", archivePath, "-C", tmpDir, "databases.sql")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar extract sql: %w", err)
	}

	sqlPath := filepath.Join(tmpDir, "databases.sql")
	sqlFile, err := os.Open(sqlPath)
	if err != nil {
		return fmt.Errorf("open sql: %w", err)
	}
	defer sqlFile.Close()

	host, port, user, pass := parseMySQLDSN(mysqlAdminDSN)
	args := []string{
		"--host=" + host,
		"--port=" + port,
		"--user=" + user,
		"--one-database",
		dbName,
	}
	mysqlCmd := exec.Command("mysql", args...)
	mysqlCmd.Env = append(os.Environ(), "MYSQL_PWD="+pass)
	mysqlCmd.Stdin = sqlFile
	mysqlCmd.Stderr = os.Stderr
	if err := mysqlCmd.Run(); err != nil {
		return fmt.Errorf("mysql import: %w", err)
	}
	return nil
}

// parseMySQLDSN pulls user/pass/host/port out of a Go MySQL driver DSN like
// `root:secret@tcp(127.0.0.1:3306)/?...`. We do this by string slicing
// rather than importing the driver's parser to keep the agent free of the
// SQL driver dependency it doesn't otherwise need.
func parseMySQLDSN(dsn string) (host, port, user, pass string) {
	host, port = "127.0.0.1", "3306"
	at := strings.LastIndex(dsn, "@")
	if at < 0 {
		return
	}
	creds := dsn[:at]
	rest := dsn[at+1:]
	if colon := strings.Index(creds, ":"); colon >= 0 {
		user = creds[:colon]
		pass = creds[colon+1:]
	} else {
		user = creds
	}
	open := strings.Index(rest, "(")
	close := strings.Index(rest, ")")
	if open >= 0 && close > open {
		addr := rest[open+1 : close]
		if c := strings.Index(addr, ":"); c >= 0 {
			host = addr[:c]
			port = addr[c+1:]
		} else {
			host = addr
		}
	}
	return
}
