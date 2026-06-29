package handlers

import (
	"bufio"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/store"
)

type SystemHandler struct {
	users     *store.UserStore
	domains   *store.DomainStore
	databases *store.DatabaseStore
	agentSock string
	version   string // set at binary build time via -ldflags
}

func NewSystemHandler(users *store.UserStore, domains *store.DomainStore, databases *store.DatabaseStore, agentSock, version string) *SystemHandler {
	return &SystemHandler{users: users, domains: domains, databases: databases, agentSock: agentSock, version: version}
}

// Version returns the running binary version (release tag or "dev").
func (h *SystemHandler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": h.version})
}

// CheckUpdate asks the agent to fetch the remote and report whether the
// installed source tree is behind. The handler is a thin shim; the
// actual git work happens in the agent because the API process can't
// write to /opt/zenspanel/src.
func (h *SystemHandler) CheckUpdate(c *gin.Context) {
	var resp interface{}
	if err := agent.NewClient(h.agentSock).Call("update.check", nil, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RunUpdate kicks off the async update on the agent. The agent runs
// at most one update at a time; concurrent calls return the in-flight
// status instead of starting a second run. The optional download_url
// in the request body switches the agent from build-from-source to
// download-the-tarball — the latter is the default path now and the
// only one that fits in 1 GB of RAM.
func (h *SystemHandler) RunUpdate(c *gin.Context) {
	var req struct {
		DownloadURL string `json:"download_url"`
		Checksum    string `json:"checksum"`
	}
	_ = c.ShouldBindJSON(&req)
	var resp interface{}
	if err := agent.NewClient(h.agentSock).Call("update.run", map[string]interface{}{
		"download_url": req.DownloadURL,
		"checksum":     req.Checksum,
	}, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, resp)
}

// UpdateStatus is what the UI polls every few seconds to render the
// progress bar and log tail.
func (h *SystemHandler) UpdateStatus(c *gin.Context) {
	var resp interface{}
	if err := agent.NewClient(h.agentSock).Call("update.status", nil, &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Stats returns host-level metrics for the admin dashboard. CPU and RAM
// come from /proc directly because they are server-wide, not per-user —
// no need to bounce through the agent. Service status comes from
// `systemctl is-active`, which is cheap and accurate.
func (h *SystemHandler) Stats(c *gin.Context) {
	users, _ := h.usersStats()
	domainsStats, _ := h.domainsStats()
	dbCount, _ := h.databasesCount()

	cpuPct, _ := readCPUPercent()
	ramUsed, ramTotal, _ := readMemInfo()
	diskUsed, diskTotal := readDiskUsage()
	load1, load5, load15 := readLoadAvg()

	c.JSON(http.StatusOK, gin.H{
		"users":      users,
		"domains":    domainsStats,
		"databases":  gin.H{"total": dbCount},
		"cpu_pct":    cpuPct,
		"ram_used":   ramUsed,
		"ram_total":  ramTotal,
		"disk_used":  diskUsed,
		"disk_total": diskTotal,
		"load_1":     load1,
		"load_5":     load5,
		"load_15":    load15,
		"services": gin.H{
			"nginx": serviceActive("nginx"),
			"mysql": serviceActive("mysql"),
			"redis": serviceActive("redis-server"),
		},
		"uptime_seconds": readUptime(),
	})
}

func (h *SystemHandler) usersStats() (gin.H, error) {
	all, _, err := h.users.List(store.UserFilter{Limit: 1000000})
	if err != nil {
		return nil, err
	}
	active, suspended := 0, 0
	for _, u := range all {
		switch u.Status {
		case "active":
			active++
		case "suspended":
			suspended++
		}
	}
	return gin.H{"total": len(all), "active": active, "suspended": suspended}, nil
}

func (h *SystemHandler) domainsStats() (gin.H, error) {
	all, err := h.domains.ListAll()
	if err != nil {
		return nil, err
	}
	active := 0
	for _, d := range all {
		if d.Status == "active" {
			active++
		}
	}
	return gin.H{"total": len(all), "active": active}, nil
}

func (h *SystemHandler) databasesCount() (int, error) {
	// We don't have a "list all databases" store method (database listings
	// are user-scoped by design), so we count via the domains-style
	// pattern: iterate users and sum CountByUserID. For a control panel
	// this stays cheap because the user count is bounded.
	users, _, err := h.users.List(store.UserFilter{Limit: 1000000})
	if err != nil {
		return 0, err
	}
	total := 0
	for _, u := range users {
		n, _ := h.databases.CountByUserID(u.ID)
		total += n
	}
	return total, nil
}

// readCPUPercent samples /proc/stat twice 200ms apart and returns the
// non-idle delta as a percentage. Lightweight (two file reads), good
// enough for a dashboard refresh — for sub-second-accurate metrics we'd
// keep a long-lived sampler instead.
func readCPUPercent() (float64, error) {
	a, err := readCPUTotals()
	if err != nil {
		return 0, err
	}
	time.Sleep(200 * time.Millisecond)
	b, err := readCPUTotals()
	if err != nil {
		return 0, err
	}
	totalDelta := float64(b.total - a.total)
	idleDelta := float64(b.idle - a.idle)
	if totalDelta <= 0 {
		return 0, nil
	}
	return (1 - idleDelta/totalDelta) * 100, nil
}

type cpuTotals struct {
	total int64
	idle  int64
}

func readCPUTotals() (cpuTotals, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuTotals{}, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return cpuTotals{}, sc.Err()
	}
	fields := strings.Fields(sc.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuTotals{}, nil
	}
	var total int64
	for _, v := range fields[1:] {
		n, _ := strconv.ParseInt(v, 10, 64)
		total += n
	}
	idle, _ := strconv.ParseInt(fields[4], 10, 64)
	return cpuTotals{total: total, idle: idle}, nil
}

// readMemInfo returns (used, total) in bytes from /proc/meminfo. We treat
// "used" as MemTotal - MemAvailable so cached pages count as usable, which
// is the figure most dashboards mean by "RAM in use".
func readMemInfo() (used, total int64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	var memTotal, memAvail int64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			memTotal = parseKB(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			memAvail = parseKB(line)
		}
		if memTotal > 0 && memAvail > 0 {
			break
		}
	}
	if memTotal == 0 {
		return 0, 0, sc.Err()
	}
	return memTotal - memAvail, memTotal, nil
}

func parseKB(line string) int64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	kb, _ := strconv.ParseInt(fields[1], 10, 64)
	return kb * 1024
}

func readUptime() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	secs, _ := strconv.ParseFloat(fields[0], 64)
	return int64(secs)
}

// readDiskUsage returns (used, total) bytes for the root filesystem via statfs.
func readDiskUsage() (used, total int64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0, 0
	}
	total = int64(stat.Blocks) * int64(stat.Bsize)
	avail := int64(stat.Bavail) * int64(stat.Bsize)
	return total - avail, total
}

// readLoadAvg returns 1, 5, and 15-minute load averages from /proc/loadavg.
func readLoadAvg() (l1, l5, l15 float64) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0, 0, 0
	}
	l1, _ = strconv.ParseFloat(fields[0], 64)
	l5, _ = strconv.ParseFloat(fields[1], 64)
	l15, _ = strconv.ParseFloat(fields[2], 64)
	return l1, l5, l15
}

func serviceActive(name string) string {
	out, err := exec.Command("systemctl", "is-active", name).Output()
	status := strings.TrimSpace(string(out))
	if err != nil && status == "" {
		return "unknown"
	}
	return status
}

// Maintenance runs a one-shot admin action on the server. Actions are
// whitelisted — no arbitrary command execution. Admin-only route.
func (h *SystemHandler) Maintenance(c *gin.Context) {
	var req struct {
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	type result struct {
		Output string `json:"output"`
		Error  string `json:"error,omitempty"`
	}

	run := func(name string, args ...string) result {
		out, err := exec.Command(name, args...).CombinedOutput()
		r := result{Output: strings.TrimSpace(string(out))}
		if err != nil {
			r.Error = err.Error()
		}
		return r
	}

	switch req.Action {
	case "clamav_install":
		// apt-get update first — ClamAV may not be in the local cache.
		run("apt-get", "update", "-qq")
		r := run("apt-get", "install", "-y", "clamav", "clamav-daemon")
		if r.Error == "" {
			run("systemctl", "enable", "clamav-daemon", "clamav-freshclam")
			run("systemctl", "start", "clamav-daemon", "clamav-freshclam")
		}
		c.JSON(http.StatusOK, r)

	case "clamav_update":
		// Update virus definitions via freshclam.
		run("systemctl", "stop", "clamav-freshclam")
		r := run("freshclam")
		run("systemctl", "start", "clamav-freshclam")
		c.JSON(http.StatusOK, r)

	case "clamav_restart":
		r := run("systemctl", "restart", "clamav-daemon")
		c.JSON(http.StatusOK, r)

	case "fail2ban_restart":
		r := run("systemctl", "restart", "fail2ban")
		c.JSON(http.StatusOK, r)

	case "nginx_reload":
		r := run("nginx", "-s", "reload")
		c.JSON(http.StatusOK, r)

	case "service_status":
		services := []string{"clamav-daemon", "clamav-freshclam", "fail2ban", "nginx", "mysql", "redis-server"}
		statuses := map[string]string{}
		for _, svc := range services {
			statuses[svc] = serviceActive(svc)
		}
		// Also check inotifywait + rclone availability.
		statuses["inotify-tools"] = func() string {
			if _, err := exec.LookPath("inotifywait"); err == nil {
				return "active"
			}
			return "not installed"
		}()
		statuses["rclone"] = func() string {
			if _, err := exec.LookPath("rclone"); err == nil {
				return "active"
			}
			return "not installed"
		}()
		c.JSON(http.StatusOK, gin.H{"services": statuses})

	case "install_tools":
		// Install inotify-tools + rclone for antivirus realtime + S3 backups.
		run("apt-get", "update", "-qq")
		r1 := run("apt-get", "install", "-y", "inotify-tools")
		r2 := run("bash", "-c", "which rclone || curl -fsSL https://rclone.org/install.sh | bash")
		output := r1.Output + "\n" + r2.Output
		errMsg := ""
		if r1.Error != "" {
			errMsg += "inotify-tools: " + r1.Error + " "
		}
		if r2.Error != "" {
			errMsg += "rclone: " + r2.Error
		}
		c.JSON(http.StatusOK, gin.H{"output": output, "error": errMsg})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown action: " + req.Action})
	}
}
