package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/zenspanel/zenspanel/agent/safe"
)

// Run creates a full, files-only, or db-only backup for a panel user.
// Archives land at <homeBase>/<username>/backups/ so they count toward the
// user's own disk quota rather than a separate backup volume.
//
// kind: "full", "files", or "db"
// dbNames: MySQL database names to dump (unused when kind="files")
// mysqlAdminDSN: Go-driver DSN used to authenticate mysqldump
func Run(username, homeBase, kind string, dbNames []string, mysqlAdminDSN string) (archivePath string, size int64, err error) {
	if err := safe.Username(username); err != nil {
		return "", 0, err
	}

	backupDir := filepath.Join(homeBase, username, "backups")
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return "", 0, fmt.Errorf("mkdir backups: %w", err)
	}

	stamp := time.Now().Format("20060102-150405")
	archivePath = filepath.Join(backupDir, fmt.Sprintf("%s-%s.tar.gz", stamp, kind))

	tmpDir, err := os.MkdirTemp("", "zp-backup-*")
	if err != nil {
		return "", 0, fmt.Errorf("mktemp: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tarArgs := []string{"-czf", archivePath, "--no-same-owner", "--no-same-permissions"}

	if kind == "db" || kind == "full" {
		dumpPath := filepath.Join(tmpDir, "databases.sql")
		f, err := os.Create(dumpPath)
		if err != nil {
			return "", 0, fmt.Errorf("create dump file: %w", err)
		}
		host, port, user, pass := parseMySQLDSN(mysqlAdminDSN)
		for _, db := range dbNames {
			if err := safe.DBIdent(db); err != nil {
				f.Close()
				return "", 0, fmt.Errorf("invalid db name %q: %w", db, err)
			}
			dumpCmd := exec.Command("mysqldump",
				"--host="+host, "--port="+port, "--user="+user,
				"--single-transaction", "--skip-lock-tables", db)
			dumpCmd.Env = append(os.Environ(), "MYSQL_PWD="+pass)
			dumpCmd.Stdout = f
			dumpCmd.Stderr = os.Stderr
			if err := dumpCmd.Run(); err != nil {
				f.Close()
				return "", 0, fmt.Errorf("mysqldump %s: %w", db, err)
			}
		}
		f.Close()
		if len(dbNames) > 0 {
			tarArgs = append(tarArgs, "-C", tmpDir, "databases.sql")
		}
	}

	if kind == "files" || kind == "full" {
		homeDir := filepath.Join(homeBase, username)
		// Exclude the backups subdir to prevent archiving old backups inside the new one.
		tarArgs = append(tarArgs,
			"--exclude="+filepath.Join(homeDir, "backups"),
			"-C", homeBase, username)
	}

	if out, runErr := exec.Command("tar", tarArgs...).CombinedOutput(); runErr != nil {
		return "", 0, fmt.Errorf("tar: %w: %s", runErr, out)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		return "", 0, fmt.Errorf("stat: %w", err)
	}
	return archivePath, info.Size(), nil
}
