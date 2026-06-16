# ZensPanel API — Request & Response Conventions

---

## Request Format

All request bodies: `Content-Type: application/json`

Exception — file upload: `Content-Type: multipart/form-data` (`POST /files/upload`)

---

## Response Format

### Success

```json
{ "data": [...], "total": 42 }    // paginated list
{ "data": [...] }                  // non-paginated list
{ "id": 1, "username": "alice" }  // single object (varies by endpoint)
{ "message": "updated" }          // mutation with no return body
```

### Error

```json
{ "error": "description of what went wrong" }
```

### Warnings (non-fatal side-effect failures)

Some operations (user create, delete, suspend) succeed but report partial agent failures in a `warnings` array:

```json
{
  "id": 5,
  "username": "alice",
  "warnings": ["cgroups: slice already exists"]
}
```

---

## HTTP Status Codes

| Code | Meaning |
|------|---------|
| `200` | OK |
| `201` | Created |
| `202` | Accepted — async job started, poll for status |
| `302` | Redirect (phpMyAdmin SSO) |
| `400` | Bad request / validation failure |
| `401` | Missing or invalid token |
| `403` | Forbidden (wrong role, resource not owned, feature disabled) |
| `404` | Resource not found |
| `409` | Conflict (e.g. duplicate FQDN) |
| `410` | Gone (one-time token expired or already used) |
| `413` | Payload too large (file upload exceeds 64 MiB) |
| `429` | Rate limited or package quota exceeded |
| `500` | Server or agent error |
| `503` | Dependency unavailable (e.g. Redis required for phpMyAdmin SSO) |

---

## Pagination

List endpoints that support pagination accept these query params:

| Param | Default | Description |
|-------|---------|-------------|
| `page` | `1` | Page number (1-based) |
| `limit` | `20` | Items per page |
| `search` | — | Full-text search (where supported) |
| `sort` | `id` | Column to sort by |
| `order` | `asc` | `asc` or `desc` |

Paginated responses always include a `total` field:

```json
{ "data": [...], "total": 142 }
```

---

## Null Fields

`sql.NullTime` and `sql.NullInt64` serialize as nested objects. **Always check `Valid` before using the inner value** — never read `Time` or `Int64` directly when `Valid` is false.

```json
{ "ssl_expires_at": { "Time": "2026-09-01T00:00:00Z", "Valid": true  } }
{ "ssl_expires_at": { "Time": "0001-01-01T00:00:00Z", "Valid": false } }
{ "package_id":     { "Int64": 2,                     "Valid": true  } }
{ "package_id":     { "Int64": 0,                     "Valid": false } }
```

`sql.NullString` follows the same pattern with a `String` inner field:

```json
{ "error_msg": { "String": "disk full", "Valid": true  } }
{ "error_msg": { "String": "",          "Valid": false } }
```
