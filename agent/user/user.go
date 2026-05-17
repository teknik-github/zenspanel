package user

import (
	"fmt"
	"os/exec"
)

func Create(username string, uid int, homeBase string) error {
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
	cmd := exec.Command("userdel", "-r", username)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("userdel failed: %w: %s", err, out)
	}
	return nil
}
