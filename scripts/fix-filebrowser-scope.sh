#!/bin/bash
# fix-filebrowser-scope.sh
#
# Fix existing FileBrowser users to use relative scopes.
# Background: scope must be relative to FileBrowser's Root config
# (which is /var/lib/zenspanel/home). An absolute scope like
# "/var/lib/zenspanel/home/rundisk" gets joined with Root and
# resolves to a non-existent path, leaving the file list empty.
#
# Run as root on the server. Idempotent — safe to run multiple times.

set -euo pipefail

DB="/var/lib/zenspanel/filebrowser.db"
HOME_BASE="/var/lib/zenspanel/home"
SERVICE="zenspanel-filebrowser"

if [[ ! -f "$DB" ]]; then
    echo "ERROR: FileBrowser DB not found at $DB" >&2
    exit 1
fi

if [[ ! -x /usr/local/bin/filebrowser ]]; then
    echo "ERROR: filebrowser binary not found at /usr/local/bin/filebrowser" >&2
    exit 1
fi

echo "==> Stopping $SERVICE (CLI needs exclusive DB lock)..."
systemctl stop "$SERVICE" 2>/dev/null || true
sleep 1

# Make sure the auth method is proxy and the header is X-Auth-User.
# These persist in the DB so re-running is fine.
echo "==> Setting auth.method=proxy + auth.header=X-Auth-User..."
filebrowser --database "$DB" config set \
    --auth.method=proxy \
    --auth.header=X-Auth-User >/dev/null

# Walk every Linux user under HOME_BASE; if they have a matching
# FileBrowser record, rewrite scope to the relative form (just the
# username). Users that don't exist yet are added.
echo "==> Syncing FileBrowser users with $HOME_BASE..."
for d in "$HOME_BASE"/*; do
    [[ -d "$d" ]] || continue
    u=$(basename "$d")

    # Skip system users that shouldn't appear in FileBrowser.
    case "$u" in
        lost+found|.*) continue ;;
    esac

    # Does the FileBrowser user already exist?
    if filebrowser --database "$DB" users ls 2>/dev/null | awk '{print $2}' | grep -Fxq "$u"; then
        echo "    update $u → scope=$u"
        filebrowser --database "$DB" users update "$u" --scope "$u" >/dev/null
    else
        echo "    add    $u → scope=$u"
        filebrowser --database "$DB" users add "$u" "${u}-no-login" \
            --scope "$u" \
            --perm.admin=false >/dev/null
    fi
done

# Make sure there's an admin user so the agent's HTTP API calls work.
# scope=/ keeps admin able to manage all users; --perm.admin=true is
# the elevation flag required for /api/users mutations.
if ! filebrowser --database "$DB" users ls 2>/dev/null | awk '{print $2}' | grep -Fxq "admin"; then
    echo "==> Adding admin user (for agent API access)..."
    filebrowser --database "$DB" users add admin admin \
        --perm.admin=true \
        --scope / >/dev/null
fi

echo "==> Resulting users:"
filebrowser --database "$DB" users ls

echo "==> Starting $SERVICE..."
systemctl start "$SERVICE"
sleep 2

systemctl is-active --quiet "$SERVICE" \
    && echo "==> $SERVICE is active." \
    || { echo "ERROR: $SERVICE failed to start"; systemctl status "$SERVICE" --no-pager; exit 1; }

echo
echo "Done. Refresh the File Manager tab in your browser to verify."
