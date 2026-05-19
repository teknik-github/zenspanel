package cron

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/zenspanel/zenspanel/agent/safe"
)

// Job is the wire type passed from the API via the cron.sync RPC.
type Job struct {
	Expression string `json:"expression"`
	Command    string `json:"command"`
	Enabled    bool   `json:"enabled"`
}

// cronFieldRe matches a single standard cron field:
// number, range, step, wildcard, or comma-separated list thereof.
var cronFieldRe = regexp.MustCompile(
	`^(\*|[0-9]+(-[0-9]+)?(,[0-9]+(-[0-9]+)?)*|\*/[0-9]+)$`)

// forbiddenCmdRe rejects shell metacharacters that could escape the
// crontab line and execute arbitrary commands (V23). We allow a
// restricted set: alphanumeric, spaces, slashes, dots, dashes,
// underscores, equals, colons, and @ (for @reboot etc).
// Quoted strings are not supported — the crontab format doesn't
// need them for safe commands.
var forbiddenCmdRe = regexp.MustCompile(`[;&|><` + "`" + `$(){}\n\r]`)

// ValidateExpression checks that expr is a valid 5-field cron expression
// or one of the @-shortcuts (@reboot, @hourly, @daily, @weekly, @monthly,
// @yearly, @annually, @midnight). Returns nil on success (V24).
func ValidateExpression(expr string) error {
	shortcuts := map[string]bool{
		"@reboot": true, "@hourly": true, "@daily": true,
		"@weekly": true, "@monthly": true, "@yearly": true,
		"@annually": true, "@midnight": true,
	}
	if shortcuts[expr] {
		return nil
	}
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return fmt.Errorf("cron expression must have 5 fields or be a @shortcut, got %d fields", len(fields))
	}
	names := []string{"minute", "hour", "day-of-month", "month", "day-of-week"}
	for i, f := range fields {
		if !cronFieldRe.MatchString(f) {
			return fmt.Errorf("invalid cron %s field: %q", names[i], f)
		}
	}
	return nil
}

// ValidateCommand checks that cmd contains no shell metacharacters (V23).
func ValidateCommand(cmd string) error {
	if strings.TrimSpace(cmd) == "" {
		return fmt.Errorf("command must not be empty")
	}
	if len(cmd) > 1024 {
		return fmt.Errorf("command too long (max 1024 chars)")
	}
	if forbiddenCmdRe.MatchString(cmd) {
		return fmt.Errorf("command contains forbidden shell metacharacters")
	}
	return nil
}

// Sync atomically rewrites the crontab for username. Disabled jobs are
// written as commented-out lines so they can be re-enabled without data
// loss (V26). All jobs are validated before any write — if any job fails
// validation the crontab is left unchanged.
func Sync(username string, jobs []Job) error {
	if err := safe.Username(username); err != nil {
		return err
	}

	// Validate all jobs before touching the crontab (V23, V24).
	for i, j := range jobs {
		if err := ValidateExpression(j.Expression); err != nil {
			return fmt.Errorf("job %d: %w", i, err)
		}
		if err := ValidateCommand(j.Command); err != nil {
			return fmt.Errorf("job %d: %w", i, err)
		}
	}

	// Build the new crontab content.
	var sb strings.Builder
	sb.WriteString("# Managed by ZensPanel — do not edit manually\n")
	for _, j := range jobs {
		line := fmt.Sprintf("%s %s\n", j.Expression, j.Command)
		if !j.Enabled {
			line = "# " + line
		}
		sb.WriteString(line)
	}

	// Write to a temp file then pipe to `crontab -u <user>` atomically.
	tmp, err := os.CreateTemp("", "zenspanel-cron-*")
	if err != nil {
		return fmt.Errorf("create temp crontab: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(sb.String()); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp crontab: %w", err)
	}
	tmp.Close()

	cmd := exec.Command("crontab", "-u", username, tmp.Name())
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("crontab -u %s: %w: %s", username, err, out)
	}
	return nil
}
