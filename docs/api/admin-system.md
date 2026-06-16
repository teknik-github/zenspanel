# ZensPanel API — Admin: System & Terminal

All endpoints require `Authorization: Bearer <token>` with role `admin`.

See [overview.md](./overview.md) for auth format, null-field conventions, and status codes.

---

## System

### GET /system/stats

Host-level server stats and service health.

**Response 200:**
```json
{
  "users":     { "total": 42, "active": 40, "suspended": 2 },
  "domains":   { "total": 87, "active": 85 },
  "databases": { "total": 63 },
  "cpu_percent": 23.5,
  "ram_used": 1073741824,
  "ram_total": 4294967296,
  "services": {
    "nginx": "running",
    "mysql": "running",
    "redis": "running"
  },
  "uptime_seconds": 864000
}
```

---

### GET /system/version

**Response 200:** `{ "version": "dev" }`

`version` is set at build time via `-ldflags "-X main.version=vX.Y.Z"`. Falls back to `"dev"` when running from source.

---

### GET /system/update/check

Check for available updates by comparing HEAD with origin/main. Agent pass-through.

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

---

### POST /system/update/run

Start a panel update (async). Poll `GET /system/update/status` for progress.

```json
{ "download_url": "https://..." }
```

`download_url` is optional — omit to pull from git instead of a release archive.

**Response 202:** agent pass-through map.

---

### GET /system/update/status

Poll update progress.

**Response 200:**
```json
{
  "phase": "building_api",
  "log": ["pulling...", "go build ./cmd/api..."],
  "done": false,
  "error": ""
}
```

Phase progression: `pulling` → `building_api` → `building_agent` → `building_frontend` → `deploying_frontend` → `restarting` → `done` / `failed`

---

### POST /system/maintenance

Trigger a one-off maintenance action.

```json
{ "action": "nginx_reload" }
```

Allowed `action` values:

| Action | Effect |
|--------|--------|
| `clamav_install` | Install/update ClamAV |
| `clamav_update` | Update virus signatures |
| `clamav_restart` | Restart clamd |
| `fail2ban_restart` | Restart fail2ban |
| `nginx_reload` | Reload nginx configuration |
| `service_status` | Query service statuses |
| `install_tools` | Install system tools (rclone, etc.) |

**Response 200 (most actions):**
```json
{ "output": "nginx reloaded", "error": "" }
```

**Response 200 (`service_status`):**
```json
{ "services": { "nginx": "running", "mysql": "running", "redis": "stopped" } }
```

---

## Admin Terminal

The admin terminal connects as the `zenspanel` system user by default, or as any named panel user. Uses the same two-step WebSocket flow as the user terminal.

### POST /admin/terminal/token

Mint a one-time WebSocket token.

```json
{ "username": "alice" }
```

`username` is optional. Omit to connect as the `zenspanel` system user.

**Response 200:**
```json
{ "token": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6" }
```

Token is **single-use** and expires in **60 seconds**.

Then connect:
```
WS /ws/terminal?token=<token>
```

See [Overview — WebSocket Terminal](./overview.md#websocket--terminal) for the full frame protocol.
