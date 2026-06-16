# ZensPanel API — Overview Index

Start here. This page links to all reference files.

**Base URL:** `https://<your-panel-domain>/api/v1`  
**Dev proxy:** Vue dev servers proxy `/api` and `/ws` → `http://127.0.0.1:8080` automatically.

---

## Reference Files

### Overview (this folder)

| File | Covers |
|------|--------|
| [overview-auth.md](./overview-auth.md) | Base URL, JWT auth, API key auth, roles, 2FA login flow |
| [overview-conventions.md](./overview-conventions.md) | Request format, response shapes, HTTP status codes, pagination, null fields |
| [overview-realtime.md](./overview-realtime.md) | WebSocket terminal (frame protocol), phpMyAdmin SSO, file upload limit, external API (billing), certbot hook |

### Admin Panel

| File | Covers |
|------|--------|
| [admin.md](./admin.md) | Admin index + quick reference |
| [admin-users.md](./admin-users.md) | Login, impersonate, users CRUD + suspend/metrics, domains admin-view |
| [admin-server.md](./admin-server.md) | Packages, PHP versions, PHP extensions, backup targets |
| [admin-security.md](./admin-security.md) | Firewall, IP allowlist, API keys, audit logs |
| [admin-system.md](./admin-system.md) | System stats/update/maintenance, admin terminal |

### User Panel

| File | Covers |
|------|--------|
| [user.md](./user.md) | User index + quick reference |
| [user-auth.md](./user-auth.md) | Login, 2FA, profile, resource usage |
| [user-domains.md](./user-domains.md) | Domains, subdomains, SSL, redirects, hotlink, logs |
| [user-databases.md](./user-databases.md) | Databases, FTP, phpMyAdmin SSO |
| [user-files.md](./user-files.md) | File manager, backups, cron jobs |
| [user-misc.md](./user-misc.md) | PHP extensions, terminal, installer, antivirus |

---

## At a Glance

```
Authorization: Bearer <token>        ← JWT (login → /auth/login)
X-API-Key: zp_live_<32hex>          ← external billing routes only
zenspanel_token cookie               ← set automatically on login
```

**Null fields** use nested objects — always check `Valid`:
```json
{ "ssl_expires_at": { "Time": "2026-09-01T00:00:00Z", "Valid": true } }
{ "package_id":     { "Int64": 0,                     "Valid": false } }
```

**Async ops** return `202` — poll the relevant list endpoint for `status`.
