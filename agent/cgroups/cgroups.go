package cgroups

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const cgroupBase = "/sys/fs/cgroup/zenspanel"

func slicePath(username string) string {
	return filepath.Join(cgroupBase, username)
}

func CreateSlice(username string, cpuQuota int, memoryLimit int64) error {
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
	if err := os.WriteFile(filepath.Join(path, "memory.swap.max"), []byte("0"), 0644); err != nil {
		return fmt.Errorf("write memory.swap.max: %w", err)
	}
	return nil
}

func UpdateSlice(username string, cpuQuota int, memoryLimit int64) error {
	return CreateSlice(username, cpuQuota, memoryLimit)
}

func DeleteSlice(username string) error {
	return os.RemoveAll(slicePath(username))
}

func AddPID(username string, pid int) error {
	path := filepath.Join(slicePath(username), "cgroup.procs")
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}
