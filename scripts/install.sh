#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# ZensPanel Installer
# Supports: Ubuntu 22.04 / 24.04
# Usage: sudo bash install.sh
# =============================================================================

ZENSPANEL_VERSION="1.0.0"
ZENSPANEL_DIR="/opt/zenspanel"
ZENSPANEL_DATA="/var/lib/zenspanel"
ZENSPANEL_LOG="/var/log/zenspanel"
ZENSPANEL_CONF="/etc/zenspanel"
ZENSPANEL_REPO="https://github.com/teknik-github/zenspanel"
PANEL_PORT="8888"
GO_VERSION="1.22.3"
GO_ARCH="amd64"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

log_info()    { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_section() { echo -e "\n${BLUE}${BOLD}==> $*${NC}"; }
die()         { log_error "$*"; exit 1; }

# =============================================================================
# STEP 1: Pre-flight checks
# =============================================================================
preflight_checks() {
    log_section "Pre-flight checks"

    [[ $EUID -eq 0 ]] || die "This installer must be run as root. Use: sudo bash install.sh"

    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        [[ "$ID" == "ubuntu" ]] || die "Unsupported OS: $ID. ZensPanel requires Ubuntu 22.04 or 24.04."
        if [[ "$VERSION_ID" != "22.04" && "$VERSION_ID" != "24.04" ]]; then
            log_warn "Untested Ubuntu version: $VERSION_ID. Proceeding anyway..."
        fi
        log_info "OS: Ubuntu $VERSION_ID ✓"
    else
        die "Cannot detect OS. /etc/os-release not found."
    fi

    local ram_mb
    ram_mb=$(awk '/MemTotal/ {print int($2/1024)}' /proc/meminfo)
    [[ $ram_mb -ge 1024 ]] || log_warn "Low RAM: ${ram_mb}MB. Minimum recommended is 1024MB."
    log_info "RAM: ${ram_mb}MB ✓"

    local disk_gb
    disk_gb=$(df -BG / | awk 'NR==2 {print int($4)}')
    [[ $disk_gb -ge 5 ]] || die "Insufficient disk space: ${disk_gb}GB free. Minimum required is 5GB."
    log_info "Disk: ${disk_gb}GB free ✓"

    if ss -tlnp 2>/dev/null | grep -q ":${PANEL_PORT} "; then
        die "Port ${PANEL_PORT} is already in use."
    fi
    log_info "Port ${PANEL_PORT}: available ✓"

    if [[ -d "$ZENSPANEL_DIR" ]]; then
        log_warn "Existing ZensPanel installation found at $ZENSPANEL_DIR"
        read -rp "Reinstall? This will overwrite existing files. [y/N]: " confirm
        [[ "$confirm" =~ ^[Yy]$ ]] || die "Installation cancelled."
    fi
}

# =============================================================================
# STEP 2: Collect configuration
# =============================================================================
collect_config() {
    log_section "Configuration"

    local default_ip
    default_ip=$(hostname -I | awk '{print $1}')

    echo ""
    echo -e "${BOLD}Panel Configuration${NC}"
    echo "────────────────────────────────────────"
    read -rp "Panel domain or IP [${default_ip}]: " PANEL_HOST
    PANEL_HOST="${PANEL_HOST:-$default_ip}"

    read -rp "Panel port [${PANEL_PORT}]: " input_port
    PANEL_PORT="${input_port:-$PANEL_PORT}"

    echo ""
    echo -e "${BOLD}MySQL Configuration${NC}"
    read -rp "MySQL root password (leave blank to auto-generate): " MYSQL_ROOT_PASS
    if [[ -z "$MYSQL_ROOT_PASS" ]]; then
        MYSQL_ROOT_PASS=$(openssl rand -base64 16 | tr -dc 'a-zA-Z0-9' | head -c 16)
        log_info "Generated MySQL root password: ${MYSQL_ROOT_PASS}"
    fi
    MYSQL_PANEL_PASS=$(openssl rand -base64 16 | tr -dc 'a-zA-Z0-9' | head -c 16)
    JWT_SECRET=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)

    echo ""
    echo -e "${BOLD}Admin Account${NC}"
    read -rp "Admin username [admin]: " ADMIN_USER
    ADMIN_USER="${ADMIN_USER:-admin}"

    read -rp "Admin email: " ADMIN_EMAIL
    while [[ -z "$ADMIN_EMAIL" ]]; do
        read -rp "Admin email (required): " ADMIN_EMAIL
    done

    while true; do
        read -rsp "Admin password: " ADMIN_PASS
        echo ""
        read -rsp "Confirm password: " ADMIN_PASS2
        echo ""
        [[ "$ADMIN_PASS" == "$ADMIN_PASS2" ]] && break
        log_warn "Passwords do not match. Try again."
    done

    echo ""
    read -rp "Let's Encrypt email [${ADMIN_EMAIL}]: " LE_EMAIL
    LE_EMAIL="${LE_EMAIL:-$ADMIN_EMAIL}"

    log_info "Configuration collected ✓"
}

# =============================================================================
# STEP 3: Install system dependencies
# =============================================================================
install_dependencies() {
    log_section "Installing system dependencies"

    export DEBIAN_FRONTEND=noninteractive

    log_info "Updating package lists..."
    apt-get update -qq

    log_info "Installing base packages..."
    apt-get install -y -qq \
        curl wget git unzip tar \
        build-essential \
        software-properties-common \
        apt-transport-https \
        ca-certificates \
        gnupg lsb-release \
        ufw quota acl openssl jq bc \
        iproute2

    log_info "Installing Nginx..."
    apt-get install -y -qq nginx

    log_info "Installing MySQL..."
    debconf-set-selections <<< "mysql-server mysql-server/root_password password ${MYSQL_ROOT_PASS}"
    debconf-set-selections <<< "mysql-server mysql-server/root_password_again password ${MYSQL_ROOT_PASS}"
    apt-get install -y -qq mysql-server

    log_info "Installing Redis..."
    apt-get install -y -qq redis-server

    log_info "Adding PHP repository..."
    add-apt-repository -y ppa:ondrej/php > /dev/null 2>&1
    apt-get update -qq

    log_info "Installing PHP 8.3, 8.2, 8.1..."
    for ver in 8.3 8.2 8.1; do
        apt-get install -y -qq \
            php${ver}-fpm php${ver}-cli php${ver}-mysql \
            php${ver}-curl php${ver}-gd php${ver}-mbstring \
            php${ver}-xml php${ver}-zip php${ver}-bcmath \
            php${ver}-intl 2>/dev/null || log_warn "Some PHP ${ver} extensions unavailable, skipping..."
    done

    log_info "Installing Certbot..."
    apt-get install -y -qq certbot python3-certbot-nginx

    log_info "Installing phpMyAdmin..."
    debconf-set-selections <<< "phpmyadmin phpmyadmin/dbconfig-install boolean true"
    debconf-set-selections <<< "phpmyadmin phpmyadmin/app-password-confirm password ${MYSQL_ROOT_PASS}"
    debconf-set-selections <<< "phpmyadmin phpmyadmin/mysql/admin-pass password ${MYSQL_ROOT_PASS}"
    debconf-set-selections <<< "phpmyadmin phpmyadmin/mysql/app-pass password ${MYSQL_ROOT_PASS}"
    debconf-set-selections <<< "phpmyadmin phpmyadmin/reconfigure-webserver multiselect none"
    apt-get install -y -qq phpmyadmin || log_warn "phpMyAdmin install had issues, continuing..."

    log_info "Dependencies installed ✓"
}

# =============================================================================
# STEP 4: Install Go
# =============================================================================
install_go() {
    log_section "Installing Go ${GO_VERSION}"

    export PATH=$PATH:/usr/local/go/bin

    if command -v go &>/dev/null; then
        log_info "Go $(go version | awk '{print $3}') already installed ✓"
        return
    fi

    local go_tar="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    log_info "Downloading Go ${GO_VERSION}..."
    wget -q "https://go.dev/dl/${go_tar}" -O "/tmp/${go_tar}"

    rm -rf /usr/local/go
    tar -C /usr/local -xzf "/tmp/${go_tar}"
    rm "/tmp/${go_tar}"

    echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
    export PATH=$PATH:/usr/local/go/bin

    log_info "Go $(go version) installed ✓"
}

# =============================================================================
# STEP 5: Install Node.js + pnpm
# =============================================================================
install_node() {
    log_section "Installing Node.js"

    if ! command -v node &>/dev/null; then
        log_info "Installing Node.js 20 LTS..."
        curl -fsSL https://deb.nodesource.com/setup_20.x | bash - > /dev/null 2>&1
        apt-get install -y -qq nodejs
    fi

    log_info "Installing pnpm..."
    npm install -g pnpm --quiet

    log_info "Node.js $(node --version) + pnpm $(pnpm --version) ✓"
}

# =============================================================================
# STEP 6: Clone and build ZensPanel
# =============================================================================
build_zenspanel() {
    log_section "Building ZensPanel"

    mkdir -p "$ZENSPANEL_DIR"/{bin,frontend}
    mkdir -p "$ZENSPANEL_DATA"/{home,backups}
    mkdir -p "$ZENSPANEL_LOG"
    mkdir -p "$ZENSPANEL_CONF"
    mkdir -p /etc/nginx/zenspanel
    mkdir -p /etc/nginx/ssl/zenspanel

    local src="$ZENSPANEL_DIR/src"

    # Use local source if running from repo, otherwise clone
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local repo_root
    repo_root="$(dirname "$script_dir")"

    if [[ -f "$repo_root/go.mod" ]] && grep -q "zenspanel" "$repo_root/go.mod" 2>/dev/null; then
        log_info "Using local source at $repo_root..."
        if [[ "$repo_root" != "$src" ]]; then
            rm -rf "$src"
            cp -r "$repo_root" "$src"
        fi
    else
        if [[ -d "$src/.git" ]]; then
            log_info "Updating existing source..."
            git -C "$src" pull --quiet
        else
            log_info "Cloning ZensPanel from GitHub..."
            git clone --quiet "$ZENSPANEL_REPO" "$src" || \
                die "Failed to clone repository. Check your internet connection."
        fi
    fi

    # Ensure all Go dependencies are present
    log_info "Resolving Go dependencies..."
    (cd "$src" && /usr/local/go/bin/go mod tidy) || \
        die "Failed to resolve Go dependencies"

    # Build Go binaries
    log_info "Building zenspanel-api..."
    (cd "$src" && /usr/local/go/bin/go build -o "$ZENSPANEL_DIR/bin/zenspanel-api" ./cmd/api) || \
        die "Failed to build zenspanel-api"

    log_info "Building zenspanel-agent..."
    (cd "$src" && /usr/local/go/bin/go build -o "$ZENSPANEL_DIR/bin/zenspanel-agent" ./cmd/agent) || \
        die "Failed to build zenspanel-agent"

    # Build frontend
    log_info "Building frontend (this may take a few minutes)..."
    (cd "$src/frontend" && \
        pnpm install 2>&1 | grep -v "^Progress" && \
        pnpm approve-builds --yes 2>/dev/null || true && \
        pnpm -r build 2>&1) || die "Failed to build frontend"

    cp -r "$src/frontend/apps/admin/dist" "$ZENSPANEL_DIR/frontend/admin"
    cp -r "$src/frontend/apps/user/dist"  "$ZENSPANEL_DIR/frontend/user"

    log_info "ZensPanel built ✓"
}

# =============================================================================
# STEP 7: Setup MySQL
# =============================================================================
setup_mysql() {
    log_section "Setting up MySQL"

    systemctl start mysql
    systemctl enable mysql --quiet

    # Wait for MySQL to be ready
    local retries=10
    while ! mysqladmin ping --silent 2>/dev/null; do
        retries=$((retries - 1))
        [[ $retries -eq 0 ]] && die "MySQL did not start in time"
        sleep 2
    done

    # Detect working connection method
    local mysql_cmd=""
    if mysql -u root -e "SELECT 1;" > /dev/null 2>&1; then
        mysql_cmd="mysql -u root"
    elif mysql -u root -p"${MYSQL_ROOT_PASS}" -e "SELECT 1;" > /dev/null 2>&1; then
        mysql_cmd="mysql -u root -p${MYSQL_ROOT_PASS}"
    elif mysql -u root --socket=/var/run/mysqld/mysqld.sock -e "SELECT 1;" > /dev/null 2>&1; then
        mysql_cmd="mysql -u root --socket=/var/run/mysqld/mysqld.sock"
    else
        # All methods failed — reset root password via skip-grant-tables
        log_warn "Cannot connect to MySQL root. Resetting root password..."
        systemctl stop mysql 2>/dev/null || true
        sleep 2

        # Kill any remaining MySQL processes
        pkill -f mysqld_safe 2>/dev/null || true
        pkill -f mysqld 2>/dev/null || true
        sleep 2

        # Ensure socket directory exists
        mkdir -p /var/run/mysqld
        chown mysql:mysql /var/run/mysqld

        # Start with skip-grant-tables in background
        mysqld_safe --skip-grant-tables --skip-networking --user=mysql > /dev/null 2>&1 &
        local mysqld_pid=$!
        sleep 8

        # Reset password
        mysql -u root <<RESETEOF 2>/dev/null || true
FLUSH PRIVILEGES;
ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${MYSQL_ROOT_PASS}';
FLUSH PRIVILEGES;
RESETEOF

        # Kill mysqld_safe and all child mysqld processes cleanly
        kill $mysqld_pid 2>/dev/null || true
        pkill -f mysqld_safe 2>/dev/null || true
        pkill -f mysqld 2>/dev/null || true
        sleep 5

        # Remove stale socket/pid files
        rm -f /var/run/mysqld/mysqld.sock /var/run/mysqld/mysqld.pid 2>/dev/null || true

        # Restart MySQL normally
        systemctl start mysql
        sleep 3

        if mysql -u root -p"${MYSQL_ROOT_PASS}" -e "SELECT 1;" > /dev/null 2>&1; then
            mysql_cmd="mysql -u root -p${MYSQL_ROOT_PASS}"
            log_info "MySQL root password reset ✓"
        else
            die "Failed to reset MySQL root password. Please reset manually and re-run."
        fi
    fi

    log_info "Connected to MySQL ✓"

    $mysql_cmd <<EOF
ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${MYSQL_ROOT_PASS}';
DELETE FROM mysql.user WHERE User='';
DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost','127.0.0.1','::1');
DROP DATABASE IF EXISTS test;
DELETE FROM mysql.db WHERE Db='test' OR Db='test\\_%';
CREATE DATABASE IF NOT EXISTS zenspanel CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'zenspanel'@'localhost' IDENTIFIED BY '${MYSQL_PANEL_PASS}';
GRANT ALL PRIVILEGES ON zenspanel.* TO 'zenspanel'@'localhost';
FLUSH PRIVILEGES;
EOF

    log_info "MySQL configured ✓"
}

# =============================================================================
# STEP 8: Write configuration file
# =============================================================================
write_config() {
    log_section "Writing configuration"

    cat > "$ZENSPANEL_CONF/config.yaml" <<EOF
server:
  host: "127.0.0.1"
  port: 8080

database:
  dsn: "zenspanel:${MYSQL_PANEL_PASS}@tcp(127.0.0.1:3306)/zenspanel?parseTime=true&multiStatements=true"

redis:
  addr: "127.0.0.1:6379"

jwt:
  secret: "${JWT_SECRET}"
  expiry: "24h"
  refresh_expiry: "720h"

agent:
  socket: "/run/zenspanel/agent.sock"

paths:
  home_base: "${ZENSPANEL_DATA}/home"
  nginx_conf: "/etc/nginx/zenspanel"
  ssl_base: "/etc/nginx/ssl/zenspanel"
  backup_base: "${ZENSPANEL_DATA}/backups"
  php_pool_base: "/etc/php"

letsencrypt:
  email: "${LE_EMAIL}"
  staging: false
EOF

    chmod 600 "$ZENSPANEL_CONF/config.yaml"

    # Symlink config into src dir so API finds it at startup
    ln -sf "$ZENSPANEL_CONF/config.yaml" "$ZENSPANEL_DIR/src/config.yaml"

    log_info "Configuration written ✓"
}

# =============================================================================
# STEP 9: Run database migrations
# =============================================================================
run_migrations() {
    log_section "Running database migrations"

    # Run API briefly — it auto-migrates on startup then we kill it
    log_info "Running migrations via API startup..."
    (cd "$ZENSPANEL_DIR/src" && \
        timeout 30 "$ZENSPANEL_DIR/bin/zenspanel-api" 2>&1 | grep -E "migration|starting|error" || true)

    # Verify tables exist
    local table_count
    table_count=$(mysql -u zenspanel -p"${MYSQL_PANEL_PASS}" zenspanel \
        -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='zenspanel';" \
        -s -N 2>/dev/null || echo "0")

    if [[ "$table_count" -ge 10 ]]; then
        log_info "Migrations completed — ${table_count} tables created ✓"
    else
        log_warn "Expected 10+ tables, found ${table_count}. Check logs at ${ZENSPANEL_LOG}/api.log"
    fi
}

# =============================================================================
# STEP 10: Create admin user
# =============================================================================
create_admin_user() {
    log_section "Creating admin user"

    # Hash password using PHP (always available since we installed PHP)
    local pass_hash
    pass_hash=$(php -r "echo password_hash('${ADMIN_PASS}', PASSWORD_BCRYPT);")

    mysql -u zenspanel -p"${MYSQL_PANEL_PASS}" zenspanel 2>/dev/null <<EOF
INSERT INTO users (username, email, password_hash, role, linux_uid, status, terminal_enabled, backup_enabled)
VALUES ('${ADMIN_USER}', '${ADMIN_EMAIL}', '${pass_hash}', 'admin', 9000, 'active', TRUE, TRUE)
ON DUPLICATE KEY UPDATE
    email=VALUES(email),
    password_hash=VALUES(password_hash),
    role='admin',
    status='active';
EOF

    log_info "Admin user '${ADMIN_USER}' created ✓"
}

# =============================================================================
# STEP 11: Setup systemd services
# =============================================================================
setup_services() {
    log_section "Setting up systemd services"

    # Create zenspanel system user
    id zenspanel &>/dev/null || useradd -r -s /bin/false -d "$ZENSPANEL_DIR" zenspanel

    # Socket directory
    mkdir -p /run/zenspanel
    chown root:zenspanel /run/zenspanel
    chmod 750 /run/zenspanel

    # Persist socket dir across reboots
    cat > /etc/tmpfiles.d/zenspanel.conf <<EOF
d /run/zenspanel 0750 root zenspanel -
EOF

    # zenspanel-agent (root)
    cat > /etc/systemd/system/zenspanel-agent.service <<EOF
[Unit]
Description=ZensPanel Agent
After=network.target mysql.service
Wants=mysql.service

[Service]
Type=simple
User=root
ExecStart=${ZENSPANEL_DIR}/bin/zenspanel-agent
WorkingDirectory=${ZENSPANEL_DIR}/src
Restart=always
RestartSec=5
StandardOutput=append:${ZENSPANEL_LOG}/agent.log
StandardError=append:${ZENSPANEL_LOG}/agent-error.log

[Install]
WantedBy=multi-user.target
EOF

    # zenspanel-api (non-root)
    cat > /etc/systemd/system/zenspanel-api.service <<EOF
[Unit]
Description=ZensPanel API
After=network.target mysql.service zenspanel-agent.service
Wants=mysql.service zenspanel-agent.service

[Service]
Type=simple
User=zenspanel
Group=zenspanel
ExecStart=${ZENSPANEL_DIR}/bin/zenspanel-api
WorkingDirectory=${ZENSPANEL_DIR}/src
Restart=always
RestartSec=5
StandardOutput=append:${ZENSPANEL_LOG}/api.log
StandardError=append:${ZENSPANEL_LOG}/api-error.log

[Install]
WantedBy=multi-user.target
EOF

    # Permissions
    chown -R zenspanel:zenspanel "$ZENSPANEL_DIR/bin"
    chown -R zenspanel:zenspanel "$ZENSPANEL_LOG"
    chown zenspanel:zenspanel "$ZENSPANEL_CONF/config.yaml"
    # Agent needs to read config too
    chmod 640 "$ZENSPANEL_CONF/config.yaml"
    chown root:zenspanel "$ZENSPANEL_CONF/config.yaml"

    systemctl daemon-reload
    systemctl enable zenspanel-agent zenspanel-api --quiet

    log_info "Starting zenspanel-agent..."
    systemctl start zenspanel-agent
    sleep 3

    log_info "Starting zenspanel-api..."
    systemctl start zenspanel-api
    sleep 3

    # Verify both services are running
    if systemctl is-active --quiet zenspanel-agent; then
        log_info "zenspanel-agent: running ✓"
    else
        log_warn "zenspanel-agent failed to start. Check: journalctl -u zenspanel-agent"
    fi

    if systemctl is-active --quiet zenspanel-api; then
        log_info "zenspanel-api: running ✓"
    else
        log_warn "zenspanel-api failed to start. Check: journalctl -u zenspanel-api"
    fi
}

# =============================================================================
# STEP 12: Setup Nginx reverse proxy
# =============================================================================
setup_nginx() {
    log_section "Setting up Nginx"

    cat > /etc/nginx/sites-available/zenspanel <<EOF
server {
    listen ${PANEL_PORT};
    server_name ${PANEL_HOST};

    client_max_body_size 100M;

    # Admin Panel
    location /admin {
        alias ${ZENSPANEL_DIR}/frontend/admin;
        try_files \$uri \$uri/ /admin/index.html;
        index index.html;
    }

    # User Panel (default)
    location / {
        root ${ZENSPANEL_DIR}/frontend/user;
        try_files \$uri \$uri/ /index.html;
        index index.html;
    }

    # API proxy
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_read_timeout 300;
        proxy_connect_timeout 10;
    }

    # WebSocket terminal
    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_read_timeout 3600;
    }

    # phpMyAdmin
    location /phpmyadmin {
        alias /usr/share/phpmyadmin;
        index index.php;
        location ~ ^/phpmyadmin(.+\.php)$ {
            alias /usr/share/phpmyadmin\$1;
            fastcgi_pass unix:/run/php/php8.3-fpm.sock;
            fastcgi_index index.php;
            include fastcgi_params;
            fastcgi_param SCRIPT_FILENAME /usr/share/phpmyadmin\$1;
        }
        location ~* ^/phpmyadmin(.+\.(jpg|jpeg|gif|css|png|js|ico|html|xml|txt))$ {
            alias /usr/share/phpmyadmin\$1;
        }
    }

    access_log ${ZENSPANEL_LOG}/nginx-access.log;
    error_log  ${ZENSPANEL_LOG}/nginx-error.log;
}
EOF

    # Include zenspanel user vhosts in main nginx config
    if ! grep -q "zenspanel" /etc/nginx/nginx.conf; then
        sed -i '/http {/a\\tinclude /etc/nginx/zenspanel/*.conf;' /etc/nginx/nginx.conf
    fi

    ln -sf /etc/nginx/sites-available/zenspanel /etc/nginx/sites-enabled/zenspanel
    rm -f /etc/nginx/sites-enabled/default

    nginx -t 2>/dev/null && systemctl reload nginx
    systemctl enable nginx --quiet

    log_info "Nginx configured ✓"
}

# =============================================================================
# STEP 13: Setup firewall
# =============================================================================
setup_firewall() {
    log_section "Configuring firewall"

    ufw --force reset > /dev/null 2>&1
    ufw default deny incoming > /dev/null 2>&1
    ufw default allow outgoing > /dev/null 2>&1
    ufw allow ssh > /dev/null 2>&1
    ufw allow 80/tcp > /dev/null 2>&1
    ufw allow 443/tcp > /dev/null 2>&1
    ufw allow "${PANEL_PORT}/tcp" > /dev/null 2>&1
    ufw --force enable > /dev/null 2>&1

    log_info "Firewall configured ✓"
}

# =============================================================================
# STEP 14: Setup cgroups v2
# =============================================================================
setup_cgroups() {
    log_section "Setting up cgroups v2"

    if [[ -f /sys/fs/cgroup/cgroup.controllers ]]; then
        mkdir -p /sys/fs/cgroup/zenspanel
        log_info "cgroups v2 available ✓"
    else
        log_warn "cgroups v2 not available. Add 'systemd.unified_cgroup_hierarchy=1' to GRUB_CMDLINE_LINUX and reboot."
    fi
}

# =============================================================================
# STEP 15: Setup log rotation
# =============================================================================
setup_logrotate() {
    cat > /etc/logrotate.d/zenspanel <<EOF
${ZENSPANEL_LOG}/*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 zenspanel zenspanel
    sharedscripts
    postrotate
        systemctl reload zenspanel-api > /dev/null 2>&1 || true
    endscript
}
EOF
    log_info "Log rotation configured ✓"
}

# =============================================================================
# STEP 16: Save installation info
# =============================================================================
save_install_info() {
    cat > "$ZENSPANEL_CONF/install.info" <<EOF
INSTALL_DATE=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
ZENSPANEL_VERSION=${ZENSPANEL_VERSION}
PANEL_HOST=${PANEL_HOST}
PANEL_PORT=${PANEL_PORT}
ADMIN_USER=${ADMIN_USER}
ADMIN_EMAIL=${ADMIN_EMAIL}
MYSQL_ROOT_PASS=${MYSQL_ROOT_PASS}
MYSQL_PANEL_PASS=${MYSQL_PANEL_PASS}
LE_EMAIL=${LE_EMAIL}
EOF
    chmod 600 "$ZENSPANEL_CONF/install.info"
}

# =============================================================================
# Print summary
# =============================================================================
print_summary() {
    local api_status agent_status
    api_status=$(systemctl is-active zenspanel-api 2>/dev/null || echo "unknown")
    agent_status=$(systemctl is-active zenspanel-agent 2>/dev/null || echo "unknown")

    echo ""
    echo -e "${GREEN}${BOLD}╔══════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}${BOLD}║        ZensPanel Installation Complete!          ║${NC}"
    echo -e "${GREEN}${BOLD}╚══════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BOLD}Panel URLs:${NC}"
    echo -e "  User Panel:   ${BLUE}http://${PANEL_HOST}:${PANEL_PORT}${NC}"
    echo -e "  Admin Panel:  ${BLUE}http://${PANEL_HOST}:${PANEL_PORT}/admin${NC}"
    echo -e "  phpMyAdmin:   ${BLUE}http://${PANEL_HOST}:${PANEL_PORT}/phpmyadmin${NC}"
    echo ""
    echo -e "${BOLD}Admin Credentials:${NC}"
    echo -e "  Username: ${YELLOW}${ADMIN_USER}${NC}"
    echo -e "  Password: ${YELLOW}${ADMIN_PASS}${NC}"
    echo ""
    echo -e "${BOLD}MySQL:${NC}"
    echo -e "  Root password: ${YELLOW}${MYSQL_ROOT_PASS}${NC}"
    echo ""
    echo -e "${BOLD}Service Status:${NC}"
    echo -e "  zenspanel-api:   ${api_status}"
    echo -e "  zenspanel-agent: ${agent_status}"
    echo ""
    echo -e "${BOLD}Useful Commands:${NC}"
    echo -e "  systemctl status zenspanel-api"
    echo -e "  systemctl status zenspanel-agent"
    echo -e "  tail -f ${ZENSPANEL_LOG}/api.log"
    echo -e "  tail -f ${ZENSPANEL_LOG}/agent.log"
    echo ""
    echo -e "${YELLOW}All credentials saved to: ${ZENSPANEL_CONF}/install.info${NC}"
    echo ""
}

# =============================================================================
# Main
# =============================================================================
main() {
    clear
    echo -e "${BLUE}${BOLD}"
    cat << 'BANNER'
  ______          ______                    _
 |___  /         (_____ \                  | |
    / / ___ ____  _____) )____ ____   ____ | |
   / / / _ \|  _ \|  ____/ _  |  _ \ / _  )| |
  / /_|  __/| | | | |   ( ( | | | | ( (/ / | |
 /_____\___)|_| |_|_|    \_||_|_| |_|\____)|_|
BANNER
    echo -e "${NC}"
    echo -e "  ${BOLD}Version ${ZENSPANEL_VERSION} Installer${NC}"
    echo ""

    preflight_checks
    collect_config
    install_dependencies
    install_go
    install_node
    build_zenspanel
    setup_mysql
    write_config
    run_migrations
    create_admin_user
    setup_services
    setup_nginx
    setup_firewall
    setup_cgroups
    setup_logrotate
    save_install_info
    print_summary
}

main "$@"
