package handlers

import (
	"fmt"
	"regexp"
)

// usernameRe matches a Linux username we are willing to provision: lowercase
// alphanumerics and underscore, 3-32 chars, must start with a letter. We are
// stricter than POSIX because the username feeds useradd, MySQL identifiers,
// nginx vhost paths and cgroup paths — every step has its own allowed-charset
// surprises. Whitelisting the intersection avoids surprises.
var usernameRe = regexp.MustCompile(`^[a-z][a-z0-9_]{2,31}$`)

// reservedUsernames lists names that exist as system or service accounts on a
// fresh Ubuntu install. Provisioning a panel user with these names would
// either fail useradd or worse, hijack a system account.
var reservedUsernames = map[string]struct{}{
	"root": {}, "daemon": {}, "bin": {}, "sys": {}, "sync": {}, "games": {},
	"man": {}, "lp": {}, "mail": {}, "news": {}, "uucp": {}, "proxy": {},
	"www-data": {}, "backup": {}, "list": {}, "irc": {}, "gnats": {},
	"nobody": {}, "systemd-network": {}, "systemd-resolve": {}, "messagebus": {},
	"sshd": {}, "syslog": {}, "_apt": {}, "uuidd": {}, "tcpdump": {},
	"landscape": {}, "pollinate": {}, "fwupd-refresh": {}, "usbmux": {},
	"mysql": {}, "redis": {}, "nginx": {}, "ubuntu": {},
	"zenspanel": {}, "admin": {}, "phpmyadmin": {},
}

// ValidateUsername enforces the username constraints described above.
func ValidateUsername(u string) error {
	if !usernameRe.MatchString(u) {
		return fmt.Errorf("username must be 3-32 chars, start with a letter, lowercase a-z 0-9 _ only")
	}
	if _, reserved := reservedUsernames[u]; reserved {
		return fmt.Errorf("username %q is reserved", u)
	}
	return nil
}

// domainRe matches a hostname we are willing to provision. RFC 1035 labels
// (lowercase, hyphen-separated, no leading/trailing hyphen), 2-253 chars
// total, at least one dot.
var domainRe = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)

// ValidateDomain rejects any hostname that nginx or the filesystem could
// misinterpret. Rejecting here, before the row hits the DB or DocumentRoot is
// constructed, avoids path traversal via dotted segments.
func ValidateDomain(d string) error {
	if len(d) < 2 || len(d) > 253 {
		return fmt.Errorf("domain length must be 2-253 chars")
	}
	if !domainRe.MatchString(d) {
		return fmt.Errorf("domain %q is not a valid hostname", d)
	}
	return nil
}

// subdomainLabelRe matches a single DNS label per RFC 1035: 1–63 chars,
// lowercase alphanumeric or hyphen, ! starts/ends with hyphen.
var subdomainLabelRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// reservedSubdomainLabels are labels that conflict with conventional DNS
// records (mail, _dmarc, etc.). Letting users register these as panel
// subdomains would shadow legitimate infra-level records.
var reservedSubdomainLabels = map[string]struct{}{
	"www":  {}, "mail": {}, "smtp": {}, "imap": {}, "pop": {},
	"ns": {}, "ns1": {}, "ns2": {},
	"_dmarc": {}, "_domainkey": {},
}

// ValidateSubdomainLabel checks the leftmost label of a subdomain against
// RFC 1035 and our reserved-label list. Caller is responsible for joining
// it with the parent domain.
func ValidateSubdomainLabel(label string) error {
	if len(label) < 1 || len(label) > 63 {
		return fmt.Errorf("subdomain label length must be 1-63 chars")
	}
	if !subdomainLabelRe.MatchString(label) {
		return fmt.Errorf("subdomain label %q is not a valid DNS label (lowercase alphanumeric, hyphens, ! start/end w/ hyphen)", label)
	}
	if _, ok := reservedSubdomainLabels[label]; ok {
		return fmt.Errorf("subdomain label %q is reserved", label)
	}
	return nil
}
