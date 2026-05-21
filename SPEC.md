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
- `GET /api/v1/admin/php-extensions` admin JWT → 200 `{data: [{id, name, php_version, enabled}]}`
- `PUT /api/v1/admin/php-extensions/:id` admin JWT body `{enabled: bool}` → 200
- `GET /api/v1/php-extensions` user JWT → 200 `{data: [{name, php_version, admin_enabled, user_enabled}]}`
- `PUT /api/v1/php-extensions` user JWT body `{name: str, php_version: str, enabled: bool}` → 200 | 403 admin-disabled

### agent rpc

(reuse, no new RPCs for subdomains)
- `nginx.create_vhost {domain, username, php_version, doc_root}`
- `nginx.delete_vhost {domain}`
- `ssl.issue_letsencrypt {domain, email, staging}`
- `phpfpm.enable_extension {username, php_version, ext_name}` — write per-user ext ini, reload pool
- `phpfpm.disable_extension {username, php_version, ext_name}` — remove per-user ext ini, reload pool

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

migration `000013_php_extensions.up.sql`:
```sql
-- global catalog: admin controls which exts are available + default state
CREATE TABLE php_extensions (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(64) NOT NULL,
  php_version VARCHAR(8) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  UNIQUE KEY uk_ext_ver (name, php_version)
);
-- per-user override: only rows where user differs from global default
CREATE TABLE user_php_extensions (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  ext_id BIGINT UNSIGNED NOT NULL,
  enabled BOOLEAN NOT NULL,
  UNIQUE KEY uk_user_ext (user_id, ext_id),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (ext_id) REFERENCES php_extensions(id) ON DELETE CASCADE
);
```

### frontend

- `frontend/apps/user/src/pages/Domains.vue`: per-row `+ Subdomain` button → modal
- modal fields: `subdomain` (label only), `php_version` select, `doc_root` (default `public_html/<sub>.<parent>`)
- per-row expandable subdomain list under each parent
- `frontend/apps/admin/src/pages/PhpExtensions.vue`: table of all exts grouped by php version, toggle per row
- `frontend/apps/user/src/pages/PhpSettings.vue`: "Extensions" section — toggle per ext (admin-allowed only)

### firewall / IP security

- `GET  /api/v1/admin/firewall/blocked`              admin JWT → 200 `{data: [{ip, reason, blocked_at, source}]}`
- `POST /api/v1/admin/firewall/block`                admin JWT body `{ip, reason?}` → 200
- `POST /api/v1/admin/firewall/unblock`              admin JWT body `{ip}` → 200
- `GET  /api/v1/admin/firewall/fail2ban/jails`       admin JWT → 200 `{data: [{name, enabled, ban_count, currently_banned}]}`
- `PUT  /api/v1/admin/firewall/fail2ban/jails/:name` admin JWT body `{enabled: bool}` → 200
- agent rpc `firewall.list_blocked`                  → `{ips: [{ip, reason, source}]}`
- agent rpc `firewall.block   {ip, reason}`          → nil  (ipset add + iptables rule)
- agent rpc `firewall.unblock {ip}`                  → nil  (ipset del)
- agent rpc `fail2ban.list_jails`                    → `{jails: [{name, enabled, ban_count, currently_banned}]}`
- agent rpc `fail2ban.set_jail {name, enabled}`      → nil  (edit jail.d config + fail2ban-client reload)

### cron jobs

- `GET /api/v1/cron-jobs` user JWT → 200 `{data: [{id, expression, command, enabled, last_run_at}]}`
- `POST /api/v1/cron-jobs` user JWT body `{expression, command, enabled?}` → 201 | 400 | 429 (quota)
- `PUT /api/v1/cron-jobs/:id` user JWT body `{expression?, command?, enabled?}` → 200
- `DELETE /api/v1/cron-jobs/:id` user JWT → 200
- agent rpc `cron.sync {username, jobs:[{expression,command,enabled}]}` — rewrites user crontab atomically

### 2fa

- `POST /api/v1/auth/2fa/setup` JWT → 200 `{secret, qr_url, recovery_codes[8]}`
- `POST /api/v1/auth/2fa/confirm` JWT body `{code}` → 200 (activates 2FA) | 400 bad code
- `DELETE /api/v1/auth/2fa` JWT body `{code}` → 200 (disables 2FA)
- `POST /api/v1/auth/login` — if user has 2FA: return `{requires_2fa: true, temp_token}` instead of full JWT; client must POST `{temp_token, code}` to `/auth/2fa/verify` → full JWT
- `POST /api/v1/auth/2fa/verify` public body `{temp_token, code}` → 200 `{token, user}` | 401
- `POST /api/v1/auth/2fa/recover` public body `{temp_token, recovery_code}` → 200 `{token, user}` | 401

### logs

- `GET /api/v1/domains/:id/logs?type=nginx|fpm&lines=100` JWT → 200 `{lines: [str]}`
- agent rpc `logs.tail {log_path, lines}` → `{lines: [str]}`

### installer

- `GET /api/v1/installer/apps` JWT → 200 `{data: [{id, name, version, description}]}`
- `POST /api/v1/installer/install` JWT body `{app_id, domain_id, db_name?, db_user?, db_pass?, overwrite?}` → 202 `{job_id}`
- `GET /api/v1/installer/status/:job_id` JWT → 200 `{phase, log, done, error}`
- agent rpc `installer.run {app_id, username, docroot, db_name, db_user, db_pass}` — async, streams status via sync.Map

### app installer (softaculous-style)

- `GET /api/v1/installer/apps` — existing, extend catalog: add Joomla, Drupal, PrestaShop, CodeIgniter, plain PHP
- installer catalog: `{id, name, version, description, requires_db, download_url}`
- agent: each app has `install_<app>()` func; WordPress already done; add others

### redirect manager

- `GET  /api/v1/domains/:id/redirects`          user JWT → 200 `{data: [{id, source_path, dest_url, type, enabled}]}`
- `POST /api/v1/domains/:id/redirects`          user JWT body `{source_path, dest_url, type: "301"|"302", enabled?}` → 201
- `PUT  /api/v1/domains/:id/redirects/:rid`     user JWT → 200
- `DELETE /api/v1/domains/:id/redirects/:rid`   user JWT → 200
- agent rpc `nginx.sync_redirects {domain, redirects:[{source_path,dest_url,type}]}` — rewrite redirect block in vhost

### hotlink protection

- `GET  /api/v1/domains/:id/hotlink`            user JWT → 200 `{enabled, allowed_domains: [str]}`
- `PUT  /api/v1/domains/:id/hotlink`            user JWT body `{enabled: bool, allowed_domains: [str]}` → 200
- agent rpc `nginx.set_hotlink {domain, enabled, allowed_domains}` — write valid_referers block in vhost

### antivirus realtime

- agent rpc `antivirus.watch_start {username}` → `{watch_id}` — inotifywait on user home, scan new/modified files
- agent rpc `antivirus.watch_stop {watch_id}` → nil
- `GET /api/v1/antivirus/alerts` user JWT → 200 `{data: [{id, path, threat, detected_at}]}`
- WS push: `{type:"antivirus_alert", path, threat}` on detection

### admin terminal

- `POST /api/v1/admin/terminal/token` admin JWT body `{username?}` → 200 `{token}` (username empty = zenspanel system user)
- WS `/ws/terminal` reuse existing — token carries admin role + target username

### s3 backup

- `GET  /api/v1/admin/backup-targets`              admin JWT → 200 `{data: [{id, name, type, bucket, prefix, enabled}]}`
- `POST /api/v1/admin/backup-targets`              admin JWT body `{name, type, bucket, prefix, access_key, secret_key_enc, region, endpoint?}` → 201
- `PUT  /api/v1/admin/backup-targets/:id`          admin JWT → 200
- `DELETE /api/v1/admin/backup-targets/:id`        admin JWT → 200
- `POST /api/v1/admin/backup-targets/:id/test`     admin JWT → 200 `{ok, error?}`
- agent rpc `backup.upload_s3 {backup_path, target_id}` — upload file to S3-compatible target

### package MB units

- `POST/PUT /api/v1/packages` — `disk_quota_mb` + `memory_limit_mb` fields (MB integers). API converts → bytes before storing. existing `disk_quota`/`memory_limit` (bytes) kept in DB unchanged

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
V18: php ext toggle ! affect only target php ver pool. ⊥ cross-user side-effect
V19: ext enable/disable ! validate ext name `^[a-z0-9_]+$`. ⊥ shell injection via ext name
V20: admin-disabled ext ! user cannot re-enable. user toggle ⊂ admin-allowed set
V21: ext change → php-fpm pool reload (! restart). ⊥ kill live requests
V22: ext state stored per-user per-phpver in DB. agent reads DB state on pool create/reload
V23: cron job command ! shell metachar injection. validate: printable ASCII only, no `; & | > < ` $ ( ) { }` outside quotes
V24: cron expr ! validated server-side (5-field standard cron). ⊥ accept free-form string
V25: max_cron_jobs per user ≤ package.max_cron_jobs. 0 = unlimited
V26: disabled cron job → commented out in crontab (`#`-prefix). ! deleted. re-enable restores
V27: 2FA TOTP secret stored encrypted @ rest (AES-256-GCM, key from config). ⊥ plaintext in DB
V28: 2FA enforce: if user has 2FA enabled → every login requires valid TOTP code. ⊥ bypass via API key
V29: 2FA recovery codes: 8 single-use codes generated at setup. each code hashed in DB (bcrypt). consumed on use
V30: log viewer ! serve files outside `/var/log/nginx/` (nginx) or `/var/log/php<ver>-fpm/` (fpm). path jail
V31: log viewer ! tail only (last N lines). ⊥ stream full file. max 500 lines per request
V32: installer ! overwrite existing files in docroot w/o explicit `overwrite=true`. ⊥ silent data loss
V33: installer runs as Linux user (agent drops privs via `su -s /bin/sh -c ... <user>`). ⊥ root-owned files in home
V34: IP block/unblock ! exec via `iptables`/`ipset` arg array. ⊥ shell string interpolation. validate IP/CIDR before any exec
V35: fail2ban jail config ! written outside `/etc/fail2ban/jail.d/`. ⊥ path traversal via jail name
V36: blocked IP list read from `iptables -L` + `ipset list` — agent is authoritative. ⊥ panel DB as source of truth
V37: unblock ! require confirmation token (admin-only route). ⊥ accidental mass-unblock via stale UI state
V38: WS CheckOrigin ! check X-Forwarded-Host when behind reverse proxy. ⊥ reject valid same-origin WS from proxied installs
V39: fail2ban banned IPs ! merged into firewall blocked list w/ source="fail2ban". ⊥ separate UI for panel vs fail2ban bans
V40: antivirus scan ! run as panel user (! root). scan path ! ⊂ user home jail. ⊥ scan arbitrary filesystem paths
V41: DB isolation: each panel user's MySQL databases ! accessible only by that user's MySQL account. ⊥ cross-user DB access
V42: disk quota enforcement ! use package.disk_quota as hard limit. quota applied at user create + package change. 0 = unlimited
V43: installer `runAs` ! build shell string via fmt.Sprintf w/ untrusted path. use exec.Command arg array OR validate path contains no shell metachar before interpolation
V44: phpext AdminUpdate disable → ! call agent `phpfpm.reload` for every user who has ext enabled. ⊥ silent no-op on global disable
V45: "Login as" URL ! use user panel base path (served at `/`). ⊥ hardcode `/user/` prefix
V46: antivirus realtime → inotifywait watch on user home. ! scan outside user home jail (V40 extends). alert stored in DB + pushed via WS
V47: S3/remote backup ! store credentials in DB plaintext. encrypt w/ AES-256-GCM (same pattern as TOTP key V27). support S3-compatible + rclone targets
V48: admin terminal ! run as root. spawn bash as `zenspanel` system user or specific panel user. ⊥ arbitrary root shell
V49: package disk_quota + memory_limit ! stored + displayed in MB in UI. converted to bytes before passing to agent (×1024×1024). ⊥ raw bytes in form fields
V50: NPROC limit via cgroup pids.max. default 200. ⊥ fork bomb. 0 = unlimited
V51: I/O throttle via cgroup io.max. 0 = unlimited. unit MB/s in UI → bytes/s in agent
V52: installer app catalog ! hardcode URLs. store version + download URL in catalog struct. ⊥ broken installs on upstream rename
V53: redirect rule source ! ⊂ user's own domains. ⊥ user redirect other users' domains. external destination OK
V54: hotlink protection ! affect API/WS paths. only static asset extensions (jpg,png,gif,css,js,woff,etc). ⊥ break panel
V55: nginx config write for redirect/hotlink ! shell string interpolation. use text/template + safe.Domain. ⊥ nginx config injection

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
| T25 | x | migration 000013: `php_extensions` table (global catalog) + `user_php_extensions` table (per-user override state) | I.db |
| T26 | x | `internal/store/phpextensions.go`: PHPExtension + UserPHPExtension models + store (List, SetGlobal, GetUserState, SetUserState) | I.db,V18,V20 |
| T27 | x | `agent/phpfpm/phpfpm.go`: `EnableExtension(username, phpVer, extName)` + `DisableExtension(...)` — write per-user ext ini, reload pool | V18,V19,V21 |
| T28 | x | register `phpfpm.enable_extension` + `phpfpm.disable_extension` RPCs @ `cmd/agent/main.go` | V19 |
| T29 | x | `internal/api/handlers/phpextensions.go`: admin routes (global catalog CRUD) + user routes (per-user toggle, gated by admin-allowed) | I.api,V20 |
| T30 | x | wire routes: `GET/PUT /api/v1/admin/php-extensions` (admin) + `GET/PUT /api/v1/php-extensions` (user, scoped to self) | I.api |
| T31 | x | admin panel: PHP Extensions page — table of all known exts, per-version enable/disable toggle | I.frontend |
| T32 | x | user panel: PHP Settings page — add "Extensions" section below shell PHP card, per-ext toggle (only admin-allowed exts shown) | I.frontend |
| T33 | x | `make build` + `pnpm -r build` clean | — |
| T34 | x | migration 000014: `cron_jobs` table (id, user_id, expression, command, enabled, last_run_at) | I.db |
| T35 | x | `internal/store/cronjobs.go`: CronJob model + store (List, Create, GetByID, Update, Delete, ListByUserID) | I.db,V25 |
| T36 | x | `agent/cron/cron.go`: `Sync(username string, jobs []Job)` — build crontab lines, validate each (V23,V24), write via `crontab -u <user>` | V23,V24,V26 |
| T37 | x | register `cron.sync` RPC @ `cmd/agent/main.go` | V23 |
| T38 | x | `internal/api/handlers/cronjobs.go`: List/Create/Update/Delete — ownership check, quota check (V25), call `cron.sync` after every mutation | I.api,V25,V26 |
| T39 | x | wire cron routes + construct CronJobHandler @ `cmd/api/main.go` + `internal/api/router.go` | I.api |
| T40 | x | user panel: Cron Jobs page — table + add/edit modal (expression builder or free-form + validate), enable/disable toggle | I.frontend,V24 |
| T41 | x | migration 000015: add `totp_secret_enc`, `totp_enabled`, `totp_recovery_codes` (JSON array of bcrypt hashes) to `users` table | I.db |
| T42 | x | `internal/store/users.go`: add SetTOTP, GetTOTPSecret, ConsumeRecoveryCode methods | I.db,V27,V29 |
| T43 | x | `internal/api/handlers/auth.go`: Setup/Confirm/Disable 2FA handlers; modify Login to return `requires_2fa+temp_token`; add Verify + Recover handlers | I.api,V27,V28,V29 |
| T44 | x | wire 2FA routes @ `internal/api/router.go` (public: verify, recover; JWT: setup, confirm, disable) | I.api |
| T45 | x | user panel: 2FA setup flow — QR code display, confirm code input, recovery codes download; login page: TOTP step after password | I.frontend,V28 |
| T46 | x | `agent/logs/logs.go`: `Tail(logPath string, lines int) ([]string, error)` — path jail (V30), max 500 lines (V31) | V30,V31 |
| T47 | x | register `logs.tail` RPC @ `cmd/agent/main.go` | V30 |
| T48 | x | `internal/api/handlers/logs.go`: `DomainLogs` handler — resolve nginx + fpm log paths from domain row, call agent `logs.tail` | I.api,V30,V31 |
| T49 | x | wire `GET /api/v1/domains/:id/logs` route | I.api |
| T50 | x | user panel: Logs viewer — per-domain dropdown (nginx/fpm), line count selector, auto-refresh toggle | I.frontend |
| T51 | x | `agent/installer/installer.go`: app catalog (WordPress, Laravel skeleton, plain HTML); `Run(appID, username, docroot, db*)` — download, extract, configure, chown (V32,V33) | V32,V33 |
| T52 | x | register `installer.run` + `installer.status` RPCs @ `cmd/agent/main.go` | V33 |
| T53 | x | `internal/api/handlers/installer.go`: ListApps, Install (async → job_id), Status — ownership check, domain lookup | I.api,V32 |
| T54 | x | wire installer routes + construct InstallerHandler @ `cmd/api/main.go` + `internal/api/router.go` | I.api |
| T55 | x | user panel: Website Installer page — app cards (WP, Laravel, HTML), domain select, DB fields, install progress log | I.frontend,V32 |
| T56 | x | `make build` + `pnpm -r build` clean | — |
| T57 | x | `scripts/install.sh`: install fail2ban + ipset; create zenspanel ipset chain; configure fail2ban jails (nginx-http-auth, nginx-limit-req, sshd) | V35 |
| T58 | x | `agent/firewall/firewall.go`: `ListBlocked()`, `Block(ip, reason)`, `Unblock(ip)` — ipset + iptables, validate IP/CIDR (V34,V36) | V34,V36 |
| T59 | x | `agent/firewall/fail2ban.go`: `ListJails()`, `SetJail(name, enabled)` — parse `fail2ban-client status`, write jail.d snippet (V35) | V35 |
| T60 | x | register `firewall.*` + `fail2ban.*` RPCs @ `cmd/agent/main.go` | V34 |
| T61 | x | `internal/api/handlers/firewall.go`: ListBlocked, Block, Unblock, ListJails, SetJail — admin-only, call agent | I.api,V37 |
| T62 | x | wire 5 firewall routes + construct FirewallHandler @ `cmd/api/main.go` + `internal/api/router.go` | I.api |
| T63 | x | admin panel: Firewall page — blocked IPs table (block/unblock), fail2ban jails table (enable/disable toggle) | I.frontend,V37 |
| T64 | x | `make build` + `pnpm -r build` clean | — |
| T65 | x | fix terminal WS: nginx config — add `proxy_set_header X-Forwarded-Host $host` to `/ws/` location in `install.sh` | V38 |
| T66 | x | firewall page: merge fail2ban currently-banned IPs into blocked list (call `fail2ban-client banned` per jail, tag source="fail2ban") | V39 |
| T67 | x | `agent/antivirus/antivirus.go`: `Scan(username, homeBase, path string) ([]string, error)` — run `clamscan` as panel user, path jail (V40) | V40 |
| T68 | x | register `antivirus.scan` RPC + `antivirus.status` (clamav daemon running?) @ `cmd/agent/main.go` | V40 |
| T69 | x | `internal/api/handlers/antivirus.go`: Scan (async job), Status — ownership check, path jail | I.api,V40 |
| T70 | x | wire antivirus routes + install ClamAV in `install.sh` | I.api |
| T71 | x | user panel: Antivirus page — scan button, path input, results list (infected files), status indicator | I.frontend,V40 |
| T72 | x | DB isolation audit: verify each `CREATE USER` grants only on `<user>_%` pattern. add `REVOKE ALL ON *.* FROM` before grant in `agent/mysql/mysql.go` | V41 |
| T73 | x | storage quota: wire `package.disk_quota` into `quota.set` on user create + package change (already partially done — verify end-to-end, fix if broken) | V42 |
| T74 | x | `make build` + `pnpm -r build` clean | — |
| T75 | x | fix B8: `agent/installer/installer.go` — replace `runAs` shell-string interpolation w/ `exec.Command` arg array for `php artisan key:generate`; validate DocRoot contains no shell metachar | V43 |
| T76 | x | fix B9: `internal/api/handlers/phpextensions.go:AdminUpdate` — on global disable, enumerate users w/ ext enabled via store, call `phpfpm.disable_extension` per user | V44 |
| T77 | x | fix B10: `internal/api/handlers/users.go:Update` — if GetByID fails after php_version update, surface warning in response instead of silent skip | — |
| T78 | x | `make build` + `pnpm -r build` clean | — |
| T79 | x | migration 000017: `antivirus_alerts` table (id, user_id, path, threat, detected_at) | I.db |
| T80 | x | `agent/antivirus/antivirus.go`: add `WatchStart(username, homeBase)` — inotifywait loop, scan on CREATE/MODIFY, store alert via callback (V40,V46) | V40,V46 |
| T81 | x | register `antivirus.watch_start` + `antivirus.watch_stop` RPCs @ `cmd/agent/main.go` | V46 |
| T82 | x | `internal/api/handlers/antivirus.go`: add `Alerts` (list) + WS push on new alert | I.api,V46 |
| T83 | x | user panel: Antivirus page — add realtime alerts section, WS listener for `antivirus_alert` events | I.frontend,V46 |
| T84 | x | admin panel: sidebar "Updates" menu item → existing Settings page update card (just add nav shortcut) | I.frontend |
| T85 | x | admin panel: Terminal page — `POST /admin/terminal/token` + reuse WS terminal (V48) | I.api,V48 |
| T86 | x | migration 000018: `backup_targets` table (id, name, type, bucket, prefix, access_key, secret_key_enc, region, endpoint, enabled) | I.db |
| T87 | x | `internal/store/backuptargets.go`: BackupTarget model + store (List, Create, Update, Delete, GetByID) | I.db,V47 |
| T88 | x | `agent/backup/s3.go`: `UploadS3(filePath string, target BackupTarget) error` — aws-sdk-go-v2 or rclone subprocess (V47) | V47 |
| T89 | x | register `backup.upload_s3` RPC @ `cmd/agent/main.go` | V47 |
| T90 | x | `internal/api/handlers/backuptargets.go`: CRUD + Test endpoint (V47) | I.api,V47 |
| T91 | x | wire backup-targets routes + construct handler @ `cmd/api/main.go` + `internal/api/router.go` | I.api |
| T92 | x | admin panel: Backup Targets page — list targets, add/edit modal (S3 creds), test connection button | I.frontend,V47 |
| T93 | x | extend existing backup flow: after local backup completes, if target configured → call `backup.upload_s3` | I.api |
| T94 | x | `internal/api/handlers/packages.go`: accept `disk_quota_mb` + `memory_limit_mb` in Create/Update, convert MB→bytes before store (V49) | V49 |
| T95 | x | admin panel: Packages page — change disk_quota + memory_limit inputs to MB with unit label | I.frontend,V49 |
| T96 | x | `make build` + `pnpm -r build` clean | — |
| T97 | . | extend `agent/installer/installer.go` catalog: add Joomla 5, Drupal 10, PrestaShop 8, CodeIgniter 4, plain PHP starter (V52) | V52 |
| T98 | . | implement `installJoomla`, `installDrupal`, `installPrestaShop`, `installCodeIgniter` in agent/installer (V33,V52) | V33,V52 |
| T99 | . | user panel: Installer page — add new app cards, show version badges | I.frontend |
| T100 | . | migration 000020: `domain_redirects` table (id, domain_id, source_path, dest_url, type, enabled) | I.db |
| T101 | . | `internal/store/redirects.go`: DomainRedirect model + store (List, Create, Update, Delete) | I.db,V53 |
| T102 | . | `agent/nginx/nginx.go`: `SyncRedirects(domain string, redirects []Redirect)` — rewrite redirect block in vhost (V55) | V55 |
| T103 | . | register `nginx.sync_redirects` RPC @ `cmd/agent/main.go` | V55 |
| T104 | . | `internal/api/handlers/redirects.go`: List/Create/Update/Delete — ownership check (V53), call `nginx.sync_redirects` after mutation | I.api,V53 |
| T105 | . | wire redirect routes + construct RedirectHandler @ `cmd/api/main.go` + `internal/api/router.go` | I.api |
| T106 | . | user panel: Redirect Manager page — table per domain, add/edit modal (source path, dest URL, 301/302), enable/disable toggle | I.frontend,V53 |
| T107 | . | `agent/nginx/nginx.go`: `SetHotlinkProtection(domain string, enabled bool, allowedDomains []string)` — write valid_referers block (V54,V55) | V54,V55 |
| T108 | . | register `nginx.set_hotlink` RPC @ `cmd/agent/main.go` | V54 |
| T109 | . | `internal/api/handlers/hotlink.go`: Get/Set — ownership check, call agent | I.api,V54 |
| T110 | . | wire hotlink routes + construct HotlinkHandler @ `cmd/api/main.go` + `internal/api/router.go` | I.api |
| T111 | . | user panel: Domains page — add "Hotlink Protection" toggle per domain row; Redirect Manager link per domain | I.frontend,V54 |
| T112 | . | `make build` + `pnpm -r build` clean | — |

## §B BUGS

| id | date | cause | fix |
|----|------|-------|-----|
| B1 | 2026-05-19 | `users.GetUsage` ! ownership check ∴ ∀ user can read other users' RAM/disk/CPU/quota via `:id` enumeration. CRITICAL IDOR | V12 |
| B2 | 2026-05-19 | WS `upgrader.CheckOrigin: return true` + cookie auth ∴ cross-origin shell hijack possible (CSWSH). HIGH | V14 |
| B3 | 2026-05-19 | subdomain `Update` accepts free-form `document_root` w/o re-running jail check ∴ DB drift to `/etc/...`; later php_version change writes vhost outside user home. MEDIUM | V15 |
| B4 | 2026-05-19 | login cookie `Secure=false` & `SameSite=Lax` ∴ JWT exposed on plain-HTTP & cross-site state-changing requests reachable. HIGH | V13 |
| B5 | 2026-05-19 | `/terminal/token` ! rate limit ∴ token enumeration / spam possible. MEDIUM | V17 |
| B6 | 2026-05-19 | ownership pattern `role != "admin"` ∴ api_key callers (id=0) fall through to "owner" branch on future routes. brittle, current routes safe-by-accident. MEDIUM | V16 |
| B7 | 2026-05-20 | WS terminal fails on direct-port access (`:8888`): nginx `/ws/` location missing `proxy_set_header X-Forwarded-Host $host` ∴ `r.Host` = `127.0.0.1:8080` but browser Origin = `103.150.92.61:8888` → CheckOrigin rejects upgrade. Fix already in code (V38); nginx config needs the header. | V38 |
| B8 | 2026-05-20 | `agent/installer/installer.go:296` — `runAs` builds shell string via `fmt.Sprintf("cd %q && ...", p.DocRoot)`. Go `%q` does NOT escape `$` or backticks ∴ DocRoot containing `$(...)` executes arbitrary commands as panel user. DocRoot flows from `domain.document_root` which user can set via PUT /domains/:id. HIGH shell injection. | V43 |
| B9 | 2026-05-20 | `internal/api/handlers/phpextensions.go:AdminUpdate` — dead propagation branch: `if !req.Enabled` block fetches agent client but never calls any RPC ∴ disabling ext globally does NOT reload running FPM pools. Ext stays loaded until next pool reload. Contradicts V20 intent. LOW. | V44 |
| B10 | 2026-05-20 | `internal/api/handlers/users.go:Update` — if `GetByID` fails after DB write, `user.setup_bin` is silently skipped and response is still 200. Shell PHP symlink stays stale. No warning surfaced to caller. LOW. | — |
| B11 | 2026-05-20 | "Login as" opens `/user/#impersonate=<token>` but nginx serves user panel at `/` (root). URL 404s ∴ token never read, user lands on login page. FIXED: changed to `/#impersonate=<token>`. | V45 |
