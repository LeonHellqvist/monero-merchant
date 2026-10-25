# wallet-rpc

Docker image for `monero-wallet-rpc`, using the official release binaries (amd64 and arm64).

During the build, the signing key is checked against the fingerprint in the Dockerfile, `hashes.txt` is verified with it, and the downloaded tarball is checked against `hashes.txt`. See the [Monero verification guide](https://www.getmonero.org/resources/user-guides/verification-allos-advanced.html).

Set `MONERO_DAEMON_RPC_USERNAME` and `MONERO_DAEMON_RPC_PASSWORD` if your node uses `--rpc-login`.

## Updating

1. Replace `hashes.txt` with https://www.getmonero.org/downloads/hashes.txt
2. Bump `MONERO_VERSION` in the Dockerfile and the image tag in `docker-compose.yml`
3. Once merged, tag `main` with `wallet-rpc-v<version>`

CI publishes `ghcr.io/<owner>/monero-merchant-wallet-rpc:<version>`. Verify an image with:

```sh
gh attestation verify oci://ghcr.io/monero-merchant/monero-merchant-wallet-rpc@<digest> --owner monero-merchant
```
