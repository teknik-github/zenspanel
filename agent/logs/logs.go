package logs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// allowedLogDirs lists the only directories from which the agent will
// serve log lines. Any path that does not resolve under one of these
// prefixes is rejected (V30).
var allowedLogDirs = []string{
	"/var/log/nginx/",
	"/var/log/php",  // covers /var/log/php8.1-fpm.log, /var/log/php8.3-fpm/ etc.
}

const maxLines = 500 // V31

// Tail returns the last `lines` lines of logPath. Returns an error if
// logPath resolves outside the allowed log directories (V30) or if
// lines > maxLines (V31).
func Tail(logPath string, lines int) ([]string, error) {
	if lines <= 0 {
		lines = 100
	}
	if lines > maxLines {
		return nil, fmt.Errorf("lines must be ≤ %d (got %d)", maxLines, lines)
	}

	// Resolve symlinks so a crafted path like /var/log/nginx/../../etc/passwd
	// can't escape the jail (V30).
	resolved, err := filepath.EvalSymlinks(logPath)
	if err != nil {
		// File may not exist yet (log rotation, fresh domain) — return empty.
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("resolve log path: %w", err)
	}
	resolved = filepath.Clean(resolved)

	allowed := false
	for _, dir := range allowedLogDirs {
		if strings.HasPrefix(resolved, dir) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("log path %q is outside allowed directories", logPath)
	}

	f, err := os.Open(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("open log: %w", err)
	}
	defer f.Close()

	// Read all lines into a ring buffer of size `lines` so we only keep
	// the tail without loading the whole file into memory.
	ring := make([]string, lines)
	pos := 0
	count := 0
	sc := bufio.NewScanner(f)
	// Increase scanner buffer for long log lines (e.g. nginx access with
	// large query strings).
	sc.Buffer(make([]byte, 256*1024), 256*1024)
	for sc.Scan() {
		ring[pos%lines] = sc.Text()
		pos++
		count++
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan log: %w", err)
	}

	if count == 0 {
		return []string{}, nil
	}

	// Reconstruct in order from the ring buffer.
	result := make([]string, 0, lines)
	if count <= lines {
		result = append(result, ring[:count]...)
	} else {
		start := pos % lines
		for i := 0; i < lines; i++ {
			result = append(result, ring[(start+i)%lines])
		}
	}
	return result, nil
}
