package user

import (
	"fmt"
	"os/exec"

	"github.com/zenspanel/zenspanel/agent/safe"
)

func Create(username string, uid int, homeBase string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	if uid < 1000 || uid > 65000 {
		return fmt.Errorf("agent: uid %d outside allowed range 1000-65000", uid)
	}
	homeDir := homeBase + "/" + username
	cmd := exec.Command("useradd",
		"-u", fmt.Sprintf("%d", uid),
		"-m",
		"-d", homeDir,
		"-s", "/bin/rbash",
		username,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("useradd failed: %w: %s", err, out)
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
