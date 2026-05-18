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

func ReloadFPM(phpVersion string) error {
	if err := safe.PHPVersion(phpVersion); err != nil {
		return err
	}
	svc := fmt.Sprintf("php%s-fpm", phpVersion)
	cmd := exec.Command("systemctl", "reload", svc)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("reload %s failed: %w: %s", svc, err, out)
	}
	return nil
}
