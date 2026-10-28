# Monero Merchant for Umbrel

An [Umbrel](https://umbrel.com) community app store with one app, Monero
Merchant. It runs the released Monero Merchant images against the Umbrel
**Monero** node app.

The store repo is generated from the `umbrel/` folder of
[Monero-Merchant/monero-merchant](https://github.com/Monero-Merchant/monero-merchant).
Make changes there. CI copies them here on every push to `main`.

- **Admin dashboard** at `/admin/`, behind Umbrel login plus the password Umbrel
  shows for the app.
- **Vendor portal** at `/`, and the vendor and POS parts of the API, reachable
  without Umbrel login so vendors and POS devices can connect.

## Layout

```
umbrel/
  umbrel-app-store.yml       store id and name
  monero-merchant-app/
    umbrel-app.yml           app manifest
    docker-compose.yml       gateway, frontend, backend, postgres, wallet-rpc, MoneroPay
    exports.sh               secrets derived by Umbrel, wallet-rpc network flag
    data/                    bind-mount folders for postgres, the wallet and MoneroPay
    icon.png, 1.jpg..3.jpg   icon and gallery
```

## Install

Install the Monero app first. Then, in the Umbrel app store, open Community App
Stores, add `https://github.com/Monero-Merchant/umbrel-app-store` and install
Monero Merchant.

After installing, back up the wallet seed under Settings in the admin dashboard.

## Updating to a new release

After a `v*` release, set the backend, frontend and gateway images in
`monero-merchant-app/docker-compose.yml` to the new version and the digests from
the release summary, and set `version:` in `monero-merchant-app/umbrel-app.yml`
to match.
