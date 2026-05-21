package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/zenspanel/zenspanel/agent/safe"
)

// BackupDomain tars only the domain's docroot into <backupBase>/<username>/.
// V58: scoped to docroot only — no full home, no cross-domain data.
// --no-same-owner and --no-same-permissions prevent setuid attacks in the archive.
func BackupDomain(username, docRoot, backupBase, domainName string) (archivePath string, size int64, err error) {
	if err := safe.Username(username); err != nil {
		return "", 0, err
	}

	dir := filepath.Join(backupBase, username)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", 0, fmt.Errorf("mkdir: %w", err)
	}

	stamp := time.Now().Format("20060102-150405")
	archivePath = filepath.Join(dir, fmt.Sprintf("%s-%s-domain.tar.gz", stamp, domainName))

	cmd := exec.Command("tar", "-czf", archivePath,
		"--no-same-owner", "--no-same-permissions",
		"-C", filepath.Dir(docRoot), filepath.Base(docRoot))
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", 0, fmt.Errorf("tar: %w", err)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		return "", 0, fmt.Errorf("stat: %w", err)
	}
	return archivePath, info.Size(), nil
}
