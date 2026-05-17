#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# ZensPanel Installer
# Supports: Ubuntu 22.04 / 24.04
# Usage: bash install.sh
# =============================================================================

ZENSPANEL_VERSION="1.0.0"
ZENSPANEL_DIR="/opt/zenspanel"
ZENSPANEL_DATA="/var/lib/zenspanel"
ZENSPANEL_LOG="/var/log/zenspanel"
ZENSPANEL_CONF="/etc/zenspanel"
ZENSPANEL_REPO="https://github.com/zenspanel/zenspanel"
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

    # Root check
    [[ $EUID -eq 0 ]] || die "This installer must be run as root. Use: sudo bash install.sh"

    # OS check
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        if [[ "$ID" != "ubuntu" ]]; then
            die "Unsupported OS: $ID. ZensPanel requires Ubuntu 22.04 or 24.04."
        fi
        if [[ "$VERSION_ID" != "22.04" && "$VERSION_ID" != "24.04" ]]; then
            log_warn "Untested Ubuntu version: $VERSION_ID. Proceeding anyway..."
        fi
        log_info "OS: Ubuntu $VERSION_ID ✓"
    else
        die "Cannot detect OS. /etc/os-release not found."
    fi

    # RAM check (minimum 1GB)
    local ram_mb
    ram_mb=$(awk '/MemTotal/ {print int($2/1024)}' /proc/meminfo)
    if [[ $ram_mb -lt 1024 ]]; then
        log_warn "Low RAM: ${ram_mb}MB detected. Minimum recommended is 1024MB."
    else
        log_info "RAM: ${ram_mb}MB ✓"
    fi

    # Disk check (minimum 5GB free)
    local disk_gb
    disk_gb=$(df -BG / | awk 'NR==2 {print int($4)}')
    if [[ $disk_gb -lt 5 ]]; then
        die "Insufficient disk space: ${disk_gb}GB free. Minimum required is 5GB."
    fi
    log_info "Disk: ${disk_gb}GB free ✓"

    # Port check
    if ss -tlnp | grep -q ":${PANEL_PORT} "; then
        die "Port ${PANEL_PORT} is already in use. Please free it before installing."
    fi
    log_info "Port ${PANEL_PORT}: available ✓"

    # Check for existing installation
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

    echo ""
    echo -e "${BOLD}ZensPanel Configuration${NC}"
    echo "────────────────────────────────────────"

    # Panel domain/IP
    local default_ip
    default_ip=$(hostname -I | awk '{print $1}')
    read -rp "Panel domain or IP [${default_ip}]: " PANEL_HOST
    PANEL_HOST="${PANEL_HOST:-$default_ip}"

    # Panel port
    read -rp "Panel port [${PANEL_PORT}]: " input_port
    PANEL_PORT="${input_port:-$PANEL_PORT}"

    # MySQL config
    echo ""
    echo -e "${BOLD}MySQL Configuration${NC}"
    read -rp "MySQL root password (leave blank to auto-generate): " MYSQL_ROOT_PASS
    if [[ -z "$MYSQL_ROOT_PASS" ]]; then
        MYSQL_ROOT_PASS=$(openssl rand -base64 16 | tr -dc 'a-zA-Z0-9' | head -c 16)
        log_info "Generated MySQL root password: ${MYSQL_ROOT_PASS}"
    fi

    MYSQL_PANEL_PASS=$(openssl rand -base64 16 | tr -dc 'a-zA-Z0-9' | head -c 16)
    JWT_SECRET=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)

    # Admin user
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

    # Let's Encrypt
    echo ""
    read -rp "Let's Encrypt email for SSL certificates: " LE_EMAIL
    LE_EMAIL="${LE_EMAIL:-$ADMIN_EMAIL}"

    echo ""
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
        ufw \
        quota \
        acl \
        openssl \
        jq \
        bc

    # Nginx
    log_info "Installing Nginx..."
    apt-get install -y -qq nginx

    # MySQL
    log_info "Installing MySQL..."
    debconf-set-selections <<< "mysql-server mysql-server/root_password password ${MYSQL_ROOT_PASS}"
    debconf-set-selections <<< "mysql-server mysql-server/root_password_again password ${MYSQL_ROOT_PASS}"
    apt-get install -y -qq mysql-server

    # Redis
    log_info "Installing Redis..."
    apt-get install -y -qq redis-server

    # PHP versions
    log_info "Adding PHP repository (ondrej/php)..."
    add-apt-repository -y ppa:ondrej/php > /dev/null 2>&1
    apt-get update -qq

    log_info "Installing PHP 8.3, 8.2, 8.1..."
    for ver in 8.3 8.2 8.1; do
        apt-get install -y -qq \
            php${ver}-fpm \
            php${ver}-cli \
            php${ver}-mysql \
            php${ver}-curl \
            php${ver}-gd \
            php${ver}-mbstring \
            php${ver}-xml \
            php${ver}-zip \
            php${ver}-bcmath \
            php${ver}-intl \
            php${ver}-redis \
            php${ver}-imagick 2>/dev/null || log_warn "Some PHP ${ver} extensions not available, skipping..."
    done

    # Certbot
    log_info "Installing Certbot..."
    apt-get install -y -qq certbot python3-certbot-nginx

    # phpMyAdmin (non-interactive)
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

    if command -v go &>/dev/null; then
        local current_ver
        current_ver=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "Go ${current_ver} already installed, skipping..."
        return
    fi

    local go_tar="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    local go_url="https://go.dev/dl/${go_tar}"

    log_info "Downloading Go ${GO_VERSION}..."
    wget -q "$go_url" -O "/tmp/${go_tar}"

    log_info "Installing Go..."
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "/tmp/${go_tar}"
    rm "/tmp/${go_tar}"

    # Add to PATH
    if ! grep -q '/usr/local/go/bin' /etc/profile.d/go.sh 2>/dev/null; then
        echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
    fi
    export PATH=$PATH:/usr/local/go/bin

    log_info "Go $(go version) installed ✓"
}

# =============================================================================
# STEP 5: Install Node.js + pnpm
# =============================================================================
install_node() {
    log_section "Installing Node.js"

    if command -v node &>/dev/null; then
        log_info "Node.js $(node --version) already installed, skipping..."
    else
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

    # Create directories
    mkdir -p "$ZENSPANEL_DIR"/{bin,frontend}
    mkdir -p "$ZENSPANEL_DATA"/{home,backups,nginx,ssl,php}
    mkdir -p "$ZENSPANEL_LOG"
    mkdir -p "$ZENSPANEL_CONF"

    # Clone source
    if [[ -d "$ZENSPANEL_DIR/src" ]]; then
        log_info "Updating existing source..."
        git -C "$ZENSPANEL_DIR/src" pull --quiet
    else
        log_info "Cloning ZensPanel source..."
        # For local install from current directory
        if [[ -f "$(pwd)/go.mod" ]] && grep -q "zenspanel" "$(pwd)/go.mod" 2>/dev/null; then
            log_info "Using local source at $(pwd)..."
            cp -r "$(pwd)" "$ZENSPANEL_DIR/src"
        else
            git clone --quiet "$ZENSPANEL_REPO" "$ZENSPANEL_DIR/src" || \
                die "Failed to clone repository. Check your internet connection."
        fi
    fi

    local src="$ZENSPANEL_DIR/src"

    # Build Go binaries
    log_info "Building zenspanel-api..."
    (cd "$src" && go build -o "$ZENSPANEL_DIR/bin/zenspanel-api" ./cmd/api) || \
        die "Failed to build zenspanel-api"

    log_info "Building zenspanel-agent..."
    (cd "$src" && go build -o "$ZENSPANEL_DIR/bin/zenspanel-agent" ./cmd/agent) || \
        die "Failed to build zenspanel-agent"

    # Build frontend
    log_info "Building Admin Panel frontend..."
    (cd "$src/frontend" && pnpm install --silent && \
        pnpm --filter @zenspanel/admin build --silent) || \
        die "Failed to build admin frontend"

    log_info "Building User Panel frontend..."
    (cd "$src/frontend" && \
        pnpm --filter @zenspanel/user build --silent) || \
        die "Failed to build user frontend"

    # Copy frontend dist
    cp -r "$src/frontend/apps/admin/dist" "$ZENSPANEL_DIR/frontend/admin"
    cp -r "$src/frontend/apps/user/dist"  "$ZENSPANEL_DIR/frontend/user"

    log_info "ZensPanel built ✓"
}

# =============================================================================
# STEP 7: Setup MySQL
# =============================================================================
setup_mysql() {
    log_section "Setting up MySQL"

    # Start MySQL
    systemctl start mysql
    systemctl enable mysql --quiet

    # Secure MySQL and create panel database
    mysql -u root -p"${MYSQL_ROOT_PASS}" <<EOF 2>/dev/null || \
    mysql -u root <<EOF
ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${MYSQL_ROOT_PASS}';
DELETE FROM mysql.user WHERE User='';
DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');
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

    # Create nginx conf directory
    mkdir -p /etc/nginx/zenspanel
    mkdir -p /etc/nginx/ssl/zenspanel

    log_info "Configuration written ✓"
}

# =============================================================================
# STEP 9: Run database migrations
# =============================================================================
run_migrations() {
    log_section "Running database migrations"

    local src="$ZENSPANEL_DIR/src"
    (cd "$src" && \
        ZENSPANEL_CONFIG="$ZENSPANEL_CONF/config.yaml" \
        "$ZENSPANEL_DIR/bin/zenspanel-api" migrate 2>/dev/null) || \
    # Fallback: run API briefly to trigger auto-migration
    (cd "$src" && \
        cp "$ZENSPANEL_CONF/config.yaml" config.yaml && \
        timeout 10 "$ZENSPANEL_DIR/bin/zenspanel-api" || true)

    log_info "Migrations completed ✓"
}

# =============================================================================
# STEP 10: Create admin user
# =============================================================================
create_admin_user() {
    log_section "Creating admin user"

    local pass_hash
    pass_hash=$(python3 -c "import bcrypt; print(bcrypt.hashpw('${ADMIN_PASS}'.encode(), bcrypt.gensalt()).decode())" 2>/dev/null) || \
    pass_hash=$(php -r "echo password_hash('${ADMIN_PASS}', PASSWORD_BCRYPT);" 2>/dev/null) || \
    pass_hash=$(openssl passwd -6 "${ADMIN_PASS}")

    mysql -u zenspanel -p"${MYSQL_PANEL_PASS}" zenspanel <<EOF
INSERT INTO users (username, email, password_hash, role, linux_uid, status, terminal_enabled, backup_enabled)
VALUES ('${ADMIN_USER}', '${ADMIN_EMAIL}', '${pass_hash}', 'admin', 9000, 'active', TRUE, TRUE)
ON DUPLICATE KEY UPDATE email='${ADMIN_EMAIL}', password_hash='${pass_hash}';
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

    # Create socket directory
    mkdir -p /run/zenspanel
    chown root:zenspanel /run/zenspanel
    chmod 750 /run/zenspanel

    # zenspanel-agent service (runs as root)
    cat > /etc/systemd/system/zenspanel-agent.service <<EOF
[Unit]
Description=ZensPanel Agent
After=network.target mysql.service

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

    # zenspanel-api service (runs as zenspanel user)
    cat > /etc/systemd/system/zenspanel-api.service <<EOF
[Unit]
Description=ZensPanel API
After=network.target mysql.service zenspanel-agent.service

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

    # Set permissions
    chown -R zenspanel:zenspanel "$ZENSPANEL_DIR/bin"
    chown -R zenspanel:zenspanel "$ZENSPANEL_LOG"
    chown zenspanel:zenspanel "$ZENSPANEL_CONF/config.yaml"

    systemctl daemon-reload
    systemctl enable zenspanel-agent zenspanel-api --quiet
    systemctl start zenspanel-agent

    sleep 2

    systemctl start zenspanel-api

    log_info "Services started ✓"
}

# =============================================================================
# STEP 12: Setup Nginx reverse proxy
# =============================================================================
setup_nginx() {
    log_section "Setting up Nginx"

    # Main panel nginx config
    cat > /etc/nginx/sites-available/zenspanel <<EOF
server {
    listen ${PANEL_PORT};
    server_name ${PANEL_HOST};

    # Admin Panel
    location /admin {
        alias ${ZENSPANEL_DIR}/frontend/admin;
        try_files \$uri \$uri/ /admin/index.html;
    }

    # User Panel (default)
    location / {
        root ${ZENSPANEL_DIR}/frontend/user;
        try_files \$uri \$uri/ /index.html;
    }

    # API proxy
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_read_timeout 300;
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
        location ~ \.php$ {
            fastcgi_pass unix:/run/php/php8.3-fpm.sock;
            fastcgi_index index.php;
            include fastcgi_params;
            fastcgi_param SCRIPT_FILENAME \$request_filename;
        }
    }

    access_log ${ZENSPANEL_LOG}/nginx-access.log;
    error_log  ${ZENSPANEL_LOG}/nginx-error.log;
}
EOF

    # Include zenspanel vhosts
    if ! grep -q "zenspanel" /etc/nginx/nginx.conf; then
        sed -i '/http {/a\\tinclude /etc/nginx/zenspanel/*.conf;' /etc/nginx/nginx.conf
    fi

    ln -sf /etc/nginx/sites-available/zenspanel /etc/nginx/sites-enabled/zenspanel
    rm -f /etc/nginx/sites-enabled/default

    nginx -t && systemctl reload nginx

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

    # Check if cgroups v2 is available
    if [[ -f /sys/fs/cgroup/cgroup.controllers ]]; then
        log_info "cgroups v2 available ✓"
        mkdir -p /sys/fs/cgroup/zenspanel
    else
        log_warn "cgroups v2 not available. Resource isolation will be limited."
        log_warn "To enable: add 'systemd.unified_cgroup_hierarchy=1' to GRUB_CMDLINE_LINUX in /etc/default/grub"
    fi

    # Enable disk quota on root filesystem
    if ! grep -q "usrquota" /etc/fstab; then
        log_warn "Disk quota not enabled in /etc/fstab. Add 'usrquota,grpquota' to your root mount options."
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
    echo -e "${BOLD}Services:${NC}"
    echo -e "  systemctl status zenspanel-api"
    echo -e "  systemctl status zenspanel-agent"
    echo ""
    echo -e "${BOLD}Logs:${NC}"
    echo -e "  ${ZENSPANEL_LOG}/api.log"
    echo -e "  ${ZENSPANEL_LOG}/agent.log"
    echo ""
    echo -e "${YELLOW}Credentials saved to: ${ZENSPANEL_CONF}/install.info${NC}"
    echo -e "${YELLOW}Keep this file secure!${NC}"
    echo ""
}

# =============================================================================
# Main
# =============================================================================
main() {
    clear
    echo -e "${BLUE}${BOLD}"
    echo "  ______          ______                    _ "
    echo " |___  /         (_____ \                  | |"
    echo "    / / ___ ____  _____) )____ ____   ____ | |"
    echo "   / / / _ \|  _ \|  ____/ _  |  _ \ / _  )| |"
    echo "  / /_|  __/| | | | |   ( ( | | | | ( (/ / | |"
    echo " /_____\___)|_| |_|_|    \_||_|_| |_|\____)|_|"
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
