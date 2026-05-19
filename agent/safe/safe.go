package safe

import (
	"fmt"
	"regexp"
)

// usernameRe is the agent-side mirror of the API username allowlist. The
// agent must NOT trust that callers validated their input — it runs as root.
var usernameRe = regexp.MustCompile(`^[a-z][a-z0-9_]{2,31}$`)

// dbIdentRe matches MySQL identifiers we accept for db_name / db_user.
// MySQL allows much more, but we restrict to the conservative subset that
// can be safely interpolated into backtick-quoted SQL with no escape pass.
var dbIdentRe = regexp.MustCompile(`^[a-zA-Z0-9_]{1,63}$`)

// dbPasswordRe matches passwords we accept for MySQL CREATE USER. The
// password is wrapped in single quotes inside the SQL statement, so we
// reject every byte that could break out of that quoting (',\,;) plus the
// non-printables that have no business being in a password anyway.
var dbPasswordRe = regexp.MustCompile(`^[A-Za-z0-9._\-!@#$%^&*()+=]{8,128}$`)

// domainRe is the agent-side mirror of the API domain allowlist.
var domainRe = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)

// phpVersionRe matches the few values we ever pass to systemctl reload
// php<ver>-fpm or interpolate into pool paths.
var phpVersionRe = regexp.MustCompile(`^[0-9]\.[0-9]$`)

// extNameRe matches PHP extension names we accept. Restricted to
// lowercase alphanumeric + underscore — no dots, slashes, or spaces
// that could escape the ini file or the filesystem path (V19).
var extNameRe = regexp.MustCompile(`^[a-z0-9_]+$`)

func Username(u string) error {
	if !usernameRe.MatchString(u) {
		return fmt.Errorf("agent: invalid username %q", u)
	}
	return nil
}

func DBIdent(s string) error {
	if !dbIdentRe.MatchString(s) {
		return fmt.Errorf("agent: invalid mysql identifier %q", s)
	}
	return nil
}

func DBPassword(p string) error {
	if !dbPasswordRe.MatchString(p) {
		return fmt.Errorf("agent: mysql password contains forbidden characters or is too short/long")
	}
	return nil
}

func Domain(d string) error {
	if len(d) < 2 || len(d) > 253 || !domainRe.MatchString(d) {
		return fmt.Errorf("agent: invalid domain %q", d)
	}
	return nil
}

func PHPVersion(v string) error {
	if !phpVersionRe.MatchString(v) {
		return fmt.Errorf("agent: invalid php version %q", v)
	}
	return nil
}

func ExtName(e string) error {
	if len(e) == 0 || len(e) > 64 || !extNameRe.MatchString(e) {
		return fmt.Errorf("agent: invalid extension name %q", e)
	}
	return nil
}
