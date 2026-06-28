#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

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
    # Let's Encrypt rejects reserved/example domains as account contact
    # emails (example.com, test.com, localhost, …). Catch the obvious
    # placeholders here so the operator doesn't hit a 500 later when they
    # click "Issue SSL" in the panel.
    while true; do
        read -rp "Let's Encrypt email [${ADMIN_EMAIL}]: " LE_EMAIL
        LE_EMAIL="${LE_EMAIL:-$ADMIN_EMAIL}"
        case "${LE_EMAIL,,}" in
            *@example.com|*@example.net|*@example.org|*@test.com|*@localhost|*@invalid|*@localdomain)
                log_warn "Let's Encrypt rejects '${LE_EMAIL##*@}' as a contact email domain. Use a real domain you control."
                continue
                ;;
            *@*.*)
                break
                ;;
            *)
                log_warn "'${LE_EMAIL}' is not a valid email address."
                ;;
        esac
    done

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
        iproute2 inotify-tools rclone \
        bubblewrap

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

    # Configure phpMyAdmin for SSO (signon auth_type). This eliminates
    # the login form — credentials are passed via PHP session from our
    # SSO bridge script, so users click "phpMyAdmin" and land directly
    # in their database without seeing a login page.
    PMA_CONF="/etc/phpmyadmin/conf.d/zenspanel-sso.php"
    cat > "$PMA_CONF" <<'PHPEOF'
<?php
$cfg['Servers'][1]['auth_type'] = 'signon';
$cfg['Servers'][1]['SignonSession'] = 'SignonSession';
$cfg['Servers'][1]['SignonURL'] = '/phpmyadmin/signon.php';
$cfg['Servers'][1]['LogoutURL'] = '/phpmyadmin/signon.php?logout=1';
PHPEOF

    # SSO bridge script: reads credentials from Redis (set by Go API),
    # sets the PHP session vars phpMyAdmin expects, then redirects.
    cat > /usr/share/phpmyadmin/signon.php <<'PHPEOF'
<?php
// ZensPanel phpMyAdmin SSO bridge.
// The Go API stores credentials in Redis under pma_bridge:<token>
// and redirects here with ?bridge=<token>. We read the credentials,
// set the PHP session, and redirect to phpMyAdmin which auto-logs in.

session_name('SignonSession');
session_start();

if (isset($_GET['logout'])) {
    $_SESSION = [];
    session_destroy();
    header('Location: /');
    exit;
}

// If a bridge token is present, fetch credentials from Redis.
if (isset($_GET['bridge'])) {
    $token = preg_replace('/[^a-f0-9]/', '', $_GET['bridge']);
    if (strlen($token) === 32) {
        $redis = new Redis();
        if ($redis->connect('127.0.0.1', 6379)) {
            $key = 'pma_bridge:' . $token;
            $val = $redis->get($key);
            if ($val !== false) {
                $redis->del($key); // one-time use
                $data = json_decode($val, true);
                if (!empty($data['DBUser']) && !empty($data['Password'])) {
                    $_SESSION['PMA_single_signon_user']     = $data['DBUser'];
                    $_SESSION['PMA_single_signon_password'] = $data['Password'];
                    $_SESSION['PMA_single_signon_host']     = '127.0.0.1';
                    session_write_close();
                    header('Location: /phpmyadmin/');
                    exit;
                }
            }
        }
    }
}

// No valid credentials — redirect back to panel.
header('Location: /');
exit;
PHPEOF
    chmod 644 /usr/share/phpmyadmin/signon.php

    # Composer phar lives at a stable path; the agent's per-user
    # ~/bin/composer wrapper execs it via the user's pinned PHP. Without
    # this file, `composer` in the terminal returns "command not found".
    if [[ ! -f /usr/local/bin/composer.phar ]]; then
        log_info "Installing Composer..."
        curl -sS https://getcomposer.org/installer | php -- \
            --install-dir=/usr/local/bin --filename=composer.phar --quiet \
            || log_warn "Composer install failed, continuing..."
        chmod +x /usr/local/bin/composer.phar 2>/dev/null || true
    fi

    log_info "Dependencies installed ✓"
}

# =============================================================================
# STEP 3b: Install fail2ban + ipset (firewall hardening)
# =============================================================================
install_firewall() {
    log_section "Installing Firewall (fail2ban + ipset)"

    apt-get install -y -qq fail2ban ipset iptables-persistent

    # Create a persistent ipset for panel-managed blocks. The set survives
    # reboots via iptables-persistent + ipset save/restore.
    if ! ipset list zenspanel-blocked &>/dev/null; then
        ipset create zenspanel-blocked hash:ip timeout 0 comment
        log_info "Created ipset zenspanel-blocked"
    fi

    # Wire the ipset into iptables INPUT chain if not already present.
    if ! iptables -C INPUT -m set --match-set zenspanel-blocked src -j DROP 2>/dev/null; then
        iptables -I INPUT 1 -m set --match-set zenspanel-blocked src -j DROP
        log_info "Added iptables rule for zenspanel-blocked"
    fi

    # Persist iptables rules across reboots.
    netfilter-persistent save 2>/dev/null || iptables-save > /etc/iptables/rules.v4 2>/dev/null || true

    # Save ipset so it's restored on boot.
    mkdir -p /etc/ipset
    ipset save > /etc/ipset/zenspanel.conf

    # Restore ipset on boot via a systemd service.
    cat > /etc/systemd/system/zenspanel-ipset.service <<'EOF'
[Unit]
Description=ZensPanel ipset restore
Before=network.target iptables.service
DefaultDependencies=no

[Service]
Type=oneshot
ExecStart=/sbin/ipset restore -f /etc/ipset/zenspanel.conf
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF
    systemctl enable zenspanel-ipset --quiet

    # fail2ban jails for ZensPanel. Written to jail.d so they survive
    # fail2ban package upgrades (V35 — never write outside jail.d).
    cat > /etc/fail2ban/jail.d/zenspanel.conf <<'EOF'
[DEFAULT]
bantime  = 3600
findtime = 600
maxretry = 10

[sshd]
enabled  = true
port     = ssh
logpath  = %(sshd_log)s
backend  = %(sshd_backend)s
maxretry = 5

[nginx-http-auth]
enabled  = true
port     = http,https
logpath  = /var/log/nginx/error.log
maxretry = 10

[nginx-limit-req]
enabled  = true
port     = http,https
logpath  = /var/log/nginx/error.log
maxretry = 20
EOF

    # Recidive jail — permanent ban for repeat offenders.
    # If an IP gets banned 3+ times across any jail within 12 hours,
    # it gets a permanent ban (bantime = -1) until manually unblocked
    # via Admin → Firewall.
    cat > /etc/fail2ban/jail.d/zenspanel-recidive.conf <<'EOF'
[recidive]
enabled  = true
logpath  = /var/log/fail2ban.log
banaction = %(banaction_allports)s
bantime  = -1
findtime = 43200
maxretry = 3
EOF

    systemctl enable fail2ban --quiet
    systemctl restart fail2ban || log_warn "fail2ban restart failed, continuing..."

    log_info "Firewall (fail2ban + ipset) installed ✓"
}

# =============================================================================
# STEP 3c: Install ClamAV antivirus
# =============================================================================
install_clamav() {
    log_section "Installing ClamAV"

    # Refresh apt cache first — ClamAV packages may not be in the local
    # cache if this is a fresh system or the cache is stale.
    apt-get update -qq

    apt-get install -y -qq clamav clamav-daemon || {
        log_warn "ClamAV install failed — trying with universe repo..."
        add-apt-repository -y universe 2>/dev/null || true
        apt-get update -qq
        apt-get install -y -qq clamav clamav-daemon || {
            log_warn "ClamAV install failed, skipping. Install manually: apt-get install clamav clamav-daemon"
            return 0
        }
    }

    # Update virus definitions before starting the daemon.
    log_info "Updating ClamAV virus definitions..."
    systemctl stop clamav-freshclam 2>/dev/null || true
    freshclam --quiet || log_warn "freshclam update failed, continuing..."
    systemctl enable clamav-freshclam --quiet
    systemctl start clamav-freshclam || log_warn "clamav-freshclam start failed"

    systemctl enable clamav-daemon --quiet
    systemctl start clamav-daemon || log_warn "clamav-daemon start failed (will retry on next boot)"

    log_info "ClamAV installed ✓"
}

setup_vsftpd() {
    log_section "Setting up vsftpd (FTP server)"

    apt-get install -y -qq vsftpd db-util libpam-pwdfile || {
        log_warn "vsftpd install failed, skipping FTP support"
        return 0
    }

    # Create virtual user system account (no login shell, no home)
    if ! id vsftpd_virtual &>/dev/null; then
        useradd -r -d /dev/null -s /sbin/nologin vsftpd_virtual || true
    fi

    # Create vsftpd config directory for per-user configs
    mkdir -p /etc/vsftpd/users
    touch /etc/vsftpd/virtual_users.txt
    chmod 600 /etc/vsftpd/virtual_users.txt

    # Build initial (empty) PAM DB
    db_load -T -t hash -f /etc/vsftpd/virtual_users.txt /etc/vsftpd/virtual_users.db 2>/dev/null || true
    chmod 600 /etc/vsftpd/virtual_users.db

    # PAM config for vsftpd virtual users
    cat > /etc/pam.d/vsftpd <<'PAMEOF'
auth    required pam_userdb.so db=/etc/vsftpd/virtual_users
account required pam_userdb.so db=/etc/vsftpd/virtual_users
PAMEOF

    # vsftpd main config
    cat > /etc/vsftpd.conf <<VSFTPDEOF
listen=YES
listen_ipv6=NO
anonymous_enable=NO
local_enable=YES
write_enable=YES
local_umask=022
dirmessage_enable=YES
use_localtime=YES
xferlog_enable=YES
connect_from_port_20=YES
xferlog_file=/var/log/vsftpd.log
xferlog_std_format=YES
idle_session_timeout=600
data_connection_timeout=120
ftpd_banner=FTP Service Ready

# Virtual users
guest_enable=YES
guest_username=vsftpd_virtual
virtual_use_local_privs=YES
pam_service_name=vsftpd
user_config_dir=/etc/vsftpd/users

# Passive mode — adjust PASV_ADDRESS to your server's public IP
pasv_enable=NO
# pasv_address=YOUR_PUBLIC_IP

# Chroot virtual users to their home dir
chroot_local_user=YES
allow_writeable_chroot=YES

# SSL/TLS (optional — enable after placing certs)
# ssl_enable=YES
# rsa_cert_file=/etc/ssl/certs/vsftpd.pem
# rsa_private_key_file=/etc/ssl/private/vsftpd.key
VSFTPDEOF

    systemctl enable vsftpd --quiet
    systemctl restart vsftpd || log_warn "vsftpd start failed"

    # fail2ban jail for vsftpd brute-force protection
    mkdir -p /etc/fail2ban/jail.d
    # Create log file so fail2ban doesn't fail with "no log file found"
    touch /var/log/vsftpd.log
    chmod 640 /var/log/vsftpd.log
    cat > /etc/fail2ban/jail.d/zenspanel-vsftpd.conf <<'EOF'
[vsftpd]
enabled      = true
port         = ftp,ftp-data,ftps,ftps-data
filter       = vsftpd
logpath      = /var/log/vsftpd.log
maxretry     = 5
bantime      = 3600
findtime     = 600
allowmissing = true
EOF
    systemctl reload fail2ban 2>/dev/null || systemctl restart fail2ban 2>/dev/null || true

    log_info "vsftpd installed ✓ (FTP on port 21, fail2ban enabled)"
}

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
        log_info "Installing Node.js 22 LTS..."
        curl -fsSL https://deb.nodesource.com/setup_22.x | bash - > /dev/null 2>&1
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
    log_section "Installing ZensPanel"

    mkdir -p "$ZENSPANEL_DIR"/{bin,frontend,src}
    mkdir -p "$ZENSPANEL_DATA"/{home,backups}
    mkdir -p "$ZENSPANEL_LOG"
    mkdir -p "$ZENSPANEL_CONF"
    mkdir -p /etc/nginx/zenspanel
    mkdir -p /etc/nginx/ssl/zenspanel

    # Create empty admin allowlist (allow all by default)
    if [[ ! -f /etc/nginx/zenspanel/admin-allowlist.conf ]]; then
        echo "# Managed by ZensPanel — admin IP allowlist" > /etc/nginx/zenspanel/admin-allowlist.conf
        echo "# No restrictions — all IPs allowed" >> /etc/nginx/zenspanel/admin-allowlist.conf
    fi

    getent group zenspanel >/dev/null || groupadd -r zenspanel

    if id zenspanel &>/dev/null; then
        chown -R zenspanel:zenspanel "$ZENSPANEL_DATA"
        chmod 751 "$ZENSPANEL_DATA/home"
        chmod 750 "$ZENSPANEL_DATA/backups"
    fi

    # Create home dir for the zenspanel system user so the admin terminal works.
    mkdir -p "$ZENSPANEL_DATA/home/zenspanel"
    chown zenspanel:zenspanel "$ZENSPANEL_DATA/home/zenspanel" 2>/dev/null || true
    chmod 0711 "$ZENSPANEL_DATA/home/zenspanel"

    # Try to download the latest pre-built release from GitHub.
    # Falls back to build-from-source only if no release is available
    # (e.g. running from a dev branch with no tag yet).
    local release_url
    release_url=$(curl -fsSL "https://api.github.com/repos/teknik-github/zenspanel/releases/latest" \
        2>/dev/null | grep '"browser_download_url"' | grep '\.tar\.gz"' | head -1 | cut -d'"' -f4)

    if [[ -n "$release_url" ]]; then
        log_info "Downloading latest release..."
        local tarball="/tmp/zenspanel-release.tar.gz"
        curl -fsSL "$release_url" -o "$tarball" || die "Failed to download release"

        log_info "Extracting release..."
        local tmpdir="/tmp/zenspanel-extract"
        rm -rf "$tmpdir"
        mkdir -p "$tmpdir"
        tar -xzf "$tarball" -C "$tmpdir" || die "Failed to extract release"
        rm -f "$tarball"

        # Stop services before replacing binaries — Linux refuses to
        # overwrite a running executable ("Text file busy").
        systemctl stop zenspanel-api zenspanel-agent zenspanel-dashboard 2>/dev/null || true

        # Bundle layout: zenspanel/bin/, zenspanel/frontend/, zenspanel/migrations/
        local bundle="$tmpdir/zenspanel"
        install -m 0755 "$bundle/bin/zenspanel-api"   "$ZENSPANEL_DIR/bin/zenspanel-api"   2>/dev/null || \
            cp "$bundle/bin/zenspanel-api"   "$ZENSPANEL_DIR/bin/zenspanel-api"
        install -m 0755 "$bundle/bin/zenspanel-agent" "$ZENSPANEL_DIR/bin/zenspanel-agent" 2>/dev/null || \
            cp "$bundle/bin/zenspanel-agent" "$ZENSPANEL_DIR/bin/zenspanel-agent"
        install -m 0755 "$bundle/bin/zenspanel-cli"   "$ZENSPANEL_DIR/bin/zenspanel-cli"   2>/dev/null || \
            cp "$bundle/bin/zenspanel-cli"   "$ZENSPANEL_DIR/bin/zenspanel-cli" 2>/dev/null || true
        chmod +x "$ZENSPANEL_DIR/bin/"*
        rm -rf "$ZENSPANEL_DIR/frontend/dashboard"
        cp -r "$bundle/frontend/dashboard" "$ZENSPANEL_DIR/frontend/dashboard"

        # Keep source for migrations + self-updater.
        local src="$ZENSPANEL_DIR/src"
        if [[ -d "$src/.git" ]]; then
            log_info "Updating source tree..."
            git -C "$src" pull --quiet || true
        else
            log_info "Cloning source tree (for migrations + updates)..."
            git clone --quiet "$ZENSPANEL_REPO" "$src" || \
                log_warn "Source clone failed — migrations will use bundled copy"
        fi
        # Copy bundled migrations as fallback if clone failed.
        if [[ -d "$bundle/migrations" ]] && [[ ! -d "$src/migrations" ]]; then
            cp -r "$bundle/migrations" "$src/migrations"
        fi

        rm -rf "$tmpdir"
        ln -sf "$ZENSPANEL_DIR/bin/zenspanel-cli" /usr/local/bin/zenspanel-cli
        log_info "ZensPanel installed from release ✓"
    else
        log_warn "No pre-built release found — building from source (requires Go + pnpm, may use 1-2 GB RAM)"
        _build_from_source
    fi
}

_build_from_source() {
    local src="$ZENSPANEL_DIR/src"
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

    log_info "Resolving Go dependencies..."
    (cd "$src" && /usr/local/go/bin/go mod tidy) || die "Failed to resolve Go dependencies"

    log_info "Building zenspanel-api..."
    (cd "$src" && /usr/local/go/bin/go build -o "$ZENSPANEL_DIR/bin/zenspanel-api" ./cmd/api) || \
        die "Failed to build zenspanel-api"

    log_info "Building zenspanel-agent..."
    (cd "$src" && /usr/local/go/bin/go build -o "$ZENSPANEL_DIR/bin/zenspanel-agent" ./cmd/agent) || \
        die "Failed to build zenspanel-agent"

    log_info "Building zenspanel-cli..."
    (cd "$src" && /usr/local/go/bin/go build -o "$ZENSPANEL_DIR/bin/zenspanel-cli" ./cmd/cli) || \
        die "Failed to build zenspanel-cli"

    ln -sf "$ZENSPANEL_DIR/bin/zenspanel-cli" /usr/local/bin/zenspanel-cli

    log_info "Building Dashboard (this may take a few minutes)..."
    (cd "$src/Dashboard" && \
        pnpm install --ignore-scripts 2>&1 | grep -v "^Progress" && \
        pnpm build 2>&1) || die "Failed to build Dashboard"

    rm -rf "$ZENSPANEL_DIR/frontend/dashboard"
    cp -r "$src/Dashboard/.output" "$ZENSPANEL_DIR/frontend/dashboard"

    log_info "ZensPanel built from source ✓"
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
  password: ""
  db: 0

jwt:
  secret: "${JWT_SECRET}"
  expiry: "24h"
  refresh_expiry: "720h"

agent:
  socket: "/run/zenspanel/agent.sock"
  socket_group: "zenspanel"
  mysql_admin_dsn: "root:${MYSQL_ROOT_PASS}@tcp(127.0.0.1:3306)/?parseTime=true&multiStatements=true"

paths:
  home_base: "${ZENSPANEL_DATA}/home"
  nginx_conf: "/etc/nginx/zenspanel"
  ssl_base: "/etc/nginx/ssl/zenspanel"
  backup_base: "${ZENSPANEL_DATA}/backups"
  php_pool_base: "/etc/php"
  src_dir: "${ZENSPANEL_DIR}/src"
  bin_dir: "${ZENSPANEL_DIR}/bin"
  frontend_dir: "${ZENSPANEL_DIR}/frontend"

letsencrypt:
  email: "${LE_EMAIL}"
  staging: false
  hook_secret: "$(openssl rand -hex 32)"
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

    # Create zenspanel system user + group. groupadd is explicit so the
    # config field agent.socket_group: "zenspanel" and the chown on
    # /run/zenspanel below have a guaranteed-existing group to point at,
    # rather than relying on useradd's implicit group creation behaviour
    # which differs between distros.
    getent group zenspanel >/dev/null || groupadd -r zenspanel
    id zenspanel &>/dev/null || useradd -r -g zenspanel -s /bin/false -d "$ZENSPANEL_DIR" zenspanel

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

    # zenspanel-dashboard (Nuxt SSR — panel UI)
    cat > /etc/systemd/system/zenspanel-dashboard.service <<EOF
[Unit]
Description=ZensPanel Dashboard (Nuxt SSR)
After=network.target zenspanel-api.service
Wants=zenspanel-api.service

[Service]
Type=simple
User=zenspanel
Group=zenspanel
Environment=PORT=3000
Environment=HOST=127.0.0.1
Environment=NUXT_BACKEND_URL=http://127.0.0.1:8080
ExecStart=/usr/bin/node ${ZENSPANEL_DIR}/frontend/dashboard/server/index.mjs
WorkingDirectory=${ZENSPANEL_DIR}/frontend/dashboard
Restart=always
RestartSec=5
StandardOutput=append:${ZENSPANEL_LOG}/dashboard.log
StandardError=append:${ZENSPANEL_LOG}/dashboard-error.log

[Install]
WantedBy=multi-user.target
EOF

    # FileBrowser — third-party Go binary. Installed once; the agent
    # spawns one per-user instance (zenspanel-fb-<username>.service)
    # running as the panel user so all created files are owned correctly.
    if [[ ! -x /usr/local/bin/filebrowser ]]; then
        log_info "Installing FileBrowser..."
        curl -fsSL https://raw.githubusercontent.com/filebrowser/get/master/get.sh | bash
    fi

    # Seed an empty Nginx port map so the map{} variable $fb_port is
    # defined before any panel user exists. The agent overwrites this
    # file on every user create/delete and reloads Nginx.
    mkdir -p /etc/nginx/zenspanel /etc/nginx/conf.d
    if [[ ! -f /etc/nginx/conf.d/zenspanel-fb-ports.conf ]]; then
        cat > /etc/nginx/conf.d/zenspanel-fb-ports.conf <<'MAPEOF'
# Auto-generated by ZensPanel agent — do not edit.
map $fb_user $fb_port {
    default 0;
}
MAPEOF
    fi
    chown -R zenspanel:zenspanel "$ZENSPANEL_LOG"
    chown -R zenspanel:zenspanel "$ZENSPANEL_DATA"
    chmod 751 "$ZENSPANEL_DATA/home"
    chmod 750 "$ZENSPANEL_DATA/backups"
    chown zenspanel:zenspanel "$ZENSPANEL_CONF/config.yaml"
    # Agent needs to read config too
    chmod 640 "$ZENSPANEL_CONF/config.yaml"
    chown root:zenspanel "$ZENSPANEL_CONF/config.yaml"

    systemctl daemon-reload
    systemctl enable zenspanel-agent zenspanel-api zenspanel-dashboard --quiet

    log_info "Starting zenspanel-agent..."
    systemctl start zenspanel-agent
    sleep 3

    log_info "Starting zenspanel-api..."
    systemctl start zenspanel-api
    sleep 3

    log_info "Starting zenspanel-dashboard..."
    systemctl start zenspanel-dashboard
    sleep 3

    # Per-user FileBrowser instances (zenspanel-fb-<username>.service)
    # are started by the agent when panel users are created — nothing
    # to start here at install time.

    # Verify services are running
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

    if systemctl is-active --quiet zenspanel-dashboard; then
        log_info "zenspanel-dashboard: running ✓"
    else
        log_warn "zenspanel-dashboard failed to start. Check: journalctl -u zenspanel-dashboard"
    fi
}

# =============================================================================
# STEP 12: Setup Nginx reverse proxy
# =============================================================================
setup_nginx() {
    log_section "Setting up Nginx"

    # "Sorry!" catch-all on port 80 — cPanel-style. Triggers in two
    # cases: (1) someone visits the panel hostname on :80 (panel
    # itself runs on $PANEL_PORT, not :80), (2) someone resolves a
    # random hostname or raw IP to this server before any user vhost
    # claims it. Each user vhost added later via the panel listens on
    # :80 too with its own server_name, so this default_server only
    # catches the leftovers.
    cat > /etc/nginx/sites-available/zenspanel-sorry <<'EOF'
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;

    access_log off;
    error_log /dev/null;

    location / {
        return 200 '<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Sorry!</title>
<style>
  body{font-family:system-ui,-apple-system,Segoe UI,Roboto,sans-serif;
    background:#f9fafb;color:#374151;display:flex;align-items:center;
    justify-content:center;min-height:100vh;margin:0;padding:1rem}
  .box{text-align:center;max-width:480px}
  .ico{width:72px;height:72px;border-radius:50%;background:#fef3c7;
    color:#d97706;display:inline-flex;align-items:center;justify-content:center;
    margin-bottom:1rem;font-size:2.25rem;font-weight:700}
  h1{font-size:1.5rem;font-weight:700;margin:.25rem 0 .75rem;color:#111827}
  p{color:#6b7280;line-height:1.6;font-size:.95rem;margin:.5rem 0}
  .hint{margin-top:1.5rem;font-size:.75rem;color:#9ca3af}
</style></head>
<body><div class="box">
<div class="ico">!</div>
<h1>Sorry!</h1>
<p>The website you are looking for is not currently available.</p>
<p>This may be because the domain has not been pointed to this server yet, or because the site is still being set up.</p>
<div class="hint">Powered by ZensPanel</div>
</div></body></html>';
        add_header Content-Type "text/html; charset=utf-8";
    }
}
EOF

    cat > /etc/nginx/sites-available/zenspanel <<EOF
server {
    listen ${PANEL_PORT};
    server_name ${PANEL_HOST};

    client_max_body_size 100M;

    # Hide nginx version for security.
    server_tokens off;

    # Security headers
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Admin routes — IP allowlist enforced before proxying to Nuxt dashboard
    location ^~ /admin {
        include /etc/nginx/zenspanel/admin-allowlist.conf;
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_read_timeout 300;
        proxy_connect_timeout 10;
    }

    # WebSocket terminal — must stay direct (Nuxt cannot proxy WS upgrades)
    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Forwarded-Host \$host;
        proxy_read_timeout 3600;
    }

    # FileBrowser auth bridge — internal endpoint nginx hits to learn
    # which panel user is on the other end of the request. The handler
    # validates the JWT cookie set at login and replies with
    # X-Auth-User; we forward that header to FileBrowser, which is
    # configured for proxy auth.
    location = /api/v1/auth/filebrowser {
        internal;
        proxy_pass http://127.0.0.1:8080;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
        proxy_set_header X-Original-URI \$request_uri;
        proxy_set_header Cookie \$http_cookie;
    }

    # FileBrowser — full-featured file manager iframed into the User
    # Panel. auth_request gates access; without a valid panel session
    # the request never reaches the FileBrowser process.
    #
    # Each panel user has a dedicated FileBrowser instance running as
    # that user (zenspanel-fb-<username>.service). The agent maintains
    # /etc/nginx/conf.d/zenspanel-fb-ports.conf which maps \$fb_user
    # to the correct per-user listen port (\$fb_port). Because the
    # instance runs as the panel user, every created file is owned by
    # that user immediately — no chown timer or root-owned files.
    location /filebrowser/ {
        auth_request /api/v1/auth/filebrowser;
        auth_request_set \$fb_user \$upstream_http_x_auth_user;
        proxy_pass http://127.0.0.1:\$fb_port;
        proxy_set_header X-Auth-User \$fb_user;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        client_max_body_size 1024M;
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

    # Dashboard (Nuxt SSR) — all other traffic
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_read_timeout 300;
        proxy_connect_timeout 10;
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
    ln -sf /etc/nginx/sites-available/zenspanel-sorry /etc/nginx/sites-enabled/zenspanel-sorry
    rm -f /etc/nginx/sites-enabled/default

    if nginx -t 2>&1 | tee -a "${ZENSPANEL_LOG}/install.log"; then
        systemctl reload nginx
    else
        log_warn "nginx config test failed — details above and in ${ZENSPANEL_LOG}/install.log"
        log_warn "Fix any issues and run: nginx -t && systemctl reload nginx"
    fi
    systemctl enable nginx --quiet

    log_info "Nginx configured ✓"
}

# =============================================================================
# STEP 13: Setup firewall
# =============================================================================
setup_firewall() {
    log_section "Configuring firewall"

    ufw --force reset           > /dev/null 2>&1 || true
    ufw default deny incoming   > /dev/null 2>&1 || true
    ufw default allow outgoing  > /dev/null 2>&1 || true
    ufw allow ssh               > /dev/null 2>&1 || true
    ufw allow 80/tcp            > /dev/null 2>&1 || true
    ufw allow 443/tcp           > /dev/null 2>&1 || true
    ufw allow "${PANEL_PORT}/tcp" > /dev/null 2>&1 || true
    ufw allow 21/tcp            > /dev/null 2>&1 || true
    ufw --force enable          > /dev/null 2>&1 || log_warn "ufw enable failed — configure firewall manually"

    log_info "Firewall configured ✓ (SSH, HTTP, HTTPS, panel, FTP)"
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
# STEP 14b: Setup filesystem disk quota
# =============================================================================
# Linux quota subsystem enforces a hard limit at the kernel level — once a
# user hits package.disk_quota bytes, every write returns EDQUOT. The panel's
# du-based metric is just for display; this is the actual fence.
#
# Setup is per-filesystem: the device that homeBase lives on must be
# mounted with usrquota, and (for ext4) initialised with quotacheck +
# quotaon. XFS has quota built into the filesystem, so quotacheck is a
# no-op there. Idempotent — safe to re-run.
setup_quota() {
    log_section "Setting up filesystem disk quota"

    # Resolve the filesystem mount point that holds homeBase. Walk up
    # until findmnt recognises the path as a mountpoint — homeBase
    # itself almost certainly isn't one.
    local home_base="${ZENSPANEL_DATA}/home"
    local fs_mount
    fs_mount=$(df --output=target "$home_base" | tail -n1)
    local fs_type
    fs_type=$(findmnt -n -o FSTYPE "$fs_mount")
    local fs_dev
    fs_dev=$(findmnt -n -o SOURCE "$fs_mount")

    log_info "homeBase=$home_base → mount=$fs_mount ($fs_type on $fs_dev)"

    case "$fs_type" in
        ext4|ext3|ext2)
            local mount_opt="usrquota"
            ;;
        xfs)
            # XFS quota is enabled at mount time; remount-with-option
            # only works at boot. We add to fstab so reboots take
            # effect; the running mount stays as-is and quotas activate
            # on next boot. setquota still works on running XFS that
            # was mounted with uquota or pquota.
            local mount_opt="uquota"
            ;;
        *)
            log_warn "Filesystem $fs_type on $fs_mount — disk quota enforcement may not be supported. Skipping."
            return 0
            ;;
    esac

    # Add usrquota (or uquota for XFS) to the fstab entry for this
    # mountpoint, only if not already present. We anchor on the device
    # column so we don't accidentally rewrite some other entry.
    if grep -E "^\s*${fs_dev//\//\\/}\s" /etc/fstab | grep -q "$mount_opt"; then
        log_info "fstab already has $mount_opt for $fs_dev ✓"
    else
        log_info "Adding $mount_opt to fstab for $fs_dev"
        # Backup before mutating
        cp /etc/fstab "/etc/fstab.bak.$(date +%s)"
        # Append ,$mount_opt to the options column (4th field) of the
        # matching device line. awk lets us touch only that line.
        awk -v dev="$fs_dev" -v opt="$mount_opt" '
            $1 == dev {
                if ($4 == "defaults") { $4 = "defaults,"opt }
                else if (index($4, opt) == 0) { $4 = $4","opt }
                print; next
            }
            { print }
        ' /etc/fstab > /etc/fstab.new && mv /etc/fstab.new /etc/fstab
    fi

    # ext4: remount live + quotacheck + quotaon. quotacheck needs
    # exclusive read on the filesystem, so on root mounts it warns
    # but still produces aquota.user; that's why the -m (mounted) flag
    # is there. Failures are surfaced to the operator (not silenced)
    # because a quota that's "on but not really" leads to confusing
    # 500s when create-user calls setquota later.
    if [[ "$fs_type" =~ ^ext ]]; then
        # Make sure the kernel module is loaded. On stock Ubuntu/Debian
        # quota_v2 is built-in but on minimal images it can be a module.
        modprobe quota_v2 2>/dev/null || true

        log_info "Remounting $fs_mount with usrquota..."
        if ! mount -o "remount,$mount_opt" "$fs_mount"; then
            log_warn "Live remount failed. Quota will activate after reboot once /etc/fstab is honoured."
            log_warn "Skipping quotacheck/quotaon — they would fail without an active quota mount."
            return 0
        fi

        # Verify the remount actually picked up the option. On some
        # kernels remount silently ignores quota flags on the rootfs
        # and only a real reboot enables them. Check before quotacheck.
        if ! mount | grep -q "on $fs_mount type $fs_type.*$mount_opt"; then
            log_warn "Mount $fs_mount is not advertising $mount_opt yet. A reboot is required to activate the quota subsystem on this filesystem."
            log_warn "After reboot, re-run: quotacheck -cum $fs_mount && quotaon -u $fs_mount"
            return 0
        fi

        log_info "Running quotacheck (this may take a moment)..."
        if ! quotacheck -cum "$fs_mount" 2>&1 | tee -a "${ZENSPANEL_LOG}/install.log"; then
            log_warn "quotacheck reported issues — see ${ZENSPANEL_LOG}/install.log"
        fi

        log_info "Enabling quota..."
        if quotaon -u "$fs_mount" 2>&1; then
            log_info "Quota enabled on $fs_mount ✓"
        else
            log_warn "quotaon failed. Run 'quotaon -u $fs_mount' manually after reboot."
        fi
    fi

    log_info "Disk quota setup complete ✓"
}

# =============================================================================
# STEP 15: Setup log rotation
# =============================================================================
setup_certbot_hook() {
    log_section "Certbot deploy hook"
    HOOK_DIR="/etc/letsencrypt/renewal-hooks/deploy"
    HOOK_SCRIPT="${HOOK_DIR}/zenspanel-update.sh"
    mkdir -p "$HOOK_DIR"
    cp "${SCRIPT_DIR}/certbot-deploy-hook.sh" "$HOOK_SCRIPT"
    chmod +x "$HOOK_SCRIPT"
    log_info "Certbot deploy hook installed ✓ (auto-updates ssl_expires_at after renewal)"
}

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
    echo -e "  zenspanel-agent:     ${agent_status}"
    echo -e "  zenspanel-api:       ${api_status}"
    local dashboard_status
    dashboard_status=$(systemctl is-active zenspanel-dashboard 2>/dev/null || echo "unknown")
    echo -e "  zenspanel-dashboard: ${dashboard_status}"
    echo ""
    echo -e "${BOLD}Useful Commands:${NC}"
    echo -e "  systemctl status zenspanel-api"
    echo -e "  systemctl status zenspanel-agent"
    echo -e "  systemctl status zenspanel-dashboard"
    echo -e "  tail -f ${ZENSPANEL_LOG}/api.log"
    echo -e "  tail -f ${ZENSPANEL_LOG}/agent.log"
    echo -e "  tail -f ${ZENSPANEL_LOG}/dashboard.log"
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
    install_firewall
    install_clamav
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
    setup_quota
    setup_vsftpd
    setup_logrotate
    setup_certbot_hook
    save_install_info
    print_summary
}

main "$@"
