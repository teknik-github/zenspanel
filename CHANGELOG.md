# Changelog

All notable changes to ZensPanel will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- SSL handler: new `internal/api/handlers/ssl.go` plus `POST /domains/:id/ssl` and `DELETE /domains/:id/ssl` routes. Issue branches on `type`: `letsencrypt` calls `agent.ssl.issue_letsencrypt` with the configured email/staging flags and records a 90-day expiry; `custom` accepts `cert_pem`/`key_pem`, calls `agent.ssl.write_custom_cert`, and parses NotAfter from the cert when possible. Remove calls `agent.ssl.remove_cert` and clears `ssl_type`/`ssl_expires_at`. Frontend SSL Manager already speaks this contract — wires up without UI changes
- Database create/delete now provision the actual MySQL database and user. `POST /databases` calls `agent.mysql.create_database` after the DB row is inserted, rolling the row back on failure. The password from the request is forwarded to MySQL and returned in the 201 response (shown once); it is never stored in the panel DB. `DELETE /databases/:id` calls `agent.mysql.drop_database` first so MySQL state matches the panel
- Domain create/delete now provision the actual nginx vhost. `POST /domains` calls `agent.nginx.create_vhost` after the DB insert and rolls the row back on failure (so the unique domain constraint doesn't block retries). After successful provision, the domain status is flipped from `pending` to `active`. `DELETE /domains/:id` now calls `agent.nginx.delete_vhost` first; if the agent fails the panel row is still removed because an orphan row blocks recreate
- Admin Panel: "Add User" button + modal on the Users page. Form covers username, email, password, package, and the terminal/backup override flags. Uses the existing `POST /api/v1/users` endpoint
- API: `users.Create` now provisions system resources after the DB insert — calls `agent.user.create` (Linux user + home dir) and, when a package is assigned, `agent.cgroups.create_slice` and `agent.phpfpm.create_pool`. If the Linux user step fails the panel row is rolled back. cgroups and phpfpm failures surface as a `warnings` array in the response so the row stands and the admin can retry by reassigning the package
- API: `users.Create` request body now accepts `terminal_enabled` and `backup_enabled` so the admin can toggle these at creation time

### Security
- Admin Panel route guard now enforces `role === 'admin'` on every protected route (previously the guard checked only for the presence of a token, so any authenticated panel user who guessed `/admin/` could browse the admin UI). The guard awaits `auth.fetchMe()` if the user is not yet loaded and logs the session out before redirecting on role mismatch

### Fixed
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
