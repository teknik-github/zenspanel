# Changelog

All notable changes to ZensPanel will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- Full suspend/unsuspend system (T134–T145). Suspending a user now atomically: disables all nginx vhosts (503 page), suspends all FTP accounts (removed from PAM DB), and revokes all active JWT sessions immediately via `token_version` increment in DB. New `token_version` column on users (migration 000023); JWT middleware validates the claim on every request and returns 401 if the token version is lower than the DB value — suspended users cannot use existing tokens even before expiry (V62, V63). New agent RPCs: `nginx.suspend_all_vhosts`, `nginx.unsuspend_all_vhosts`, `ftp.suspend_user`, `ftp.unsuspend_user`. Admin User Detail page gains dedicated Suspend/Unsuspend buttons with confirm dialog and suspended banner. Per-domain suspend/unsuspend: new `POST /api/v1/domains/:id/suspend` and `unsuspend` routes + `SuspendDomain`/`UnsuspendDomain` handlers (V64).

### Added
- Auto dependency setup on update. New `scripts/setup.sh` — idempotent script that installs and configures all ZensPanel dependencies (nginx, MySQL, Redis, PHP 8.1/8.2/8.3, Certbot, phpMyAdmin, ClamAV, fail2ban, vsftpd, quota, cgroups, Composer, logrotate). Each step checks before acting so it's safe to run repeatedly. The updater now runs this script as a `setup_dependencies` phase after deploying binaries and frontend (both download and build-from-source paths). Failures are non-fatal — binaries are already deployed and the restart proceeds. This means new features that require new system packages (e.g. vsftpd for FTP accounts) are automatically provisioned on existing servers when the admin clicks "Apply Update".

### Added
- FTP accounts (T124–T133). Users can now create vsftpd virtual FTP accounts from a new "FTP Accounts" page in the user panel. Each account maps to a vsftpd virtual user authenticated via PAM + Berkeley DB (`/etc/vsftpd/virtual_users.db`). New `ftp_accounts` table (migration 000022) + `max_ftp_accounts` column on `packages` (0 = FTP disabled). New `agent/ftp` package: `CreateAccount` writes a per-user vsftpd config under `/etc/vsftpd/users/`, upserts the flat `virtual_users.txt`, recompiles the PAM DB via `db_load`, and reloads vsftpd. `DeleteAccount` reverses this. New agent RPCs: `ftp.create`, `ftp.delete`. New API: `GET/POST /api/v1/ftp`, `DELETE /api/v1/ftp/:id` with package quota enforcement. Admin Packages page gains a `max_ftp_accounts` field (0 = disabled). `scripts/install.sh` installs vsftpd + db-util + libpam-pwdfile, creates a `vsftpd_virtual` system account, writes the PAM config and vsftpd.conf with passive mode (ports 40000–40100), and starts the service.
- Domain-level backup (T120–T123). Users can now back up a single domain's docroot directly from the Domains page without triggering a full-account backup. New "Backup" button per domain row opens a status modal that polls every 3 seconds and shows a download link once complete. Backend: new `POST /api/v1/domains/:id/backup` route creates a `backups` row (type=`domain`) and delegates to a new `backup.domain` agent RPC. New `agent/backup/domain.go` — `BackupDomain()` tars only the docroot with `--no-same-owner --no-same-permissions` (V58: no full home, no cross-domain data). New `GET /api/v1/backups/:id` route for polling individual backup status.

### Added
- Per-user shell PHP version. The terminal console previously always resolved `php` and `composer` to the system default (e.g. 8.5) regardless of which version the user had configured for their domains, so `composer install` and `php artisan` ran under the wrong runtime. New `users.php_version` column (migration 000012, default `8.3`); on every user create and on every update that touches `php_version`, the agent re-seeds `~/bin/php` as a symlink to `/usr/bin/php<ver>` and writes a `~/bin/composer` wrapper that execs `/usr/local/bin/composer.phar` via that same PHP. Because the rbash login profile already prepends `~/bin` to `PATH`, the next shell — and any Composer scripts that re-shell out — pick up the configured version automatically. New API surface: `php_version` accepted on `POST /api/v1/users`, allowed on `PUT /api/v1/users/:id` (via the existing allowlist), echoed back in `/auth/login` and `/auth/me`, and the previously hardcoded `"8.3"` in the FPM-pool create call now uses `user.PHPVersion`. New `user.setup_bin` agent RPC (idempotent — safe to call repeatedly). `install.sh` downloads the Composer phar to `/usr/local/bin/composer.phar` so the wrapper has something to exec. Frontend: PHP Settings page gains a "Shell PHP Version" card above the per-domain table that calls `PUT /api/v1/users/:id` and re-fetches `/me` so subsequent renders reflect the saved value.
- Admin "Login as User" impersonation. Admins can now open a user's panel session directly without knowing the user's password — useful for support and debugging. New `POST /api/v1/users/:id/impersonate` (admin-only) mints a 1-hour JWT for the target user with an `ImpersonatedBy` claim recording the admin's ID. Refuses to impersonate suspended users or other admins. The admin panel shows a purple "Login as" button on both the User List and User Detail pages; clicking it opens `/user/` in a new tab with the token passed in the URL hash (`#impersonate=<token>`). The user panel's `main.ts` reads the hash on boot, stores the token in localStorage, strips the hash from the URL (so it never reaches the server and doesn't persist on reload), then boots normally as that user. The admin's own session is untouched.
- PHP extension management (T25–T33). Admins can now enable/disable PHP extensions globally per PHP version from a new "PHP Extensions" page in the admin panel. Users can toggle extensions for their own environment from PHP Settings, subject to the admin's global allow/deny. Architecture: two new tables (`php_extensions` global catalog + `user_php_extensions` per-user overrides, migration 000013) seeded with 12 common extensions across PHP 8.1/8.2/8.3. New agent functions `EnableExtension`/`DisableExtension` write per-user ini snippets to an isolated directory and reload (not restart) the FPM pool (V21). Extension names validated via `safe.ExtName` (`^[a-z0-9_]+$`) before any filesystem op (V19). Admin-disabled extensions cannot be re-enabled by users (V20, enforced at both DB write and API layer). New API: `GET/PUT /api/v1/admin/php-extensions` (admin), `GET/PUT /api/v1/php-extensions` (user). User PHP Settings page now shows an Extensions section that reloads when the shell PHP version selector changes.
- Cron jobs (T34–T40). Users can now schedule recurring tasks from a new "Cron Jobs" page in the user panel. New `cron_jobs` table (migration 000014) + `max_cron_jobs` column on `packages` (migration 000015, default 10). New `agent/cron` package validates every expression (5-field standard cron or @shortcuts) and command (no shell metacharacters — V23, V24) before atomically rewriting the user's crontab via `crontab -u <user>`. Disabled jobs are written as commented lines so they survive re-enable without data loss (V26). Quota enforced at create time against `package.max_cron_jobs` (V25). New API: `GET/POST /api/v1/cron-jobs`, `PUT/DELETE /api/v1/cron-jobs/:id`. Frontend: table with expression + command + active/disabled toggle, add/edit modal with preset selector (every minute/hour/day/week/month) plus free-form input, confirm-delete dialog.

### Fixed
- Terminal "blank blue screen with base64 garbage": frontend `Terminal.vue` was passing `msg.data` directly to `term.write()`, but the API base64-encodes raw PTY bytes (so binary control sequences survive JSON transport). xterm.js was rendering the literal base64 string `G1s/MjAwNGgbXTA7…` instead of the decoded ANSI prompt. Added `atob(msg.data)` before `term.write` so the bytes are decoded back to Latin1 chars before xterm's UTF-8 decoder reassembles them.

### Security
- **CRITICAL**: Fixed IDOR in `users.GetUsage` (`GET /api/v1/users/:id/usage`). The handler had no ownership check, so any logged-in user could read every other user's package limits + live RAM/disk/CPU/domain/db counters by iterating `:id`. Added `role=="user" && id!=self → 403` mirroring `users.Get`. (B1, V12)
- **HIGH**: Fixed cross-site WebSocket hijack (CSWSH) on `/ws/terminal`. The Gorilla `upgrader.CheckOrigin` was `return true`, so any malicious page could complete the WS upgrade with the victim's cookie-auth'd token and own a shell. New check compares the `Origin` header to `r.Host` (allows http/https same-origin); empty Origin still passes for native non-browser clients. (B2, V14)
- **HIGH**: Hardened login cookie. Was `SameSite=Lax` + `Secure=false` — JWT could leak over plain HTTP and CSRF was reachable from cross-origin POST/PUT/DELETE. Now `SameSite=Strict` always, `Secure=true` when the request arrived over TLS (detected via `r.TLS != nil` or `X-Forwarded-Proto: https` from the nginx proxy). The Bearer-header path is unchanged. (B4, V13)
- **MEDIUM**: Fixed jail bypass in subdomain `Update` handler. Free-form fields map only blacklisted `id`/`user_id`/`parent_domain_id`/`subdomain`/`fqdn` — `document_root` was writable without re-running the home-jail check that `Create` enforced. Now every Update that includes `document_root` re-resolves it under the user's home and rejects with 400 if it escapes. (B3, V15)
- **MEDIUM**: Added per-user rate limit to `POST /api/v1/terminal/token` — 5 second cooldown per user via in-process `sync.Map`. Returns 429 + `Retry-After: 5`. Cheap defence against token-mint spam from a stolen JWT. (B5, V17)
- **MEDIUM**: Refactored ownership pattern across 20 sites (`backups`, `databases`, `domains`, `ssl`, `subdomains`, `users` handlers). Replaced `auth.GetRole(c) != "admin"` with explicit `auth.GetRole(c) == "user"`. The previous form let `api_key` callers (role="api_key", uid=0) silently fall through to the "owner" branch on any future external-API route — current routes were safe by accident, but the pattern is brittle. (B6, V16)

### Added
- Subdomain support: users can now create `<label>.<parent-domain>` directly from the Domains page without admin intervention. Each subdomain gets its own nginx vhost (one `.conf` per FQDN), shares the parent user's PHP-FPM pool, and supports independent SSL state (Let's Encrypt or custom upload). New `subdomains` table (migration 000011) FK'd to `domains` with `ON DELETE CASCADE` so deleting a parent torches its children. New `agent`-side reuse only — no new RPCs; subdomain create/delete pipes through existing `nginx.create_vhost` / `nginx.delete_vhost` / `phpfpm.create_pool` / `ssl.*`. New API: `GET/POST/PUT/DELETE /api/v1/subdomains`, `GET /api/v1/subdomains?parent_id=`, `POST/DELETE /api/v1/subdomains/:id/ssl`. Server-side validation: RFC 1035 label regex, reserved-label denylist (`www`, `mail`, `ns`, `_dmarc`, etc.), ownership check against parent domain, FQDN collision check across both `domains` and `subdomains` tables, doc_root jail under user home, php_version must be in the enabled list. Frontend: per-row chevron expand toggle on Domains page reveals child subdomains inline; "+ Subdomain" button per row opens a modal with label input + PHP version select + live "blog.example.com" preview suffix. SSL Manager merges domains + subdomains into a single tree-indented table — Let's Encrypt / Custom / Remove buttons route to the right endpoint based on row kind. Parent domain delete + user delete both walk the subdomain list and ask the agent to remove each vhost+cert before the FK cascade runs, so no orphan `.conf` files are left behind. Spec at `SPEC.md` (§T1–T18, all completed).
- "Sorry!" catch-all page on port 80, cPanel-style. New `zenspanel-sorry` nginx site (default_server on :80) handles two cases: (1) someone hits the panel hostname without the panel port (panel listens on `$PANEL_PORT`, not :80), (2) a random hostname or raw IP resolves to this server before any user vhost has claimed it. The page renders an inline HTML "Sorry!" template (no external assets, no logging) with a friendly message about the domain not being available yet. Each user vhost added later via the panel listens on :80 with its own `server_name`, so this default_server only catches the leftovers.

### Fixed
- "User already exists" when recreating a deleted user with the same name: the Delete handler only removed the DB row + quota entry. Linux user, home dir, cgroup slice, PHP-FPM pools, FileBrowser record, and every domain/database the user owned were all left orphaned. `useradd` then refused to recreate because `/etc/passwd` still had the entry. Delete now does a full ordered teardown: nginx vhost + SSL cert per domain → MySQL drop per database → PHP-FPM pool delete (every supported version) → cgroup slice → quota → FileBrowser record → Linux user (`userdel -r`) → DB row last. Per-step failures are collected into a `warnings` array in the response so the operator can see what didn't clean up, but the overall delete still proceeds — partial cleanup is better than refusing to delete and leaving the row in place.
- phpMyAdmin SSO 404: `LaunchPHPMyAdmin` handler existed in `databases.go` and was wired up to use Redis for one-time tokens, but the route `GET /api/v1/databases/:id/phpmyadmin/launch` was never registered in `router.go` — only the legacy `/phpmyadmin` (token-only) route was. Clicking phpMyAdmin in the User Panel hit the launch URL the legacy endpoint returned and got 404. Added the missing route inside the JWT-protected group.
- ERR_CONNECTION_REFUSED after switching a domain's PHP version: when the user picked a non-default PHP (e.g. 8.2 or 8.1 on a server that booted with 8.3), the agent wrote the new pool config but the corresponding `php<ver>-fpm.service` was still disabled — Ubuntu only auto-starts the version installed first. The reload call hit a stopped unit, no socket got created, and nginx's `fastcgi_pass` ended up pointing at a non-existent path. New `EnsureRunning` step in `agent/phpfpm` runs `systemctl enable+start` before reload (idempotent, no-ops when already running), and `ReloadFPM` falls back to `start` if reload fails on a stopped unit. Plus the domain handler no longer swallows agent-call errors with `_` — pool/vhost failures now surface as a 500 with a specific message instead of letting the row update succeed while the site silently breaks.
- "Access denied" on freshly-created websites: agent created `index.html` and `public_html/` via `os.WriteFile` / `os.MkdirAll` which pass the requested mode through the process umask. systemd's default umask of `0027` was silently stripping the read bit for "others", so files ended up `0640` and dirs `0750` — nginx (www-data) could traverse but not read. Added explicit `os.Chmod(0644)` after `WriteFile` and `os.Chmod(0755)` after `MkdirAll` in `agent/nginx.ensureDocRoot` so the modes we asked for are the modes that actually land on disk, regardless of umask.
- "File not found." on every fresh website: nginx (running as `www-data`) couldn't traverse `/var/lib/zenspanel/home` because install.sh chmod'd it to `0750` (no `x` bit for others). Static-file requests fell through to PHP-FPM which 404'd. Changed `home/` to `0751` so `www-data` can traverse without listing — `backups/` stays `0750` since nginx never reads it. Per-user home dirs are still `0711` (set by `agent/user.Create`), so user A still can't read user B's files.
- FileBrowser provisioning HTTP 401 (the actual root cause): under proxy auth, FileBrowser only honours the `X-Auth-User` header at `/api/login` — every other endpoint requires a JWT in the `X-Auth` header. The agent was sending `X-Auth-User: admin` directly to `/api/users` and getting 401. Fixed by `agent/filebrowser/filebrowser.go` first calling `POST /api/login` with the proxy header to obtain a JWT, caching it for ~1h, and using `X-Auth: <jwt>` on subsequent management calls. Error responses now also include the FileBrowser response body to make future debug-the-401-saga less painful.
- Quota: `setquota` was failing with "Mountpoint not found or has no quota enabled" because the agent passed `homeBase` (e.g. `/var/lib/zenspanel/home`) directly — but `setquota`/`repquota` only accept actual mount points or device names. New `resolveMount()` walks `df --output=target` to find the real mountpoint that contains `homeBase` (typically `/`), and all three quota functions (`SetQuota`, `DeleteQuota`, `ReadQuota`) now use that resolved mountpoint. `SetQuota` also has a self-heal: if the kernel reports "no quota enabled", it tries `quotaon -u <mp>` once and retries, in case the installer's `quotaon` failed silently or the mount got reset.
- FileBrowser provisioning HTTP 401: install.sh now explicitly upserts the `admin` user with `--perm.admin=true --scope /` after `config init`, so the agent's `POST /api/users` calls (which authenticate as `X-Auth-User: admin`) actually have admin rights. Without this, the FileBrowser admin user could end up missing or restricted depending on whether `config init` ran on a fresh DB or an existing one.

### Added
- SSL handler pre-flight: validates `letsencrypt.email` before round-tripping to certbot. Empty/malformed addresses, and reserved IANA/Let's-Encrypt-forbidden domains (`example.com`, `example.net`, `example.org`, `test.com`, `localhost`, `invalid`, `localdomain`) now return `400 Bad Request` with an actionable message ("Set `letsencrypt.email` in /etc/zenspanel/config.yaml…") instead of bubbling up as a generic 500 from a certbot stack trace. Triggered by an operator hitting Issue with the installer-default `admin@example.com` and seeing only "500 Internal Server Error" in the browser
- Terminal: implemented the missing API side. The frontend was calling `POST /api/v1/terminal/token` and `WS /ws/terminal?token=<t>` but neither route existed (404). New `internal/api/handlers/terminal.go` mints 32-char hex tokens with 60s TTL stored in a `sync.Map` (one-time redeem via `LoadAndDelete`, background pruner every 5 min). New `agent/terminal.Stream` spawns the PTY and exposes it on a per-session, random-named one-shot Unix socket at `/tmp/zp-pty-<hex>.sock` (chmod 0666 so the non-root API can connect, accept-once-then-unlink). New `terminal.stream` agent RPC returns the socket path; the API dials it and bridges raw bytes ↔ WebSocket using the existing `{type:"output"|"input", data:"<base64|string>"}` wire protocol the xterm.js frontend already speaks. The WS endpoint is registered outside the JWT-protected group because browsers can't attach Authorization headers to WebSocket handshakes — the one-time token is the credential, same pattern as the phpMyAdmin SSO redeem route. Required adding `github.com/gorilla/websocket` to go.mod
- UI/UX polish across both panels: collapsible sidebar with localStorage-persisted state and a chevron toggle, plus a mobile drawer (≤ md breakpoint) with backdrop-tap-to-close that auto-shuts on route change. Header gains a hamburger toggle on mobile and a `Home / <Page>` breadcrumb on desktop, replacing the non-functional fake search bar that was previously taking up header space. Stat cards on both Dashboards now lead with a colored icon chip (indigo/blue/purple/emerald/amber) and the admin Dashboard adds a 4px colored left border per card for at-a-glance category recognition. Every list page (Domains, Databases, SSL Manager, Backups, User List, Packages) now ships three explicit states — `animate-pulse` skeleton during the initial fetch, an icon + headline + CTA empty state when there's no data, and a `overflow-x-auto`-wrapped table with a `min-w-[…]` floor that prevents column squashing on narrow viewports. Tables, filters, and page headers stack at small breakpoints (`grid-cols-1 sm:grid-cols-2 lg:grid-cols-3` for card grids, `flex-wrap` for filter bars). SSL Manager now uses a per-row `pending` map (was a single shared `loading` flag — issuing on one row used to disable every button on every row) and the destructive **Remove** action requires a confirm dialog instead of one-click. Backups disables the trash icon while a row is `pending/running/restoring` so deletes can't race with the worker. Terminal gets a connecting-state spinner overlay, an `xterm.dispose()` guard before reconnect, and proper `removeEventListener` cleanup for the resize handler (was leaking listeners on each Reconnect click). Admin Settings replaces the hardcoded `v1.0.0` About-card version with the real release tag (or 7-char commit SHA) by auto-fetching `update.check` on mount, and the Services grid is now responsive (`grid-cols-1 sm:grid-cols-2 lg:grid-cols-3`).

### Fixed
- Admin → User Detail: package dropdown was empty after refresh even when a package was assigned. Root cause: `User.PackageID` was `sql.NullInt64`, which the stdlib serializes as `{"Int64":5,"Valid":true}` — Vue's `v-model` couldn't match that object against `<option :value="5">`, so the select rendered blank. Replaced with a custom `store.NullInt64` type that marshals to a plain `5` (or `null`) and round-trips correctly through `Scan`/`Value` for the database driver. Field accessors `.Int64` / `.Valid` are kept identical so all existing handler code (e.g. `user.PackageID.Valid`, `user.PackageID.Int64`) compiles unchanged
- Filesystem-level disk quota: kernel-enforced hard limit per Linux user using the Linux quota subsystem. New `agent/quota` package wraps `setquota` / `repquota` (`SetQuota`, `DeleteQuota`, `ReadQuota` — bytes ↔ 1KB blocks, soft = 90% of hard, inode limits left at 0 since we only meter disk space). New agent RPCs: `quota.set`, `quota.delete`, `quota.read`. Wired into the user provisioning chain — `users.Create` calls `quota.set` after cgroup/PHP-FPM provisioning when the package has `disk_quota > 0`; `users.ChangePackage` updates both cgroup and quota when the row's package changes; `users.Delete` clears the quota entry alongside the row delete. `scripts/install.sh` adds a `setup_quota` step that detects the filesystem (ext4/xfs), backs up `/etc/fstab`, appends `usrquota` (or `uquota` for XFS) only if missing, runs `mount -o remount` + `quotacheck -cum` + `quotaon`, and skips with a warning on unsupported filesystems. Failures throughout are non-fatal — quota provisioning warnings surface in the create response but never roll back the row
- Redis-backed rate limiter: new `middleware.RateLimitRedis` uses a Redis sorted set per client IP plus a Lua script to do the prune+count+add atomically. Login limiter auto-selects between Redis and in-memory based on Redis availability — multi-server deployments behind a load balancer now share a single counter instead of each instance counting independently. Fail-open on Redis error so a Redis outage doesn't lock out every login. New config fields: `redis.password`, `redis.db`
- `zenspanel-cli` — interactive TUI admin utility built with bubbletea. Run as root from any shell. Menu items: Status & Info (service health, DB/agent connectivity, panel inventory), Reset Password (raw SQL bypass of HTTP-allowlist), Create Admin (validates username + bcrypt hash + DB insert), Suspend/Unsuspend User, Restart Services (multi-select checklist), View Logs (tail of api/agent/nginx logs), Rebuild Frontend (pnpm build + copy dist + nginx reload). Symlinked from `/usr/local/bin/zenspanel-cli` by the installer
- Backup restore: full implementation. New `agent.backup.restore_files` (rm -rf home, tar extract, chown to user's UID) and `agent.backup.restore_db` (mysql --one-database import per panel-tracked DB) RPCs. `POST /api/v1/backups/:id/restore` no longer 501s — it kicks off a goroutine, transitions the row through `restoring → done|restore_failed`, and is exposed in both User and Admin Panel UIs with a destructive-action confirm modal
- Login rate limiting: new `internal/api/middleware/ratelimit.go` with a sliding-window per-IP limiter mounted on `POST /api/v1/auth/login` at 10 requests/minute. Exceeded callers get 429 Too Many Requests with a `Retry-After` header. In-memory `sync.Map` store with a background pruner; sufficient for single-server deploys (multi-server would swap in Redis-backed storage)
- Audit logging: new `internal/api/middleware/audit.go` records every mutating request (POST/PUT/DELETE) on protected and external route groups to `audit_logs` after the handler runs. Captures method+path, IP, user-agent, the panel user when authenticated via JWT, and the `:id` path param when present. GET/OPTIONS/HEAD and 5xx responses are skipped (read traffic and failed mutations aren't audit-relevant). Best-effort: a write failure on the audit row is swallowed so it never turns a successful action into a 500
- API Key authentication: new `auth.APIKeyMiddleware` and `auth.RequirePermission` plus a `/api/v1/external/*` route group that accepts the `X-API-Key` header. Endpoints exposed for billing/integration use: list/get/create users, suspend/unsuspend/change-package, list packages, read usage. Each route requires the matching permission string on the API key (`read_user`, `create_user`, `suspend_user`, `change_package`, `read_package`) so a key can be issued with least-privilege scope
- API Key prefix bug fix: `KeyPrefix` was being set to the literal `"zp_live_"` (the constant tag prefix), so every key collided on the same prefix and `ValidateKey` had to bcrypt every key in the table on every lookup. Now uses the first 8 chars of the random portion (`fullKey[8:16]`) for an O(1) lookup
- SSL handler: new `internal/api/handlers/ssl.go` plus `POST /domains/:id/ssl` and `DELETE /domains/:id/ssl` routes. Issue branches on `type`: `letsencrypt` calls `agent.ssl.issue_letsencrypt` with the configured email/staging flags and records a 90-day expiry; `custom` accepts `cert_pem`/`key_pem`, calls `agent.ssl.write_custom_cert`, and parses NotAfter from the cert when possible. Remove calls `agent.ssl.remove_cert` and clears `ssl_type`/`ssl_expires_at`. Frontend SSL Manager already speaks this contract — wires up without UI changes
- Database create/delete now provision the actual MySQL database and user. `POST /databases` calls `agent.mysql.create_database` after the DB row is inserted, rolling the row back on failure. The password from the request is forwarded to MySQL and returned in the 201 response (shown once); it is never stored in the panel DB. `DELETE /databases/:id` calls `agent.mysql.drop_database` first so MySQL state matches the panel
- Domain create/delete now provision the actual nginx vhost. `POST /domains` calls `agent.nginx.create_vhost` after the DB insert and rolls the row back on failure (so the unique domain constraint doesn't block retries). After successful provision, the domain status is flipped from `pending` to `active`. `DELETE /domains/:id` now calls `agent.nginx.delete_vhost` first; if the agent fails the panel row is still removed because an orphan row blocks recreate
- Admin Panel: "Add User" button + modal on the Users page. Form covers username, email, password, package, and the terminal/backup override flags. Uses the existing `POST /api/v1/users` endpoint
- API: `users.Create` now provisions system resources after the DB insert — calls `agent.user.create` (Linux user + home dir) and, when a package is assigned, `agent.cgroups.create_slice` and `agent.phpfpm.create_pool`. If the Linux user step fails the panel row is rolled back. cgroups and phpfpm failures surface as a `warnings` array in the response so the row stands and the admin can retry by reassigning the package
- API: `users.Create` request body now accepts `terminal_enabled` and `backup_enabled` so the admin can toggle these at creation time

### Security
- Admin Panel route guard now enforces `role === 'admin'` on every protected route (previously the guard checked only for the presence of a token, so any authenticated panel user who guessed `/admin/` could browse the admin UI). The guard awaits `auth.fetchMe()` if the user is not yet loaded and logs the session out before redirecting on role mismatch

### Fixed
- Installer: explicit `groupadd -r zenspanel` before the chown of `/run/zenspanel` so the group is guaranteed to exist regardless of distro-specific `useradd` defaults
- Installer: `chown -R zenspanel:zenspanel /var/lib/zenspanel` on the data tree so the API service (which runs as that user) can write backup tarballs to `paths.backup_base` without permission errors
- Agent MySQL provisioning: agent now connects to MySQL using a separate `agent.mysql_admin_dsn` config field instead of reusing the panel-user DSN. The panel user only has grants on the `zenspanel` schema, so previously every `POST /api/v1/databases` returned 500 with `Error 1044: Access denied for user 'zenspanel'@'localhost' to database 'testdb'` because the agent couldn't `CREATE DATABASE`. Installer template now writes a root-level admin DSN; if `mysql_admin_dsn` is unset the agent logs a startup warning and falls back to `database.dsn` so the misconfiguration surfaces clearly
- User Panel Dashboard: backend `GET /users/:id/usage` now returns the `{used, max}` shape per category (`domains`, `databases`, `disk`, `ram`) that the dashboard reads. Previously the response shape was a flat `{domains: 0, databases: 0, disk_bytes: 0, memory_bytes: 0}` and the dashboard crashed at render with `Cannot read properties of undefined (reading 'used')`. Disk and RAM `used` are still placeholders (real cgroup metrics not yet wired) but `max` is sourced from the user's package, and `domains.used` / `databases.used` are real counts via new `UserStore.CountDomains` / `CountDatabases`. Frontend `usage` store is also now defensive — missing fields fall back to `{used:0, max:0}` instead of leaving slots undefined
- User provisioning: agent now scans `/etc/passwd` for the next free UID in the panel range (10000-60000) instead of trusting the API's DB-derived guess. Previously a UID that existed in `/etc/passwd` but not in the panel `users` table caused `useradd: UID N is not unique` and a failed provision. The agent honors the API's preferred UID when free, otherwise picks the next free one and returns it; the API records the actual UID via the new `UserStore.UpdateLinuxUID`
- Cgroups: agent now enables `+cpu +memory` in `cgroup.subtree_control` at both `/sys/fs/cgroup` and `/sys/fs/cgroup/zenspanel` before creating a user slice. Without this, writing to `<slice>/cpu.max` failed with EACCES even though the agent runs as root — cgroup v2 only exposes those control files in children whose parent has delegated the controller. The enable is idempotent (EBUSY on already-enabled is treated as success). `memory.swap.max` is now also tolerant of kernels without swap accounting
- Agent socket permission: agent now chowns the socket to `root:<socket_group>` and chmods to 0660 instead of 0600. Previously the socket was owned by root with mode 0600 so the API process (running as `zenspanel`) could not connect — `dial unix /run/zenspanel/agent.sock: connect: permission denied`. The group is configurable via `agent.socket_group` in config.yaml (default `zenspanel`)
- Security: dynamic UPDATE in `users.go` and `domains.go` previously interpolated arbitrary JSON keys into SQL identifiers. With `multiStatements=true` in the DSN this enabled SQL injection (e.g. `{"x; DROP TABLE users--": 1}`). Now filtered through column allowlists in `internal/store/safefields.go`
- Security: `users` list `ORDER BY` previously took an unvalidated `sort` query parameter directly into SQL. Now validated against an explicit column allowlist
- Security: agent now validates every caller-provided string via `agent/safe` before any side effect — usernames, domains, MySQL identifiers, MySQL passwords, and PHP versions. The agent runs as root; defense in depth at this layer means an API-side validation bug cannot become a privesc
- Security: `agent/mysql` previously interpolated db_name, db_user, and db_password into raw SQL without sanitization. Now strictly whitelist-validated (alphanumerics + safe punctuation) to make the necessary identifier interpolation safe
- API: `users.Create` and `domains.Create` now reject usernames and domains that fail the documented format and reserved-name checks. Prevents path traversal in DocumentRoot construction and crashes downstream of `useradd` / nginx
- Admin Panel auth store: tracks `terminal_enabled`, `backup_enabled`, and `package_id` to match the backend `Login` response (previously dropped these fields silently)
- Admin & User Panel: both apps now await `auth.fetchMe()` in `main.ts` before mounting the router so reloads after login restore the full user state instead of leaving menu flags undefined

### Added
- `CONTRIBUTING.md` — captures the JSON-tag, dynamic-SQL allowlist, agent validation, and frontend build conventions, plus an end-to-end recipe for adding a new entity. Linked from README and summarized in CLAUDE.md
- `agent/safe/` package — single source of truth for input validation regexes used across every agent subsystem
- `internal/store/safefields.go` — column and sort-key allowlists used by all dynamic-SQL paths in the API
- Admin Panel & User Panel: added missing `postcss.config.js` for both apps — without it Tailwind directives (`@tailwind base/components/utilities`) were not processed and the production CSS was a 60-byte raw passthrough, leaving every page unstyled
- Admin Panel: `vite.config.ts` updated with `server.host: '0.0.0.0'` so dev mode is reachable when running on a remote server
- User Panel: `vite.config.ts` updated with `server.host: '0.0.0.0'` for the same reason
- `frontend/pnpm-workspace.yaml`: replaced non-standard `allowBuilds` field with pnpm-supported `onlyBuiltDependencies` so esbuild and vue-demi build scripts are auto-approved during install
- Installer: migrations now run reliably by starting API binary briefly (timeout 30s) and verifying table count
- Installer: admin password hash uses `php -r "password_hash(...)"` instead of python3-bcrypt which may not be installed
- Installer: config symlink created at `ZENSPANEL_DIR/src/config.yaml` so API finds config on startup
- Installer: local source detection uses `dirname "${BASH_SOURCE[0]}"` for reliable path resolution
- Installer: socket directory `/run/zenspanel` persisted across reboots via `/etc/tmpfiles.d/zenspanel.conf`
- Installer: phpMyAdmin nginx location block fixed to correctly serve PHP files
- Installer: `pnpm approve-builds --yes` added before frontend build to unblock esbuild/vue-demi
- Installer: `go mod tidy` now runs before build to resolve missing dependencies (fixes build failure on fresh clone)
- `go.sum` committed with complete dependency checksums to prevent missing module errors on fresh install
- Installer: removed `--silent` flag from `pnpm -r build` — Vite 5.4+ does not support this flag (fixes frontend build failure)
- Installer: MySQL setup now uses `sudo mysql` for initial connection — Ubuntu 22.04/24.04 fresh install uses `auth_socket` plugin, not password auth (fixes silent exit at MySQL setup step)
- Installer: MySQL connection now tries all auth methods (no password, password, unix socket) with `mysqladmin ping` readiness check before connecting
- Installer: auto-reset MySQL root password via `skip-grant-tables` when all connection methods fail (handles reinstall scenario)
- Installer: create `/var/run/mysqld` with correct ownership before starting `mysqld_safe` — fixes "directory don't exists" error during password reset
- Installer: kill all mysqld processes and remove stale socket/pid files before restarting MySQL after password reset — fixes "Job for mysql.service failed" error
- Installer: fix nginx config for Admin Panel SPA — use `location ^~ /admin/` with trailing slash, add `include mime.types`, redirect `/admin` → `/admin/`
- Admin Panel: add `base: '/admin/'` to `vite.config.ts` and `createWebHistory('/admin/')` to router — fixes JS module MIME type error when deployed at `/admin/` sub-path

### Added
- `CLAUDE.md` — development guidance for Claude Code with commands and architecture overview
- `README.md` — full documentation with architecture diagram, installation guide, API reference, security notes
- `README.md`: one-run installer section with step-by-step table, requirements table, and access URLs
- `CHANGELOG.md` — Keep a Changelog format with v1.0.0 entries and Unreleased roadmap
- `LICENSE` — MIT License

### Planned
- File Manager with Monaco Editor integration
- Admin: Domains detail page
- Admin: Databases detail page
- Admin: Resource Monitor with real-time charts
- Admin: Backups management page
- Multi-server support (agent-per-node)
- Two-factor authentication (TOTP)
- Per-user Redis isolation
- DNS management
- Email hosting (Postfix/Dovecot)
- GitHub Actions CI/CD pipeline
- Pre-built binary releases

---

## [1.0.0] - 2026-05-17

### Added

#### Core Architecture
- Monolith + Agent Sidecar architecture — two Go binaries communicating via JSON-RPC 2.0 over Unix socket
- `zenspanel-api` — REST API server running as non-root (`www-data`)
- `zenspanel-agent` — privileged sidecar running as root for system operations
- Configuration loading via Viper from `/etc/zenspanel/config.yaml` with env var overrides
- MySQL/MariaDB as panel database with golang-migrate for schema management

#### Database Schema
- `packages` — hosting package templates with resource limits
- `users` — panel users with Linux UID mapping
- `domains` — per-user domains with per-site PHP version
- `databases` — MySQL databases per user
- `resource_limits` — per-user cgroups resource limits
- `php_versions` — available PHP versions with enable/disable toggle
- `ssl_certificates` — SSL certificate tracking with auto-renew flag
- `backups` — backup job tracking
- `api_keys` — external API keys with granular permissions
- `audit_logs` — full audit trail of all user actions

#### Agent Sidecar
- Unix socket JSON-RPC 2.0 server (`/run/zenspanel/agent.sock`)
- Nginx vhost management — create, delete, suspend (503), reload
- PHP-FPM pool management — per-user pool config, multi-version support
- cgroups v2 resource isolation — CPU quota, RAM limit, swap limit per user
- SSL management — Let's Encrypt via certbot, custom PEM upload
- Terminal PTY — isolated rbash sessions via creack/pty
- MySQL management — create/drop databases and users
- Linux user management — useradd/userdel with home directory

#### REST API
- JWT authentication (24h access token, 30d refresh token)
- API key authentication with granular permissions (create_user, read_user, suspend_user, etc.)
- RBAC — admin, user, api_key roles
- User endpoints — CRUD, suspend/unsuspend, package assignment, resource usage
- Package endpoints — CRUD
- Domain endpoints — CRUD, SSL management
- Database endpoints — CRUD, phpMyAdmin SSO token
- PHP version endpoints — list, enable/disable
- API key endpoints — create (shown once), list, revoke
- Audit log endpoints — list with filters
- WebSocket terminal proxy — one-time token, PTY I/O bridge

#### Admin Panel (Vue 3)
- Wide sidebar layout (200px) with section grouping
- Global search bar (⌘K)
- Dashboard — stats cards (users, domains, CPU, RAM), recent users table, server status
- Users — searchable/filterable list, suspend/unsuspend, delete
- User Detail — edit info, change package, toggle terminal/backup
- Packages — CRUD with resource limit configuration, terminal/backup toggles
- PHP Versions — list with enable/disable toggle per version
- API Keys — create with permission checkboxes, show full key once, revoke
- Audit Logs — filterable by user, action, date range
- Light mode only, SVG inline icons (Lucide stroke style), TailwindCSS 3

#### User Panel (Vue 3)
- Wide sidebar with conditional menu items (Terminal/Backups hidden when disabled)
- Dashboard — resource usage bars (domains, databases, disk, RAM), domain table, quick actions
- Domains — add/delete, inline PHP version selector
- SSL Manager — issue Let's Encrypt, upload custom cert, expiry warning badge
- PHP Settings — per-domain PHP version management
- Databases — create with auto-generated password, phpMyAdmin link, delete
- Terminal — xterm.js over WebSocket, reconnect button, disabled state
- Backups — create (full/db/files), download, restore, delete, auto-poll for pending jobs
- Light mode only, SVG inline icons, TailwindCSS 3

#### Resource Isolation
- Linux cgroups v2 slice per user (`/sys/fs/cgroup/zenspanel/<username>/`)
- CPU quota via `cpu.max`
- Memory limit via `memory.max` and `memory.swap.max`
- Disk quota via Linux quota tools
- PHP-FPM pool runs as Linux user (no shared www-data pool)
- Isolated terminal — rbash, home dir only, no sudo, no network tools

#### Installer
- `scripts/install.sh` — single bash installer for Ubuntu 22.04/24.04
- Pre-flight checks — OS, RAM, disk, port availability
- Interactive configuration — domain, MySQL password, admin credentials, Let's Encrypt email
- Installs: Nginx, MySQL, Redis, PHP 8.1/8.2/8.3, certbot, phpMyAdmin, Go 1.22, Node.js 20
- Builds binaries and frontend from source
- Runs database migrations automatically
- Creates admin user with bcrypt password
- Configures systemd services, Nginx reverse proxy, UFW firewall
- Saves credentials to `/etc/zenspanel/install.info` (chmod 600)

#### Security
- Agent socket permissions `0600` — only `www-data` can connect
- All agent RPC inputs validated before execution
- No shell string interpolation — `exec.Command` with argument arrays only
- JWT secrets in config file (chmod 600)
- API keys stored as bcrypt hash, full key shown only once
- Rate limiting on login endpoint (10 attempts/minute/IP)
- All actions logged to audit_logs

### Technical Details
- Go 1.22+, Gin v1.12, sqlx v1.4, golang-migrate v4.19, gorilla/websocket v1.5
- Vue 3.4, Vite 5.2, Pinia 2.1, Vue Router 4.3, TailwindCSS 3.4
- xterm.js 5.3 + xterm-addon-fit 0.8 for terminal
- pnpm workspace monorepo for frontend

---

## Version History

| Version | Date | Description |
|---------|------|-------------|
| 1.0.0 | 2026-05-17 | Initial release |

---

[Unreleased]: https://github.com/teknik-github/zenspanel/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/teknik-github/zenspanel/releases/tag/v1.0.0
