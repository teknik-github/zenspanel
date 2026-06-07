# ZensPanel API — Overview

## Base URL

```
https://<your-panel-domain>/api/v1
```

All endpoints are prefixed with `/api/v1`. In development the API listens on port `8080`.

---

## Authentication

ZensPanel uses **JWT Bearer tokens**. After login, include the token in every request:

```
Authorization: Bearer <token>
```

The token is also set as an `HttpOnly` cookie (`zenspanel_token`) automatically on login. Axios in the frontend reads from the cookie automatically, but the `Authorization` header takes precedence if both are present.

### Roles

| Role | Description |
|------|-------------|
| `admin` | Full access to all endpoints |
| `user` | Access to own resources only |
| `api_key` | External/billing system access via `X-API-Key` header |

---

## Two-Factor Authentication Flow

When a user has 2FA enabled, login returns a `temp_token` instead of a full JWT:

```
POST /auth/login → { requires_2fa: true, temp_token: "..." }
  ↓
POST /auth/2fa/verify { temp_token, code } → { token, user }
```

If the user loses their authenticator app, they can use a recovery code:

```
POST /auth/2fa/recover { temp_token, recovery_code } → { token, user }
```

Recovery codes are single-use. Using one also disables 2FA so the user can re-enroll.

---

## Request Format

All request bodies must be `Content-Type: application/json` unless uploading files (`multipart/form-data`).

---

## Response Format

### Success

```json
{ "data": [...] }          // list responses
{ "id": 1, "name": "..." } // single object responses
{ "message": "updated" }   // mutation responses
```

### Error

```json
{ "error": "description of what went wrong" }
```

### Warnings

Some operations (user create, delete) succeed but report non-fatal side-effect failures:

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
| `202` | Accepted (async job started) |
| `400` | Bad request / validation error |
| `401` | Unauthenticated |
| `403` | Forbidden (wrong role or not owner) |
| `404` | Resource not found |
| `409` | Conflict (duplicate) |
| `413` | Payload too large |
| `429` | Rate limited |
| `500` | Server error |

---

## Pagination

List endpoints that support pagination accept these query params:

| Param | Default | Description |
|-------|---------|-------------|
| `page` | `1` | Page number |
| `limit` | `20` | Items per page |
| `search` | — | Full-text search |
| `sort` | `id` | Column to sort by |
| `order` | `asc` | `asc` or `desc` |

Paginated responses include a `total` field:

```json
{ "data": [...], "total": 142 }
```

---

## External API (Billing Integration)

A separate route group at `/api/v1/external` is authenticated via `X-API-Key` header instead of JWT. This is intended for WHMCS, FOSSBilling, or custom billing integrations.

```
X-API-Key: zpk_live_xxxxxxxxxxxxxxxx
```

Each API key has a `permissions` field (comma-separated):

| Permission | Grants |
|-----------|--------|
| `read_user` | GET /external/users, GET /external/users/:id, GET /external/users/:id/usage |
| `create_user` | POST /external/users |
| `suspend_user` | PUT /external/users/:id/suspend, PUT /external/users/:id/unsuspend |
| `change_package` | PUT /external/users/:id/package |
| `read_package` | GET /external/packages, GET /external/packages/:id |

---

## WebSocket — Terminal

The terminal uses a two-step flow because browsers cannot send custom headers on WebSocket connections:

```
1. POST /api/v1/terminal/token  (JWT required)
   → { token: "<one-time 32-char hex token>" }

2. WS /ws/terminal?token=<token>
   → raw PTY stream
```

The token is single-use and expires in **60 seconds**. Rate limited to 1 token per user per 5 seconds.

### WebSocket message format

**Input (browser → server):**
```json
{ "type": "input", "data": "ls -la\n" }
{ "type": "resize", "cols": 220, "rows": 50 }
```

**Output (server → browser):**
```json
{ "type": "output", "data": "<base64-encoded PTY bytes>" }
```

---

## File Upload Limit

Multipart file uploads are capped at **64 MiB** per request (`POST /files/upload`).

---

## phpMyAdmin SSO

```
GET /api/v1/databases/:id/phpmyadmin/launch  (JWT required)
→ { url: "/api/v1/phpmyadmin/sso/<token>" }

Browser opens that URL in a new tab:
GET /api/v1/phpmyadmin/sso/<token>  (no JWT — token is the credential)
→ 302 redirect → phpMyAdmin auto-login form
```

The SSO token is single-use and expires in **60 seconds**. Requires Redis.

---

## Null fields

`sql.NullTime` and `sql.NullInt64` fields serialize as:

```json
{ "ssl_expires_at": { "Time": "2026-01-01T00:00:00Z", "Valid": true } }
{ "ssl_expires_at": { "Time": "0001-01-01T00:00:00Z", "Valid": false } }
```

Check `Valid` before using `Time`.
