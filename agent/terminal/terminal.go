package terminal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	osuser "os/user"
	"path/filepath"
	"strconv"

	"github.com/creack/pty"

	"github.com/zenspanel/zenspanel/agent/safe"
)

type Session struct {
	PTY     *os.File
	Cmd     *exec.Cmd
	cleanup []func()
}

func bwrapPath() string {
	for _, p := range []string{"/usr/bin/bwrap", "/bin/bwrap"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func suPath() string {
	for _, p := range []string{"/usr/bin/su", "/bin/su"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "su"
}

// SpawnSession starts a PTY.
// Admin: unrestricted root bash (WHM-style, no chroot).
// User: bubblewrap chroot jail — only the user's home is visible and writable;
// /usr, /bin, /lib are bind-mounted read-only; no other users' files are accessible.
func SpawnSession(username, homeBase string, isAdmin bool) (*Session, error) {
	if isAdmin {
		return spawnAdminSession()
	}
	if err := safe.Username(username); err != nil {
		return nil, err
	}

	bwrap := bwrapPath()
	if bwrap == "" {
		return nil, fmt.Errorf("bubblewrap not found — install with: apt-get install -y bubblewrap")
	}

	homeDir := filepath.Join(homeBase, username)
	if _, err := os.Stat(homeDir); os.IsNotExist(err) {
		if err := os.MkdirAll(homeDir, 0711); err != nil {
			return nil, fmt.Errorf("mkdir home: %w", err)
		}
	}

	// Resolve the panel user's uid/gid so bwrap can drop to them inside the jail.
	u, err := osuser.Lookup(username)
	if err != nil {
		return nil, fmt.Errorf("lookup user %q: %w", username, err)
	}
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)

	// Write minimal /etc files for the jail — only the current user's entry.
	// Keeping these separate prevents leaking /etc/shadow or other users' entries.
	etcDir, err := os.MkdirTemp("", "zp-jailetc-")
	if err != nil {
		return nil, fmt.Errorf("mktemp jail etc: %w", err)
	}
	cleanupEtc := func() { os.RemoveAll(etcDir) }

	etcFiles := []struct {
		name    string
		content string
	}{
		{"passwd", fmt.Sprintf("%s:x:%d:%d::/home/%s:/bin/bash\nroot:x:0:0:root:/root:/bin/bash\n", username, uid, gid, username)},
		{"group", fmt.Sprintf("%s:x:%d:\n", username, gid)},
		{"nsswitch.conf", "passwd: files\ngroup: files\nhosts: files\n"},
	}
	for _, f := range etcFiles {
		if err := os.WriteFile(filepath.Join(etcDir, f.name), []byte(f.content), 0644); err != nil {
			cleanupEtc()
			return nil, fmt.Errorf("write jail etc/%s: %w", f.name, err)
		}
	}

	uidStr := strconv.Itoa(uid)
	gidStr := strconv.Itoa(gid)

	args := []string{
		// User home: read-write (the only writable location in the jail)
		"--bind", homeDir, "/home/" + username,
		// System filesystem: read-only.
		// ro-bind-try skips silently if the path doesn't exist on this host
		// (e.g. /bin is a symlink → usr/bin on Ubuntu 22.04, not a real dir).
		"--ro-bind", "/usr", "/usr",
		"--ro-bind-try", "/bin", "/bin",
		"--ro-bind-try", "/sbin", "/sbin",
		"--ro-bind-try", "/lib", "/lib",
		"--ro-bind-try", "/lib64", "/lib64",
		"--ro-bind-try", "/lib32", "/lib32",
		// Minimal /etc — no shadow file, no zenspanel config, no other users
		"--dir", "/etc",
		"--ro-bind", filepath.Join(etcDir, "passwd"), "/etc/passwd",
		"--ro-bind", filepath.Join(etcDir, "group"), "/etc/group",
		"--ro-bind", filepath.Join(etcDir, "nsswitch.conf"), "/etc/nsswitch.conf",
		// Virtual filesystems
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
		// Identity and working directory
		"--chdir", "/home/" + username,
		"--uid", uidStr,
		"--gid", gidStr,
		// Isolation
		"--unshare-pid",
		"--unshare-uts",
		"--hostname", "panel",
		"--die-with-parent",
		// Start with a clean environment; ~/.bash_profile sets PATH=$HOME/bin
		"--clearenv",
		"--setenv", "HOME", "/home/" + username,
		"--setenv", "USER", username,
		"--setenv", "LOGNAME", username,
		"--setenv", "TERM", "xterm-256color",
		"--setenv", "SHELL", "/bin/bash",
		// Login shell reads ~/.bash_profile (root-owned 444 — restricts PATH to ~/bin)
		"/bin/bash", "--login",
	}

	cmd := exec.Command(bwrap, args...)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		cleanupEtc()
		return nil, fmt.Errorf("pty start: %w", err)
	}
	return &Session{PTY: ptmx, Cmd: cmd, cleanup: []func(){cleanupEtc}}, nil
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
	err := s.PTY.Close()
	for _, fn := range s.cleanup {
		fn()
	}
	return err
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
