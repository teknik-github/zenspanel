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

### Frontend (Vue 3 + pnpm workspace)

```bash
cd frontend

# Install dependencies
pnpm install

# Dev servers
pnpm --filter @zenspanel/admin dev   # Admin Panel → http://localhost:3000
pnpm --filter @zenspanel/user dev    # User Panel  → http://localhost:3001

# Production build
pnpm --filter @zenspanel/admin build
pnpm --filter @zenspanel/user build

# Build both
pnpm -r build
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

Two independent Vue 3 apps in a pnpm workspace:
- `frontend/apps/admin/` — Admin Panel (port 3000 in dev)
- `frontend/apps/user/` — User Panel (port 3001 in dev)
- `frontend/packages/ui/` — Shared components (not yet populated)

Both apps proxy `/api` and `/ws` to `http://127.0.0.1:8080` in dev (see `vite.config.ts`). Each app has its own `src/api/` (axios modules), `src/stores/` (Pinia), `src/router/`, `src/layouts/`, and `src/pages/`.

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
- Frontend components use `<script setup lang="ts">` (Composition API). TailwindCSS only — no custom CSS files. SVG icons are inline Lucide-style (`stroke="currentColor"`, `fill="none"`).
- Both frontend apps require `postcss.config.js` next to `vite.config.ts`. Without it Tailwind directives are not processed and every page renders unstyled.
- Admin app deploys at `/admin/`: `vite.config.ts` declares `base: '/admin/'` AND `router/index.ts` calls `createWebHistory('/admin/')` — both must agree or the entry script fails to load with a MIME error.
- Both Vue apps await `auth.fetchMe()` in `main.ts` before mounting the router so reload-after-login restores `auth.user.*` flags.
- Terminal and Backups sidebar items in the User Panel are conditionally rendered based on `auth.user.terminal_enabled` / `auth.user.backup_enabled`.

See `CONTRIBUTING.md` for the full rationale and end-to-end recipe for adding a new entity.
