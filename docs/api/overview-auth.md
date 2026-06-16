# ZensPanel API — Authentication & Roles

---

## Base URL

```
https://<your-panel-domain>/api/v1
```

In development the API listens on `http://127.0.0.1:8080`. Both Vue dev servers proxy `/api` and `/ws` to port 8080 automatically (`vite.config.ts`).

---

## Authentication

### JWT (primary)

Obtain a token via `POST /auth/login`, then include it in every request:

```
Authorization: Bearer <token>
```

The token is also set as a `zenspanel_token` **HttpOnly, SameSite=Strict** cookie on login. The Axios client reads it from the cookie automatically; the `Authorization` header takes precedence if both are present.

**JWT payload fields:**

| Field | Type | Description |
|-------|------|-------------|
| `user_id` | uint64 | User's database ID |
| `role` | string | `admin`, `user`, or `api_key` |
| `token_version` | int | Incremented on suspend/logout — old tokens become invalid immediately |

---

### API Key (external integrations)

```
X-API-Key: zp_live_<32 hex chars>
```

Valid on `/api/v1/external/*` routes only. Each key has a `permissions` field that gates individual sub-routes. See [overview-realtime.md — External API](./overview-realtime.md#external-api-billing-integration) for the full permissions table.

---

## Roles

| Role | Description |
|------|-------------|
| `admin` | Full access to all endpoints |
| `user` | Access to own resources only |
| `api_key` | Scoped external billing access via `X-API-Key` |

---

## Two-Factor Authentication (2FA) Login Flow

```
POST /auth/login  →  { requires_2fa: true, temp_token: "..." }
         │
         ├── Have authenticator app?
         │       POST /auth/2fa/verify  { temp_token, code }         →  { token, user }
         │
         └── Lost authenticator app?
                 POST /auth/2fa/recover { temp_token, recovery_code } →  { token, user }
                 (recovery code is single-use; 2FA is disabled afterward so user can re-enroll)
```

Full endpoint details: [user-auth.md](./user-auth.md#two-factor-authentication-2fa)
