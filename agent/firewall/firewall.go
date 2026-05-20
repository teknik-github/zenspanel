package firewall

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

const ipsetName = "zenspanel-blocked"

// BlockedIP represents a single blocked entry.
type BlockedIP struct {
	IP        string `json:"ip"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`    // "panel" or "fail2ban"
	BlockedAt string `json:"blocked_at"`
}

// validateIP checks that s is a valid IPv4/IPv6 address or CIDR (V34).
func validateIP(s string) error {
	// Try plain IP first.
	if net.ParseIP(s) != nil {
		return nil
	}
	// Try CIDR.
	if _, _, err := net.ParseCIDR(s); err == nil {
		return nil
	}
	return fmt.Errorf("invalid IP address or CIDR: %q", s)
}

// ListBlocked returns all IPs currently in the zenspanel ipset (V36 —
// agent is authoritative, not the DB).
func ListBlocked() ([]BlockedIP, error) {
	out, err := exec.Command("ipset", "list", ipsetName, "-output", "save").Output()
	if err != nil {
		// ipset not installed or set doesn't exist yet — return empty.
		return []BlockedIP{}, nil
	}

	var result []BlockedIP
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		// Lines look like: add zenspanel-blocked 1.2.3.4 comment "reason" timeout 0
		if !strings.HasPrefix(line, "add "+ipsetName+" ") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		ip := parts[2]
		reason := ""
		// Extract comment if present.
		if idx := strings.Index(line, "comment \""); idx >= 0 {
			rest := line[idx+9:]
			if end := strings.Index(rest, "\""); end >= 0 {
				reason = rest[:end]
			}
		}
		result = append(result, BlockedIP{
			IP:        ip,
			Reason:    reason,
			Source:    "panel",
			BlockedAt: time.Now().Format(time.RFC3339), // ipset doesn't store time
		})
	}
	return result, nil
}

// Block adds an IP to the zenspanel ipset (V34 — arg array, no shell).
func Block(ip, reason string) error {
	if err := validateIP(ip); err != nil {
		return err
	}
	// Sanitise reason — strip quotes to avoid breaking the ipset comment.
	reason = strings.ReplaceAll(reason, `"`, `'`)
	if reason == "" {
		reason = "blocked by admin"
	}

	args := []string{"add", ipsetName, ip, "comment", reason}
	if out, err := exec.Command("ipset", args...).CombinedOutput(); err != nil {
		// "already exists" is not an error.
		if strings.Contains(string(out), "already exists") {
			return nil
		}
		return fmt.Errorf("ipset add: %w: %s", err, out)
	}

	// Persist the updated set.
	_ = exec.Command("sh", "-c", "ipset save > /etc/ipset/zenspanel.conf").Run()
	return nil
}

// Unblock removes an IP from the zenspanel ipset (V34).
func Unblock(ip string) error {
	if err := validateIP(ip); err != nil {
		return err
	}
	if out, err := exec.Command("ipset", "del", ipsetName, ip).CombinedOutput(); err != nil {
		if strings.Contains(string(out), "not exist") || strings.Contains(string(out), "not added") {
			return nil // already unblocked — idempotent
		}
		return fmt.Errorf("ipset del: %w: %s", err, out)
	}
	_ = exec.Command("sh", "-c", "ipset save > /etc/ipset/zenspanel.conf").Run()
	return nil
}
