# ZensPanel API — User Endpoints

All endpoints require `Authorization: Bearer <token>` with role `user` unless noted.

---

## Authentication

### POST /auth/login
```json
{ "username": "alice", "password": "secret" }
```
**Response 200:**
```json
{
  "token": "eyJ...",
  "user": {
    "id": 5, "username": "alice", "email": "alice@example.com",
    "role": "user", "terminal_enabled": true, "backup_enabled": true,
    "package_id": { "Int64": 2, "Valid": true },
    "php_version": "8.3", "totp_enabled": false
  }
}
```
**Response 200 (2FA enabled):**
```json
{ "requires_2fa": true, "temp_token": "a1b2c3..." }
```

### POST /auth/2fa/verify
Complete login when 2FA is enabled. Public endpoint — no JWT required.
```json
{ "temp_token": "a1b2c3...", "code": "123456" }
```
**Response 200:** same as normal login (token + user).

### POST /auth/2fa/recover
Login using a recovery code when authenticator is unavailable. Disables 2FA after use.
```json
{ "temp_token": "a1b2c3...", "recovery_code": "ab12cd34ef" }
```
**Response 200:** same as normal login.

### GET /auth/me
Get current user profile.
**Response 200:** same shape as login `user` object.

---

## Two-Factor Authentication (2FA)

### POST /auth/2fa/setup
Generate a new TOTP secret. Returns QR URL and 8 one-time recovery codes. **Save the recovery codes** — they are never shown again.
```json
// No request body
```
**Response 200:**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_url": "otpauth://totp/ZensPanel:alice?secret=...",
  "recovery_codes": ["ab12cd34ef", "gh56ij78kl", "..."]
}
```

### POST /auth/2fa/confirm
Activate 2FA by confirming with the first TOTP code from the authenticator app.
```json
{ "code": "123456" }
```
**Response 200:** `{ "message": "2FA enabled" }`

### DELETE /auth/2fa
Disable 2FA. Requires current TOTP code.
```json
{ "code": "123456" }
```
**Response 200:** `{ "message": "2FA disabled" }`

---

## User Profile

### GET /users/:id
Get user profile. Users can only get their own profile.

### GET /users/:id/usage
Get live resource usage.
**Response 200:**
```json
{
  "user_id": 5,
  "usage": {
    "domains":   { "used": 2, "max": 10 },
    "databases": { "used": 1, "max": 5 },
    "disk": { "used": 524288000, "max": 10737418240, "files": 500000000, "db": 24288000 },
    "ram":  { "used": 67108864, "max": 536870912 },
    "cpu":  { "used": 12.5, "max": 100 }
  }
}
```
All sizes in bytes. `cpu.used` is percentage (0–100).

---

## Domains

### GET /domains
List user's domains.
**Response 200:**
```json
{
  "data": [
    {
      "id": 1, "user_id": 5, "domain": "example.com",
      "document_root": "/var/lib/zenspanel/home/alice/public_html/example.com",
      "php_version": "8.3", "ssl_type": "letsencrypt",
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

### GET /domains/:id
Get a single domain.

### POST /domains
Add a domain. Provisions nginx vhost and creates document root directory.
```json
{ "domain": "example.com", "php_version": "8.3" }
```
**Response 201:** domain object.

### PUT /domains/:id
Update domain. Changing `php_version` triggers FPM pool creation and nginx vhost regeneration.
```json
{ "php_version": "8.2" }
```
**Response 200:** `{ "message": "updated" }`

### DELETE /domains/:id
Delete domain and remove nginx vhost. Also deletes all subdomains.
**Response 200:** `{ "message": "deleted" }`

### POST /domains/:id/suspend
Suspend domain (nginx returns 503).
**Response 200:** `{ "message": "domain suspended" }`

### POST /domains/:id/unsuspend
Unsuspend domain.
**Response 200:** `{ "message": "domain unsuspended" }`

### POST /domains/:id/backup
Backup domain docroot only (not full home). Async — poll backups list for status.
**Response 202:** `{ "job_id": 42, "backup_id": 42 }`

---

## Subdomains

### GET /subdomains?parent_id=:id
List subdomains for a parent domain.
**Response 200:**
```json
{
  "data": [
    {
      "id": 10, "user_id": 5, "parent_domain_id": 1,
      "subdomain": "blog", "fqdn": "blog.example.com",
      "document_root": "/var/lib/zenspanel/home/alice/public_html/blog.example.com",
      "php_version": "8.3", "ssl_type": "none",
      "ssl_expires_at": { "Time": "0001-01-01T00:00:00Z", "Valid": false },
      "status": "active",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

### GET /subdomains/:id
Get a single subdomain.

### POST /subdomains
Create a subdomain.
```json
{
  "parent_domain_id": 1,
  "subdomain": "blog",
  "php_version": "8.3",
  "doc_root": "/var/lib/zenspanel/home/alice/public_html/blog.example.com"
}
```
- `subdomain`: lowercase letters, digits, hyphens. Cannot be: `www`, `mail`, `smtp`, `imap`, `pop`, `ns`, `ns1`, `ns2`
- `doc_root`: optional, defaults to `~/public_html/<fqdn>`

**Response 201:** subdomain object.

### PUT /subdomains/:id
Update subdomain. Changing `php_version` or `document_root` re-provisions nginx.
```json
{ "php_version": "8.2" }
```

### DELETE /subdomains/:id

### POST /subdomains/:id/ssl
Issue Let's Encrypt certificate for subdomain.
```json
{ "type": "letsencrypt" }
```
**Response 200:** `{ "message": "SSL issued" }`

### DELETE /subdomains/:id/ssl
Remove SSL certificate for subdomain.

---

## SSL

### POST /domains/:id/ssl
Issue Let's Encrypt certificate for a domain.
```json
{ "type": "letsencrypt" }
```
**Response 200:** `{ "message": "SSL issued" }`

### DELETE /domains/:id/ssl
Remove SSL certificate.
**Response 200:** `{ "message": "SSL removed" }`

---

## Databases

### GET /databases
List user's databases.
**Response 200:**
```json
{
  "data": [
    { "id": 1, "user_id": 5, "db_name": "alice_wp", "db_user": "alice_wp", "created_at": "..." }
  ]
}
```

### POST /databases
Create a database and MySQL user.
```json
{ "db_name": "alice_wp", "db_user": "alice_wp", "db_password": "StrongPass123" }
```
**Response 201:**
```json
{
  "id": 1, "user_id": 5, "db_name": "alice_wp", "db_user": "alice_wp",
  "db_password": "StrongPass123",
  "note": "This password will not be shown again"
}
```
⚠️ Save the password — it is not stored in the panel.

### DELETE /databases/:id
Drop database and MySQL user.

### POST /databases/:id/reset-password
Generate and set a new random MySQL password. Old password is invalidated.
**Response 200:**
```json
{ "db_user": "alice_wp", "new_password": "Xk7mP2qNrL3s" }
```
⚠️ Copy the password — it is not stored in the panel.

### GET /databases/:id/phpmyadmin/launch
Open phpMyAdmin with auto-login (requires Redis on server).
**Response 200:**
```json
{ "url": "/api/v1/phpmyadmin/sso/a1b2c3d4..." }
```
Open this URL in a new tab. The token is single-use and expires in 60 seconds.

---

## FTP Accounts

### GET /ftp
List user's FTP accounts.
**Response 200:**
```json
{
  "data": [
    {
      "id": 1, "user_id": 5, "ftp_username": "alice_ftp",
      "home_dir": "/var/lib/zenspanel/home/alice",
      "enabled": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

### POST /ftp
Create an FTP account. Quota from package `max_ftp_accounts` (0 = disabled).
```json
{
  "ftp_username": "alice_ftp",
  "password": "StrongPass123",
  "home_dir": "/var/lib/zenspanel/home/alice"
}
```
- `password`: minimum 8 characters
- `home_dir`: optional, defaults to user's home directory

**Response 201:** FTP account object.

### DELETE /ftp/:id
Delete FTP account and remove vsftpd virtual user.
**Response 200:** `{ "message": "deleted" }`

---

## Backups

Backups are async — create returns immediately, poll the list for status changes.

`status` values: `pending`, `running`, `done`, `failed`, `restoring`, `restore_failed`

### GET /backups
List user's backups.
**Response 200:**
```json
{
  "data": [
    {
      "id": 1, "user_id": 5, "type": "full", "status": "done",
      "file_path": { "String": "/var/lib/zenspanel/backups/alice/20260101-120000-full.tar.gz", "Valid": true },
      "size_bytes": { "Int64": 104857600, "Valid": true },
      "error_msg":  { "String": "", "Valid": false },
      "created_at": "2026-01-01T12:00:00Z",
      "updated_at": "2026-01-01T12:05:00Z"
    }
  ]
}
```

### GET /backups/:id
Get a single backup.

### POST /backups
Create a new backup.
```json
{ "type": "full" }
```
`type` values: `full` (files + databases), `files` (home dir only), `db` (databases only)

**Response 202:** backup object with `status: "pending"`.

### GET /backups/:id/download
Download the backup archive. Only available when `status == "done"`.
Returns the file as a binary attachment.

### POST /backups/:id/restore
Restore from a backup. **Destructive** — overwrites existing files and databases. Only available when `status == "done"`.
**Response 202:** `{ "message": "restore started", "backup_id": 1 }`

### DELETE /backups/:id
Delete backup record and archive file.
**Response 200:** `{ "message": "deleted" }`

---

## File Manager

All paths are relative to the user's home directory. You cannot access paths outside your home.

### GET /files?path=:path
List directory contents.
```
GET /files?path=public_html/example.com
```
**Response 200:**
```json
{
  "data": [
    { "name": "index.php", "size": 1024, "mode": "0644", "is_dir": false, "modified_at": "2026-01-01T00:00:00Z" },
    { "name": "uploads",   "size": 0,    "mode": "0755", "is_dir": true,  "modified_at": "2026-01-01T00:00:00Z" }
  ]
}
```

### GET /files/content?path=:path
Read file contents as text.
**Response 200:** `{ "content": "<?php echo 'hello'; ?>" }`

### POST /files/content
Write (create or overwrite) a text file.
```json
{ "path": "public_html/example.com/index.php", "content": "<?php echo 'hello'; ?>" }
```
**Response 200:** `{ "message": "written" }`

### POST /files/mkdir
Create a directory.
```json
{ "path": "public_html/example.com/uploads" }
```
**Response 200:** `{ "message": "created" }`

### PUT /files/rename
Rename or move a file/directory.
```json
{ "src": "old-name.php", "dst": "new-name.php" }
```
**Response 200:** `{ "message": "renamed" }`

### DELETE /files?path=:path
Delete a file or directory (recursive).
```
DELETE /files?path=public_html/example.com/old-file.php
```
**Response 200:** `{ "message": "deleted" }`

### POST /files/upload
Upload a binary file. Max **64 MiB** per request. `Content-Type: multipart/form-data`.

| Field | Type | Description |
|-------|------|-------------|
| `path` | string | Destination directory path |
| `file` | file | The file to upload |

**Response 200:** `{ "message": "uploaded", "path": "public_html/example.com/image.jpg" }`

### PUT /files/chmod
Change file or directory permissions.
```json
{ "path": "public_html/example.com/script.sh", "mode": "0755" }
```
**Response 200:** `{ "message": "chmod ok" }`

### POST /files/copy
Copy a file or directory.
```json
{ "src": "public_html/example.com/config.php", "dst": "public_html/example.com/config.bak.php" }
```
**Response 200:** `{ "message": "copied" }`

### POST /files/compress
Compress a file or directory into an archive.
```json
{ "src": "public_html/example.com", "dst": "backups/example-site.zip" }
```
Supported formats: `.zip`, `.tar.gz`

**Response 200:** `{ "message": "compressed" }`

### POST /files/extract
Extract an archive.
```json
{ "archive": "backups/example-site.zip", "dst_dir": "public_html/restored" }
```
**Response 200:** `{ "message": "extracted" }`

---

## Cron Jobs

### GET /cron-jobs
List user's cron jobs.
**Response 200:**
```json
{
  "data": [
    {
      "id": 1, "user_id": 5,
      "expression": "0 * * * *",
      "command": "php /home/alice/public_html/example.com/artisan schedule:run",
      "enabled": true,
      "last_run_at": null,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

### POST /cron-jobs
Create a cron job. Quota from package `max_cron_jobs` (0 = unlimited).
```json
{
  "expression": "0 * * * *",
  "command": "php /home/alice/public_html/example.com/artisan schedule:run",
  "enabled": true
}
```
- `expression`: standard 5-field cron (`* * * * *`)
- `command`: printable ASCII only, no shell metacharacters (`;`, `&`, `|`, `>`, `<`, `` ` ``, `$`, `(`, `)`, `{`, `}`)

**Response 201:** cron job object.

### PUT /cron-jobs/:id
Update a cron job.
```json
{ "expression": "*/5 * * * *", "enabled": false }
```
Disabled jobs are commented out in the crontab (`#`-prefix) — not deleted.

**Response 200:** `{ "message": "updated" }`

### DELETE /cron-jobs/:id
Delete cron job and remove from crontab.
**Response 200:** `{ "message": "deleted" }`

---

## PHP Extensions

### GET /php-extensions
List available extensions for the user's PHP version. Only admin-enabled extensions are shown.
**Response 200:**
```json
{
  "data": [
    { "id": 1, "name": "redis", "php_version": "8.3", "admin_enabled": true, "user_enabled": true }
  ]
}
```

### PUT /php-extensions
Toggle an extension for the user's PHP-FPM pool. Cannot enable if admin has disabled it globally.
```json
{ "name": "redis", "php_version": "8.3", "enabled": true }
```
**Response 200:** `{ "message": "updated" }`

---

## Redirect Manager

### GET /domains/:id/redirects
List redirects for a domain.
**Response 200:**
```json
{
  "data": [
    {
      "id": 1, "domain_id": 1,
      "source_path": "/old-page",
      "dest_url": "https://example.com/new-page",
      "type": "301",
      "enabled": true
    }
  ]
}
```

### POST /domains/:id/redirects
```json
{
  "source_path": "/old-page",
  "dest_url": "https://example.com/new-page",
  "type": "301",
  "enabled": true
}
```
`type`: `"301"` (permanent) or `"302"` (temporary)

**Response 201:** redirect object.

### PUT /domains/:id/redirects/:rid
```json
{ "enabled": false }
```
**Response 200:** `{ "message": "updated" }`

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

### PUT /domains/:id/hotlink
```json
{
  "enabled": true,
  "allowed_domains": ["example.com", "www.example.com", "partner.com"]
}
```
**Response 200:** `{ "message": "updated" }`

---

## Domain Logs

### GET /domains/:id/logs?type=nginx&lines=100
Tail nginx or PHP-FPM logs for a domain.

| Param | Values | Default |
|-------|--------|---------|
| `type` | `nginx`, `fpm` | `nginx` |
| `lines` | 1–500 | `100` |

**Response 200:**
```json
{
  "lines": [
    "2026-01-01 12:00:00 [error] 1234#0: *1 connect() failed...",
    "2026-01-01 12:00:01 [notice] signal process started"
  ]
}
```

---

## Antivirus

Requires `antivirus_enabled` on the user's package.

### GET /antivirus/status
Check if ClamAV daemon is running.
**Response 200:** `{ "running": true, "version": "ClamAV 1.0.0" }`

### POST /antivirus/scan
Start an async scan. Scans within user's home directory only.
```json
{ "path": "public_html/example.com" }
```
**Response 202:** `{ "job_id": "abc123" }`

### GET /antivirus/scan/:job_id
Poll scan status.
**Response 200:**
```json
{
  "job_id": "abc123",
  "status": "done",
  "infected": [
    { "path": "public_html/example.com/shell.php", "threat": "Php.Webshell.Agent" }
  ]
}
```
`status` values: `running`, `done`, `failed`

### GET /antivirus/alerts
List all detected threats (stored in DB).
**Response 200:**
```json
{
  "data": [
    { "id": 1, "user_id": 5, "path": "public_html/shell.php", "threat": "Php.Webshell.Agent", "detected_at": "..." }
  ]
}
```

### POST /antivirus/watch
Start real-time file monitoring (inotifywait). Alerts pushed via polling endpoint.
**Response 200:** `{ "watch_id": "xyz789" }`

### DELETE /antivirus/watch/:watch_id
Stop real-time monitoring.
**Response 200:** `{ "message": "stopped" }`

### GET /antivirus/poll
Poll for new real-time alerts since last check.
**Response 200:** same shape as `/antivirus/alerts`

---

## Terminal

Requires `terminal_enabled` on user account.

### POST /terminal/token
Mint a one-time WebSocket token. Rate limited to 1 request per 5 seconds per user.

**Response 200:** `{ "token": "a1b2c3d4..." }`

Then open a WebSocket connection:
```
WS /ws/terminal?token=<token>
```
Token is single-use and expires in **60 seconds**.

See [Overview — WebSocket Terminal](./overview.md) for message format.

---

## Website Installer

### GET /installer/apps
List available apps.
**Response 200:**
```json
{
  "data": [
    { "id": "wordpress", "name": "WordPress", "version": "6.5", "description": "Popular CMS", "requires_db": true },
    { "id": "laravel",   "name": "Laravel",   "version": "11",  "description": "PHP framework", "requires_db": false },
    { "id": "joomla",    "name": "Joomla",    "version": "5.0", "description": "CMS", "requires_db": true },
    { "id": "drupal",    "name": "Drupal",    "version": "10",  "description": "CMS", "requires_db": true },
    { "id": "html",      "name": "Plain HTML","version": "—",   "description": "Static starter", "requires_db": false }
  ]
}
```

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
- `overwrite`: set to `true` to allow overwriting existing files in docroot

**Response 202:** `{ "job_id": "install-abc123" }`

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

## Packages (read-only)

### GET /packages
List all packages. Used to display plan details.

### GET /packages/:id
Get a single package.

See [Admin Docs — Packages](./admin.md) for full field reference.

---

## PHP Versions (read-only)

### GET /php-versions/enabled
List enabled PHP versions. Used to populate PHP version selects.
**Response 200:**
```json
{
  "data": [
    { "id": 1, "version": "8.3", "fpm_socket": "/run/php/php8.3-fpm.sock", "enabled": true }
  ]
}
```
