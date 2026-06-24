#!/bin/sh
set -e

# If SECRETS_FILE is set, generate secrets on first boot and source them.
if [ -n "${SECRETS_FILE}" ]; then
    SECRETS_DIR="$(dirname "${SECRETS_FILE}")"
    mkdir -p "${SECRETS_DIR}"

    # Generate secrets file if it does not exist
    if [ ! -f "${SECRETS_FILE}" ]; then
        echo "Generating secrets file at ${SECRETS_FILE} ..."
        cat > "${SECRETS_FILE}" <<EOF
# Auto-generated secrets for Monero Merchant
# Generated on first boot. Do not delete.
DB_PASSWORD=$(openssl rand -hex 16)
JWT_SECRET=$(openssl rand -hex 32)
JWT_REFRESH_SECRET=$(openssl rand -hex 32)
JWT_MONEROPAY_SECRET=$(openssl rand -hex 32)
JWT_LWS_TOKEN=$(openssl rand -hex 32)
WALLET_PASSWORD=$(openssl rand -hex 16)
EOF
        chmod 600 "${SECRETS_FILE}"
    fi

    # Source secrets into the environment (only if not already set)
    while IFS= read -r line || [ -n "$line" ]; do
        # skip comments and empty lines
        case "$line" in
            ""|\#*) continue ;;
        esac
        key="$(printf '%s' "$line" | cut -d '=' -f 1)"
        val="$(printf '%s' "$line" | cut -d '=' -f 2-)"
        # Only export if the env var is currently empty
        eval "current_value=\"\${$key:-}\""
        if [ -z "$current_value" ]; then
            export "$key=$val"
        fi
    done < "${SECRETS_FILE}"
fi

exec ./backend
