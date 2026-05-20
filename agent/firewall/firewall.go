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

// ListBlocked returns all IPs currently in the zenspanel ipset plus any
// IPs currently banned by fail2ban jails (V39 — merged view, source tagged).
func ListBlocked() ([]BlockedIP, error) {
	var result []BlockedIP

	// Panel-managed blocks from ipset.
	out, err := exec.Command("ipset", "list", ipsetName, "-output", "save").Output()
	if err == nil {
		sc := bufio.NewScanner(strings.NewReader(string(out)))
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if !strings.HasPrefix(line, "add "+ipsetName+" ") {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			ip := parts[2]
			reason := ""
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
				BlockedAt: time.Now().Format(time.RFC3339),
			})
		}
	}

	// fail2ban currently-banned IPs (V39).
	result = append(result, listFail2banBanned()...)
	return result, nil
}

// listFail2banBanned returns all IPs currently banned by any fail2ban jail.
func listFail2banBanned() []BlockedIP {
	// Get jail list.
	out, err := exec.Command("fail2ban-client", "status").Output()
	if err != nil {
		return nil
	}
	var jailNames []string
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := sc.Text()
		if strings.Contains(line, "Jail list:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				for _, j := range strings.Split(parts[1], ",") {
					name := strings.TrimSpace(j)
					if name != "" {
						jailNames = append(jailNames, name)
					}
				}
			}
		}
	}

	var banned []BlockedIP
	seen := map[string]bool{}
	for _, jail := range jailNames {
		jout, err := exec.Command("fail2ban-client", "status", jail).Output()
		if err != nil {
			continue
		}
		// Parse "Banned IP list: 1.2.3.4 5.6.7.8"
		sc2 := bufio.NewScanner(strings.NewReader(string(jout)))
		for sc2.Scan() {
			l := sc2.Text()
			if strings.Contains(l, "Banned IP list:") {
				parts := strings.SplitN(l, ":", 2)
				if len(parts) != 2 {
					continue
				}
				for _, ip := range strings.Fields(parts[1]) {
					ip = strings.TrimSpace(ip)
					if ip == "" || seen[ip] {
						continue
					}
					seen[ip] = true
					banned = append(banned, BlockedIP{
						IP:        ip,
						Reason:    "fail2ban: " + jail,
						Source:    "fail2ban",
						BlockedAt: time.Now().Format(time.RFC3339),
					})
				}
			}
		}
	}
	return banned
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
