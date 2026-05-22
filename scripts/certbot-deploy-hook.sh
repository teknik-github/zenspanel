#!/bin/bash
# ZensPanel certbot deploy hook.
# Called by certbot after every successful renewal.
# Updates ssl_expires_at in the panel DB via the internal API.
#
# Installed at: /etc/letsencrypt/renewal-hooks/deploy/zenspanel-update.sh
# Config read from: /etc/zenspanel/config.yaml

set -euo pipefail

CONFIG="/etc/zenspanel/config.yaml"

# Read hook_secret from config.yaml
HOOK_SECRET=$(grep -E '^\s*hook_secret:' "$CONFIG" 2>/dev/null | awk '{print $2}' | tr -d '"' || true)
API_PORT=$(grep -E '^\s*port:' "$CONFIG" 2>/dev/null | head -1 | awk '{print $2}' || echo "8080")

if [[ -z "$HOOK_SECRET" ]]; then
    echo "zenspanel-update: hook_secret not set in $CONFIG, skipping DB update"
    exit 0
fi

# RENEWED_DOMAINS is set by certbot — space-separated list of renewed domains
for domain in $RENEWED_DOMAINS; do
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -X POST "http://127.0.0.1:${API_PORT}/api/v1/system/ssl-renewed" \
        -H "Content-Type: application/json" \
        -H "X-Hook-Secret: ${HOOK_SECRET}" \
        -d "{\"domain\": \"${domain}\"}" \
        --max-time 10 || true)
    if [[ "$response" == "200" ]]; then
        echo "zenspanel-update: updated ssl_expires_at for ${domain}"
    else
        echo "zenspanel-update: failed to update ${domain} (HTTP ${response})"
    fi
done
