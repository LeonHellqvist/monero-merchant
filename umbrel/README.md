# Monero Merchant for Umbrel

This folder is a self-contained [Umbrel](https://umbrel.com) **community app
store** that packages Monero Merchant as a self-hosted Monero (XMR)
point-of-sale backend, including the backend source it is built from.

## What it does

- Connects to the official Umbrel **Monero** node for blockchain data (no third
  party), and runs its own `monero-wallet-rpc` and `MoneroPay` services.
- Serves two web interfaces from the backend container:
  - **Admin dashboard** at `/`: wallet setup and seed backup, balance, node sync
    status, vendor management, invites, transactions, and payouts.
  - **Vendor portal** at `/store`: a vendor signs up with an invite code, sets
    their Monero payout address, and creates POS devices (name + password) for
    use with the Android POS client (in [`../pos`](../pos)).
- First run shows a "set your admin password or skip" screen, so there is no
  hidden auto-generated password.

## Layout

```
umbrel/
  umbrel-app-store.yml          community app store manifest (id + name)
  monero-merchant-app/          the app
    umbrel-app.yml              app manifest (metadata, dependencies, user config)
    docker-compose.yml          services: app-proxy, backend, postgres, wallet-rpc, MoneroPay
    exports.sh                  on app start: auto-detects the Umbrel Monero node,
                                generates secrets, writes the backend .env
    icon.png, 1.jpg..3.jpg      icon and gallery
    backend/                    the backend source the image is built from
```

## Install on Umbrel

Monero Merchant depends on the Umbrel **Monero** app, so install that first.

An Umbrel community app store must live at the root of a git repository. To
install, copy the contents of this `umbrel/` folder into the root of a git repo
(so `umbrel-app-store.yml` is at the top), then on your Umbrel go to the app
store, add a community app store, and enter that repo URL. Install Monero
Merchant from the UI, then open it to set your admin password.

## Building the backend image

`docker-compose.yml` references a prebuilt image. To rebuild it from the source
in [`monero-merchant-app/backend`](monero-merchant-app/backend):

```sh
cd monero-merchant-app/backend
docker buildx build --platform linux/amd64,linux/arm64 \
  -t <your-registry>/monero-merchant-backend:<tag> --push .
```

Then update the `image:` tag in `monero-merchant-app/docker-compose.yml` and the
`version:` in `monero-merchant-app/umbrel-app.yml`.

## Demo

A walkthrough video is available here: <add demo video link>
