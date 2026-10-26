# Monero Merchant for Umbrel

This folder is an [Umbrel](https://umbrel.com) community app store with one app,
Monero Merchant. It runs the released images from this repo against the Umbrel
**Monero** node app.

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

Install the Monero app first. A community app store has to sit at the root of a
git repo, so copy this folder's contents into one, then add that repo's URL as a
community app store in the Umbrel app store and install Monero Merchant.

After installing, back up the wallet seed under Settings in the admin dashboard.

## Updating to a new release

After a `v*` release, set the backend, frontend and gateway images in
`monero-merchant-app/docker-compose.yml` to the new version and the digests from
the release summary, and set `version:` in `monero-merchant-app/umbrel-app.yml`
to match.
