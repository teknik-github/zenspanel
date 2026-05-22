#!/usr/bin/env bash
# setup.sh — idempotent dependency setup for ZensPanel.
# Run after every update to install new dependencies and apply new configs.
# Safe to run multiple times — each step checks before acting.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── helpers ──────────────────────────────────────────────────────────────────

log_info()  { echo "[INFO]  $*"; }
log_warn()  { echo "[WARN]  $*" >&2; }
log_section() { echo ""; echo "══ $* ══"; }

export DEBIAN_FRONTEND=noninteractive

# ── base packages ─────────────────────────────────────────────────────────────

setup_base_packages() {
    log_section "Base packages"
    apt-get update -qq
    apt-get install -y -qq \
        curl wget git unzip tar \
        build-essential \
        software-properties-common \
        apt-transport-https \
        ca-certificates \
        gnupg lsb-release \
        ufw quota acl openssl jq bc \
        iproute2 inotify-tools rclone \
        db-util 2>/dev/null || true
    log_info "Base packages OK"
}

# ── nginx ─────────────────────────────────────────────────────────────────────

setup_nginx() {
    log_section "nginx"
    if ! command -v nginx &>/dev/null; then
        apt-get install -y -qq nginx
        log_info "nginx installed"
    else
        log_info "nginx already installed ✓"
    fi
    systemctl enable nginx --quiet
    systemctl start nginx 2>/dev/null || true
}

# ── mysql ─────────────────────────────────────────────────────────────────────

setup_mysql() {
    log_section "MySQL"
    if ! command -v mysql &>/dev/null; then
        apt-get install -y -qq mysql-server
        log_info "MySQL installed"
    else
        log_info "MySQL already installed ✓"
    fi
    systemctl enable mysql --quiet
    systemctl start mysql 2>/dev/null || true
}

# ── redis ─────────────────────────────────────────────────────────────────────

setup_redis() {
    log_section "Redis"
    if ! command -v redis-server &>/dev/null; then
        apt-get install -y -qq redis-server
        log_info "Redis installed"
    else
        log_info "Redis already installed ✓"
    fi
    systemctl enable redis-server --quiet
    systemctl start redis-server 2>/dev/null || true
}

# ── php ───────────────────────────────────────────────────────────────────────

setup_php() {
    log_section "PHP"
    # Add ondrej/php PPA if not already present
    if ! grep -r "ondrej/php" /etc/apt/sources.list /etc/apt/sources.list.d/ &>/dev/null; then
        add-apt-repository -y ppa:ondrej/php > /dev/null 2>&1
        apt-get update -qq
        log_info "ondrej/php PPA added"
    else
        log_info "ondrej/php PPA already present ✓"
    fi
    for ver in 8.3 8.2 8.1; do
        if ! command -v php${ver} &>/dev/null; then
            apt-get install -y -qq \
                php${ver}-fpm php${ver}-cli php${ver}-mysql \
                php${ver}-curl php${ver}-gd php${ver}-mbstring \
                php${ver}-xml php${ver}-zip php${ver}-bcmath \
                php${ver}-intl 2>/dev/null || log_warn "Some PHP ${ver} extensions unavailable"
            log_info "PHP ${ver} installed"
        else
            log_info "PHP ${ver} already installed ✓"
        fi
    done
}

# ── certbot ───────────────────────────────────────────────────────────────────

setup_certbot() {
    log_section "Certbot"
    if ! command -v certbot &>/dev/null; then
        apt-get install -y -qq certbot python3-certbot-nginx
        log_info "Certbot installed"
    else
        log_info "Certbot already installed ✓"
    fi
}

# ── phpmyadmin ────────────────────────────────────────────────────────────────

setup_phpmyadmin() {
    log_section "phpMyAdmin"
    if ! dpkg -l phpmyadmin &>/dev/null 2>&1; then
        apt-get install -y -qq phpmyadmin || log_warn "phpMyAdmin install had issues, continuing"
        log_info "phpMyAdmin installed"
    else
        log_info "phpMyAdmin already installed ✓"
    fi

    # Ensure SSO config exists (idempotent write)
    PMA_CONF="/etc/phpmyadmin/conf.d/zenspanel-sso.php"
    if [[ ! -f "$PMA_CONF" ]]; then
        mkdir -p /etc/phpmyadmin/conf.d
        cat > "$PMA_CONF" <<'PHPEOF'
<?php
$cfg['Servers'][1]['auth_type'] = 'signon';
$cfg['Servers'][1]['SignonSession'] = 'SignonSession';
$cfg['Servers'][1]['SignonURL'] = '/phpmyadmin/signon.php';
$cfg['Servers'][1]['LogoutURL'] = '/phpmyadmin/signon.php?logout=1';
PHPEOF
        log_info "phpMyAdmin SSO config written"
    fi
}

# ── clamav ────────────────────────────────────────────────────────────────────

setup_clamav() {
    log_section "ClamAV"
    if ! command -v clamscan &>/dev/null; then
        apt-get install -y -qq clamav clamav-daemon || {
            add-apt-repository -y universe 2>/dev/null || true
            apt-get update -qq
            apt-get install -y -qq clamav clamav-daemon || {
                log_warn "ClamAV install failed, skipping"
                return 0
            }
        }
        log_info "ClamAV installed"
    else
        log_info "ClamAV already installed ✓"
    fi
    systemctl enable clamav-freshclam --quiet 2>/dev/null || true
    systemctl start clamav-freshclam 2>/dev/null || true
    systemctl enable clamav-daemon --quiet 2>/dev/null || true
    systemctl start clamav-daemon 2>/dev/null || true
}

# ── fail2ban + firewall ───────────────────────────────────────────────────────

setup_firewall() {
    log_section "Firewall / fail2ban"
    if ! command -v fail2ban-client &>/dev/null; then
        apt-get install -y -qq fail2ban ipset iptables-persistent
        log_info "fail2ban installed"
    else
        log_info "fail2ban already installed ✓"
    fi
    systemctl enable fail2ban --quiet 2>/dev/null || true
    systemctl start fail2ban 2>/dev/null || true
}

# ── vsftpd ────────────────────────────────────────────────────────────────────

setup_vsftpd() {
    log_section "vsftpd (FTP)"
    if ! command -v vsftpd &>/dev/null; then
        apt-get install -y -qq vsftpd db-util libpam-pwdfile || {
            log_warn "vsftpd install failed, skipping FTP support"
            return 0
        }
        log_info "vsftpd installed"
    else
        log_info "vsftpd already installed ✓"
    fi

    # Virtual user system account
    if ! id vsftpd_virtual &>/dev/null; then
        useradd -r -d /dev/null -s /sbin/nologin vsftpd_virtual || true
        log_info "vsftpd_virtual user created"
    fi

    # Config dirs + PAM DB
    mkdir -p /etc/vsftpd/users
    if [[ ! -f /etc/vsftpd/virtual_users.txt ]]; then
        touch /etc/vsftpd/virtual_users.txt
        chmod 600 /etc/vsftpd/virtual_users.txt
    fi
    if [[ ! -f /etc/vsftpd/virtual_users.db ]]; then
        db_load -T -t hash -f /etc/vsftpd/virtual_users.txt /etc/vsftpd/virtual_users.db 2>/dev/null || true
        chmod 600 /etc/vsftpd/virtual_users.db 2>/dev/null || true
        log_info "vsftpd PAM DB initialised"
    fi

    # PAM config
    if [[ ! -f /etc/pam.d/vsftpd ]] || ! grep -q "pam_userdb" /etc/pam.d/vsftpd; then
        cat > /etc/pam.d/vsftpd <<'PAMEOF'
auth    required pam_userdb.so db=/etc/vsftpd/virtual_users
account required pam_userdb.so db=/etc/vsftpd/virtual_users
PAMEOF
        log_info "vsftpd PAM config written"
    fi

    # vsftpd.conf — only write if not already configured for virtual users
    if [[ ! -f /etc/vsftpd.conf ]] || ! grep -q "guest_enable=YES" /etc/vsftpd.conf; then
        cat > /etc/vsftpd.conf <<'VSFTPDEOF'
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
guest_enable=YES
guest_username=vsftpd_virtual
virtual_use_local_privs=YES
pam_service_name=vsftpd
user_config_dir=/etc/vsftpd/users
pasv_enable=NO
chroot_local_user=YES
allow_writeable_chroot=YES
VSFTPDEOF
        log_info "vsftpd.conf written"
    fi

    systemctl enable vsftpd --quiet
    systemctl restart vsftpd || log_warn "vsftpd restart failed"

    # fail2ban jail for vsftpd brute-force protection
    JAIL_CONF="/etc/fail2ban/jail.d/zenspanel-vsftpd.conf"
    # Create log file so fail2ban doesn't fail with "no log file found"
    touch /var/log/vsftpd.log 2>/dev/null || true
    chmod 640 /var/log/vsftpd.log 2>/dev/null || true
    if [[ ! -f "$JAIL_CONF" ]]; then
        cat > "$JAIL_CONF" <<'EOF'
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
        log_info "fail2ban vsftpd jail enabled (5 attempts / 10 min → 1h ban)"
    else
        log_info "fail2ban vsftpd jail already configured ✓"
    fi
}

# ── quota ─────────────────────────────────────────────────────────────────────

setup_quota() {
    log_section "Disk quota"
    if ! command -v setquota &>/dev/null; then
        apt-get install -y -qq quota
        log_info "quota tools installed"
    else
        log_info "quota tools already installed ✓"
    fi

    ROOT_FS=$(findmnt -n -o FSTYPE / 2>/dev/null || echo "unknown")
    if ! grep -q "usrquota\|uquota" /etc/fstab 2>/dev/null; then
        if [[ "$ROOT_FS" == "ext4" ]]; then
            sed -i 's|\(.*\s/\s.*defaults\)|\1,usrquota|' /etc/fstab || true
            mount -o remount,usrquota / 2>/dev/null || true
            quotacheck -cum / 2>/dev/null || true
            quotaon / 2>/dev/null || true
            log_info "ext4 quota enabled"
        elif [[ "$ROOT_FS" == "xfs" ]]; then
            sed -i 's|\(.*\s/\s.*defaults\)|\1,uquota|' /etc/fstab || true
            mount -o remount,uquota / 2>/dev/null || true
            log_info "XFS quota enabled"
        else
            log_warn "Unsupported filesystem ${ROOT_FS} — quota skipped"
        fi
    else
        log_info "quota already enabled in fstab ✓"
    fi
}

# ── cgroups v2 ────────────────────────────────────────────────────────────────

setup_cgroups() {
    log_section "cgroups v2"
    CGROUP_BASE="/sys/fs/cgroup/zenspanel"
    if [[ ! -d "$CGROUP_BASE" ]]; then
        mkdir -p "$CGROUP_BASE"
        log_info "zenspanel cgroup slice created"
    else
        log_info "cgroup slice already exists ✓"
    fi
    # Enable cpu+memory controllers if not already delegated
    for ctrl in cpu memory; do
        if ! grep -q "$ctrl" /sys/fs/cgroup/cgroup.subtree_control 2>/dev/null; then
            echo "+${ctrl}" > /sys/fs/cgroup/cgroup.subtree_control 2>/dev/null || true
        fi
    done
}

# ── composer ──────────────────────────────────────────────────────────────────

setup_composer() {
    log_section "Composer"
    if [[ ! -f /usr/local/bin/composer.phar ]]; then
        curl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer.phar
        log_info "Composer installed"
    else
        log_info "Composer already installed ✓"
    fi
}

# ── logrotate ─────────────────────────────────────────────────────────────────

setup_logrotate() {
    log_section "logrotate"
    LOGROTATE_CONF="/etc/logrotate.d/zenspanel"
    if [[ ! -f "$LOGROTATE_CONF" ]]; then
        cat > "$LOGROTATE_CONF" <<'EOF'
/var/log/zenspanel/*.log {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    create 0640 zenspanel zenspanel
}
EOF
        log_info "logrotate config written"
    else
        log_info "logrotate already configured ✓"
    fi
}

# ── certbot deploy hook ───────────────────────────────────────────────────────

setup_certbot_hook() {
    log_section "Certbot deploy hook"
    HOOK_DIR="/etc/letsencrypt/renewal-hooks/deploy"
    HOOK_SCRIPT="${HOOK_DIR}/zenspanel-update.sh"
    # Find the source script relative to this setup.sh
    SRC="$(dirname "${BASH_SOURCE[0]}")/certbot-deploy-hook.sh"
    if [[ ! -f "$SRC" ]]; then
        log_warn "certbot-deploy-hook.sh not found at $SRC, skipping"
        return 0
    fi
    mkdir -p "$HOOK_DIR"
    cp "$SRC" "$HOOK_SCRIPT"
    chmod +x "$HOOK_SCRIPT"
    log_info "Certbot deploy hook installed ✓"
}

# ── main ──────────────────────────────────────────────────────────────────────

main() {
    log_section "ZensPanel dependency setup"
    setup_base_packages
    setup_nginx
    setup_mysql
    setup_redis
    setup_php
    setup_certbot
    setup_phpmyadmin
    setup_clamav
    setup_firewall
    setup_vsftpd
    setup_quota
    setup_cgroups
    setup_composer
    setup_logrotate
    setup_certbot_hook
    echo ""
    log_info "All dependencies OK ✓"
}

main "$@"
