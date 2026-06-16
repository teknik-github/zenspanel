# ZensPanel API — WebSocket, SSO & External API

---

## WebSocket — Terminal

Browsers cannot send custom headers on WebSocket connections, so auth uses a two-step token flow:

```
Step 1 — mint a one-time token (JWT required):
  POST /api/v1/terminal/token          ← user terminal
  POST /api/v1/admin/terminal/token    ← admin terminal (any username)
  → { "token": "<32-char hex>" }

Step 2 — open WebSocket:
  WS /ws/terminal?token=<token>
```

Token is **single-use** and expires in **60 seconds**. Rate limited to 1 mint per 5 seconds per user.

### Frame format

**Browser → server:**
```json
{ "type": "input",  "data": "ls -la\n" }
{ "type": "resize", "cols": 220, "rows": 50 }
```

**Server → browser:**
```json
{ "type": "output", "data": "<base64-encoded PTY bytes>" }
```

Decode `data` with `atob()` then write the raw bytes to your xterm.js terminal instance.

---

## phpMyAdmin SSO

Databases are accessed via a single-use SSO token to avoid storing MySQL passwords in the panel.

```
Step 1 (JWT required):
  GET /api/v1/databases/:id/phpmyadmin/launch
  → { "url": "/api/v1/phpmyadmin/sso/<32hex>" }

Step 2 (no JWT — the token IS the credential):
  browser opens that URL in a new tab
  → 302 redirect to phpMyAdmin with auto-login
```

Token is **single-use** and expires in **60 seconds**. Requires Redis — returns `503` if Redis is unavailable.

---

## File Upload Limit

`POST /files/upload` — maximum **64 MiB** per request (`Content-Type: multipart/form-data`).

Exceeding the limit returns `413 Payload Too Large`.

---

## External API (Billing Integration)

For WHMCS, billing portals, or other automation that needs to manage panel accounts without using an admin JWT.

**Route prefix:** `/api/v1/external/`  
**Header:** `X-API-Key: zp_live_<32hex>`

Each API key carries a `permissions` comma-separated string:

| Permission | Routes granted |
|-----------|----------------|
| `read_user` | `GET /external/users`, `GET /external/users/:id`, `GET /external/users/:id/usage` |
| `create_user` | `POST /external/users` |
| `suspend_user` | `PUT /external/users/:id/suspend`, `PUT /external/users/:id/unsuspend` |
| `change_package` | `PUT /external/users/:id/package` |
| `read_package` | `GET /external/packages`, `GET /external/packages/:id` |

Handlers are identical to the JWT-protected equivalents. Admin-only routes (firewall, audit logs, PHP versions, etc.) are never accessible via API key.

Manage API keys via the admin panel: [admin-security.md — API Keys](./admin-security.md#api-keys).

---

## Certbot SSL Hook (server-internal)

```
POST /api/v1/system/ssl-renewed
Header: X-Hook-Secret: <shared secret from config.yaml>
Body:   { "domain": "example.com" }
```

Called automatically by the certbot deploy hook after certificate renewal. Updates `ssl_expires_at` in the database. **Not for frontend use.**
