# ZensPanel API — PHP, Terminal, Installer & Antivirus

All endpoints require `Authorization: Bearer <token>` with role `user`.

See [overview.md](./overview.md) for auth format, null-field conventions, and status codes.

---

## PHP Extensions

### GET /php-extensions

List available extensions for the user's configured PHP version. Only admin-enabled extensions appear.

**Query params:** `?php_version=8.3` (defaults to user's configured version)

**Response 200:**
```json
{
  "data": [
    { "id": 1, "name": "redis",   "php_version": "8.3", "admin_enabled": true, "user_enabled": true  },
    { "id": 2, "name": "imagick", "php_version": "8.3", "admin_enabled": true, "user_enabled": false }
  ]
}
```

---

### PUT /php-extensions

Toggle an extension for the user's own PHP-FPM pool. Cannot enable if `admin_enabled` is `false`.

```json
{ "name": "redis", "php_version": "8.3", "enabled": true }
```

**Response 200:** `{ "message": "updated" }`

**Response 403:** extension is disabled globally by admin.

---

## Packages (read-only)

### GET /packages

List available packages. Users receive a customer-visible subset (no internal byte values or I/O limits).

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "name": "Starter",
      "disk_quota_mb": 10240,
      "memory_limit_mb": 512,
      "max_domains": 5,
      "max_databases": 3,
      "max_cron_jobs": 10,
      "max_ftp_accounts": 3,
      "antivirus_enabled": true,
      "php_versions_allowed": "8.3,8.2",
      "terminal_enabled": false,
      "backup_enabled": true
    }
  ]
}
```

---

### GET /packages/:id

Get a single package. Same response shape as list item.

---

## PHP Versions (read-only)

### GET /php-versions/enabled

List enabled PHP versions. Use to populate PHP version dropdowns.

**Response 200:**
```json
{
  "data": [
    { "id": 1, "version": "8.3", "fpm_socket": "/run/php/php8.3-fpm.sock", "enabled": true, "created_at": "2026-01-01T00:00:00Z" }
  ]
}
```

---

## Terminal

Requires `terminal_enabled = true` on the user account.

### POST /terminal/token

Mint a one-time WebSocket token. **Rate limited: 1 request per 5 seconds per user.**

**Response 200:**
```json
{ "token": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6" }
```

**Response 403:** terminal not enabled on account.

**Response 429:** rate limit exceeded.

Then open a WebSocket:
```
WS /ws/terminal?token=<token>
```

Token is **single-use** and expires in **60 seconds**. See [Overview — WebSocket Terminal](./overview.md#websocket--terminal).

---

## Website Installer

### GET /installer/apps

List available installable apps.

**Response 200:**
```json
{
  "data": [
    { "id": "wordpress", "name": "WordPress", "version": "6.5", "description": "Popular CMS",   "requires_db": true  },
    { "id": "laravel",   "name": "Laravel",   "version": "11",  "description": "PHP framework", "requires_db": false },
    { "id": "joomla",    "name": "Joomla",    "version": "5.0", "description": "CMS",           "requires_db": true  },
    { "id": "drupal",    "name": "Drupal",    "version": "10",  "description": "CMS",           "requires_db": true  },
    { "id": "html",      "name": "Plain HTML","version": "—",   "description": "Static starter","requires_db": false }
  ]
}
```

---

### POST /installer/install

Start installation (async).

```json
{
  "app_id": "wordpress",
  "domain_id": 1,
  "db_name": "alice_wp",
  "db_user": "alice_wp",
  "db_pass": "StrongPass123",
  "overwrite": false
}
```

- `overwrite`: set `true` to allow overwriting existing files in the docroot
- `db_name`, `db_user`, `db_pass`: required for apps with `requires_db: true`

**Response 202:**
```json
{ "job_id": "install-a1b2c3d4" }
```

---

### GET /installer/status/:job_id

Poll installation progress.

**Response 200:**
```json
{
  "phase": "configuring",
  "log": ["Downloading WordPress...", "Extracting...", "Configuring wp-config.php..."],
  "done": false,
  "error": ""
}
```

---

## Antivirus

Requires `antivirus_enabled = true` on the user's package.

### GET /antivirus/status

Check if ClamAV daemon is running.

**Response 200:**
```json
{ "running": true, "version": "ClamAV 1.0.0" }
```

---

### POST /antivirus/scan

Start an async scan within the user's home directory.

```json
{ "path": "public_html/example.com" }
```

`path` is optional — omit to scan the entire home directory.

**Response 202:**
```json
{ "job_id": "abc123def456" }
```

---

### GET /antivirus/scan/:job_id

Poll scan status.

**Response 200:**
```json
{
  "job_id": "abc123def456",
  "status": "done",
  "infected": [
    { "path": "public_html/shell.php", "threat": "Php.Webshell.Agent" }
  ]
}
```

`status` values: `running`, `done`, `failed`

---

### GET /antivirus/alerts

List stored threat alerts for the calling user. Returns last 50.

**Response 200:**
```json
{
  "data": [
    { "id": 1, "user_id": 5, "path": "public_html/shell.php", "threat": "Php.Webshell.Agent", "detected_at": "2026-01-01T00:00:00Z" }
  ]
}
```

---

### GET /antivirus/poll

Poll for new real-time alerts (from inotify watch) since last check. Persists new alerts to DB.

**Response 200:**
```json
{
  "new_alerts": 1,
  "alerts": [{ "path": "public_html/evil.php", "threat": "Php.Webshell", "detected_at": "2026-01-01T00:00:00Z" }]
}
```

---

### POST /antivirus/watch

Start real-time file monitoring (inotifywait) on the user's home directory.

**Response 200:**
```json
{ "watch_id": "xyz789abc012" }
```

---

### DELETE /antivirus/watch/:watch_id

Stop real-time monitoring.

**Response 200:** `{ "message": "stopped" }`
