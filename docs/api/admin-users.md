# ZensPanel API — Admin: Users & Auth

All endpoints require `Authorization: Bearer <token>` with role `admin` unless noted.

See [overview.md](./overview.md) for auth format, null-field conventions, and status codes.

---

## Authentication

### POST /auth/login

**No auth required.**

```json
{ "username": "admin", "password": "secret" }
```

**Response 200 (no 2FA):**
```json
{
  "token": "eyJ...",
  "user": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "admin",
    "terminal_enabled": true,
    "backup_enabled": true,
    "package_id": { "Int64": 0, "Valid": false },
    "php_version": "8.3",
    "totp_enabled": false
  }
}
```

Also sets `zenspanel_token` HttpOnly cookie.

**Response 200 (2FA enabled):**
```json
{ "requires_2fa": true, "temp_token": "a1b2c3d4..." }
```

Complete login via `POST /auth/2fa/verify`. See [User Auth Docs](./user-auth.md#two-factor-authentication-2fa).

---

### GET /auth/me

Returns current admin profile. Same shape as the `user` object in login response.

---

### POST /users/:id/impersonate

Mint a 1-hour JWT as the target panel user. Used to debug user accounts from the admin panel.

**Restrictions:** cannot impersonate other admins or suspended users.

**Response 200:**
```json
{
  "token": "eyJ...",
  "user": { "id": 5, "username": "alice", "role": "user" }
}
```

**Response 403:** target is admin or suspended.

---

## Users

### GET /users

List all users with optional filters.

**Query params:**

| Param | Type | Description |
|-------|------|-------------|
| `search` | string | Search by username or email |
| `status` | string | `active` or `suspended` |
| `package_id` | integer | Filter by package ID |
| `sort` | string | `id`, `username`, `email`, `created_at`, `status` |
| `order` | string | `asc` or `desc` (default `asc`) |
| `page` | integer | Page number (default 1) |
| `limit` | integer | Items per page (default 20) |

**Response 200:**
```json
{
  "data": [
    {
      "id": 5,
      "username": "alice",
      "email": "alice@example.com",
      "role": "user",
      "linux_uid": 10001,
      "package_id": { "Int64": 2, "Valid": true },
      "status": "active",
      "terminal_enabled": true,
      "backup_enabled": false,
      "php_version": "8.3",
      "totp_enabled": false,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ],
  "total": 42
}
```

Non-admin callers receive a restricted subset (no `linux_uid`, `role`, `status`) and can only fetch their own ID. See [User Auth Docs](./user-auth.md#get-usersid).

---

### GET /users/:id

Get a single user. Same response shape as the list item above.

---

### POST /users

Create a new hosting user. Provisions: Linux user account, cgroup v2 slice, PHP-FPM pool, disk quota. Row is rolled back if Linux user creation fails.

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "StrongPass123",
  "package_id": 2,
  "terminal_enabled": true,
  "backup_enabled": false,
  "php_version": "8.3"
}
```

Field rules:
- `username`: lowercase letters, digits, underscores; 3–32 chars; cannot be a reserved system name (`root`, `admin`, `www-data`, etc.)
- `php_version`: defaults to `"8.3"` if omitted

**Response 201:**
```json
{
  "id": 5,
  "username": "alice",
  "email": "alice@example.com",
  "role": "user",
  "linux_uid": 10001,
  "package_id": { "Int64": 2, "Valid": true },
  "status": "active",
  "terminal_enabled": true,
  "backup_enabled": false,
  "php_version": "8.3",
  "created_at": "2026-01-01T00:00:00Z",
  "warnings": []
}
```

Non-fatal provisioning failures (e.g. cgroup slice already exists) are reported in `warnings` — the user is still created.

---

### PUT /users/:id

Partial update. Send only the fields you want to change.

Accepted fields: `email`, `role`, `package_id`, `status`, `terminal_enabled`, `backup_enabled`, `php_version`.

Protected fields (`id`, `linux_uid`, `password_hash`) are silently stripped even if sent.

Changing `php_version` triggers a `~/bin/php` symlink update via the agent (reported in `warning` if it fails).

```json
{ "email": "newemail@example.com", "terminal_enabled": false }
```

**Response 200:** `{ "message": "updated" }`

or on symlink failure: `{ "message": "updated", "warning": "setup bin: ..." }`

---

### DELETE /users/:id

Delete user and tear down all resources in order:

subdomains → domains (nginx vhosts + SSL) → databases (MySQL schema + user) → PHP-FPM pools → cgroup slice → disk quota → FileBrowser record → Linux user + home directory → database row

Non-fatal agent failures are reported in `warnings`.

**Response 200:** `{ "message": "deleted", "warnings": [] }`

---

### PUT /users/:id/suspend

Suspends user immediately:
- Replaces all nginx vhosts with 503 pages
- Disables all FTP accounts
- Bumps `token_version` — all existing JWT sessions are immediately invalidated

**Response 200:** `{ "message": "suspended", "warnings": [] }`

---

### PUT /users/:id/unsuspend

Restores user:
- Rebuilds nginx vhosts from DB
- Re-enables FTP accounts
- User must log in again to get a new token

**Response 200:** `{ "message": "unsuspended", "warnings": [] }`

---

### PUT /users/:id/package

Change the user's hosting package. Takes effect immediately: cgroup CPU/RAM limits, I/O limits, and disk quota are updated via the agent.

```json
{ "package_id": 3 }
```

**Response 200:** `{ "message": "package updated" }`

---

### GET /users/:id/usage

Get live resource usage for a user. Non-admin users can call this for their own ID only.

**Response 200:**
```json
{
  "user_id": 5,
  "usage": {
    "domains":   { "used": 2,         "max": 10           },
    "databases": { "used": 1,         "max": 5            },
    "disk":      { "used": 524288000, "max": 10737418240, "files": 500000000, "db": 24288000 },
    "ram":       { "used": 67108864,  "max": 536870912    },
    "cpu":       { "used": 12.5,      "max": 100          }
  }
}
```

All sizes in bytes. `cpu.used` is a percentage (0–100).

---

### GET /users/metrics

Get live cgroup metrics for all active users. Used for the Resource Monitor page. Fetches up to 200 users in parallel with a 3-second timeout per user.

**Response 200:**
```json
{
  "data": [
    {
      "id": 5,
      "username": "alice",
      "ram_used": 67108864,
      "ram_max": 536870912,
      "disk_used": 524288000,
      "disk_max": 10737418240,
      "cpu_pct": 12.5
    }
  ]
}
```

---

## Domains (Admin View)

Admins can read and modify any user's domains. All domain, subdomain, SSL, redirect, hotlink, and log endpoints are documented in [User Domains Docs](./user-domains.md). Key differences for admins:

1. `GET /domains` returns domains for **all users**, not scoped to the caller.
2. `POST /domains` accepts an optional `user_id` to assign the domain to a specific user:
   ```json
   { "domain": "example.com", "php_version": "8.3", "user_id": 5 }
   ```
3. Ownership checks are bypassed — admins can read and modify any resource.
