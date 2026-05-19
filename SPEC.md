# SPEC

## §G GOAL

subdomain support: user create/manage `<sub>.<parent-domain>` via panel, reuse nginx vhost + FPM pool infra.

## §C CONSTRAINTS

- lang: Go 1.22 (api+agent), Vue 3 + Tailwind (frontend)
- arch: api ↔ agent via JSON-RPC unix socket. !  add agent calls thru `internal/agent/client.go`
- DB: MySQL/MariaDB. ! migration via `migrations/`, numbered monotonic
- nginx: vhost per FQDN @ `/etc/nginx/zenspanel/<fqdn>.conf`. reuse `agent/nginx.CreateVhost`
- PHP-FPM: pool per user (! per subdomain). socket `/run/php/zenspanel-<user>-<phpver>.sock`
- file jail: subdomain docroot ! ⊂ `/var/lib/zenspanel/home/<user>/`
- SSL: per-FQDN. reuse `agent/ssl.IssueLetsEncrypt` + Let's Encrypt rate limit (50 cert/week/parent-domain)
- ! ⊥ subdomain on parent-domain user ≠ own
- backwards compat: existing `domains` table rows = parent domains. subdomains = new rows w/ `parent_id` FK

## §I INTERFACES

### api

- `POST /api/v1/subdomains` JWT
  body: `{parent_domain_id: u64, subdomain: str, php_version: str, doc_root?: str}`
  → 201 `{id, domain, document_root, php_version, status}` | 400 invalid | 403 not owner | 409 exists
- `GET /api/v1/subdomains?parent_id=<id>` JWT → 200 `{data: [...]}`
- `GET /api/v1/subdomains/:id` JWT → 200 row | 404
- `PUT /api/v1/subdomains/:id` JWT body `{php_version?, doc_root?}` → 200
- `DELETE /api/v1/subdomains/:id` JWT → 200

### agent rpc

(reuse, no new RPCs)
- `nginx.create_vhost {domain, username, php_version, doc_root}`
- `nginx.delete_vhost {domain}`
- `ssl.issue_letsencrypt {domain, email, staging}`

### db

migration `000011_create_subdomains.up.sql`:
```sql
CREATE TABLE subdomains (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  parent_domain_id BIGINT UNSIGNED NOT NULL,
  subdomain VARCHAR(63) NOT NULL,        -- label only, eg "blog"
  fqdn VARCHAR(253) NOT NULL UNIQUE,     -- "blog.example.com"
  document_root VARCHAR(512) NOT NULL,
  php_version VARCHAR(8) NOT NULL,
  ssl_type VARCHAR(32) NOT NULL DEFAULT 'none',
  ssl_expires_at DATETIME NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (parent_domain_id) REFERENCES domains(id) ON DELETE CASCADE,
  UNIQUE KEY uk_parent_sub (parent_domain_id, subdomain)
);
```

### frontend

- `frontend/apps/user/src/pages/Domains.vue`: per-row `+ Subdomain` button → modal
- modal fields: `subdomain` (label only), `php_version` select, `doc_root` (default `public_html/<sub>.<parent>`)
- per-row expandable subdomain list under each parent

## §V INVARIANTS

V1: ∀ subdomain create → parent_domain.user_id == requester.user_id (! admin bypass)
V2: subdomain label ! match `^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$` (RFC 1035)
V3: fqdn = `<subdomain>.<parent.domain>`. fqdn UNIQUE @ DB ! collide w/ `domains.domain`
V4: doc_root ! ⊂ `homeBase + "/" + username + "/"`. resolve symlinks before check
V5: nginx vhost write @ `<nginx_conf>/<fqdn>.conf` ! ⊥ overwrite parent .conf
V6: ! cascade delete: parent domain delete → all subdomain rows + nginx vhosts + SSL certs deleted
V7: ! reserved labels rejected: `www, mail, smtp, imap, pop, ns, ns1, ns2, _dmarc, _domainkey`
V8: SSL issue → Let's Encrypt rate-aware. ! per-fqdn cert (no wildcard yet)
V9: php_version ! ∈ enabled list (`php_versions` where `enabled=true`)
V10: agent nginx.delete_vhost failure on subdomain delete → row stays (! orphan row block recreate)
V11: subdomain status flip → suspended/active via existing `nginx.suspend_vhost` (parity w/ parent domains)
V12: ∀ JWT route on `/users/:id/*` → ! ownership check `role=="admin" | id==self`. ! exception `users.GetUsage`
V13: cookie auth `zenspanel_token` ! `Secure=true` & `SameSite=Strict` (HTTP plain ⊥ in prod)
V14: WS upgrader `CheckOrigin` ! same-origin (Host header == r.Host) | explicit allowlist. ⊥ `return true`
V15: Update handlers w/ free-form fields ! re-validate jail/safety on every mutating field (ex: `document_root` jail check ! Create-only)
V16: ownership pattern ! `role == "user" && id != self → 403`. ⊥ `role != "admin"` (api_key falls through w/ id=0)
V17: `/terminal/token` ! per-user rate limit (≥ 1 req/sec/user)

## §T TASKS

| id | status | task | cites |
|----|--------|------|-------|
| T1 | x | migration 000011 create subdomains table | I.db |
| T2 | x | `internal/store/subdomains.go`: Subdomain model + SubdomainStore (Create/Get/List/ListByParent/ListByUser/Update/Delete) | I.db,V3 |
| T3 | x | `internal/api/handlers/subdomains.go`: SubdomainHandler full CRUD | I.api,V1,V2,V4,V7,V9 |
| T4 | x | wire SubdomainHandler @ `cmd/api/main.go` + `internal/api/router.go` (5 routes) | I.api |
| T5 | x | api validation: subdomain regex, reserved labels, ownership check, fqdn collision | V1,V2,V7 |
| T6 | x | api Create: agent `nginx.create_vhost` + `phpfpm.create_pool` (idempotent) + `ensureDocRoot` | V4,V9 |
| T7 | x | api Update: re-issue vhost on php_version OR doc_root change (mirror domains.Update logic) | V9 |
| T8 | x | api Delete: agent `nginx.delete_vhost` → `ssl.remove_cert` → row delete | V6,V10 |
| T9 | x | parent domain delete cascade: extend `DomainHandler.Delete` to enumerate subdomains, delete each | V6 |
| T10 | x | extend `users.Delete` teardown: also list+delete subdomains per user | V6 |
| T11 | x | extend SSL handler: accept subdomain id (path differentiator or new route `/subdomains/:id/ssl`) | V8 |
| T12 | x | `frontend/apps/user/src/api/subdomains.ts`: 5-method client | I.api |
| T13 | x | `Domains.vue`: per-row expand toggle → list subdomains + "+ Subdomain" button | I.frontend |
| T14 | x | Subdomain modal: validate label client-side, php select, doc_root default = `public_html/<fqdn>` | V2,V7 |
| T15 | x | subdomain row actions: Manage Files (deep-link), Delete confirm, SSL Manager link | V6,V8 |
| T16 | x | extend SSL Manager page to show subdomains alongside parent domains | V8 |
| T17 | . | add subdomain count to user usage / package limits (optional, gate behind package.max_subdomains) | C |
| T18 | x | end-to-end test: create user → parent domain → subdomain → curl FQDN → 200 OK | V1-V10 |
| T19 | x | fix `users.GetUsage` IDOR: add `role=="admin" \| id==self` ownership check @ `internal/api/handlers/users.go:GetUsage` | V12 |
| T20 | x | fix WS CSWSH: tighten `upgrader.CheckOrigin` @ `internal/api/handlers/terminal.go` to same-host (compare Origin → r.Host) | V14 |
| T21 | x | fix subdomain Update jail bypass: re-run docroot-jail check when `document_root` ∈ payload @ `internal/api/handlers/subdomains.go:Update` | V15 |
| T22 | x | harden cookie: set `Secure=true` (when scheme=https) + `SameSite=Strict` @ `internal/api/handlers/auth.go:Login` cookie set | V13 |
| T23 | x | rate-limit `/terminal/token`: per-user limiter (sync.Map or Redis), 1 req/sec/user, 429 on exceed | V17 |
| T24 | x | refactor ownership pattern: replace `role != "admin"` w/ explicit `role == "user" && id != self` across all handlers | V16 |

## §B BUGS

| id | date | cause | fix |
|----|------|-------|-----|
| B1 | 2026-05-19 | `users.GetUsage` ! ownership check ∴ ∀ user can read other users' RAM/disk/CPU/quota via `:id` enumeration. CRITICAL IDOR | V12 |
| B2 | 2026-05-19 | WS `upgrader.CheckOrigin: return true` + cookie auth ∴ cross-origin shell hijack possible (CSWSH). HIGH | V14 |
| B3 | 2026-05-19 | subdomain `Update` accepts free-form `document_root` w/o re-running jail check ∴ DB drift to `/etc/...`; later php_version change writes vhost outside user home. MEDIUM | V15 |
| B4 | 2026-05-19 | login cookie `Secure=false` & `SameSite=Lax` ∴ JWT exposed on plain-HTTP & cross-site state-changing requests reachable. HIGH | V13 |
| B5 | 2026-05-19 | `/terminal/token` ! rate limit ∴ token enumeration / spam possible. MEDIUM | V17 |
| B6 | 2026-05-19 | ownership pattern `role != "admin"` ∴ api_key callers (id=0) fall through to "owner" branch on future routes. brittle, current routes safe-by-accident. MEDIUM | V16 |
