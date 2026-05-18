package terminal

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/creack/pty"

	"github.com/zenspanel/zenspanel/agent/safe"
)

type Session struct {
	PTY *os.File
	Cmd *exec.Cmd
}

func SpawnSession(username, homeBase string) (*Session, error) {
	if err := safe.Username(username); err != nil {
		return nil, err
	}
	homeDir := homeBase + "/" + username
	cmd := exec.Command("su", "-s", "/bin/rbash", "-", username)
	cmd.Env = []string{
		"HOME=" + homeDir,
		"USER=" + username,
		"LOGNAME=" + username,
		"PATH=" + homeDir + "/bin",
		"TERM=xterm-256color",
	}
	cmd.Dir = homeDir

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("pty start: %w", err)
	}
	return &Session{PTY: ptmx, Cmd: cmd}, nil
}

func (s *Session) Close() error {
	if s.Cmd.Process != nil {
		s.Cmd.Process.Kill()
	}
	return s.PTY.Close()
}
