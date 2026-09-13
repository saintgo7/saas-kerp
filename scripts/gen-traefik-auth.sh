#!/usr/bin/env bash
# Generate the Traefik basic-auth users file from the TRAEFIK_DASHBOARD_AUTH
# secret. Run on the deployment host before `docker compose up`.
#
#   TRAEFIK_DASHBOARD_AUTH must contain a full htpasswd line, e.g.
#     admin:$2y$05$....
#   Produce one with:  htpasswd -nbB admin "$(openssl rand -base64 24)"
#
# Exits non-zero when the secret is missing or is still the placeholder, so a
# deployment cannot silently fall back to an open or well-known dashboard.

set -euo pipefail

OUT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/deployments/docker/traefik/secrets"
OUT_FILE="${OUT_DIR}/dashboard-users.htpasswd"

: "${TRAEFIK_DASHBOARD_AUTH:?TRAEFIK_DASHBOARD_AUTH is not set}"

if [[ "$TRAEFIK_DASHBOARD_AUTH" == "change-me-in-production" ]]; then
    echo "ERROR: TRAEFIK_DASHBOARD_AUTH is still the placeholder value" >&2
    exit 1
fi

if [[ "$TRAEFIK_DASHBOARD_AUTH" != *:* ]]; then
    echo "ERROR: TRAEFIK_DASHBOARD_AUTH must be a htpasswd line (user:hash)" >&2
    exit 1
fi

case "$TRAEFIK_DASHBOARD_AUTH" in
    *':$apr1$'*|*':$2y$'*|*':$2a$'*|*':$2b$'*|*':{SHA}'*) ;;
    *)
        echo "ERROR: TRAEFIK_DASHBOARD_AUTH does not look like a hashed htpasswd entry" >&2
        exit 1
        ;;
esac

mkdir -p "$OUT_DIR"
umask 077
printf '%s\n' "$TRAEFIK_DASHBOARD_AUTH" > "$OUT_FILE"
chmod 600 "$OUT_FILE"
echo "Wrote ${OUT_FILE}"
