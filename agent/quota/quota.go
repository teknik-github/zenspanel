package quota

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/zenspanel/zenspanel/agent/safe"
)

// resolveMount finds the actual filesystem mount point that contains the
// given path. setquota / repquota refuse anything that isn't a mount
// point or device, so passing the homeBase ("/var/lib/zenspanel/home")
// directly fails when home_base is a subdirectory of the root mount.
//
// We use `df` because it walks /proc/mounts the same way the kernel
// does and handles bind mounts, btrfs subvolumes, etc. Falls back to
// "/" if df fails — which is almost always correct when the home dir
// lives on the root filesystem.
func resolveMount(path string) string {
	out, err := exec.Command("df", "--output=target", path).Output()
	if err != nil {
		return "/"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return "/"
	}
	mp := strings.TrimSpace(lines[len(lines)-1])
	if mp == "" {
		return "/"
	}
	return mp
}

// SetQuota sets disk quota for username on the filesystem containing
// homeBase. hardBytes is the hard limit in bytes; soft is 90% of hard.
// Passing hardBytes=0 removes the quota (sets unlimited).
//
// setquota expects 1KB blocks, so we divide bytes by 1024. Inode limits
// stay at 0/0 (unlimited) — we only care about disk space, not file count.
//
// If setquota fails with "no quota enabled" (which happens when the
// installer's quotaon failed silently or the mount was remounted
// without quota since), we try to turn it on with `quotaon -u <mp>`
// then retry once. This is a self-heal path; the installer remains
// the canonical place to enable quota.
func SetQuota(username, homeBase string, hardBytes int64) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	mp := resolveMount(homeBase)
	hardKB := hardBytes / 1024
	softKB := hardKB * 9 / 10

	args := []string{"-u", username,
		strconv.FormatInt(softKB, 10),
		strconv.FormatInt(hardKB, 10),
		"0", "0", mp}

	out, err := exec.Command("setquota", args...).CombinedOutput()
	if err != nil && strings.Contains(string(out), "no quota enabled") {
		// Best-effort: try to flip quota on, then retry. Failures here
		// are reported via the original setquota error, not the
		// quotaon error — the user cares about why their quota call
		// didn't take effect.
		_, _ = exec.Command("quotaon", "-u", mp).CombinedOutput()
		out, err = exec.Command("setquota", args...).CombinedOutput()
	}
	if err != nil {
		return fmt.Errorf("setquota: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// DeleteQuota clears all quota limits for username. Used during user
// deletion so the username is freed up if reused later.
func DeleteQuota(username, homeBase string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	mp := resolveMount(homeBase)
	cmd := exec.Command("setquota", "-u", username, "0", "0", "0", "0", mp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("setquota delete: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ReadQuota returns current usage and hard limit in bytes for username
// on the filesystem containing homeBase. Parses `repquota -u <fs>` since
// it produces stable, parseable output across distros (vs `quota -u` which
// human-formats the result).
//
// Returns (0, 0, nil) if the user has no quota entry yet — that's the
// expected state right before SetQuota is called for the first time.
func ReadQuota(username, homeBase string) (usedBytes, hardBytes int64, err error) {
	if err := safe.Username(username); err != nil {
		return 0, 0, err
	}
	mp := resolveMount(homeBase)
	cmd := exec.Command("repquota", "-u", mp)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("repquota: %w", err)
	}
	// repquota output:
	//   User    used    soft    hard    grace   used  soft  hard  grace
	//   ----------------------------------------------------------------
	//   alice  --    1234       0  104857  ...
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != username {
			continue
		}
		// fields[2] = used (KB), fields[4] = hard (KB)
		used, _ := strconv.ParseInt(fields[2], 10, 64)
		hard, _ := strconv.ParseInt(fields[4], 10, 64)
		return used * 1024, hard * 1024, nil
	}
	return 0, 0, nil
}
