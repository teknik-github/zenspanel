# ZensPanel API — User Panel Index

This page is a navigation index. Each section links to its dedicated reference file.

See [overview.md](./overview.md) for base URL, authentication format, null-field conventions, WebSocket terminal protocol, and phpMyAdmin SSO flow.

---

## Sections

| File | Covers |
|------|--------|
| [user-auth.md](./user-auth.md) | Login, 2FA setup/confirm/disable/recover, GET /auth/me, user profile, resource usage |
| [user-domains.md](./user-domains.md) | Domains CRUD + suspend/unsuspend/backup, Subdomains CRUD, SSL (Let's Encrypt + custom), Redirects, Hotlink protection, Domain logs |
| [user-databases.md](./user-databases.md) | Databases CRUD + password reset + phpMyAdmin SSO, FTP accounts CRUD |
| [user-files.md](./user-files.md) | File manager API (list/read/write/mkdir/rename/delete/upload/chmod/copy/compress/extract), FileBrowser web UI (auth bridge + provisioning), Backups (async + download + restore), Cron jobs |
| [user-misc.md](./user-misc.md) | PHP extensions toggle, Packages (read-only), PHP versions (read-only), Terminal WebSocket token, Website installer, Antivirus |

---

## Quick Reference

### Authentication

```
Authorization: Bearer <token>
```

Or via `zenspanel_token` HttpOnly cookie set on login. API key (`X-API-Key`) is for admin external routes only — not user panel endpoints.

### 2FA Login Flow

```
POST /auth/login  →  { requires_2fa: true, temp_token: "..." }
         │
         ├── Have app?   POST /auth/2fa/verify  { temp_token, code }          →  { token, user }
         └── Lost app?   POST /auth/2fa/recover { temp_token, recovery_code } →  { token, user }
```

### Async Operations

Several operations return `202 Accepted`. Poll the relevant endpoint for status:

| Operation | Poll endpoint | Status field |
|-----------|--------------|--------------|
| `POST /backups` | `GET /backups/:id` | `status` |
| `POST /domains/:id/backup` | `GET /backups/:id` | `status` |
| `POST /backups/:id/restore` | `GET /backups/:id` | `status` |
| `POST /installer/install` | `GET /installer/status/:job_id` | `done` / `error` |
| `POST /antivirus/scan` | `GET /antivirus/scan/:job_id` | `status` |

### Feature Flags

Some endpoints require flags set on the user account:

| Flag | Required by |
|------|-------------|
| `terminal_enabled` | `POST /terminal/token` |
| `backup_enabled` | `POST /backups` |
| `antivirus_enabled` (package flag) | All `/antivirus/*` endpoints |
