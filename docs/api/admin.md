# ZensPanel API — Admin Endpoints

All endpoints require `Authorization: Bearer <token>` with role `admin`.

---

## Authentication

### POST /auth/login
Login as admin.

**Request:**
```json
{ "username": "admin", "password": "secret" }
```

**Response 200:**
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
    "package_id": null,
    "php_version": "8.3",
    "totp_enabled": false
  }
}
```

**Response 200 (2FA enabled):**
```json
{ "requires_2fa": true, "temp_token": "a1b2c3..." }
```

---

### GET /auth/me
Get current logged-in user info.

**Response 200:** same shape as login `user` object.

---

### POST /users/:id/impersonate
Mint a 1-hour token to log in as a panel user. Returns to user panel at `/#impersonate=<token>`.

**Response 200:**
```json
{
  "token": "eyJ...",
  "user": { "id": 5, "username": "alice", "role": "user", ... }
}
```

---

## Users

### GET /users
List all users with optional filters.

**Query params:**
| Param | Type | Description |
|-------|------|-------------|
| `search` | string | Search by username or email |
| `status` | string | `active` or `suspended` |
| `package_id` | integer | Filter by package |
| `sort` | string | `id`, `username`, `email`, `created_at`, `status` |
| `order` | string | `asc` or `desc` |
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
      "linux_uid": 2001,
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

---

### GET /users/:id
Get a single user by ID.

**Response 200:** same shape as list item above.

---

### POST /users
Create a new hosting user. Provisions Linux user, cgroup slice, PHP-FPM pool, and filesystem quota.

**Request:**
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

- `username`: lowercase letters, digits, hyphens, 3–32 chars
- `php_version`: defaults to `"8.3"` if omitted

**Response 201:**
```json
{
  "id": 5,
  "username": "alice",
  "email": "alice@example.com",
  "role": "user",
  "linux_uid": 2001,
  "package_id": { "Int64": 2, "Valid": true },
  "status": "active",
  "terminal_enabled": true,
  "backup_enabled": false,
  "php_version": "8.3",
  "created_at": "2026-01-01T00:00:00Z",
  "warnings": []
}
```

Non-fatal provisioning failures are reported in `warnings` but the user is still created.

---

### PUT /users/:id
Update user fields. Accepted fields: `email`, `role`, `package_id`, `status`, `terminal_enabled`, `backup_enabled`, `php_version`.

**Request:**
```json
{ "email": "newemail@example.com", "terminal_enabled": false }
```

**Response 200:**
```json
{ "message": "updated" }
```

---

### DELETE /users/:id
Delete user and tear down all associated resources (nginx vhosts, databases, PHP-FPM pools, cgroup slice, Linux user, home directory).

**Response 200:**
```json
{ "message": "deleted", "warnings": [] }
```

---

### PUT /users/:id/suspend
Suspend user. Disables all nginx vhosts, FTP accounts, and immediately revokes all active JWT sessions (token_version bump).

**Response 200:**
```json
{ "message": "suspended", "warnings": [] }
```

---

### PUT /users/:id/unsuspend
Unsuspend user. Re-enables nginx vhosts and FTP accounts. User must log in again to get a new token.

**Response 200:**
```json
{ "message": "unsuspended", "warnings": [] }
```

---

### PUT /users/:id/package
Change user's hosting package. Updates cgroup limits and disk quota immediately.

**Request:**
```json
{ "package_id": 3 }
```

**Response 200:**
```json
{ "message": "package updated" }
```

---

### GET /users/:id/usage
Get live resource usage for a user.

**Response 200:**
```json
{
  "user_id": 5,
  "usage": {
    "domains":   { "used": 2, "max": 10 },
    "databases": { "used": 1, "max": 5 },
    "disk":      { "used": 524288000, "max": 10737418240, "files": 500000000, "db": 24288000 },
    "ram":       { "used": 67108864, "max": 536870912 },
    "cpu":       { "used": 12.5, "max": 100 }
  }
}
```

All sizes are in bytes. `cpu.used` is a percentage (0–100).

---

### GET /users/metrics
Get live cgroup metrics for all active users. Used for the Resource Monitor page.

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

## Packages

### GET /packages
List all packages (accessible by all authenticated users).

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "name": "Starter",
      "cpu_quota": 50,
      "disk_quota": 10737418240,
      "disk_quota_mb": 10240,
      "memory_limit": 536870912,
      "memory_limit_mb": 512,
      "max_domains": 5,
      "max_databases": 3,
      "max_cron_jobs": 10,
      "max_procs": 50,
      "io_read_bps": 0,
      "io_read_mbps": 0,
      "io_write_bps": 0,
      "io_write_mbps": 0,
      "antivirus_enabled": true,
      "max_ftp_accounts": 3,
      "php_versions_allowed": "8.3,8.2",
      "terminal_enabled": false,
      "backup_enabled": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### GET /packages/:id
Get a single package.

---

### POST /packages
Create a package. Send sizes in MB — they are stored as bytes internally.

**Request:**
```json
{
  "name": "Pro",
  "cpu_quota": 100,
  "disk_quota_mb": 20480,
  "memory_limit_mb": 1024,
  "max_domains": 20,
  "max_databases": 10,
  "max_cron_jobs": 20,
  "max_procs": 100,
  "io_read_mbps": 50,
  "io_write_mbps": 50,
  "antivirus_enabled": true,
  "max_ftp_accounts": 5,
  "php_versions_allowed": "8.3,8.2,8.1",
  "terminal_enabled": true,
  "backup_enabled": true
}
```

**Response 201:** same shape as GET /packages/:id.

---

### PUT /packages/:id
Update a package. Same body as Create.

**Response 200:**
```json
{ "message": "updated" }
```

---

### DELETE /packages/:id

**Response 200:**
```json
{ "message": "deleted" }
```

---

## PHP Versions

### GET /php-versions
List all PHP versions.

**Response 200:**
```json
{
  "data": [
    { "id": 1, "version": "8.3", "fpm_socket": "/run/php/php8.3-fpm.sock", "enabled": true, "created_at": "..." }
  ]
}
```

### GET /php-versions/enabled
List only enabled PHP versions. Used to populate PHP version selects.

### POST /php-versions
Add a PHP version.

**Request:**
```json
{ "version": "8.4" }
```

### PUT /php-versions/:id/enable
Enable a PHP version.

### PUT /php-versions/:id/disable
Disable a PHP version.

### DELETE /php-versions/:id
Remove a PHP version.

---

## PHP Extensions

### GET /admin/php-extensions
List all extensions in the global catalog, grouped by PHP version.

**Response 200:**
```json
{
  "data": [
    { "id": 1, "name": "redis", "php_version": "8.3", "enabled": true }
  ]
}
```

### POST /admin/php-extensions
Add an extension to the catalog.

**Request:**
```json
{ "name": "imagick", "php_version": "8.3", "enabled": true }
```

### PUT /admin/php-extensions/:id
Enable or disable an extension globally. Disabling re-loads all affected user FPM pools immediately.

**Request:**
```json
{ "enabled": false }
```

### DELETE /admin/php-extensions/:id
Remove extension from catalog.

### POST /admin/php-extensions/seed
Auto-populate the catalog with common extensions for all enabled PHP versions.

---

## Firewall

### GET /admin/firewall/blocked
List all blocked IPs (panel blocks + fail2ban bans merged).

**Response 200:**
```json
{
  "data": [
    { "ip": "1.2.3.4", "reason": "brute force", "source": "panel", "blocked_at": "2026-01-01T00:00:00Z" },
    { "ip": "5.6.7.8", "reason": "fail2ban-sshd", "source": "fail2ban", "blocked_at": "..." }
  ]
}
```

### POST /admin/firewall/block
Block an IP or CIDR.

**Request:**
```json
{ "ip": "1.2.3.4", "reason": "manual block" }
```

### POST /admin/firewall/unblock
Unblock an IP.

**Request:**
```json
{ "ip": "1.2.3.4" }
```

### GET /admin/firewall/fail2ban/jails
List fail2ban jails and their status.

**Response 200:**
```json
{
  "data": [
    { "name": "sshd", "enabled": true, "ban_count": 5, "currently_banned": 2 }
  ]
}
```

### PUT /admin/firewall/fail2ban/jails/:name
Enable or disable a fail2ban jail.

**Request:**
```json
{ "enabled": false }
```

---

## IP Allowlist

Restrict `/admin/` access to specific IPs. Empty list = allow all.

### GET /admin/ip-allowlist

**Response 200:**
```json
{
  "data": [
    { "id": 1, "ip": "203.0.113.0/24", "note": "office", "created_at": "..." }
  ]
}
```

### POST /admin/ip-allowlist

**Request:**
```json
{ "ip": "203.0.113.0/24", "note": "office" }
```

### DELETE /admin/ip-allowlist/:id

---

## Backup Targets

S3-compatible remote backup destinations.

### GET /admin/backup-targets

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "name": "Wasabi",
      "type": "s3",
      "bucket": "my-backups",
      "prefix": "zenspanel/",
      "access_key": "AKIAIOSFODNN7EXAMPLE",
      "region": "us-east-1",
      "endpoint": "https://s3.wasabisys.com",
      "enabled": true
    }
  ]
}
```

### POST /admin/backup-targets

**Request:**
```json
{
  "name": "Wasabi",
  "type": "s3",
  "bucket": "my-backups",
  "prefix": "zenspanel/",
  "access_key": "AKIAIOSFODNN7EXAMPLE",
  "secret_key_enc": "plaintext-secret-will-be-encrypted",
  "region": "us-east-1",
  "endpoint": "https://s3.wasabisys.com"
}
```

### PUT /admin/backup-targets/:id
Update a target. Same body as Create.

### DELETE /admin/backup-targets/:id

### POST /admin/backup-targets/:id/test
Test connectivity to the backup target.

**Response 200:**
```json
{ "ok": true }
```

**Response 200 (failure):**
```json
{ "ok": false, "error": "connection refused" }
```

---

## API Keys

### GET /api-keys
List all API keys (key hash is never returned; only prefix is shown).

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "name": "WHMCS Integration",
      "key_prefix": "zpk_live_a1b2",
      "permissions": "read_user,create_user,suspend_user",
      "last_used_at": { "Time": "2026-01-01T00:00:00Z", "Valid": true },
      "expires_at": { "Time": "0001-01-01T00:00:00Z", "Valid": false },
      "created_by": 1,
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

### POST /api-keys
Create an API key. The full key is only shown once.

**Request:**
```json
{
  "name": "WHMCS Integration",
  "permissions": "read_user,create_user,suspend_user,change_package,read_package"
}
```

**Response 201:**
```json
{
  "id": 1,
  "name": "WHMCS Integration",
  "key": "zpk_live_a1b2c3d4e5f6...",
  "key_prefix": "zpk_live_a1b2",
  "permissions": "read_user,create_user,suspend_user,change_package,read_package"
}
```

### DELETE /api-keys/:id
Revoke an API key.

---

## Audit Logs

### GET /audit-logs
List audit log entries.

**Query params:** `page`, `limit`, `search`

**Response 200:**
```json
{
  "data": [
    {
      "id": 100,
      "user_id": { "Int64": 1, "Valid": true },
      "action": "POST /api/v1/users",
      "resource": { "String": "user:5", "Valid": true },
      "ip_address": "203.0.113.10",
      "user_agent": { "String": "Mozilla/5.0...", "Valid": true },
      "meta": { "String": "", "Valid": false },
      "created_at": "2026-01-01T00:00:00Z"
    }
  ],
  "total": 500
}
```

---

## System

### GET /system/stats
Get host-level stats (CPU, RAM, disk, load, service status).

**Response 200:**
```json
{
  "cpu_pct": 23.5,
  "ram_used": 1073741824,
  "ram_total": 4294967296,
  "disk_used": 21474836480,
  "disk_total": 107374182400,
  "load_1": 0.42,
  "load_5": 0.38,
  "load_15": 0.35,
  "services": {
    "nginx": "running",
    "mysql": "running",
    "php8.3-fpm": "running"
  }
}
```

### GET /system/version
Get panel version info.

**Response 200:**
```json
{
  "current_sha": "345602c",
  "branch": "main",
  "version": "v2.0.0"
}
```

### GET /system/update/check
Check for available updates by comparing HEAD with origin/main.

**Response 200:**
```json
{
  "current_sha": "345602c",
  "latest_sha": "abc1234",
  "behind_by": 3,
  "changelog": "- fix: package fields\n- feat: suspend user",
  "current_branch": "main",
  "download_url": "https://github.com/.../releases/download/v2.0.1/...",
  "release_tag": "v2.0.1"
}
```

### POST /system/update/run
Trigger a panel update (async). Poll `/system/update/status` for progress.

**Response 200:**
```json
{ "started": true }
```

### GET /system/update/status

**Response 200:**
```json
{
  "phase": "building_api",
  "log": ["pulling...", "go build ./cmd/api..."],
  "done": false,
  "error": ""
}
```

Phases: `pulling` → `building_api` → `building_agent` → `building_frontend` → `deploying_frontend` → `restarting` → `done` / `failed`

### POST /system/maintenance
Toggle maintenance mode.

**Request:**
```json
{ "enabled": true }
```

---

## Admin Terminal

### POST /admin/terminal/token
Mint a terminal token for the panel system user or a specific panel user.

**Request:**
```json
{ "username": "alice" }
```

Leave `username` empty to open a shell as the `zenspanel` system user.

**Response 200:**
```json
{ "token": "a1b2c3d4..." }
```

Then connect via `WS /ws/terminal?token=<token>`. See [Overview — WebSocket Terminal](./overview.md).

---

## Domains (Admin view)

Admins can see all domains. See [User Docs — Domains](./user.md) for full endpoint shapes. Key difference: `GET /domains` returns all domains across all users when called as admin.

When creating a domain as admin, pass `user_id` in the body to assign to a specific user:
```json
{ "domain": "example.com", "php_version": "8.3", "user_id": 5 }
```
