# ZensPanel API — Admin: Packages, PHP & Backup Targets

All endpoints require `Authorization: Bearer <token>` with role `admin` unless noted.

See [overview.md](./overview.md) for auth format, null-field conventions, and status codes.

---

## Packages

### GET /packages

List all packages. Admins receive the full response including internal byte values and I/O limits. Regular users receive a restricted subset — see [User Misc Docs](./user-misc.md#packages-read-only).

**Admin response 200:**
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
      "io_read_bps": 52428800,
      "io_read_mbps": 50,
      "io_write_bps": 52428800,
      "io_write_mbps": 50,
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

Get a single package. Same response shape as list item.

---

### POST /packages

Create a package. Send sizes in **MB** — the API converts to bytes internally (`mb × 1024²`). The response returns both MB and raw byte fields.

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

**Response 201:** full package object (same shape as GET /packages/:id).

---

### PUT /packages/:id

Update a package. Same body as Create. Existing users on this package are not retroactively re-provisioned — changes apply at next agent interaction.

**Response 200:** `{ "message": "updated" }`

---

### DELETE /packages/:id

**Response 200:** `{ "message": "deleted" }`

---

## PHP Versions

### GET /php-versions

List all PHP versions (enabled and disabled). Available to all authenticated users.

**Response 200:**
```json
{
  "data": [
    { "id": 1, "version": "8.3", "fpm_socket": "/run/php/php8.3-fpm.sock", "enabled": true, "created_at": "2026-01-01T00:00:00Z" }
  ]
}
```

---

### GET /php-versions/enabled

List only enabled PHP versions. Available to all authenticated users. Use to populate PHP version dropdowns.

---

### POST /php-versions

**Admin only.** Register a new PHP version.

```json
{ "version": "8.4", "fpm_socket": "/run/php/php8.4-fpm.sock" }
```

`fpm_socket` defaults to `/run/php/php<version>-fpm.sock` if omitted.

**Response 201:** PHP version object.

---

### PUT /php-versions/:id/enable

**Admin only.** Enable a version — makes it selectable by users.

**Response 200:** `{ "message": "enabled" }`

---

### PUT /php-versions/:id/disable

**Admin only.** Disable a version — existing users keep it, new selections blocked.

**Response 200:** `{ "message": "disabled" }`

---

### DELETE /php-versions/:id

**Admin only.**

**Response 200:** `{ "message": "deleted" }`

---

## PHP Extensions

### GET /admin/php-extensions

List the global extension catalog. Query param: `?php_version=8.3`

**Response 200:**
```json
{
  "data": [
    { "id": 1, "name": "redis",   "php_version": "8.3", "enabled": true  },
    { "id": 2, "name": "imagick", "php_version": "8.3", "enabled": false }
  ]
}
```

---

### POST /admin/php-extensions

Add an extension to the catalog.

```json
{ "name": "imagick", "php_version": "8.3", "enabled": true }
```

**Response 201:** PHP extension object.

---

### PUT /admin/php-extensions/:id

Enable or disable an extension globally. **Disabling immediately reloads all affected user PHP-FPM pools** via agent.

```json
{ "enabled": false }
```

**Response 200:** `{ "message": "updated" }`

---

### DELETE /admin/php-extensions/:id

**Response 200:** `{ "message": "deleted" }`

---

### POST /admin/php-extensions/seed

Seed the default extension catalog for all PHP versions. Idempotent — no-op if the table is already populated. Covers PHP 8.1–8.4 × 17 common extensions (68 rows total).

**Response 200:**
```json
{ "message": "seeded", "count": 68 }
```
or
```json
{ "message": "already seeded" }
```

---

## Backup Targets

S3-compatible remote backup destinations. The `secret_key` is stored AES-256-GCM encrypted and is **never returned** in any response.

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

---

### POST /admin/backup-targets

```json
{
  "name": "Wasabi",
  "type": "s3",
  "bucket": "my-backups",
  "prefix": "zenspanel/",
  "access_key": "AKIAIOSFODNN7EXAMPLE",
  "secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
  "region": "us-east-1",
  "endpoint": "https://s3.wasabisys.com",
  "enabled": true
}
```

`type` defaults to `"s3"`. `region` defaults to `"us-east-1"`.

**Response 201:** backup target object (without `secret_key`).

---

### PUT /admin/backup-targets/:id

Partial update. Omitting `secret_key` (or sending empty string) keeps the existing encrypted secret unchanged.

**Response 200:** `{ "message": "updated" }`

---

### DELETE /admin/backup-targets/:id

**Response 200:** `{ "message": "deleted" }`

---

### POST /admin/backup-targets/:id/test

Test connectivity by listing the bucket via rclone. Agent pass-through.

**Response 200 (success):** `{ "ok": true }`

**Response 200 (failure):** `{ "ok": false, "error": "connection refused" }`
