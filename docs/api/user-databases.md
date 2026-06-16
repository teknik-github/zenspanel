# ZensPanel API — Databases & FTP

All endpoints require `Authorization: Bearer <token>` with role `user`.

See [overview.md](./overview.md) for auth format, null-field conventions, and status codes.

---

## Databases

### GET /databases

List the calling user's databases.

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "user_id": 5,
      "db_name": "alice_wp",
      "db_user": "alice_wp",
      "created_at": "2026-01-01T00:00:00Z"
    }
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

Drops MySQL schema and user via agent, then removes the row.

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

Mint a 60-second single-use phpMyAdmin SSO token. Open the returned URL in a new browser tab.

**Response 200:**
```json
{ "url": "/api/v1/phpmyadmin/sso/a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6" }
```

**Response 503:** Redis is not configured on the server.

Full SSO flow: see [Overview — phpMyAdmin SSO](./overview.md#phpmyadmin-sso).

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

Removes the vsftpd virtual user via agent and deletes the row.

**Response 200:** `{ "message": "deleted" }`
