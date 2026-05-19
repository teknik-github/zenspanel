package phpfpm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/zenspanel/zenspanel/agent/safe"
)

const poolTmpl = `[zenspanel-{{.Username}}]
user = {{.Username}}
group = {{.Username}}
listen = /run/php/zenspanel-{{.Username}}-{{.PHPVersion}}.sock
listen.owner = www-data
listen.group = www-data
pm = dynamic
pm.max_children = 5
pm.start_servers = 1
pm.min_spare_servers = 1
pm.max_spare_servers = 3
`

type poolData struct {
	Username   string
	PHPVersion string
}

func poolPath(phpPoolBase, username, phpVersion string) string {
	return filepath.Join(phpPoolBase, phpVersion, "fpm", "pool.d", fmt.Sprintf("zenspanel-%s.conf", username))
}

func CreatePool(phpPoolBase, username, phpVersion string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	if err := safe.PHPVersion(phpVersion); err != nil {
		return err
	}
	tmpl := template.Must(template.New("pool").Parse(poolTmpl))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, poolData{Username: username, PHPVersion: phpVersion}); err != nil {
		return fmt.Errorf("template: %w", err)
	}
	path := poolPath(phpPoolBase, username, phpVersion)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir pool dir: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write pool: %w", err)
	}
	// The php<ver>-fpm service may be installed but disabled — Ubuntu
	// only auto-starts the version installed first. EnsureRunning is
	// idempotent and runs before reload so the socket the pool just
	// created actually appears on disk.
	if err := EnsureRunning(phpVersion); err != nil {
		return fmt.Errorf("ensure fpm running: %w", err)
	}
	return ReloadFPM(phpVersion)
}

func DeletePool(phpPoolBase, username, phpVersion string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	if err := safe.PHPVersion(phpVersion); err != nil {
		return err
	}
	path := poolPath(phpPoolBase, username, phpVersion)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove pool: %w", err)
	}
	return ReloadFPM(phpVersion)
}

// EnsureRunning makes sure php<ver>-fpm is enabled+started before we
// try to reload it. Default Ubuntu only auto-starts the version that
// was installed first; the others sit disabled. Without this, switching
// a domain to a version whose service is stopped leaves nginx pointing
// at a socket that doesn't exist (ERR_CONNECTION_REFUSED in the
// browser).
func EnsureRunning(phpVersion string) error {
	if err := safe.PHPVersion(phpVersion); err != nil {
		return err
	}
	svc := fmt.Sprintf("php%s-fpm", phpVersion)

	// Idempotent: enable+start are no-ops if already on. We don't gate
	// on `is-active` because a "failed" state needs explicit `start`
	// to clear, not a check.
	if out, err := exec.Command("systemctl", "enable", svc).CombinedOutput(); err != nil {
		return fmt.Errorf("enable %s: %w: %s", svc, err, out)
	}
	if out, err := exec.Command("systemctl", "start", svc).CombinedOutput(); err != nil {
		return fmt.Errorf("start %s: %w: %s", svc, err, out)
	}
	return nil
}

// ReloadFPM reloads the FPM master so it re-reads pool configs. Falls
// back to a full restart if reload fails — happens when the service
// was stopped (reload only works on a running unit).
func ReloadFPM(phpVersion string) error {
	if err := safe.PHPVersion(phpVersion); err != nil {
		return err
	}
	svc := fmt.Sprintf("php%s-fpm", phpVersion)
	if out, err := exec.Command("systemctl", "reload", svc).CombinedOutput(); err != nil {
		// Reload fails on a stopped unit — try start instead so the
		// caller's "ensure pool is live" intent is honoured. Any
		// error here is reported back unchanged.
		if out2, err2 := exec.Command("systemctl", "start", svc).CombinedOutput(); err2 != nil {
			return fmt.Errorf("reload %s failed (%s); fallback start failed: %w: %s",
				svc, string(out), err2, out2)
		}
	}
	return nil
}
