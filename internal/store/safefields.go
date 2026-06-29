package store

import "fmt"

// allowedUserUpdate lists columns the API may update on the users table via
// the dynamic Update path. Anything not listed is silently dropped — the only
// safe way to defend against attacker-controlled JSON keys flowing into raw
// SQL identifiers (which cannot be parameterized).
var allowedUserUpdate = map[string]struct{}{
	"email":            {},
	"role":             {},
	"package_id":       {},
	"status":           {},
	"terminal_enabled":  {},
	"backup_enabled":    {},
	"antivirus_enabled": {},
	"php_version":      {},
}

// allowedDomainUpdate lists columns the API may update on the domains table.
var allowedDomainUpdate = map[string]struct{}{
	"php_version":    {},
	"document_root":  {},
	"ssl_type":       {},
	"ssl_expires_at": {},
	"status":         {},
}

// allowedSubdomainUpdate mirrors allowedDomainUpdate — subdomains share the
// same mutable surface (PHP version, docroot, SSL state, status). subdomain
// label and parent_domain_id are intentionally not in this list: changing
// them would invalidate the fqdn and detach the row from its parent's
// cascade-delete chain.
var allowedSubdomainUpdate = map[string]struct{}{
	"php_version":    {},
	"document_root":  {},
	"ssl_type":       {},
	"ssl_expires_at": {},
	"status":         {},
}

// allowedUserSort lists columns the API may use in ORDER BY for the users
// list endpoint. ORDER BY identifiers cannot be parameterized; whitelisting
// is the only safe option.
var allowedUserSort = map[string]struct{}{
	"id":         {},
	"username":   {},
	"email":      {},
	"created_at": {},
	"updated_at": {},
	"status":     {},
}

// allowedCronJobUpdate lists columns the API may update on the cron_jobs table.
var allowedCronJobUpdate = map[string]struct{}{
	"expression": {},
	"command":    {},
	"enabled":    {},
}
// allowlist. Used to sanitize attacker-controlled JSON before it lands in
// dynamic SQL.
func filterAllowed(fields map[string]interface{}, allowed map[string]struct{}) map[string]interface{} {
	out := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		if _, ok := allowed[k]; ok {
			out[k] = v
		}
	}
	return out
}

// safeSort returns sort if present in the allowlist, otherwise fallback.
func safeSort(sort, fallback string, allowed map[string]struct{}) string {
	if _, ok := allowed[sort]; ok {
		return sort
	}
	return fallback
}

// errNoAllowedFields is returned by Update helpers when every supplied key was
// rejected by the allowlist. Callers should treat this as a no-op rather than
// surface it to the API caller as 500.
var errNoAllowedFields = fmt.Errorf("no allowed fields to update")
