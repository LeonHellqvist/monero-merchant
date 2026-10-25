# Official Monero release binaries, checked against binaryFate's signed hashes.txt.
# To update: copy https://www.getmonero.org/downloads/hashes.txt over hashes.txt
# and bump MONERO_VERSION.

ARG MONERO_VERSION=0.18.5.4

FROM --platform=$BUILDPLATFORM debian:trixie-slim@sha256:62bd423aeee912639ca8ee681c2b762cb3e25137e0a92b20be455898a9f034ad AS download
ARG MONERO_VERSION
ARG TARGETARCH
# https://www.getmonero.org/resources/user-guides/verification-allos-advanced.html
ARG SIGNING_KEY_FINGERPRINT=81AC591FE9C4B65C5806AFC3F0AF4D462A0BDF92
SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

RUN apt-get update \
    && apt-get install -y --no-install-recommends bzip2 ca-certificates curl gpg gpgv \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /verify
COPY binaryfate.asc hashes.txt ./

RUN fingerprints="$(gpg --batch --with-colons --show-keys binaryfate.asc | awk -F: '$1 == "pub" { getline; print $10 }')"; \
    if [ "$fingerprints" != "$SIGNING_KEY_FINGERPRINT" ]; then \
        echo "unexpected key in binaryfate.asc: $fingerprints" >&2; \
        exit 1; \
    fi; \
    gpg --batch --dearmor --output keyring.gpg binaryfate.asc; \
    # only the signed part ends up in hashes.verified
    gpgv --keyring ./keyring.gpg --output hashes.verified hashes.txt

RUN case "$TARGETARCH" in \
        amd64) platform=linux-x64 ;; \
        arm64) platform=linux-armv8 ;; \
        *) echo "unsupported architecture: $TARGETARCH" >&2; exit 1 ;; \
    esac; \
    tarball="monero-${platform}-v${MONERO_VERSION}.tar.bz2"; \
    grep -E "^[0-9a-f]{64}  ${tarball}\$" hashes.verified > tarball.sha256; \
    test "$(wc -l < tarball.sha256)" -eq 1; \
    curl -fsSL --proto '=https' --tlsv1.2 -o "$tarball" "https://downloads.getmonero.org/cli/${tarball}"; \
    sha256sum -c tarball.sha256; \
    mkdir extracted; \
    tar -xjf "$tarball" -C extracted --strip-components=1; \
    install -m 0755 extracted/monero-wallet-rpc /monero-wallet-rpc

FROM debian:trixie-slim@sha256:62bd423aeee912639ca8ee681c2b762cb3e25137e0a92b20be455898a9f034ad

# curl is for the compose healthcheck
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home --uid 1000 monero

COPY --from=download /monero-wallet-rpc /usr/local/bin/monero-wallet-rpc
COPY --chmod=0755 entrypoint.sh /usr/local/bin/entrypoint.sh

LABEL org.opencontainers.image.title="monero-wallet-rpc" \
      org.opencontainers.image.licenses="BSD-3-Clause"

USER monero
WORKDIR /home/monero
EXPOSE 28081
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
