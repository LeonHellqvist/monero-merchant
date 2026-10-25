#!/bin/sh
set -eu

case "${1:-}" in
    --version|--help)
        exec monero-wallet-rpc "$@"
        ;;
esac

for arg in "$@"; do
    case "$arg" in
        --disable-rpc-login|--rpc-login|--rpc-login=*)
            echo "Do not pass ${arg%%=*}; set MONERO_WALLET_RPC_USERNAME and MONERO_WALLET_RPC_PASSWORD instead" >&2
            exit 1
            ;;
    esac
done

if [ -z "${MONERO_WALLET_RPC_USERNAME:-}" ] || [ -z "${MONERO_WALLET_RPC_PASSWORD:-}" ]; then
    echo "MONERO_WALLET_RPC_USERNAME and MONERO_WALLET_RPC_PASSWORD must be set" >&2
    exit 1
fi

set -- "$@" "--rpc-login=${MONERO_WALLET_RPC_USERNAME}:${MONERO_WALLET_RPC_PASSWORD}"

if [ -n "${MONERO_DAEMON_RPC_USERNAME:-}" ]; then
    set -- "$@" "--daemon-login=${MONERO_DAEMON_RPC_USERNAME}:${MONERO_DAEMON_RPC_PASSWORD:-}"
fi

exec monero-wallet-rpc "$@"
