package cgroups

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zenspanel/zenspanel/agent/safe"
)

const (
	cgroupRoot = "/sys/fs/cgroup"
	cgroupBase = "/sys/fs/cgroup/zenspanel"
)

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

	wanted := []string{"cpu", "memory"}
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
	// memory.swap.max is optional — kernels built without swap accounting
	// don't expose it, and the file may also be missing on some distros.
	// We treat its absence as non-fatal.
	swapPath := filepath.Join(path, "memory.swap.max")
	if _, err := os.Stat(swapPath); err == nil {
		if err := os.WriteFile(swapPath, []byte("0"), 0644); err != nil {
			return fmt.Errorf("write memory.swap.max: %w", err)
		}
	}
	return nil
}

func UpdateSlice(username string, cpuQuota int, memoryLimit int64) error {
	return CreateSlice(username, cpuQuota, memoryLimit)
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
