package filebrowser

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	osuser "os/user"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"github.com/zenspanel/zenspanel/agent/safe"
)

const (
	// nginxMapConf is placed in conf.d/ so nginx includes it in the http
	// context — the only context where map{} directives are valid.
	nginxMapConf = "/etc/nginx/conf.d/zenspanel-fb-ports.conf"

	// portMapJSON is the JSON source of truth for port assignments.
	// We derive the Nginx map file from it on every change instead of
	// parsing Nginx syntax.
	portMapJSON = "/etc/nginx/zenspanel/fb-ports.json"

	fbBin = "/usr/local/bin/filebrowser"
)

// unitPath returns the systemd service unit file path for a panel user's
// dedicated FileBrowser instance.
func unitPath(username string) string {
	return "/etc/systemd/system/zenspanel-fb-" + username + ".service"
}

// userPort derives the FileBrowser TCP listen port from the Linux UID.
// UIDs are assigned in range 10 000–60 000 by the agent; adding 100
// gives ports 10 100–60 100. Ports below 32 768 (the Linux ephemeral
// range floor on most systems) are guaranteed not to collide with
// ephemeral connections. Panels with more than ~22 000 users should
// raise net.ipv4.ip_local_port_range to start above 60 100.
func userPort(uid int) int {
	return uid + 100
}

// mu serialises concurrent writes to the Nginx map files.
var mu sync.Mutex

// CreateUser provisions a dedicated FileBrowser process for a panel user.
//
// Each instance runs as the panel user (User=<username> in the systemd
// unit), so every file FileBrowser creates is owned by that user from
// the moment of creation — no chown timer, no root-owned files, no race
// window where PHP cannot write to uploaded content.
//
// The FileBrowser SQLite DB is initialised via the CLI *before* the
// service starts. CLI and service both hold an exclusive write lock on
// the DB; initialising beforehand avoids the deadlock documented in the
// previous single-instance design.
// CreateUser provisions (or re-syncs) a dedicated FileBrowser instance for a
// panel user. Safe to call on existing users — config set and users update run
// every time so settings are always kept in sync with the current policy.
//
// isAdmin=true: runs as root, browse root="/", perm.admin=true (Settings page
// and Storage Usage widget visible) — mirrors cPanel WHM File Manager.
// isAdmin=false: runs as the panel user, scoped to their home dir,
// perm.admin=false (Settings and Storage Usage hidden).
func CreateUser(username, homeBase string, isAdmin bool) error {
	if err := safe.Username(username); err != nil {
		return err
	}

	u, err := osuser.Lookup(username)
	if err != nil {
		return fmt.Errorf("lookup %s: %w", username, err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("parse uid: %w", err)
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return fmt.Errorf("parse gid: %w", err)
	}

	port := userPort(uid)

	var homeDir, dbPath string
	if isAdmin {
		homeDir = "/"
		dbPath = "/root/.zenspanel-fb-" + username + ".db"
	} else {
		homeDir = filepath.Join(homeBase, username)
		dbPath = filepath.Join(homeDir, ".zenspanel-fb.db")
	}

	// Step 1: initialise a brand-new DB (config init + users add).
	// Skipped when DB already exists so we don't wipe existing sessions.
	dbIsNew := false
	if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
		dbIsNew = true
		if out, err := exec.Command(fbBin, "--database="+dbPath, "config", "init").CombinedOutput(); err != nil {
			return fmt.Errorf("filebrowser config init: %w: %s", err, out)
		}
	}

	// Step 2: always sync config — idempotent, updates existing DBs.
	// --branding.disableUsedPercentage hides the "X of Y used" storage
	// widget for panel users; omitted for admin so they see disk usage.
	configArgs := []string{
		fbBin, "--database=" + dbPath, "config", "set",
		"--address", "127.0.0.1",
		"--port", strconv.Itoa(port),
		"--root", homeDir,
		"--baseurl", "/filebrowser",
		"--auth.method=proxy",
		"--auth.header=X-Auth-User",
	}
	if !isAdmin {
		configArgs = append(configArgs, "--branding.disableUsedPercentage=true")
	}
	if out, err := exec.Command(configArgs[0], configArgs[1:]...).CombinedOutput(); err != nil {
		return fmt.Errorf("filebrowser config set: %w: %s", err, out)
	}

	// Step 3: ensure the user record has up-to-date permissions.
	// New DB → users add; existing DB → users update (fallback to add).
	permAdmin := "false"
	if isAdmin {
		permAdmin = "true"
	}
	userPerms := []string{
		"--perm.admin=" + permAdmin,
		"--perm.create=true",
		"--perm.rename=true",
		"--perm.modify=true",
		"--perm.delete=true",
		"--perm.share=false",
		"--perm.download=true",
		"--scope", "/",
	}
	if dbIsNew {
		pw := make([]byte, 16)
		if _, err := rand.Read(pw); err != nil {
			return fmt.Errorf("rand: %w", err)
		}
		addArgs := append([]string{fbBin, "--database=" + dbPath, "users", "add",
			username, hex.EncodeToString(pw)}, userPerms...)
		if out, err := exec.Command(addArgs[0], addArgs[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("filebrowser users add: %w: %s", err, out)
		}
		if !isAdmin {
			if err := os.Chown(dbPath, uid, gid); err != nil {
				return fmt.Errorf("chown db: %w", err)
			}
		}
	} else {
		// users update keeps the existing password intact.
		updateArgs := append([]string{fbBin, "--database=" + dbPath, "users", "update", username}, userPerms...)
		if out, err := exec.Command(updateArgs[0], updateArgs[1:]...).CombinedOutput(); err != nil {
			// User record may be missing (e.g. DB existed but was corrupt).
			// Fall back to add with a fresh password.
			pw := make([]byte, 16)
			rand.Read(pw) //nolint:errcheck
			addArgs := append([]string{fbBin, "--database=" + dbPath, "users", "add",
				username, hex.EncodeToString(pw)}, userPerms...)
			if out2, err2 := exec.Command(addArgs[0], addArgs[1:]...).CombinedOutput(); err2 != nil {
				return fmt.Errorf("filebrowser users update: %w: %s (add: %v: %s)", err, out, err2, out2)
			}
		}
	}

	// Admin unit: User=root (full VM access).
	// User unit:  User=<username> (files created are owned correctly).
	serviceUser := username
	if isAdmin {
		serviceUser = "root"
	}
	unit := "# Auto-generated by ZensPanel agent — do not edit.\n" +
		"[Unit]\n" +
		"Description=ZensPanel FileBrowser — " + username + "\n" +
		"After=network.target\n\n" +
		"[Service]\n" +
		"Type=simple\n" +
		"User=" + serviceUser + "\n" +
		"ExecStart=" + fbBin + " --database " + dbPath + "\n" +
		"Restart=on-failure\n" +
		"RestartSec=5\n\n" +
		"[Install]\n" +
		"WantedBy=multi-user.target\n"

	if err := os.WriteFile(unitPath(username), []byte(unit), 0644); err != nil {
		return fmt.Errorf("write unit: %w", err)
	}

	svc := "zenspanel-fb-" + username + ".service"
	for _, args := range [][]string{
		{"systemctl", "daemon-reload"},
		{"systemctl", "enable", "--quiet", svc},
		// restart (not start) so config changes take effect on existing instances.
		{"systemctl", "restart", svc},
	} {
		if out, cmdErr := exec.Command(args[0], args[1:]...).CombinedOutput(); cmdErr != nil {
			return fmt.Errorf("%v: %w: %s", args, cmdErr, out)
		}
	}

	if err := addToPortMap(username, port); err != nil {
		return fmt.Errorf("update port map: %w", err)
	}
	return reloadNginx()
}

// DeleteUser stops and removes the per-user FileBrowser service and
// removes its entry from the Nginx port map.
func DeleteUser(username string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	svc := "zenspanel-fb-" + username + ".service"
	exec.Command("systemctl", "stop", svc).Run()    //nolint:errcheck
	exec.Command("systemctl", "disable", svc).Run() //nolint:errcheck
	os.Remove(unitPath(username))
	exec.Command("systemctl", "daemon-reload").Run() //nolint:errcheck

	if err := removeFromPortMap(username); err != nil {
		return fmt.Errorf("update port map: %w", err)
	}
	return reloadNginx()
}

// ── Nginx port map ────────────────────────────────────────────────────

type portMap map[string]int

func readPortMap() (portMap, error) {
	pm := make(portMap)
	data, err := os.ReadFile(portMapJSON)
	if os.IsNotExist(err) {
		return pm, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &pm); err != nil {
		return nil, fmt.Errorf("parse port map: %w", err)
	}
	return pm, nil
}

func writePortMap(pm portMap) error {
	if err := os.MkdirAll(filepath.Dir(portMapJSON), 0755); err != nil {
		return err
	}
	data, _ := json.Marshal(pm)
	if err := os.WriteFile(portMapJSON, data, 0644); err != nil {
		return err
	}
	return writeNginxMapConf(pm)
}

// writeNginxMapConf renders the Nginx map block from the current port
// assignments. The map{} directive must be in the http context; placing
// this file in conf.d/ satisfies that — nginx.conf includes conf.d/*.conf
// inside the http block on all standard Ubuntu installs.
func writeNginxMapConf(pm portMap) error {
	if err := os.MkdirAll(filepath.Dir(nginxMapConf), 0755); err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString("# Auto-generated by ZensPanel agent — do not edit.\n")
	buf.WriteString("map $fb_user $fb_port {\n")
	buf.WriteString("    default 0;\n")
	users := make([]string, 0, len(pm))
	for u := range pm {
		users = append(users, u)
	}
	sort.Strings(users)
	for _, u := range users {
		fmt.Fprintf(&buf, "    %-30s %d;\n", u, pm[u])
	}
	buf.WriteString("}\n")
	return os.WriteFile(nginxMapConf, buf.Bytes(), 0644)
}

func addToPortMap(username string, port int) error {
	mu.Lock()
	defer mu.Unlock()
	pm, err := readPortMap()
	if err != nil {
		pm = make(portMap)
	}
	pm[username] = port
	return writePortMap(pm)
}

func removeFromPortMap(username string) error {
	mu.Lock()
	defer mu.Unlock()
	pm, err := readPortMap()
	if err != nil {
		return err
	}
	delete(pm, username)
	return writePortMap(pm)
}

func reloadNginx() error {
	if out, err := exec.Command("nginx", "-s", "reload").CombinedOutput(); err != nil {
		return fmt.Errorf("nginx reload: %w: %s", err, out)
	}
	return nil
}
