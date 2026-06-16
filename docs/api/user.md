# ZensPanel API — User Endpoints

All endpoints require `Authorization: Bearer <token>` with role `user` unless noted.

See [overview.md](./overview.md) for auth format, null-field conventions, WebSocket terminal protocol, and phpMyAdmin SSO flow.

---

## Authentication

### POST /auth/login

**No auth required. Rate limited: 10 requests/min per IP.**

```json
{ "username": "alice", "password": "secret" }
```

**Response 200 (no 2FA):**
```json
{
  "token": "eyJ...",
  "user": {
    "id": 5,
    "username": "alice",
    "email": "alice@example.com",
    "role": "user",
    "terminal_enabled": true,
    "backup_enabled": true,
    "package_id": { "Int64": 2, "Valid": true },
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

**Response 401:** `{ "error": "invalid credentials" }`

**Response 403:** `{ "error": "account suspended" }`

---

### POST /auth/2fa/verify

Complete login when 2FA is enabled. **No auth required.**

```json
{ "temp_token": "a1b2c3d4...", "code": "123456" }
```

**Response 200:** same as normal login (token + user object). Sets cookie.

**Response 401:** invalid or expired temp_token, or invalid TOTP code.

---

### POST /auth/2fa/recover

Login using a backup recovery code when the authenticator app is unavailable. **No auth required.**

The recovery code is **single-use**. After use, 2FA is disabled so the user can re-enroll.

```json
{ "temp_token": "a1b2c3d4...", "recovery_code": "ab12cd34ef" }
```

**Response 200:** same as normal login (token + user object). Sets cookie.

---

### GET /auth/me

Get current user profile.

**Response 200:**
```json
{
  "id": 5,
  "username": "alice",
  "email": "alice@example.com",
  "role": "user",
  "terminal_enabled": true,
  "backup_enabled": true,
  "package_id": { "Int64": 2, "Valid": true },
  "php_version": "8.3",
  "totp_enabled": false
}
```

---

## Two-Factor Authentication (2FA)

### POST /auth/2fa/setup

Generate a new TOTP secret. Returns a QR URL and 8 one-time recovery codes. **Save the recovery codes — they are never shown again.**

**Response 200:**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_url": "otpauth://totp/ZensPanel:alice?secret=JBSWY3DPEHPK3PXP&issuer=ZensPanel",
  "recovery_codes": ["ab12cd34ef", "gh56ij78kl", "mn90op12qr", "st34uv56wx", "yz78ab90cd", "ef12gh34ij", "kl56mn78op", "qr90st12uv"]
}
```

2FA is **not yet active** after this call — confirm with the next step.

---

### POST /auth/2fa/confirm

Activate 2FA by confirming with the first TOTP code from the authenticator app.

```json
{ "code": "123456" }
```

**Response 200:** `{ "message": "2FA enabled" }`

**Response 400:** no pending setup session, or invalid code.

---

### DELETE /auth/2fa

Disable 2FA. Requires the current TOTP code as proof of possession.

```json
{ "code": "123456" }
```

**Response 200:** `{ "message": "2FA disabled" }`

**Response 401:** invalid code.

---

## User Profile

### GET /users/:id

Get user profile. Users can only fetch their own ID. Admins can fetch any user (and receive additional fields — see [Admin Docs](./admin.md#get-usersid)).

**Response 200:**
```json
{
  "id": 5,
  "username": "alice",
  "email": "alice@example.com",
  "package_id": { "Int64": 2, "Valid": true },
  "terminal_enabled": true,
  "backup_enabled": true,
  "php_version": "8.3",
  "totp_enabled": false,
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

Note: `role`, `status`, and `linux_uid` are **not** returned to non-admin callers.

---

### GET /users/:id/usage

Get live resource usage. Users can only fetch their own ID.

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

## Domains

### GET /domains

List domains for the calling user.

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "user_id": 5,
      "domain": "example.com",
      "document_root": "/home/zenspanel/alice/public_html/example.com",
      "php_version": "8.3",
      "ssl_type": "letsencrypt",
      "ssl_expires_at": { "Time": "2026-09-01T00:00:00Z", "Valid": true },
      "status": "active",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

`status` values: `pending`, `active`, `suspended`

`ssl_type` values: `none`, `letsencrypt`, `custom`

---

### GET /domains/:id

Get a single domain. Returns `403` if the domain belongs to another user.

---

### POST /domains

Add a domain. Provisions nginx vhost and creates document root directory via agent. Row is rolled back on agent failure.

```json
{ "domain": "example.com", "php_version": "8.3" }
```

**Response 201:** domain object (same shape as list item).

---

### PUT /domains/:id

Partial update. Send only the fields you want to change. Changing `php_version` triggers PHP-FPM pool recreation and nginx vhost regeneration via agent.

```json
{ "php_version": "8.2" }
```

**Response 200:** `{ "message": "updated" }`

---

### DELETE /domains/:id

Delete domain. Tears down child subdomains (nginx vhost + SSL cert) first, then the parent vhost.

**Response 200:** `{ "message": "deleted" }`

---

### POST /domains/:id/suspend

Replace nginx vhost with a 503 page.

**Response 200:** `{ "message": "domain suspended" }`

---

### POST /domains/:id/unsuspend

Rebuild nginx vhost from DB.

**Response 200:** `{ "message": "domain unsuspended" }`

---

### POST /domains/:id/backup

Async backup of the domain's document root only (not full home directory). Poll the backup list for status.

**Response 202:**
```json
{ "job_id": 42, "backup_id": 42 }
```

---

## Subdomains

### GET /subdomains

List subdomains. Use `?parent_id=<domain_id>` to scope to one parent domain.

**Response 200:**
```json
{
  "data": [
    {
      "id": 10,
      "user_id": 5,
      "parent_domain_id": 1,
      "subdomain": "blog",
      "fqdn": "blog.example.com",
      "document_root": "/home/zenspanel/alice/public_html/blog.example.com",
      "php_version": "8.3",
      "ssl_type": "none",
      "ssl_expires_at": { "Time": "0001-01-01T00:00:00Z", "Valid": false },
      "status": "active",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### GET /subdomains/:id

Get a single subdomain.

---

### POST /subdomains

Create a subdomain. Provisions PHP-FPM pool and nginx vhost via agent. Row is rolled back on agent failure.

```json
{
  "parent_domain_id": 1,
  "subdomain": "blog",
  "php_version": "8.3",
  "doc_root": "/home/zenspanel/alice/public_html/blog.example.com"
}
```

Field rules:
- `subdomain`: lowercase letters, digits, hyphens only (no dots). Cannot be: `www`, `mail`, `smtp`, `imap`, `pop`, `ns`, `ns1`, `ns2`
- `doc_root`: optional, defaults to `~/public_html/<fqdn>`. Must be inside the user's home directory
- `php_version`: defaults to the parent domain's version if omitted

**Response 201:** subdomain object (same shape as list item).

**Response 409:** FQDN already in use.

---

### PUT /subdomains/:id

Partial update. Immutable fields (`id`, `user_id`, `parent_domain_id`, `subdomain`, `fqdn`) are silently stripped. Changing `php_version` or `document_root` triggers agent re-provisioning.

```json
{ "php_version": "8.2" }
```

**Response 200:** `{ "message": "updated" }`

---

### DELETE /subdomains/:id

Tears down nginx vhost and SSL cert via agent (failures are logged but non-fatal).

**Response 200:** `{ "message": "deleted" }`

---

## SSL

### POST /domains/:id/ssl

Issue or install an SSL certificate for a domain.

**Let's Encrypt:**
```json
{ "type": "letsencrypt" }
```

Calls certbot via agent. Sets `ssl_expires_at` to now+90 days.

**Custom certificate:**
```json
{
  "type": "custom",
  "cert_pem": "-----BEGIN CERTIFICATE-----\n...",
  "key_pem": "-----BEGIN PRIVATE KEY-----\n..."
}
```

Parses actual `NotAfter` from the certificate PEM.

**Response 200:** updated domain object.

---

### DELETE /domains/:id/ssl

Remove certificate files via agent. Sets `ssl_type = "none"`, `ssl_expires_at = null`.

**Response 200:** updated domain object.

---

### POST /subdomains/:id/ssl

Same contract as domain SSL — issue Let's Encrypt or install custom certificate for a subdomain.

**Response 200:** updated subdomain object.

---

### DELETE /subdomains/:id/ssl

Remove SSL certificate from subdomain.

**Response 200:** updated subdomain object.

---

## Redirects

### GET /domains/:id/redirects

List HTTP redirects for a domain.

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "domain_id": 1,
      "source_path": "/old-page",
      "dest_url": "https://example.com/new-page",
      "type": "301",
      "enabled": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /domains/:id/redirects

```json
{
  "source_path": "/old-page",
  "dest_url": "https://example.com/new-page",
  "type": "301",
  "enabled": true
}
```

`type`: `"301"` (permanent) or `"302"` (temporary). Defaults to `"301"`.

`enabled` defaults to `true`.

**Response 201:** redirect object. Syncs all redirects to nginx immediately.

---

### PUT /domains/:id/redirects/:rid

Partial update. Any field may be omitted.

```json
{ "enabled": false }
```

**Response 200:** `{ "message": "updated" }`

---

### DELETE /domains/:id/redirects/:rid

**Response 200:** `{ "message": "deleted" }`

---

## Hotlink Protection

### GET /domains/:id/hotlink

**Response 200:**
```json
{
  "enabled": true,
  "allowed_domains": ["example.com", "www.example.com"]
}
```

---

### PUT /domains/:id/hotlink

```json
{
  "enabled": true,
  "allowed_domains": ["example.com", "www.example.com", "partner.com"]
}
```

`allowed_domains` is the **complete replacement list** — it replaces the existing list, not appended.

**Response 200:** `{ "message": "updated", "enabled": true }`

---

## Domain Logs

### GET /domains/:id/logs

Tail nginx or PHP-FPM logs for a domain.

**Query params:**

| Param | Values | Default |
|-------|--------|---------|
| `type` | `nginx`, `nginx-access`, `nginx-error`, `fpm` | `nginx` |
| `lines` | integer (1–500) | `100` |

**Response 200:**
```json
{
  "domain": "example.com",
  "type": "nginx",
  "log_path": "/var/log/nginx/example.com-error.log",
  "lines": [
    "2026-01-01 12:00:00 [error] 1234#0: *1 connect() failed...",
    "2026-01-01 12:00:01 [notice] signal process started"
  ]
}
```

---

## Databases

### GET /databases

List the calling user's databases.

**Response 200:**
```json
{
  "data": [
    { "id": 1, "user_id": 5, "db_name": "alice_wp", "db_user": "alice_wp", "created_at": "2026-01-01T00:00:00Z" }
  ]
}
```

---

### POST /databases

Create a MySQL database and user. The password is **never stored** in the panel — copy it immediately.

```json
{ "db_name": "alice_wp", "db_user": "alice_wp", "db_password": "StrongPass123" }
```

**Response 201:**
```json
{
  "id": 1,
  "user_id": 5,
  "db_name": "alice_wp",
  "db_user": "alice_wp",
  "db_password": "StrongPass123",
  "note": "This password will not be shown again"
}
```

Row is rolled back if MySQL provisioning via agent fails.

---

### DELETE /databases/:id

Drops MySQL schema and user via agent, then removes the database row.

**Response 200:** `{ "message": "deleted" }`

---

### POST /databases/:id/reset-password

Generate and set a new random 16-char alphanumeric MySQL password. The new password is **not stored** in the panel.

**Response 200:**
```json
{ "db_user": "alice_wp", "new_password": "Xk7mP2qNrL3sAb4d" }
```

---

### GET /databases/:id/phpmyadmin/launch

Mint a 60-second single-use phpMyAdmin SSO token. Open the returned URL in a new tab.

**Response 200:**
```json
{ "url": "/api/v1/phpmyadmin/sso/a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6" }
```

**Response 503:** Redis is not configured on the server.

---

## FTP Accounts

### GET /ftp

List the calling user's FTP accounts.

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "user_id": 5,
      "ftp_username": "alice_ftp",
      "home_dir": "/home/zenspanel/alice",
      "enabled": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /ftp

Create an FTP account. Enforces `max_ftp_accounts` package quota.

```json
{
  "ftp_username": "alice_ftp",
  "password": "StrongPass123",
  "home_dir": "/home/zenspanel/alice/public_html"
}
```

- `password`: minimum 8 characters
- `home_dir`: optional, defaults to the user's home directory

**Response 201:** FTP account object.

**Response 403:** package quota reached.

---

### DELETE /ftp/:id

Removes the vsftpd virtual user via agent and deletes the database row.

**Response 200:** `{ "message": "deleted" }`

---

## Backups

Backups are async — create returns immediately, poll via `GET /backups` for status changes.

`status` values: `pending`, `running`, `done`, `failed`, `restoring`, `restore_failed`

`type` values: `full` (files + databases), `files` (home directory only), `db` (databases only), `domain` (domain docroot only)

### GET /backups

List the calling user's backups.

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "user_id": 5,
      "type": "full",
      "status": "done",
      "file_path": { "String": "/var/backups/zenspanel/alice/20260101-120000-full.tar.gz", "Valid": true  },
      "size_bytes": { "Int64": 104857600,                                                   "Valid": true  },
      "error_msg":  { "String": "",                                                          "Valid": false },
      "created_at": "2026-01-01T12:00:00Z",
      "updated_at": "2026-01-01T12:05:00Z"
    }
  ]
}
```

---

### POST /backups

Create a backup. Requires `backup_enabled = true` on the user account.

```json
{ "type": "full" }
```

**Response 202:** backup object with `status: "pending"`. Poll `GET /backups` for completion.

**Response 403:** backup not enabled on account.

---

### GET /backups/:id

Get a single backup.

---

### GET /backups/:id/download

Download the backup archive. Only available when `status == "done"`.

**Response 200:** binary file stream with `Content-Disposition: attachment; filename=<archive name>`.

**Response 400:** backup not in `done` status.

---

### POST /backups/:id/restore

Restore from a backup. **Destructive** — overwrites existing files and databases. Only available when `status == "done"`. Async.

**Response 202:**
```json
{ "message": "restore started", "backup_id": 1 }
```

---

### DELETE /backups/:id

Delete backup record and archive file from disk.

**Response 200:** `{ "message": "deleted" }`

---

## File Manager

All paths are relative to the user's home directory. The agent rejects any path that would escape the home directory.

### GET /files?path=:path

List directory contents.

```
GET /files?path=public_html/example.com
```

**Response 200:**
```json
{
  "entries": [
    { "name": "index.php", "size": 1024, "mode": "0644", "is_dir": false, "modified_at": "2026-01-01T00:00:00Z" },
    { "name": "uploads",   "size": 0,    "mode": "0755", "is_dir": true,  "modified_at": "2026-01-01T00:00:00Z" }
  ]
}
```

---

### GET /files/content?path=:path

Read a text file's contents.

**Response 200:** `{ "content": "<?php echo 'hello'; ?>" }`

---

### POST /files/content

Write (create or overwrite) a text file.

```json
{ "path": "public_html/example.com/index.php", "content": "<?php echo 'hello'; ?>" }
```

**Response 200:** `{ "message": "saved" }`

---

### POST /files/mkdir

Create a directory.

```json
{ "path": "public_html/example.com/uploads" }
```

**Response 200:** `{ "message": "created" }`

---

### PUT /files/rename

Rename or move a file or directory.

```json
{ "old_path": "public_html/old-name.php", "new_path": "public_html/new-name.php" }
```

**Response 200:** `{ "message": "renamed" }`

---

### DELETE /files?path=:path

Delete a file or directory (recursive).

```
DELETE /files?path=public_html/example.com/old-file.php
```

**Response 200:** `{ "message": "deleted" }`

---

### POST /files/upload

Upload a binary file. Maximum **64 MiB** per request. `Content-Type: multipart/form-data`.

| Form field | Type | Description |
|------------|------|-------------|
| `path` | string | Destination directory (relative to home) |
| `file` | binary | The file to upload |

**Response 200:**
```json
{ "message": "uploaded", "path": "public_html/example.com/image.jpg", "size": 52428 }
```

**Response 413:** file exceeds 64 MiB.

---

### PUT /files/chmod

Change file or directory permissions.

```json
{ "path": "public_html/example.com/script.sh", "mode": "0755" }
```

`mode` accepts octal strings: `"0755"`, `"755"`, or symbolic `"rwxr-xr-x"`.

**Response 200:** `{ "message": "permissions updated" }`

---

### POST /files/copy

Copy a file or directory.

```json
{ "src": "public_html/example.com/config.php", "dst": "public_html/example.com/config.bak.php" }
```

**Response 200:** `{ "message": "copied" }`

---

### POST /files/compress

Compress a file or directory into an archive.

```json
{ "src": "public_html/example.com", "dst": "backups/example-site.zip" }
```

Supported output formats: `.zip`, `.tar.gz`

**Response 200:** `{ "message": "compressed" }`

---

### POST /files/extract

Extract an archive into a directory.

```json
{ "archive": "backups/example-site.zip", "dst_dir": "public_html/restored" }
```

**Response 200:** `{ "message": "extracted" }`

---

## Cron Jobs

### GET /cron-jobs

List the calling user's cron jobs.

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "user_id": 5,
      "expression": "0 * * * *",
      "command": "php /home/zenspanel/alice/public_html/example.com/artisan schedule:run",
      "enabled": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /cron-jobs

Create a cron job. Enforces `max_cron_jobs` package quota (0 = unlimited).

```json
{
  "expression": "0 * * * *",
  "command": "php /home/zenspanel/alice/public_html/example.com/artisan schedule:run",
  "enabled": true
}
```

Field rules:
- `expression`: standard 5-field cron (`* * * * *`)
- `command`: printable ASCII only. Forbidden characters: `;`, `&`, `|`, `>`, `<`, `` ` ``, `$`, `(`, `)`, `{`, `}`
- `enabled`: defaults to `true`

**Response 201:** `{ "data": <cron job object> }` (with optional `"warning"` on crontab sync failure)

**Response 429:** package cron job quota exceeded.

---

### PUT /cron-jobs/:id

Partial update. Disabled jobs are commented out in the system crontab (not deleted).

```json
{ "expression": "*/5 * * * *", "enabled": false }
```

**Response 200:** `{ "message": "updated" }`

---

### DELETE /cron-jobs/:id

Delete cron job and remove from system crontab.

**Response 200:** `{ "message": "deleted" }`

---

## PHP Extensions

### GET /php-extensions

List available extensions for the user's configured PHP version. Only admin-enabled extensions appear. Query param: `?php_version=8.3` (defaults to user's configured version).

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

List available packages. User receives a customer-visible subset of fields (no internal byte values or I/O limits).

**User response 200:**
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

Then open a WebSocket connection:
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
    { "id": "wordpress", "name": "WordPress", "version": "6.5", "description": "Popular CMS",    "requires_db": true  },
    { "id": "laravel",   "name": "Laravel",   "version": "11",  "description": "PHP framework",  "requires_db": false },
    { "id": "joomla",    "name": "Joomla",    "version": "5.0", "description": "CMS",            "requires_db": true  },
    { "id": "drupal",    "name": "Drupal",    "version": "10",  "description": "CMS",            "requires_db": true  },
    { "id": "html",      "name": "Plain HTML","version": "—",   "description": "Static starter", "requires_db": false }
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

- `overwrite`: set `true` to allow overwriting existing files in docroot
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

**Response 200:** agent pass-through. Example:
```json
{ "running": true, "version": "ClamAV 1.0.0" }
```

---

### POST /antivirus/scan

Start an async scan. Scans within the user's home directory only.

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
{ "new_alerts": 1, "alerts": [{ "path": "public_html/evil.php", "threat": "Php.Webshell", "detected_at": "2026-01-01T00:00:00Z" }] }
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
