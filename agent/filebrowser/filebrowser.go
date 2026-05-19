package filebrowser

import (
	"fmt"
	"os/exec"

	"github.com/zenspanel/zenspanel/agent/safe"
)

// Default DB path. Match the systemd unit so we operate on the same
// database FileBrowser is serving from.
const DefaultDB = "/var/lib/zenspanel/filebrowser.db"

// CreateUser provisions a FileBrowser user scoped to <homeBase>/<username>/.
// Used immediately after we successfully add a Linux user via agent.user.create
// so the panel user can land in FileBrowser and only see their own files.
//
// FileBrowser's proxy auth maps the X-Auth-User header to a user record;
// if no record exists for that name the request falls back to the default
// admin which sees everything. Pre-creating the user with the right
// scope is the only way to get per-user isolation under proxy auth.
//
// Idempotent: re-running with an existing username returns success-ish
// from the underlying command; we tolerate non-zero exit because the
// expected error is "user already exists" which is exactly the state we
// want to end up in.
func CreateUser(username, homeBase string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	scope := homeBase + "/" + username
	// FileBrowser still requires a password column even under proxy
	// auth. We set a long random one — it's never used because the
	// proxy header is the credential.
	password := username + "-no-login"

	cmd := exec.Command("/usr/local/bin/filebrowser",
		"--database", DefaultDB,
		"users", "add", username, password,
		"--scope", scope,
		"--perm.admin=false",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// "user already exists" is acceptable — that means a previous
		// run already provisioned them. Anything else is a real error.
		s := string(out)
		if containsAny(s, "already exists", "AlreadyExists", "duplicate") {
			return nil
		}
		return fmt.Errorf("filebrowser users add: %w: %s", err, s)
	}
	return nil
}

// DeleteUser removes the FileBrowser user record. Called from
// agent.user.delete so a recreated panel user with the same name
// doesn't inherit the previous occupant's session/scope.
func DeleteUser(username string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	cmd := exec.Command("/usr/local/bin/filebrowser",
		"--database", DefaultDB,
		"users", "rm", username,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		s := string(out)
		if containsAny(s, "not found", "no rows", "doesn't exist") {
			return nil
		}
		return fmt.Errorf("filebrowser users rm: %w: %s", err, s)
	}
	return nil
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if indexOf(haystack, n) >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
