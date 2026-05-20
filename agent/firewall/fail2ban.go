package firewall

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const jailDir = "/etc/fail2ban/jail.d"

// Jail describes a fail2ban jail and its current state.
type Jail struct {
	Name           string `json:"name"`
	Enabled        bool   `json:"enabled"`
	BanCount       int    `json:"ban_count"`
	CurrentlyBanned int   `json:"currently_banned"`
}

// jailNameRe validates jail names — alphanumeric + dash/underscore only.
// Prevents path traversal when constructing the jail.d file path (V35).
var jailNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func validateJailName(name string) error {
	if !jailNameRe.MatchString(name) {
		return fmt.Errorf("invalid jail name %q", name)
	}
	return nil
}

// ListJails returns all known fail2ban jails with their status.
func ListJails() ([]Jail, error) {
	// Get list of active jails from fail2ban-client.
	out, err := exec.Command("fail2ban-client", "status").Output()
	if err != nil {
		return []Jail{}, nil // fail2ban not running — return empty
	}

	var jailNames []string
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := sc.Text()
		if strings.Contains(line, "Jail list:") {
			// Format: "|- Jail list:   sshd, nginx-http-auth"
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

	var jails []Jail
	for _, name := range jailNames {
		j := Jail{Name: name, Enabled: true}
		// Get per-jail stats.
		if jout, err := exec.Command("fail2ban-client", "status", name).Output(); err == nil {
			sc2 := bufio.NewScanner(strings.NewReader(string(jout)))
			for sc2.Scan() {
				l := sc2.Text()
				if strings.Contains(l, "Total banned:") {
					parts := strings.SplitN(l, ":", 2)
					if len(parts) == 2 {
						j.BanCount, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
					}
				}
				if strings.Contains(l, "Currently banned:") {
					parts := strings.SplitN(l, ":", 2)
					if len(parts) == 2 {
						j.CurrentlyBanned, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
					}
				}
			}
		}
		jails = append(jails, j)
	}

	// Also include disabled jails from jail.d files.
	entries, _ := os.ReadDir(jailDir)
	activeSet := map[string]bool{}
	for _, j := range jails {
		activeSet[j.Name] = true
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".conf")
		if name == "zenspanel" || activeSet[name] {
			continue
		}
		// Check if enabled=false in the file.
		data, err := os.ReadFile(filepath.Join(jailDir, e.Name()))
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "enabled = false") {
			jails = append(jails, Jail{Name: name, Enabled: false})
		}
	}

	return jails, nil
}

// SetJail enables or disables a fail2ban jail by writing a snippet to
// jail.d and reloading fail2ban-client (V35 — only writes to jail.d).
func SetJail(name string, enabled bool) error {
	if err := validateJailName(name); err != nil {
		return err
	}

	// Write a minimal override snippet to jail.d.
	confPath := filepath.Join(jailDir, name+"-zenspanel-override.conf")
	// Ensure we only write inside jail.d (V35).
	if !strings.HasPrefix(filepath.Clean(confPath), jailDir+"/") {
		return fmt.Errorf("jail config path escapes jail.d directory")
	}

	enabledStr := "true"
	if !enabled {
		enabledStr = "false"
	}
	content := fmt.Sprintf("[%s]\nenabled = %s\n", name, enabledStr)
	if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write jail config: %w", err)
	}

	// Reload fail2ban to pick up the change.
	if out, err := exec.Command("fail2ban-client", "reload").CombinedOutput(); err != nil {
		return fmt.Errorf("fail2ban reload: %w: %s", err, out)
	}
	return nil
}
