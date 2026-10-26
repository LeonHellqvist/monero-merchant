export APP_MONERO_MERCHANT_APP_DB_PASSWORD="$(derive_entropy "${app_entropy_identifier}-db-password")"
export APP_MONERO_MERCHANT_APP_JWT_SECRET="$(derive_entropy "${app_entropy_identifier}-jwt-secret")"
export APP_MONERO_MERCHANT_APP_JWT_REFRESH_SECRET="$(derive_entropy "${app_entropy_identifier}-jwt-refresh-secret")"
export APP_MONERO_MERCHANT_APP_JWT_MONEROPAY_SECRET="$(derive_entropy "${app_entropy_identifier}-jwt-moneropay-secret")"
export APP_MONERO_MERCHANT_APP_WALLET_RPC_PASSWORD="$(derive_entropy "${app_entropy_identifier}-wallet-rpc-password")"

# wallet-rpc has to run on the same network as the Monero app's node
case "${APP_MONERO_NETWORK:-mainnet}" in
  testnet|stagenet) export APP_MONERO_MERCHANT_APP_NETWORK_FLAG="--${APP_MONERO_NETWORK}" ;;
  *) export APP_MONERO_MERCHANT_APP_NETWORK_FLAG="" ;;
esac
