# ZensPanel API — Auth & Profile

All endpoints use `Authorization: Bearer <token>` unless marked **No auth required**.

See [overview.md](./overview.md) for auth format, null-field conventions, and status codes.

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

Complete login via `POST /auth/2fa/verify` or `POST /auth/2fa/recover`.

**Response 401:** `{ "error": "invalid credentials" }`

**Response 403:** `{ "error": "account suspended" }`

---

### POST /auth/2fa/verify

Complete login when 2FA is enabled. **No auth required.**

```json
{ "temp_token": "a1b2c3d4...", "code": "123456" }
```

**Response 200:** same as normal login (token + user object). Sets cookie.

**Response 401:** invalid/expired `temp_token`, or wrong TOTP code.

---

### POST /auth/2fa/recover

Login using a backup recovery code. **No auth required.**

Recovery codes are **single-use**. After use, 2FA is disabled so the user can re-enroll.

```json
{ "temp_token": "a1b2c3d4...", "recovery_code": "ab12cd34ef" }
```

**Response 200:** same as normal login (token + user object). Sets cookie.

---

### GET /auth/me

Get current authenticated user's profile.

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

Generate a new TOTP secret. Returns a QR URL and 8 one-time recovery codes.

**Save the recovery codes immediately — they are never shown again.**

**Response 200:**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_url": "otpauth://totp/ZensPanel:alice?secret=JBSWY3DPEHPK3PXP&issuer=ZensPanel",
  "recovery_codes": [
    "ab12cd34ef", "gh56ij78kl", "mn90op12qr", "st34uv56wx",
    "yz78ab90cd", "ef12gh34ij", "kl56mn78op", "qr90st12uv"
  ]
}
```

2FA is **not yet active** after this call. Confirm with the next step.

---

### POST /auth/2fa/confirm

Activate 2FA by supplying the first TOTP code from the authenticator app.

```json
{ "code": "123456" }
```

**Response 200:** `{ "message": "2FA enabled" }`

**Response 400:** no pending setup session, or invalid code.

---

### DELETE /auth/2fa

Disable 2FA. Requires a valid current TOTP code as proof of possession.

```json
{ "code": "123456" }
```

**Response 200:** `{ "message": "2FA disabled" }`

**Response 401:** invalid code.

---

## User Profile

### GET /users/:id

Get a user profile. Users can only fetch their own ID.

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

Note: `role`, `status`, and `linux_uid` are **not** returned to non-admin callers. Admins receive the full object — see [Admin Docs](./admin.md#get-usersid).

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
