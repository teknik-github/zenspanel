# ZensPanel Architecture

## Overview

ZensPanel is a Go-based web hosting panel with a Vue 3 frontend. Three binaries, one database, one Unix socket.

```
┌────────────────────────────────────────────────────┐
│ Browser (Vue 3 SPA)                                │
│  Admin: /admin/*  User: /*                         │
└──────────┬─────────────────────────────────────────┘
           │ HTTP (REST + WebSocket)
           ▼
┌──────────────────────────────────────────────────────┐
│ Nginx (reverse proxy, TLS termination)               │
│  /api/v1/* → zenspanel-api :8080                     │
│  /ws/*     → zenspanel-api :8080                     │
│  /filebrowser/ → filebrowser :8081                   │
│  /admin/   → static SPA files                        │
│  /*        → static user SPA files                   │
└──────────┬───────────────────────────────────────────┘
           │
           ▼
┌──────────────────────────────────────────────────────┐
│ zenspanel-api (Gin, runs as www-data)                │
│  Auth (JWT), routing, handlers, store (sqlx),        │
│  agent client (JSON-RPC over Unix socket)            │
└──────────┬───────────────────────────────────────────┘
           │ JSON-RPC 2.0 over Unix socket
           │ /run/zenspanel/agent.sock
           ▼
┌──────────────────────────────────────────────────────┐
│ zenspanel-agent (runs as root)                       │
│  System operations: nginx, PHP-FPM, cgroups,         │
│  SSL (certbot), MySQL, Linux users, firewall,        │
│  filemanager, terminal (PTY), quota (setquota)       │
└──────────────────────────────────────────────────────┘
```

---

## Three-Binary Design

| Binary | User | Purpose |
|--------|------|---------|
| `zenspanel-api` | `www-data` | HTTP API, auth, DB, business logic |
| `zenspanel-agent` | `root` | System operations, listens on Unix socket |
| `zenspanel-cli` | any | Bubble Tea TUI for server config |

API and agent communicate via **JSON-RPC 2.0 over a Unix socket** at `/run/zenspanel/agent.sock`. The API never runs privileged operations directly — it calls the agent, which validates the request and executes it as root.

This is privilege separation without a network boundary: both processes are on the same host, but the agent runs as a different user (root) and validates every input through `agent/safe/` before any side effect.

---

## Request Flow

```
Browser (admin or user)
  → Nginx (TLS, static files, reverse proxy)
  → Gin router (internal/api/router.go)
    → JWT middleware (internal/auth/middleware.go)
    → Role check (RequireRole)
    → Audit middleware (internal/api/middleware/audit.go)
    → Handler (internal/api/handlers/<entity>.go)
      → Store (internal/store/<entity>.go) — DB read/write
      → Agent client (internal/agent/client.go) — system ops
        → agent/server.go dispatches to agent/<subsystem>/<subsystem>.go
  → JSON response
```

### Middleware pipeline (in order)

1. **JWT middleware** — extracts Bearer token or cookie, validates signature, checks `token_version` against DB for session revocation
2. **Role check** (`RequireRole`) — gates admin-only routes (`"admin"`), user routes (`"user"`), or public API key routes (`"api_key"`)
3. **Audit middleware** — writes every mutating request to `audit_logs` table (method, path, user_id, IP, user agent, metadata)

### External API (billing integration)

A separate route group under `/api/v1/external` uses **X-API-Key header** auth instead of JWT. Each key has a comma-separated permissions string (`read_user`, `create_user`, `suspend_user`, `change_package`, `read_package`). Permissions are checked per-route via `RequirePermission`.

---

## Directory Structure

```
cmd/
  api/main.go           — API binary entry point
  agent/main.go         — Agent binary entry point
  cli/main.go           — CLI TUI entry point
  seed/main.go          — DB seed script (dev only)

internal/
  agent/client.go       — JSON-RPC client (dials Unix socket)
  config/config.go      — Viper-based config loading
  auth/
    jwt.go              — JWT generate/validate
    middleware.go        — Gin JWT middleware, role checks, API key auth
  api/
    router.go           — All routes wired with middleware
    middleware/
      audit.go          — Audit logging to audit_logs table
      ratelimit.go      — In-memory + Redis rate limiters
    handlers/
      packages.go       — PackageHandler (CRUD + role-scoped DTOs)
      users.go           — UserHandler (CRUD, suspend, metrics, usage)
      domains.go         — DomainHandler
      subdomains.go      — SubdomainHandler
      databases.go       — DatabaseHandler
      auth.go            — AuthHandler (login, me, 2FA, impersonate)
      files.go / filemanager.go — FileManagerHandler
      terminal.go        — TerminalHandler (token mint + WS bridge)
      ... (25 handler files, one per entity)
  store/
    models.go           — All DB structs (db + json tags)
    db.go               — sqlx connection pool
    migrate.go          — Auto-migration on startup
    safefields.go       — Allowlist for dynamic SQL identifiers
    packages.go         — PackageStore
    users.go            — UserStore
    domains.go          — DomainStore
    ... (one store file per entity)

agent/
  server.go             — JSON-RPC server (Unix socket listener)
  safe/                 — Input validators (Username, Domain, DBIdent, etc.)
  nginx/                — nginx vhost create/delete/suspend
  phpfpm/               — PHP-FPM pool create/delete
  cgroups/              — cgroups v2 slice + metrics (cpu.stat, memory.current)
  mysql/                — CREATE/DROP database, user grants, password reset
  user/                 — Linux useradd/userdel, ~/bin symlinks
  ssl/                  — certbot issue/renew
  terminal/             — PTY spawn (su -s /bin/rbash)
  backup/               — tar + rsync to S3 targets
  firewall/             — ufw + fail2ban wrappers
  files/                — In-process ops: chmod, copy, compress, extract
  ftp/                  — vsftpd virtual user management
  quota/                — setquota wrapper (ext4/XFS)
  updater/              — Git pull + Go build + pnpm build pipeline

frontend/
  apps/
    admin/              — Admin SPA (Vue 3 + Vite + Tailwind)
    user/               — User SPA (Vue 3 + Vite + Tailwind)
  packages/ui/          — Shared components (stub)

migrations/             — 24 migration pairs (48 .sql files), auto-applied
scripts/
  install.sh            — Full server provisioning script
docs/
  api/
    overview.md         — Auth, format, WebSocket protocol
    admin.md            — Admin endpoint reference
    user.md             — User endpoint reference
  seed.md               — Seed script usage guide
  architecture.md       — This document
```

---

## Auth & Session Model

### JWT tokens
- **Algorithm**: HS256 (HMAC-SHA256)
- **Claims**: `user_id`, `role`, `impersonated_by` (optional), `token_version`
- **Expiry**: configurable (default 24h)
- **Transmission**: `Authorization: Bearer <token>` header or `zenspanel_token` cookie (HttpOnly, SameSite=Strict, Secure on HTTPS)

### Session revocation
- `users.token_version` column is incremented (`BumpTokenVersion`) on suspend
- JWT carries `token_version` claim set at issue time
- Middleware compares claim vs DB — if DB version is higher, the token is rejected (401 "session revoked")
- This revokes ALL active sessions for that user instantly without a blocklist

### Roles

| Role | Source | Scope |
|------|--------|-------|
| `admin` | JWT (password login) | All admin routes + user routes |
| `user` | JWT (password login) | Own resources only (ownership check in every handler) |
| `api_key` | X-API-Key header | `/api/v1/external/*` subset, permission-gated |

### 2FA (TOTP)
- Optional per-user. Setup returns QR URL + 8 recovery codes.
- Login with 2FA enabled returns `{requires_2fa: true, temp_token: "..."}` instead of JWT
- `POST /auth/2fa/verify` with `{temp_token, code}` completes login and returns JWT
- Recovery codes are one-time-use and disable 2FA on use

---

## Database

### Schema
- MySQL/MariaDB, database name `zenspanel`
- ~20 tables: `users`, `packages`, `domains`, `subdomains`, `databases`, `ftp_accounts`, `backups`, `cron_jobs`, `php_versions`, `php_extensions`, `api_keys`, `audit_logs`, `backup_targets`, `admin_allowed_ips`, `redirects`, `hotlink_configs`, `antivirus_alerts`, `ssl_certificates`
- Auto-migration on API startup (reads `migrations/` directory, applies in order)
- `databases` table is backtick-quoted in SQL because `databases` is a MySQL reserved word

### Store layer
- `internal/store/` — one file per entity with a corresponding `*Store` struct
- All database access goes through sqlx (`jmoiron/sqlx`)
- Named queries (`:field` syntax) for inserts/updates
- `NullInt64` / `NullString` custom types for nullable columns (distinct from Go's zero values)
- `safefields.go` — allowlists for dynamic SQL identifiers (UPDATE columns, ORDER BY) sourced from user input. Identifiers cannot be parameterized; this is the only safe option

### Models
- Every struct in `internal/store/models.go` carries both `db:"snake_case"` and `json:"snake_case"` tags
- Sensitive columns (`password_hash`, `totp_secret_enc`, `totp_recovery_codes`, `api_keys.key_hash`) use `json:"-"` to prevent accidental serialization
- Without `json` tags Go produces PascalCase keys that the frontend cannot bind

---

## Agent Communication Protocol

### JSON-RPC 2.0 over Unix socket

The API calls the agent via `internal/agent/client.go`:

```go
client := agent.NewClient("/run/zenspanel/agent.sock")
var result SomeStruct
err := client.Call("nginx.create_vhost", map[string]interface{}{
    "domain":       "example.com",
    "document_root": "/home/user/public_html/example.com",
    "php_version":  "8.3",
}, &result)
```

Each call dials the socket, sends a JSON-RPC 2.0 request, reads the response, and closes the connection. No persistent connection — simplicity over performance for an admin panel where RPC calls are infrequent (one per API request).

### Registered RPC methods

```
user.create / user.delete / user.setup_bin
nginx.create_vhost / nginx.delete_vhost / nginx.suspend_all_vhosts / nginx.unsuspend_all_vhosts
phpfpm.create_pool / phpfpm.delete_pool
cgroups.create_slice / cgroups.update_slice / cgroups.delete_slice / cgroups.read_metrics
mysql.create_database / mysql.drop_database / mysql.reset_password / mysql.enforce_db_quota / mysql.get_db_size
ssl.issue_cert / ssl.remove_cert / ssl.renewed_hook
terminal.spawn / terminal.stream
backup.run / backup.restore
filemanager.chmod / filemanager.copy / filemanager.compress / filemanager.extract / filemanager.upload
ftp.create_user / ftp.delete_user / ftp.suspend_user / ftp.unsuspend_user
firewall.block / firewall.unblock / firewall.list_blocked / firewall.list_jails / firewall.set_jail
quota.set / quota.delete / quota.read
filebrowser.user_create / filebrowser.user_delete
update.check / update.run / update.status
```

### Input validation: agent/safe

Every exported function in `agent/<subsystem>/` validates its caller-provided strings via `agent/safe`:

```go
safe.Username(name)      // ^[a-z][a-z0-9-]{2,31}$
safe.Domain(domain)       // ^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*\.[a-z]{2,}$
safe.DBIdent(name)        // ^[a-zA-Z_][a-zA-Z0-9_]{0,63}$
safe.DBPassword(pw)       // printable ASCII, no spaces, 8-64 chars
safe.PHPVersion(ver)      // ^\d+\.\d+$
```

This is defense in depth — the API may not be the only caller of agent functions.

---

## Security Design

### Response splitting by role (commit `47d7ff8`)

Not all callers see the same data. Handlers return role-scoped DTOs:

| Endpoint | Admin sees | User sees |
|----------|-----------|-----------|
| `GET /users/:id` | Full: `id, username, email, role, linux_uid, package_id, status, terminal_enabled, backup_enabled, php_version, totp_enabled, created_at, updated_at` | Safe: `id, username, email, package_id, terminal_enabled, backup_enabled, php_version, totp_enabled, created_at, updated_at` |
| `GET /packages` | Full: all limits including `cpu_quota, max_procs, io_read_bps, io_write_bps` | Customer: `disk_quota_mb, memory_limit_mb, max_domains, max_databases, max_cron_jobs, max_ftp_accounts, feature flags` |
| `GET /packages/:id` | Same as List | Same as List |

Sensitive columns (`password_hash`, `totp_secret_enc`, `totp_recovery_codes`, `api_keys.key_hash`, `users.token_version`) are tagged `json:"-"` in models — they never serialize regardless of role.

### Ownership checks (IDOR prevention)

Every handler that operates on a single resource validates ownership before acting:

```go
if auth.GetRole(c) == "user" && resource.UserID != auth.GetUserID(c) {
    c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
    return
}
```

Pattern: use `auth.GetRole(c) == "user"` (not `!= "admin"`) so `api_key` callers with id=0 fall through to 403, not accidentally passing the check.

### Other security measures
- **Rate limiting**: Login endpoint — Redis sliding window (multi-server) with in-memory fallback (single-server)
- **IP allowlist**: `/admin/` access can be restricted to specific IPs/CIDRs via `admin_allowed_ips` table
- **Cookie hardening**: HttpOnly, SameSite=Strict, Secure on HTTPS
- **WebSocket origin check**: Terminal WS handler validates Origin header matches request Host
- **Agent input validation**: All system calls go through `agent/safe` validators before exec

---

## Frontend Architecture

### Two independent SPAs

```
frontend/
  apps/admin/   — port 3000 (dev), base /admin/
  apps/user/    — port 3001 (dev), base /
  packages/ui/  — shared components (stub)
```

Both use Vue 3 Composition API (`<script setup lang="ts">`), Vite, TailwindCSS, Pinia stores, and Vue Router.

### Build toolchain
- `pnpm` workspace (not npm/yarn)
- `postcss.config.js` required next to `vite.config.ts` in each app (Tailwind won't process without it)
- Admin app: `vite.config.ts` sets `base: '/admin/'` AND `router/index.ts` calls `createWebHistory('/admin/')` — both must agree
- Dev proxy: `/api` and `/ws` → `http://127.0.0.1:8080`

### Auth flow (frontend)
1. `main.ts` calls `auth.fetchMe()` before mounting the router
2. On success → router mounts, navigation guard checks `auth.user.role`
3. On failure → redirect to login
4. `auth.user.terminal_enabled` / `auth.user.backup_enabled` control sidebar menu visibility

### API client (`src/api/client.ts`)
- Axios instance with JWT interceptor
- 401 response → clear auth + redirect to login
- Base URL proxied to API in dev, same-origin in production (nginx)

---

## Known Improvement Areas

These are architectural notes, not bugs. Prioritize after feature completeness.

1. **Response wrapper standardization** — Some endpoints return `gin.H{"data": ...}`, some return bare objects, some return `gin.H{"message": ...}`. A consistent envelope (e.g., `{status, data, error}`) would simplify frontend error handling.

2. **router.go Setup() is long** — 80+ routes in one function. Could be split into `registerUserRoutes()`, `registerAdminRoutes()`, `registerExternalRoutes()`.

3. **DTO layer** — Role-scoped response functions (`userAdminResponse`, `packageUserResponse`) currently live inline in handler files. If the number grows, move to `internal/api/dto/` or `internal/api/response/`.

4. **Agent error handling consistency** — Some agent calls append errors to `warnings` (non-fatal), others return 500 immediately. No single pattern. A structured `ProvisionResult` type could unify this.

5. **Persistent agent connection** — `agent/client.go` dials the Unix socket per-call. For the terminal WebSocket bridge, a persistent connection would reduce latency. The current approach is fine for CRUD operations.

6. **`databases` reserved word** — The `databases` table name requires backtick quoting in raw SQL. Renaming to `user_databases` would be cleaner but requires migration coordination.

7. **Frontend shared components** — `frontend/packages/ui/` exists but is not populated. Admin and user apps duplicate layout patterns (sidebar, header, empty states). Extracting these would reduce drift.

---

## Key Conventions

- **Every code change updates `CHANGELOG.md`** — add entry under `[Unreleased]` in the appropriate section (`### Added`, `### Fixed`, `### Changed`, `### Security`, `### Removed`)
- **No `Co-Authored-By`** in commit messages
- **Agent commands use `exec.Command` with argument arrays** — never shell string interpolation
- **Store methods use named sqlx queries** (`:field` syntax)
- **Dynamic SQL identifiers go through `safefields.go` allowlists**
- **`databases` table always backtick-quoted in raw SQL** (`` `databases` ``)
- **Frontend uses TailwindCSS only** — no custom CSS files
- **SVG icons are Lucide-style** (`stroke="currentColor"`, `fill="none"`)

---

## Running the Stack

```bash
# Backend: build all binaries
make build                           # → bin/zenspanel-api, bin/zenspanel-agent, bin/zenspanel-cli

# Backend: dev with hot reload
make dev                             # requires: go install github.com/air-verse/air@latest

# Backend: tests
make test                            # go test ./...

# Backend: lint
make lint                            # requires: golangci-lint

# Frontend: install + dev
cd frontend && pnpm install
pnpm --filter @zenspanel/admin dev   # → http://localhost:3000/admin/
pnpm --filter @zenspanel/user dev    # → http://localhost:3001/

# Frontend: production build
pnpm --filter @zenspanel/admin build
pnpm --filter @zenspanel/user build

# Seed DB for frontend dev
go run ./cmd/seed --wipe             # reset + fill with dummy data
```

Full workflow for frontend development:

```bash
# Terminal 1: API
go run ./cmd/api                     # reads config.yaml

# Terminal 2: Seed (first time only)
go run ./cmd/seed --wipe

# Terminal 3: Frontend
cd frontend
pnpm --filter @zenspanel/admin dev
pnpm --filter @zenspanel/user dev

# Login: admin/admin123 or alice/user123
```

---

*Last updated: 2026-06-08*
