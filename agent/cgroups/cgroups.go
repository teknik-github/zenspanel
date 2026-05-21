package cgroups

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zenspanel/zenspanel/agent/safe"
)

const (
	cgroupRoot = "/sys/fs/cgroup"
	cgroupBase = "/sys/fs/cgroup/zenspanel"
)

// homeBaseDev is the block device major:minor for the filesystem that
// contains the user home directories. Detected once at startup and used
// for io.max throttling. Empty string = io throttling disabled.
var homeBaseDev string

// InitHomeBaseDev detects the block device for homeBase and caches it.
// Called by the agent at startup with cfg.Paths.HomeBase.
func InitHomeBaseDev(homeBase string) {
	out, err := exec.Command("stat", "-c", "%d", homeBase).Output()
	if err != nil {
		return
	}
	devNum, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return
	}
	major := devNum >> 8
	minor := devNum & 0xff
	homeBaseDev = fmt.Sprintf("%d:%d", major, minor)
}

func slicePath(username string) string {
	return filepath.Join(cgroupBase, username)
}

// ensureControllers makes sure cpu and memory controllers are delegated to
// children at every cgroup level we care about. In cgroup v2 a writable
// child file like cpu.max only exists under a parent whose
// `cgroup.subtree_control` enables that controller — root may inherit it
// from systemd, but `/sys/fs/cgroup/zenspanel/<user>/cpu.max` only appears
// after `+cpu` is echoed into the parent's subtree_control. Doing this in
// the agent removes a manual server-bringup step.
//
// We enable controllers we already know are available; missing controllers
// are skipped so this works on hosts that lack swap accounting or partial
// v2 support.
func ensureControllers() error {
	if err := enableSubtreeControllers(cgroupRoot); err != nil {
		return fmt.Errorf("enable controllers at %s: %w", cgroupRoot, err)
	}
	if err := os.MkdirAll(cgroupBase, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", cgroupBase, err)
	}
	if err := enableSubtreeControllers(cgroupBase); err != nil {
		return fmt.Errorf("enable controllers at %s: %w", cgroupBase, err)
	}
	return nil
}

// enableSubtreeControllers writes "+cpu +memory" (filtered to whatever the
// parent advertises in cgroup.controllers) to the parent's
// cgroup.subtree_control. Idempotent — writing a controller that's already
// enabled returns EBUSY which we treat as success.
func enableSubtreeControllers(parent string) error {
	availBytes, err := os.ReadFile(filepath.Join(parent, "cgroup.controllers"))
	if err != nil {
		return fmt.Errorf("read cgroup.controllers: %w", err)
	}
	avail := map[string]bool{}
	for _, c := range strings.Fields(string(availBytes)) {
		avail[c] = true
	}

	wanted := []string{"cpu", "memory", "pids", "io"}
	tokens := []string{}
	for _, c := range wanted {
		if avail[c] {
			tokens = append(tokens, "+"+c)
		}
	}
	if len(tokens) == 0 {
		return nil
	}
	payload := strings.Join(tokens, " ")
	err = os.WriteFile(filepath.Join(parent, "cgroup.subtree_control"), []byte(payload), 0644)
	if err != nil && !strings.Contains(err.Error(), "device or resource busy") {
		return fmt.Errorf("write cgroup.subtree_control: %w", err)
	}
	return nil
}

func CreateSlice(username string, cpuQuota int, memoryLimit int64) error {
	return CreateSliceWithLimits(username, cpuQuota, memoryLimit, 0, 0, 0)
}

// CreateSliceWithLimits creates a cgroup slice with full resource limits.
// maxProcs: max number of processes (0 = unlimited, prevents fork bombs)
// ioReadBps/ioWriteBps: I/O bandwidth limit in bytes/sec (0 = unlimited)
func CreateSliceWithLimits(username string, cpuQuota int, memoryLimit int64, maxProcs int, ioReadBps, ioWriteBps int64) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	if err := ensureControllers(); err != nil {
		return err
	}
	path := slicePath(username)
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("mkdir cgroup: %w", err)
	}
	cpuMax := fmt.Sprintf("%d 100000", cpuQuota)
	if err := os.WriteFile(filepath.Join(path, "cpu.max"), []byte(cpuMax), 0644); err != nil {
		return fmt.Errorf("write cpu.max: %w", err)
	}
	if err := os.WriteFile(filepath.Join(path, "memory.max"), []byte(strconv.FormatInt(memoryLimit, 10)), 0644); err != nil {
		return fmt.Errorf("write memory.max: %w", err)
	}
	swapPath := filepath.Join(path, "memory.swap.max")
	if _, err := os.Stat(swapPath); err == nil {
		if err := os.WriteFile(swapPath, []byte("0"), 0644); err != nil {
			return fmt.Errorf("write memory.swap.max: %w", err)
		}
	}

	// NPROC limit via pids.max — prevents fork bombs (V50).
	// Default 200 if not specified; 0 means unlimited.
	pidsMax := "200"
	if maxProcs > 0 {
		pidsMax = strconv.Itoa(maxProcs)
	} else if maxProcs < 0 {
		pidsMax = "max"
	}
	pidsPath := filepath.Join(path, "pids.max")
	if _, err := os.Stat(pidsPath); err == nil {
		_ = os.WriteFile(pidsPath, []byte(pidsMax), 0644)
	}

	// I/O throttling via io.max — prevents disk abuse (V51).
	// Requires the block device major:minor number. We detect the device
	// that homeBase lives on and apply the limit there.
	if (ioReadBps > 0 || ioWriteBps > 0) && homeBaseDev != "" {
		ioVal := fmt.Sprintf("%s rbps=%d wbps=%d riops=max wiops=max",
			homeBaseDev, ioReadBps, ioWriteBps)
		ioPath := filepath.Join(path, "io.max")
		if _, err := os.Stat(ioPath); err == nil {
			_ = os.WriteFile(ioPath, []byte(ioVal), 0644)
		}
	}

	return nil
}

func UpdateSlice(username string, cpuQuota int, memoryLimit int64) error {
	return CreateSliceWithLimits(username, cpuQuota, memoryLimit, 0, 0, 0)
}

func DeleteSlice(username string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	return os.RemoveAll(slicePath(username))
}

func AddPID(username string, pid int) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	path := filepath.Join(slicePath(username), "cgroup.procs")
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}

// ReadMetrics returns the current RAM, disk, and CPU usage for a user.
// RAM comes from the cgroup v2 memory.current pseudo-file (bytes). Disk
// is `du -sb` over the user's home directory (bytes) — there's no
// per-user cgroup disk metric, only quota tooling, and quotas may not be
// enabled. We cap du to 5s to avoid pinning the API on a giant home tree.
// CPU is a percentage (0-100+) computed from the delta of cpu.stat's
// usage_usec between this call and the previous one cached per user.
//
// A missing slice or home dir is not an error here — it just means the
// user hasn't been provisioned yet (or was just deleted), so we return 0
// for whichever one is missing rather than failing the whole call.
//
// First call per user returns 0 for CPU because there's no previous
// sample to compare against. Subsequent calls return a real percentage.
func ReadMetrics(username, homeBase string) (ramUsed, diskUsed int64, cpuPct float64, err error) {
	if err := safe.Username(username); err != nil {
		return 0, 0, 0, err
	}

	if data, readErr := os.ReadFile(filepath.Join(slicePath(username), "memory.current")); readErr == nil {
		ramUsed, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	}

	homeDir := filepath.Join(homeBase, username)
	if _, statErr := os.Stat(homeDir); statErr == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		out, runErr := exec.CommandContext(ctx, "du", "-sb", homeDir).Output()
		if runErr == nil {
			fields := strings.Fields(string(out))
			if len(fields) > 0 {
				diskUsed, _ = strconv.ParseInt(fields[0], 10, 64)
			}
		}
	}

	cpuPct = cpuPercent(username)
	return ramUsed, diskUsed, cpuPct, nil
}

// cpuSample caches the previous cpu.stat reading per user so we can turn
// the monotonic usage_usec counter into a rate. Without this we'd either
// have to sleep inside the agent (adding latency to every dashboard
// refresh) or push the math onto the frontend (which would have to track
// previous values across HTTP polls).
type cpuSample struct {
	usec int64
	at   time.Time
}

var cpuCache sync.Map // map[string]*cpuSample

// readCPUUsageUsec parses the first line of cpu.stat ("usage_usec <N>").
// Returns 0 with no error if the file is missing — that just means the
// slice hasn't been created yet, which is fine for a freshly provisioned
// user. Genuine read errors after Stat passes propagate up.
func readCPUUsageUsec(username string) (int64, error) {
	data, err := os.ReadFile(filepath.Join(slicePath(username), "cpu.stat"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "usage_usec" {
			n, _ := strconv.ParseInt(fields[1], 10, 64)
			return n, nil
		}
	}
	return 0, nil
}

// cpuPercent returns the CPU usage of the user's cgroup since the last
// time it was sampled, as a percentage of one CPU core. Values >100 are
// possible on multi-core systems where the user has multi-core quota and
// is hitting more than one core at once. The percentage is capped at the
// caller layer if the UI needs it bounded.
func cpuPercent(username string) float64 {
	now := time.Now()
	usec, err := readCPUUsageUsec(username)
	if err != nil || usec == 0 {
		return 0
	}
	stored, loaded := cpuCache.Swap(username, &cpuSample{usec: usec, at: now})
	if !loaded {
		// First sample — no delta yet.
		return 0
	}
	prev := stored.(*cpuSample)
	deltaUsec := usec - prev.usec
	deltaTime := now.Sub(prev.at).Microseconds()
	if deltaTime <= 0 || deltaUsec < 0 {
		return 0
	}
	return float64(deltaUsec) / float64(deltaTime) * 100
}
