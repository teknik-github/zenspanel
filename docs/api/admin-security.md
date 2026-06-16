# ZensPanel API — Admin: Firewall, IP Allowlist, API Keys & Audit Logs

All endpoints require `Authorization: Bearer <token>` with role `admin`.

See [overview.md](./overview.md) for auth format, null-field conventions, and status codes.

---

## Firewall

### GET /admin/firewall/blocked

List all blocked IPs — panel manual blocks and fail2ban bans merged. Agent pass-through.

**Response 200:**
```json
{
  "data": [
    { "ip": "1.2.3.4", "reason": "brute force", "source": "panel",    "blocked_at": "2026-01-01T00:00:00Z" },
    { "ip": "5.6.7.8", "reason": "fail2ban-sshd","source": "fail2ban","blocked_at": "2026-01-01T00:00:00Z" }
  ]
}
```

---

### POST /admin/firewall/block

Block an IP or CIDR.

```json
{ "ip": "1.2.3.4", "reason": "manual block" }
```

**Response 200:** `{ "message": "blocked" }`

---

### POST /admin/firewall/unblock

```json
{ "ip": "1.2.3.4" }
```

**Response 200:** `{ "message": "unblocked" }`

---

### GET /admin/firewall/fail2ban/jails

List fail2ban jails and their status. Agent pass-through.

**Response 200:**
```json
{
  "data": [
    { "name": "sshd", "enabled": true, "ban_count": 5, "currently_banned": 2 }
  ]
}
```

---

### PUT /admin/firewall/fail2ban/jails/:name

Enable or disable a fail2ban jail.

```json
{ "enabled": false }
```

**Response 200:** `{ "message": "updated" }`

---

## IP Allowlist

Restricts admin panel access to specific IPs or CIDRs. An empty list means all IPs are allowed. Changes sync to nginx immediately.

### GET /admin/ip-allowlist

**Response 200:**
```json
{
  "data": [
    { "id": 1, "ip_cidr": "203.0.113.0/24", "note": "office", "created_at": "2026-01-01T00:00:00Z" }
  ]
}
```

---

### POST /admin/ip-allowlist

```json
{ "ip_cidr": "203.0.113.0/24", "note": "office" }
```

Accepts a single IP (`203.0.113.10`) or CIDR prefix (`203.0.113.0/24`). Validated as `netip.Addr` or `netip.Prefix`.

**Response 201:** IP allowlist entry object.

---

### DELETE /admin/ip-allowlist/:id

**Response 200:** `{ "message": "deleted" }`

---

## API Keys

API keys grant external systems (e.g. WHMCS, billing portals) scoped access to admin operations without exposing admin credentials. Keys are valid on `/api/v1/external/*` routes only.

The raw key is shown **once at creation** — only a hash is stored. Identify existing keys by `key_prefix`.

### GET /api-keys

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "name": "WHMCS Integration",
      "key_prefix": "zp_live_",
      "permissions": "read_user,create_user,suspend_user",
      "last_used_at": { "Time": "2026-01-01T00:00:00Z", "Valid": true  },
      "expires_at":   { "Time": "0001-01-01T00:00:00Z", "Valid": false },
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /api-keys

**The full key is shown only once — save it immediately.**

```json
{
  "name": "WHMCS Integration",
  "permissions": "read_user,create_user,suspend_user,change_package,read_package",
  "expires_at": "2027-01-01T00:00:00Z"
}
```

`expires_at` is optional. `permissions` is a comma-separated string of allowed values:

| Permission | Grants access to |
|-----------|-----------------|
| `read_user` | `GET /external/users`, `GET /external/users/:id`, `GET /external/users/:id/usage` |
| `create_user` | `POST /external/users` |
| `suspend_user` | `PUT /external/users/:id/suspend`, `PUT /external/users/:id/unsuspend` |
| `change_package` | `PUT /external/users/:id/package` |
| `read_package` | `GET /external/packages`, `GET /external/packages/:id` |

**Response 201:**
```json
{
  "id": 1,
  "key": "zp_live_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "prefix": "zp_live_a1b2",
  "note": "Save this key — it will not be shown again"
}
```

---

### DELETE /api-keys/:id

Revoke an API key. Immediately invalidates any in-flight requests using it.

**Response 200:** `{ "message": "revoked" }`

---

## Audit Logs

Every API call is automatically written to the audit log by middleware. Logs cannot be deleted via the API.

### GET /audit-logs

**Query params:**

| Param | Type | Description |
|-------|------|-------------|
| `page` | integer | Page number (default 1) |
| `limit` | integer | Items per page (default 20) |
| `action` | string | Filter by action string (e.g. `POST /api/v1/users`) |
| `user_id` | integer | Filter by user ID |
| `date_from` | ISO datetime | Start of range |
| `date_to` | ISO datetime | End of range |

**Response 200:**
```json
{
  "data": [
    {
      "id": 100,
      "user_id":    { "Int64": 1,                 "Valid": true  },
      "action":     "POST /api/v1/users",
      "resource":   { "String": "user:5",          "Valid": true  },
      "ip_address": "203.0.113.10",
      "user_agent": { "String": "Mozilla/5.0 ...", "Valid": true  },
      "meta":       { "String": "",                "Valid": false },
      "created_at": "2026-01-01T12:00:00Z"
    }
  ],
  "total": 500
}
```
