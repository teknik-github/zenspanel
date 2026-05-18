# Contributing to ZensPanel

This document captures conventions and gotchas that have produced real bugs
in this repo. Read it before adding code that touches the boundaries listed
below — every rule here exists because something broke that wouldn't have if
the rule had been followed.

---

## Table of contents

1. [Backend ↔ Frontend JSON contract](#1-backend--frontend-json-contract)
2. [Dynamic SQL & user-controlled identifiers](#2-dynamic-sql--user-controlled-identifiers)
3. [Agent input validation (defense in depth)](#3-agent-input-validation-defense-in-depth)
4. [Frontend build & deploy invariants](#4-frontend-build--deploy-invariants)
5. [Auth state across page reloads](#5-auth-state-across-page-reloads)
6. [Adding a new entity end-to-end](#6-adding-a-new-entity-end-to-end)
7. [Verification checklist before committing](#7-verification-checklist-before-committing)
8. [Known incomplete features](#8-known-incomplete-features)

---

## 1. Backend ↔ Frontend JSON contract

**Rule:** every field on every struct in `internal/store/models.go` MUST carry
both a `db:"snake_case"` tag and a `json:"snake_case"` tag.

**Why:** Go's `encoding/json` uses the field name as-is when no `json` tag is
present. The Vue frontend (and every other JSON consumer) speaks snake_case.
A struct without `json` tags silently produces PascalCase keys on the wire,
which:

- decoder cannot match incoming snake_case → every field except those Go's
  case-insensitive matcher accidentally hits binds to its zero value
- encoder produces PascalCase → frontend reads `pkg.cpu_quota` → `undefined`

This already cost us a 500 on `POST /api/v1/packages` and a list page that
rendered cards full of `undefined` values.

**Sensitive fields** (`password_hash`, `api_keys.key_hash`) must use
`json:"-"` so they never leak to the API response.

**How to apply:** when adding a struct or column, copy the pattern in
`models.go`:

```go
type Foo struct {
    ID    uint64 `db:"id" json:"id"`
    Name  string `db:"name" json:"name"`
    Hash  string `db:"hash" json:"-"`       // never sent to clients
}
```

A model file without `json` tags must be rejected in review.

---

## 2. Dynamic SQL & user-controlled identifiers

**Rule:** any SQL identifier — table, column, ORDER BY field, UPDATE target —
that originates from a request body or query string MUST pass through an
allowlist before it reaches `Exec`/`Query`. Identifiers cannot be
parameterized; `?` placeholders only protect values.

**Why:** an attacker-controlled `key` in `UPDATE users SET <key> = ?` is
arbitrary SQL injection — and because our DSN sets `multiStatements=true`,
this includes `DROP TABLE users--`. Same applies to `ORDER BY <field>`.

**How to apply:** declare allowlists in `internal/store/safefields.go` and
filter through them. Pattern:

```go
fields = filterAllowed(fields, allowedUserUpdate)  // strips disallowed keys
sort = safeSort(sort, "created_at", allowedUserSort)
```

Whenever you add a column that the frontend should be allowed to mutate,
add it to the matching allowlist. Forgetting to add it produces a no-op,
not a security hole, and that is the point.

**Note:** the `databases` table must always be backtick-quoted in raw SQL
because `databases` is a MySQL reserved word.

---

## 3. Agent input validation (defense in depth)

**Rule:** every exported function in `agent/<subsystem>/` that consumes a
caller-provided string MUST validate it with the matching helper in
`agent/safe/safe.go` *as the first statement of the function body*. Do not
trust the API to have already validated.

**Why:** the agent runs as `root`. The API runs as `www-data` and is the
only intended caller — but anything that can write to `/run/zenspanel/agent.sock`
(misconfigured perms, a future internal tool) can call the agent directly.
Validation duplicated at both layers means a bug at one layer cannot become
a privesc.

**Why double validation despite `exec.Command` arg arrays:** `exec.Command`
protects against shell metacharacters but NOT against:

- MySQL identifiers and password literals interpolated into raw SQL
  (`agent/mysql/mysql.go` cannot use `?` placeholders for `CREATE USER` —
  identifiers and passwords must be sanitized first)
- filesystem paths constructed by string concatenation that get passed to
  `os.WriteFile` / `os.RemoveAll` (e.g. `cgroupBase/<username>/`)
- arguments that downstream tools (certbot, useradd, systemctl) parse with
  their own quirks

**How to apply:** import `github.com/zenspanel/zenspanel/agent/safe` and call
`safe.Username(s)`, `safe.Domain(s)`, `safe.DBIdent(s)`, `safe.DBPassword(s)`,
`safe.PHPVersion(s)` before any side effect. Pattern:

```go
func DeleteSlice(username string) error {
    if err := safe.Username(username); err != nil {
        return err
    }
    return os.RemoveAll(slicePath(username))
}
```

---

## 4. Frontend build & deploy invariants

**Rule 1 — `postcss.config.js` is required in every Vite app.** Without it,
Tailwind directives (`@tailwind base/components/utilities`) pass through
unprocessed and the production CSS is a 60-byte raw passthrough that
generates zero utility classes. Every page renders unstyled. Both
`apps/admin` and `apps/user` ship one.

**Rule 2 — `vite.config.ts` `base` and `vue-router` `createWebHistory` must
agree.** Admin Panel deploys at `/admin/`, so:

- `apps/admin/vite.config.ts` declares `base: '/admin/'`
- `apps/admin/src/router/index.ts` calls `createWebHistory('/admin/')`

If only one is set, the entry `<script src="...">` in `dist/index.html`
points to a path nginx serves as `index.html` (SPA fallback) → browser
rejects it with `Failed to load module script: Expected a JavaScript-or-Wasm
module script but the server responded with a MIME type of "text/html"`.

**Rule 3 — `dist/index.html` and `dist/assets/` must come from the same
build.** Atomic deploy:

```bash
rm -rf /opt/zenspanel/frontend/admin /opt/zenspanel/frontend/user
cp -r apps/admin/dist /opt/zenspanel/frontend/admin
cp -r apps/user/dist  /opt/zenspanel/frontend/user
```

Never copy only `index.html` or only `assets/`. The hash in `index.html`
references chunks in `assets/` by exact filename — mixing builds means
fallback to `index.html` and the same MIME error as above. After deploy,
verify timestamps match: `ls -la /opt/zenspanel/frontend/admin/`.

**Rule 4 — `pnpm-workspace.yaml` uses `onlyBuiltDependencies`, not
`allowBuilds`.** The latter is silently ignored by pnpm, leaving esbuild
build scripts unapproved → install loops asking for confirmation. The
correct field is `onlyBuiltDependencies`.

**Rule 5 — Vite dev server must bind to `0.0.0.0` for remote development.**
Both apps' configs do this; if you remove it, the dev server is only
reachable from localhost on the VPS.

---

## 5. Auth state across page reloads

**Rule:** both Vue apps boot by awaiting `auth.fetchMe()` if a token is in
localStorage, before mounting the router.

**Why:** Pinia state lives in memory. A page reload empties `auth.user` and
keeps only the token in localStorage. Without `fetchMe()` on boot,
components that read `auth.user.terminal_enabled` see `undefined` until the
user manually navigates somewhere that re-fetches — visible as menu items
that flicker on/off.

**How to apply:** see `apps/*/src/main.ts`. Do not move user-state-dependent
checks earlier in the boot sequence.

---

## 6. Adding a new entity end-to-end

The full path for a new resource (call it `Foo`):

1. **Migration**: `migrations/00000N_create_foos.{up,down}.sql`. Use
   `BIGINT UNSIGNED` PKs, `utf8mb4`, `InnoDB`.
2. **Model**: add `Foo` struct in `internal/store/models.go` with full
   `db` + `json` tags. (See §1.)
3. **Store**: `internal/store/foos.go` with `Create/GetByID/List/Update/Delete`.
   If `Update` accepts a `map[string]interface{}`, add a `allowedFooUpdate`
   allowlist in `safefields.go` and filter through `filterAllowed`. (See §2.)
4. **Handler**: `internal/api/handlers/foos.go`. Bind into a struct with
   explicit `json:"..."` tags rather than `map[string]interface{}` whenever
   the field set is known up front — this avoids `float64` corruption of
   numeric fields.
5. **Router**: wire in `internal/api/router.go` with the right
   `auth.RequireRole(...)` middleware. Routes that should be admin-only:
   create, delete, suspend, role/package change.
6. **Wiring**: instantiate the store and handler in `cmd/api/main.go`.
7. **Agent** (only if Foo touches root-only resources): create
   `agent/foos/foos.go`, validate every input via `agent/safe`, register
   the RPC handler in `cmd/agent/main.go`, and call from the API via
   `internal/agent/client.Client.Call()`.
8. **Frontend**: add `apps/<app>/src/api/foos.ts`, a Pinia store if state
   is shared across pages, and the page component.
9. **CHANGELOG.md**: add an entry under `[Unreleased]` describing the
   change. Per project rule, every code change updates the changelog.

---

## 7. Verification checklist before committing

Before `git commit`, run the relevant subset:

```bash
# Go compile (catches signature drift, missing imports)
make build   # produces bin/zenspanel-api and bin/zenspanel-agent

# Go tests
make test

# Frontend type-check + build (catches missing types and Tailwind issues)
cd frontend && pnpm -r build

# After a frontend build, verify CSS was actually processed
ls -la apps/admin/dist/assets/*.css   # should be ~16 KB, not 60 bytes
grep -E 'src=|href=' apps/admin/dist/index.html
# admin: src="/admin/assets/index-XXXX.js"
# user:  src="/assets/index-XXXX.js"
```

If you only touched backend code, the frontend block can be skipped. If you
only touched one frontend app, build only that filter.

For UI changes: spin up the dev servers (`pnpm --filter @zenspanel/admin dev`,
similarly for user) and click through the changed page in a browser before
calling the change done. Type checking does not verify layout or behavior.

---

## 8. Known incomplete features

Captured here so future work doesn't waste time discovering they are stubs:

- **Backups handler** is missing (`internal/api/handlers/`) even though the
  store, migration, and frontend page exist. Clicking backup actions in the
  User Panel currently 404s.
- **API key authentication** is described in README and the store has
  `ValidateKey`, but no middleware wires it up — only JWT auth is enforced.
- **Audit logging** has migration, store, and List endpoint but no producer
  side: nothing in the request flow currently calls `AuditLogStore.Create`.
- **Rate limiting** on the login endpoint is described in the README but not
  implemented.
- **`UserHandler.GetUsage`** returns hardcoded zeros — real cgroup metrics
  collection is not implemented.

When picking these up, do the work end-to-end including a §6-style
checklist and a CHANGELOG entry.
