# ZensPanel API — Domains & SSL

All endpoints require `Authorization: Bearer <token>` with role `user`.

See [overview.md](./overview.md) for auth format, null-field conventions, and status codes.

---

## Domains

### GET /domains

List domains for the calling user.

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "user_id": 5,
      "domain": "example.com",
      "document_root": "/home/zenspanel/alice/public_html/example.com",
      "php_version": "8.3",
      "ssl_type": "letsencrypt",
      "ssl_expires_at": { "Time": "2026-09-01T00:00:00Z", "Valid": true },
      "status": "active",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

`status` values: `pending`, `active`, `suspended`

`ssl_type` values: `none`, `letsencrypt`, `custom`

---

### GET /domains/:id

Get a single domain. Returns `403` if the domain belongs to another user.

---

### POST /domains

Add a domain. Provisions nginx vhost and creates document root via agent. Row is rolled back on agent failure.

```json
{ "domain": "example.com", "php_version": "8.3" }
```

**Response 201:** domain object (same shape as list item).

---

### PUT /domains/:id

Partial update. Changing `php_version` triggers PHP-FPM pool recreation and nginx vhost regeneration via agent.

```json
{ "php_version": "8.2" }
```

**Response 200:** `{ "message": "updated" }`

---

### DELETE /domains/:id

Delete domain. Tears down child subdomains first (nginx vhost + SSL), then the parent vhost.

**Response 200:** `{ "message": "deleted" }`

---

### POST /domains/:id/suspend

Replace nginx vhost with a 503 page.

**Response 200:** `{ "message": "domain suspended" }`

---

### POST /domains/:id/unsuspend

Rebuild nginx vhost from DB.

**Response 200:** `{ "message": "domain unsuspended" }`

---

### POST /domains/:id/backup

Async backup of the domain's document root (not full home directory). Poll `GET /backups` for status.

**Response 202:**
```json
{ "job_id": 42, "backup_id": 42 }
```

---

## Subdomains

### GET /subdomains

List subdomains. Use `?parent_id=<domain_id>` to scope to one parent.

**Response 200:**
```json
{
  "data": [
    {
      "id": 10,
      "user_id": 5,
      "parent_domain_id": 1,
      "subdomain": "blog",
      "fqdn": "blog.example.com",
      "document_root": "/home/zenspanel/alice/public_html/blog.example.com",
      "php_version": "8.3",
      "ssl_type": "none",
      "ssl_expires_at": { "Time": "0001-01-01T00:00:00Z", "Valid": false },
      "status": "active",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### GET /subdomains/:id

Get a single subdomain.

---

### POST /subdomains

Create a subdomain. Provisions PHP-FPM pool and nginx vhost via agent. Row is rolled back on agent failure.

```json
{
  "parent_domain_id": 1,
  "subdomain": "blog",
  "php_version": "8.3",
  "doc_root": "/home/zenspanel/alice/public_html/blog.example.com"
}
```

Field rules:
- `subdomain`: lowercase letters, digits, hyphens only. Cannot be: `www`, `mail`, `smtp`, `imap`, `pop`, `ns`, `ns1`, `ns2`
- `doc_root`: optional, defaults to `~/public_html/<fqdn>`. Must be inside the user's home directory
- `php_version`: defaults to the parent domain's version if omitted

**Response 201:** subdomain object.

**Response 409:** FQDN already in use.

---

### PUT /subdomains/:id

Partial update. Immutable fields (`id`, `user_id`, `parent_domain_id`, `subdomain`, `fqdn`) are stripped. Changing `php_version` or `document_root` triggers agent re-provisioning.

```json
{ "php_version": "8.2" }
```

**Response 200:** `{ "message": "updated" }`

---

### DELETE /subdomains/:id

Tears down nginx vhost and SSL cert via agent, then removes the row.

**Response 200:** `{ "message": "deleted" }`

---

## SSL

### POST /domains/:id/ssl

Issue or install an SSL certificate for a domain.

**Let's Encrypt:**
```json
{ "type": "letsencrypt" }
```

Calls certbot via agent. Sets `ssl_expires_at` to now+90 days.

**Custom certificate:**
```json
{
  "type": "custom",
  "cert_pem": "-----BEGIN CERTIFICATE-----\n...",
  "key_pem": "-----BEGIN PRIVATE KEY-----\n..."
}
```

Parses the actual `NotAfter` date from the certificate PEM.

**Response 200:** updated domain object.

---

### DELETE /domains/:id/ssl

Remove certificate files via agent. Sets `ssl_type = "none"`, `ssl_expires_at = null`.

**Response 200:** updated domain object.

---

### POST /subdomains/:id/ssl

Same contract as domain SSL — issue Let's Encrypt or install custom certificate for a subdomain.

**Response 200:** updated subdomain object.

---

### DELETE /subdomains/:id/ssl

Remove SSL certificate from subdomain.

**Response 200:** updated subdomain object.

---

## Redirects

### GET /domains/:id/redirects

List HTTP redirects for a domain.

**Response 200:**
```json
{
  "data": [
    {
      "id": 1,
      "domain_id": 1,
      "source_path": "/old-page",
      "dest_url": "https://example.com/new-page",
      "type": "301",
      "enabled": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /domains/:id/redirects

```json
{
  "source_path": "/old-page",
  "dest_url": "https://example.com/new-page",
  "type": "301",
  "enabled": true
}
```

`type`: `"301"` (permanent) or `"302"` (temporary). Defaults to `"301"`.

**Response 201:** redirect object. Syncs all redirects to nginx immediately.

---

### PUT /domains/:id/redirects/:rid

Partial update. Any field may be omitted.

```json
{ "enabled": false }
```

**Response 200:** `{ "message": "updated" }`

---

### DELETE /domains/:id/redirects/:rid

**Response 200:** `{ "message": "deleted" }`

---

## Hotlink Protection

### GET /domains/:id/hotlink

**Response 200:**
```json
{
  "enabled": true,
  "allowed_domains": ["example.com", "www.example.com"]
}
```

---

### PUT /domains/:id/hotlink

`allowed_domains` is the **complete replacement list**, not appended.

```json
{
  "enabled": true,
  "allowed_domains": ["example.com", "www.example.com", "partner.com"]
}
```

**Response 200:** `{ "message": "updated", "enabled": true }`

---

## Domain Logs

### GET /domains/:id/logs

Tail nginx or PHP-FPM logs.

**Query params:**

| Param | Values | Default |
|-------|--------|---------|
| `type` | `nginx`, `nginx-access`, `nginx-error`, `fpm` | `nginx` |
| `lines` | integer (1–500) | `100` |

**Response 200:**
```json
{
  "domain": "example.com",
  "type": "nginx",
  "log_path": "/var/log/nginx/example.com-error.log",
  "lines": [
    "2026-01-01 12:00:00 [error] 1234#0: *1 connect() failed...",
    "2026-01-01 12:00:01 [notice] signal process started"
  ]
}
```
