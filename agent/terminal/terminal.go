package terminal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/creack/pty"

	"github.com/zenspanel/zenspanel/agent/safe"
)

type Session struct {
	PTY *os.File
	Cmd *exec.Cmd
}

func suPath() string {
	for _, p := range []string{"/usr/bin/su", "/bin/su"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "su" // fallback to PATH
}

// SpawnSession starts a PTY. isAdmin=true spawns an unrestricted root bash
// (WHM-style server terminal); isAdmin=false spawns a bash session scoped
// to the panel user's home via PATH=~/bin and a cd() wrapper in ~/.bash_profile.
func SpawnSession(username, homeBase string, isAdmin bool) (*Session, error) {
	if isAdmin {
		return spawnAdminSession()
	}
	if err := safe.Username(username); err != nil {
		return nil, err
	}
	homeDir := homeBase + "/" + username
	if _, err := os.Stat(homeDir); os.IsNotExist(err) {
		if err := os.MkdirAll(homeDir, 0711); err != nil {
			return nil, fmt.Errorf("mkdir home: %w", err)
		}
	}
	cmd := exec.Command(suPath(), "-s", "/bin/bash", "-", username)
	cmd.Env = []string{
		"HOME=" + homeDir,
		"USER=" + username,
		"LOGNAME=" + username,
		"PATH=" + homeDir + "/bin",
		"TERM=xterm-256color",
		"SHELL=/bin/bash",
	}
	cmd.Dir = homeDir
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("pty start: %w", err)
	}
	return &Session{PTY: ptmx, Cmd: cmd}, nil
}

// spawnAdminSession spawns an unrestricted root bash shell. The agent runs
// as root so no su(1) wrapper is needed — mirrors cPanel WHM Terminal.
func spawnAdminSession() (*Session, error) {
	cmd := exec.Command("/bin/bash", "--login")
	cmd.Env = []string{
		"HOME=/root",
		"USER=root",
		"LOGNAME=root",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"TERM=xterm-256color",
		"SHELL=/bin/bash",
	}
	cmd.Dir = "/root"
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

// Stream spawns a PTY for the user, creates a one-shot Unix socket the
// API can connect to, and returns the socket path. The caller (agent
// RPC) returns this path to the API; the API then dials it and bridges
// raw bytes between that socket and the WebSocket the browser is
// connected to.
//
// The socket accepts exactly one connection then unlinks itself —
// short-lived, per-session, no shared state.
func Stream(username, homeBase string, isAdmin bool) (string, error) {
	sess, err := SpawnSession(username, homeBase, isAdmin)
	if err != nil {
		return "", err
	}

	// Random suffix avoids collisions if two users connect at the same
	// instant before the listener is created. /tmp because it needs to
	// be writable by the agent (root) and connectable by the API user.
	rb := make([]byte, 8)
	if _, err := rand.Read(rb); err != nil {
		sess.Close()
		return "", err
	}
	sockPath := filepath.Join("/tmp", "zp-pty-"+hex.EncodeToString(rb)+".sock")
	_ = os.Remove(sockPath) // tear down stale socket from a crashed prior run

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		sess.Close()
		return "", fmt.Errorf("listen unix: %w", err)
	}
	// 0666 so the API user (non-root) can connect. Path is unguessable
	// (random hex) and torn down right after the single accept.
	if err := os.Chmod(sockPath, 0666); err != nil {
		ln.Close()
		_ = os.Remove(sockPath)
		sess.Close()
		return "", fmt.Errorf("chmod sock: %w", err)
	}

	go func() {
		defer os.Remove(sockPath)
		defer sess.Close()

		conn, err := ln.Accept()
		ln.Close() // one shot; close the listener regardless
		if err != nil {
			return
		}
		defer conn.Close()

		// Bidirectional copy. Either side closing tears everything down
		// because the io.Copy on the closed half returns and signals
		// `done`, which lets the goroutine exit and runs the deferred
		// `sess.Close()` (which kills the shell and closes the PTY,
		// which makes the OTHER io.Copy also return).
		done := make(chan struct{}, 2)
		go func() {
			io.Copy(conn, sess.PTY)
			done <- struct{}{}
		}()
		go func() {
			io.Copy(sess.PTY, conn)
			done <- struct{}{}
		}()
		<-done
	}()

	return sockPath, nil
}
