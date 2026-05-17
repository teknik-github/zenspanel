# ZensPanel

Self-hosted web hosting control panel built with Go + Vue 3. Deploy and manage websites with Nginx, PHP-FPM (multi-version), MySQL, Redis, and per-user resource isolation via Linux cgroups v2 — comparable to cPanel, Plesk, or CyberPanel.

![License](https://img.shields.io/badge/license-MIT-blue)
![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)
![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js)

---

## Features

### Admin Panel
- User management with package assignment
- Package templates (CPU, RAM, disk, domain/DB limits)
- PHP version management — enable/disable per version globally
- SSL Manager — Let's Encrypt + custom SSL upload
- API Keys with granular permissions (for WHMCS/billing integration)
- Audit logs
- Resource monitor

### User Panel
- Domain management with per-site PHP version selection
- SSL — issue Let's Encrypt or upload custom certificate
- MySQL database management + phpMyAdmin SSO
- File Manager
- Web Terminal (isolated rbash, enable/disable per user)
- Backup & restore (full / DB / files, enable/disable per user)

### Resource Isolation
- Linux cgroups v2 — CPU quota, RAM limit per user
- Disk quota via Linux quota tools
- PHP-FPM pool per user (runs as Linux user)
- Isolated terminal — rbash, home dir only, no sudo

### External API
- REST API with API key authentication
- Create/list/update/delete users and packages
- Assign packages, suspend/unsuspend users
- Compatible with WHMCS, FOSSBilling, and custom billing systems

---

## Architecture

```
┌─────────────────────────────────────────┐
│           Browser (Vue 3 + Vite)        │
│         Admin Panel  |  User Panel      │
└──────────────┬──────────────────────────┘
               │ HTTPS / WebSocket
┌──────────────▼──────────────────────────┐
│     zenspanel-api  (Go / Gin)           │
│     REST API + JWT Auth + RBAC          │
│     Runs as: www-data / non-root        │
└──────────────┬──────────────────────────┘
               │ JSON-RPC 2.0 over Unix socket
               │ /run/zenspanel/agent.sock
┌──────────────▼──────────────────────────┐
│     zenspanel-agent  (Go)               │
│     Privileged system operations        │
│     Runs as: root                       │
│                                         │
│  nginx │ phpfpm │ cgroups │ ssl         │
│  mysql │ terminal │ user                │
└─────────────────────────────────────────┘
```

Two Go binaries communicate via Unix socket (JSON-RPC 2.0). The API never runs as root — only the agent does, and it validates all inputs before executing system commands.

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.22+, Gin, sqlx, golang-migrate |
| Database | MySQL 8.0 / MariaDB 10.6 |
| Cache / Queue | Redis 7 |
| Frontend | Vue 3, Vite, Pinia, Vue Router 4, TailwindCSS 3 |
| Terminal | xterm.js over WebSocket, creack/pty |
| Web Server | Nginx |
| PHP | PHP-FPM 7.4 / 8.1 / 8.2 / 8.3 |
| SSL | Let's Encrypt (certbot) |

---

## Requirements

- Ubuntu 22.04 or 24.04
- 1 GB RAM minimum (2 GB recommended)
- 5 GB free disk space
- Root access

---

## Installation

```bash
git clone https://github.com/teknik-github/zenspanel.git
cd zenspanel
sudo bash scripts/install.sh
```

The installer will:
1. Check system requirements
2. Prompt for configuration (domain, MySQL password, admin credentials)
3. Install all dependencies (Nginx, MySQL, Redis, PHP 8.1/8.2/8.3, certbot, phpMyAdmin)
4. Install Go 1.22 and Node.js 20
5. Build binaries and frontend from source
6. Run database migrations
7. Create admin user
8. Configure systemd services, Nginx, and firewall

After installation, access the panel at:
- **User Panel:** `http://your-server:8888`
- **Admin Panel:** `http://your-server:8888/admin`
- **phpMyAdmin:** `http://your-server:8888/phpmyadmin`

---

## Development

### Prerequisites

- Go 1.22+
- Node.js 20+
- pnpm
- MySQL 8.0
- Redis

### Setup

```bash
# Clone
git clone https://github.com/teknik-github/zenspanel.git
cd zenspanel

# Copy and edit config
cp config.yaml.example config.yaml
# Edit config.yaml with your local MySQL credentials

# Install Go dependencies
go mod tidy

# Run database migrations
go run ./cmd/api

# Install frontend dependencies
cd frontend && pnpm install

# Start development servers
# Terminal 1 — API
make run-api

# Terminal 2 — Agent (requires root for system operations)
sudo make run-agent

# Terminal 3 — Admin Panel
cd frontend && pnpm --filter @zenspanel/admin dev

# Terminal 4 — User Panel
cd frontend && pnpm --filter @zenspanel/user dev
```

### Build

```bash
# Build Go binaries
make build

# Build frontend
cd frontend
pnpm --filter @zenspanel/admin build
pnpm --filter @zenspanel/user build
```

### Run tests

```bash
go test ./... -v
```

---

## Project Structure

```
zenspanel/
├── cmd/
│   ├── api/          — API server entrypoint
│   └── agent/        — Agent sidecar entrypoint
├── internal/
│   ├── api/          — Gin routes, handlers, middleware
│   ├── auth/         — JWT, RBAC, API key validation
│   ├── agent/        — Unix socket client
│   ├── store/        — MySQL data access layer
│   └── config/       — Configuration loading
├── agent/
│   ├── nginx/        — Nginx vhost management
│   ├── phpfpm/       — PHP-FPM pool management
│   ├── cgroups/      — cgroups v2 resource limits
│   ├── ssl/          — SSL certificate management
│   ├── terminal/     — PTY terminal sessions
│   ├── mysql/        — MySQL database management
│   └── user/         — Linux user management
├── migrations/       — SQL migration files
├── frontend/
│   ├── apps/
│   │   ├── admin/    — Admin Panel (Vue 3)
│   │   └── user/     — User Panel (Vue 3)
│   └── packages/
│       └── ui/       — Shared components
├── scripts/
│   └── install.sh    — Bash installer
└── docs/
    └── superpowers/
        └── specs/    — Design specification
```

---

## API Reference

Authentication via `Authorization: Bearer <jwt>` or `X-API-Key: <key>`.

### Users
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users` | List users |
| POST | `/api/v1/users` | Create user |
| GET | `/api/v1/users/:id` | Get user |
| PUT | `/api/v1/users/:id` | Update user |
| DELETE | `/api/v1/users/:id` | Delete user |
| PUT | `/api/v1/users/:id/suspend` | Suspend user |
| PUT | `/api/v1/users/:id/unsuspend` | Unsuspend user |
| PUT | `/api/v1/users/:id/package` | Change package |
| GET | `/api/v1/users/:id/usage` | Resource usage |

### Packages
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/packages` | List packages |
| POST | `/api/v1/packages` | Create package |
| PUT | `/api/v1/packages/:id` | Update package |
| DELETE | `/api/v1/packages/:id` | Delete package |

Full API documentation available in `docs/superpowers/specs/2026-05-17-zenspanel-design.md`.

---

## Services

```bash
# Status
systemctl status zenspanel-api
systemctl status zenspanel-agent

# Restart
systemctl restart zenspanel-api
systemctl restart zenspanel-agent

# Logs
tail -f /var/log/zenspanel/api.log
tail -f /var/log/zenspanel/agent.log
```

---

## Security

- API runs as non-root (`www-data`); only agent runs as root
- Agent socket permissions: `0600`, owned by `root:www-data`
- All agent RPC inputs validated before shell execution
- No shell string interpolation — all commands use `exec.Command` with argument arrays
- JWT secrets stored in environment variables / config file (chmod 600)
- API keys stored as bcrypt hash, shown only once at creation
- Rate limiting on login endpoint
- All user actions logged to audit_logs table
- PHP-FPM pools run as the Linux user (no shared www-data pool)
- rbash terminal: no PATH outside home, no sudo, no network tools

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
