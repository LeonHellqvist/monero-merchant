FROM caddy:2.11.7-alpine@sha256:eb4d35b7452d1ab6171fbdbe8d9275b2301dad6dad92115d91e19fb9b24d510f

COPY Caddyfile /etc/caddy/Caddyfile
COPY site /srv

# Caddy wants writable data and config dirs
ENV XDG_DATA_HOME=/tmp XDG_CONFIG_HOME=/tmp
USER 65534:65534
EXPOSE 8080
