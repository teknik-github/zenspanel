package quota

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/zenspanel/zenspanel/agent/safe"
)

// SetQuota sets disk quota for username on the filesystem containing
// homeBase. hardBytes is the hard limit in bytes; soft is 90% of hard.
// Passing hardBytes=0 removes the quota (sets unlimited).
//
// setquota expects 1KB blocks, so we divide bytes by 1024. Inode limits
// stay at 0/0 (unlimited) — we only care about disk space, not file count.
func SetQuota(username, homeBase string, hardBytes int64) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	hardKB := hardBytes / 1024
	softKB := hardKB * 9 / 10

	cmd := exec.Command("setquota", "-u", username,
		strconv.FormatInt(softKB, 10),
		strconv.FormatInt(hardKB, 10),
		"0", "0", homeBase)
	out, err := cmd.CombinedOutput()
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
	cmd := exec.Command("setquota", "-u", username, "0", "0", "0", "0", homeBase)
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
	cmd := exec.Command("repquota", "-u", homeBase)
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
