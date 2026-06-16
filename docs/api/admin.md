# ZensPanel API — Admin Panel Index

This page is a navigation index. Each section links to its dedicated reference file.

See [overview.md](./overview.md) for base URL, authentication format, null-field conventions, WebSocket terminal protocol, and external API (billing) routes.

---

## Sections

| File | Covers |
|------|--------|
| [admin-users.md](./admin-users.md) | Login + impersonate, Users CRUD + suspend/unsuspend/package/usage/metrics, Domains admin-view note |
| [admin-server.md](./admin-server.md) | Packages CRUD, PHP Versions CRUD + enable/disable, PHP Extensions CRUD + seed, Backup Targets CRUD + test |
| [admin-security.md](./admin-security.md) | Firewall (block/unblock + fail2ban jails), IP Allowlist, API Keys, Audit Logs |
| [admin-system.md](./admin-system.md) | System stats/version/update/maintenance, Admin Terminal WebSocket token |

---

## Quick Reference

### Authentication

```
Authorization: Bearer <token>   (role: admin)
```

Login: `POST /auth/login` — same endpoint as users. Role is determined by the stored account role.

### Admin-only vs shared endpoints

Some endpoints are accessible to both admins and regular users but return different data:

| Endpoint | Admin gets | User gets |
|----------|-----------|-----------|
| `GET /users` | All users + `linux_uid`, `role`, `status` | Own record only, restricted fields |
| `GET /users/:id` | Full object | Own ID only, no `linux_uid`/`role`/`status` |
| `GET /packages` | Full limits (raw bytes, I/O limits) | Customer-visible quota fields only |
| `GET /domains` | All users' domains | Own domains only |
| `POST /domains` | Accepts `user_id` to assign ownership | Own account only |

### Impersonation

`POST /users/:id/impersonate` returns a short-lived (1h) JWT scoped to the target user. Use it to open the user panel as that user for debugging. Cannot impersonate admins or suspended users.

### Async Operations

| Operation | Poll endpoint | Status field |
|-----------|--------------|--------------|
| `POST /system/update/run` | `GET /system/update/status` | `done` / `error` |
