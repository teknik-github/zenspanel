# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend (Go)

```bash
# Build all binaries
make build
# Output: bin/zenspanel-api, bin/zenspanel-agent, bin/zenspanel-cli

# Run API server (reads config.yaml from current dir)
make run-api

# Run agent sidecar (requires root for system operations)
sudo make run-agent

# Run tests
make test

# Run a single test
go test ./internal/config/... -v -run TestLoadDefaults

# Hot reload during development
make dev   # requires: go install github.com/air-verse/air@latest

# Lint
make lint  # requires: golangci-lint
```

### Frontend (Nuxt 4 SSR — Dashboard/)

```bash
cd Dashboard

# Install dependencies
pnpm install

# Dev server (Admin + User Panel → http://localhost:3000)
pnpm dev

# Production build (output: Dashboard/.output/)
pnpm build

# Preview production build locally
pnpm preview
```

### Database

```bash
# Migrations run automatically on API startup
# To run manually against local MySQL:
go run ./cmd/api

# Rollback one migration (using golang-migrate CLI)
migrate -path migrations -database "mysql://user:pass@tcp(host)/db" down 1
```

---

## Architecture

### Three-binary design

The project compiles to three Go binaries:

- **`zenspanel-api`** (`cmd/api/`) — Gin HTTP server, runs as `www-data` (non-root). Handles all business logic, auth, and API routing. Calls the agent for any privileged operation.
- **`zenspanel-agent`** (`cmd/agent/`) — Runs as `root`. Listens on a Unix socket and executes system operations: writing nginx configs, managing PHP-FPM pools, cgroups v2, certbot, PTY terminal sessions, MySQL user/DB creation, Linux user management.
- **`zenspanel-cli`** (`cmd/cli/`) — Bubble Tea TUI for managing panel screens and server configuration interactively.

The API and agent communicate via **JSON-RPC 2.0 over a Unix socket** (`/run/zenspanel/agent.sock`).

The agent RPC server is in `agent/server.go`. All handlers are registered in `cmd/agent/main.go`. Each subsystem is a separate package under `agent/` (nginx, phpfpm, cgroups, ssl, terminal, mysql, user).

The API calls the agent via `internal/agent/client.go` which dials the socket per-call (no persistent connection).

### Request flow

```
Vue → GET/POST /api/v1/... → internal/api/router.go
  → auth middleware (internal/auth/middleware.go)
  → handler (internal/api/handlers/*.go)
  → store (internal/store/*.go) for DB reads/writes
  → agent client (internal/agent/client.go) for system ops
    → agent/server.go dispatches to agent/<subsystem>/<subsystem>.go
  → audit middleware (internal/api/middleware/audit.go) logs every API call to audit_logs
  → rate limiter (internal/api/middleware/ratelimit.go) on login endpoint
```

### Internal packages

- `internal/config/` — Viper-based config loading. Config file: `/etc/zenspanel/config.yaml` or `./config.yaml`.
- `internal/store/` — sqlx data access layer. One file per entity (`users.go`, `domains.go`, etc.) plus `models.go` for all structs, `db.go` for connection, `migrate.go` for migrations.
- `internal/auth/` — JWT generation/validation (`jwt.go`) and Gin middleware (`middleware.go`). Three roles: `admin`, `user`, `api_key`.
- `internal/api/middleware/` — Audit logging (`audit.go`) writes every API call to `audit_logs`. Rate limiting (`ratelimit.go`) on login endpoint (in-memory, single-server only — swap to Redis for multi-server).
- `internal/api/router.go` — All routes wired here with middleware. Groups: public (`/api/v1/auth/login`), JWT-protected (`/api/v1/`), WebSocket (`/ws/`).
- Redis (`go-redis/v9`) — used for session/cache, configured from config.yaml.

### Frontend structure

Single Nuxt 4 SSR app in `Dashboard/`:
- `Dashboard/app/pages/` — File-based routing. `/admin/*` = admin panel, all other routes = user panel.
- `Dashboard/app/layouts/` — Shared layout with sidebar navigation.
- `Dashboard/app/composables/useAuth.ts` — `useAuth()` shared composable managing JWT state via `useState`.
- `Dashboard/app/middleware/auth.global.ts` — Global route guard: enforces role-based access, redirects unauthenticated users.
- `Dashboard/server/api/v1/[...path].ts` — Nitro server-side proxy: forwards all `/api/v1/**` to Go API at `backendUrl` (runtime config), forwarding cookies and auth headers.

In dev, `pnpm dev` runs Nuxt on port 3000. All `/api/v1/**` calls are handled server-side by the Nitro proxy — no Vite proxy config needed.

### Database

MySQL/MariaDB. Panel DB name: `zenspanel`. Migrations in `migrations/` numbered `000001`–`000024` (48 .sql files). The `databases` table uses backtick quoting in SQL because `databases` is a MySQL reserved word — keep this in mind when writing raw queries against that table.

### Resource isolation

Each panel user maps to a Linux system user. cgroups v2 slice at `/sys/fs/cgroup/zenspanel/<username>/`. PHP-FPM pool socket: `/run/php/zenspanel-<username>-<phpversion>.sock`. Nginx vhosts in `/etc/nginx/zenspanel/<domain>.conf`.

### Key conventions

- **Every code change must update `CHANGELOG.md`** — add an entry under `[Unreleased]` (`### Added`, `### Fixed`, `### Changed`, or `### Removed`). Commit together with the change. Do NOT add `Co-Authored-By` lines to commit messages (per project policy).
- Agent commands always use `exec.Command` with argument arrays — never shell string interpolation.
- Every exported function in `agent/<subsystem>/` validates its caller-provided strings via `agent/safe` (`safe.Username`, `safe.Domain`, `safe.DBIdent`, `safe.DBPassword`, `safe.PHPVersion`) before any side effect — defense in depth, the API may not be the only caller.
- Store methods use named sqlx queries (`:field` syntax) for inserts/updates.
- Every struct in `internal/store/models.go` carries both `db:"snake_case"` and `json:"snake_case"` tags. Sensitive columns (`password_hash`, `api_keys.key_hash`) use `json:"-"`. Without `json` tags Go produces PascalCase keys that the frontend cannot bind.
- Dynamic SQL identifiers (UPDATE columns, ORDER BY) sourced from request input must pass through allowlists in `internal/store/safefields.go` (`filterAllowed`, `safeSort`). Identifiers cannot be parameterized; this is the only safe option.
- The `databases` table must always be quoted as `` `databases` `` in raw SQL.
- Dashboard components use `<script setup lang="ts">` (Composition API). `@nuxt/ui` for UI primitives; Lucide icons via `@iconify-json/lucide`. TailwindCSS 4 — no custom CSS beyond `Dashboard/app/assets/css/main.css`.
- Admin routes are prefixed `/admin/`. The global middleware (`auth.global.ts`) enforces this — do not use Nuxt `definePageMeta` role checks; add routes to `ADMIN_ROUTES` or `USER_ROUTES` in the middleware instead.
- All API calls use `$fetch('/api/v1/...')` with relative paths. The Nitro proxy (`server/api/v1/[...path].ts`) handles forwarding to Go API on both server-side (SSR) and client-side renders — never call Go API directly from components.
- `useAuth()` composable manages session state. Call `auth.fetchMe()` to restore session after page reload; `auth.user.value` is `null` when unauthenticated.
- `terminal_enabled` and `backup_enabled` on `auth.user` control sidebar item visibility.

See `CONTRIBUTING.md` for the full rationale and end-to-end recipe for adding a new entity.
