#!/bin/sh
set -eu

if [ -n "${MONERO_DAEMON_RPC_USERNAME:-}" ]; then
    set -- "$@" "--daemon-login=${MONERO_DAEMON_RPC_USERNAME}:${MONERO_DAEMON_RPC_PASSWORD:-}"
fi

exec monero-wallet-rpc "$@"
