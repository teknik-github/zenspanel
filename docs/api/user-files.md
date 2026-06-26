# ZensPanel API — File Manager, FileBrowser, Backups & Cron Jobs

All endpoints require `Authorization: Bearer <token>` with role `user`.

See [overview.md](./overview.md) for auth format, null-field conventions, and status codes.

---

## File Manager

All paths are relative to the user's home directory. The agent rejects any path that would escape the home directory (path traversal is blocked at the agent boundary).

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

`mode` accepts octal strings (`"0755"`, `"755"`) or symbolic (`"rwxr-xr-x"`).

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

## FileBrowser

FileBrowser is a full-featured web-based file manager (third-party open-source binary, [filebrowser.xyz](https://filebrowser.xyz)) embedded in the user panel. It runs as a separate service (`zenspanel-filebrowser`) on port `8081` and is proxied by Nginx at `/filebrowser/`.

> **Note:** FileBrowser is a browser UI, not a REST API that frontend code calls directly. The ZensPanel API's only role here is the auth bridge used by Nginx.

### Access URL

```
https://<panel-domain>/filebrowser/
```

The user opens this URL in a browser (or in an iframe within the panel). No API call is needed from the frontend to launch FileBrowser.

### Authentication flow

Access is gated by Nginx `auth_request`. The request never reaches the FileBrowser process unless the panel session is valid:

```
Browser → GET /filebrowser/*
  → Nginx auth_request → GET /api/v1/auth/filebrowser  (internal — not client-callable)
      Validates ZensPanel JWT cookie
      Returns 200 + X-Auth-User: <username>   (valid session)
              401                               (no/invalid session)
  → On 200: Nginx forwards request to FileBrowser at 127.0.0.1:8081
            with header X-Auth-User: <username>
  → FileBrowser uses proxy-auth mode — no separate login required
  → On 401: Nginx returns 401 to browser; panel redirects to login
```

### GET /api/v1/auth/filebrowser

This endpoint is `internal` in Nginx — it can only be reached via `auth_request`, not from the internet or from frontend `$fetch` calls.

**Auth:** JWT cookie (`zenspanel_token`) set at panel login — standard user session.

**Response 200:** empty body; sets `X-Auth-User: <username>` response header for Nginx to forward.

**Response 401:** no valid session.

### User provisioning

FileBrowser users are managed automatically by the ZensPanel agent (`agent/filebrowser/`). When a panel user is created or their password changes, the agent calls the FileBrowser HTTP API to create/update the corresponding FileBrowser account. Each user's root directory inside FileBrowser is jailed to their panel home (`/home/zenspanel/<username>`), so they cannot browse outside their own files.

| FileBrowser config | Value |
|--------------------|-------|
| Base URL | `/filebrowser` |
| Listen address | `127.0.0.1:8081` |
| Auth mode | Proxy (`X-Auth-User` header) |
| DB | `$ZENSPANEL_DATA/filebrowser.db` (SQLite) |
| Max upload size | 1024 MiB (set in Nginx `client_max_body_size`) |
| Systemd service | `zenspanel-filebrowser.service` |

### Difference from the File Manager API

| | File Manager API | FileBrowser |
|-|-----------------|-------------|
| Interface | JSON REST API (Monaco editor integration) | Full web UI |
| Max file read | 4 MiB | No limit |
| Max upload | 64 MiB | 1024 MiB |
| Use case | Inline code editing, small file ops | Large uploads, drag-and-drop, bulk operations |
| URL | `/api/v1/files/*` | `/filebrowser/` |

---

## Backups

Backups are async — create returns immediately, poll `GET /backups` for status.

`status` values: `pending`, `running`, `done`, `failed`, `restoring`, `restore_failed`

`type` values: `full` (files + databases), `files` (home dir only), `db` (databases only), `domain` (domain docroot only)

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

**Response 202:** backup object with `status: "pending"`.

**Response 403:** backup not enabled on account.

---

### GET /backups/:id

Get a single backup.

---

### GET /backups/:id/download

Download the backup archive. Only available when `status == "done"`.

**Response 200:** binary file stream with `Content-Disposition: attachment; filename=<archive>`.

**Response 400:** backup not in `done` status.

---

### POST /backups/:id/restore

Restore from a backup. **Destructive — overwrites existing files and databases.** Only available when `status == "done"`. Async.

**Response 202:**
```json
{ "message": "restore started", "backup_id": 1 }
```

---

### DELETE /backups/:id

Delete backup record and archive file from disk.

**Response 200:** `{ "message": "deleted" }`

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

**Response 201:** `{ "data": <cron job object> }` — may include `"warning"` on crontab sync failure (job is still saved).

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
