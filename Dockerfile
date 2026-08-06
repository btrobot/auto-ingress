# Auto Ingress - Generic Domain + Service + Auto HTTPS Gateway
# Supports multiple DNS providers via build args

ARG CADDY_VERSION=2.10.2
ARG ALPINE_VERSION=3.20

# ── Forge base image args ──────────────────────────────────────────
ARG CADDY_BUILDER_IMAGE=caddy:${CADDY_VERSION}-builder-alpine
ARG CADDY_IMAGE=caddy:${CADDY_VERSION}-alpine
ARG ALPINE_IMAGE=alpine:${ALPINE_VERSION}

# ── Forge mirror args ──────────────────────────────────────────────
ARG APK_MIRROR_HOST=

# ── Builder Stage ──────────────────────────────────────────────────
FROM ${CADDY_BUILDER_IMAGE} AS builder

ARG CADDY_VERSION
ARG DNS_PLUGINS="tencentcloud"
ARG EXTRA_CADDY_MODULES=""
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org

ENV GOMAXPROCS=2 \
    GOMEMLIMIT=1GiB \
    GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB}

# Build Caddy with specified DNS plugins
RUN <<BUILD_SCRIPT
#!/bin/sh
set -e

XCADDY_CMD="xcaddy build v${CADDY_VERSION}"

# Add DNS plugins
for plugin in ${DNS_PLUGINS}; do
    case "$plugin" in
        tencentcloud)
            XCADDY_CMD="$XCADDY_CMD --with github.com/caddy-dns/tencentcloud"
            ;;
        aliyun)
            XCADDY_CMD="$XCADDY_CMD --with github.com/caddy-dns/alidns"
            ;;
        cloudflare)
            XCADDY_CMD="$XCADDY_CMD --with github.com/caddy-dns/cloudflare"
            ;;
        route53)
            XCADDY_CMD="$XCADDY_CMD --with github.com/caddy-dns/route53"
            ;;
        digitalocean)
            XCADDY_CMD="$XCADDY_CMD --with github.com/caddy-dns/digitalocean"
            ;;
        googlecloud)
            XCADDY_CMD="$XCADDY_CMD --with github.com/caddy-dns/googlecloud"
            ;;
        *)
            echo "Warning: Unknown DNS plugin: $plugin"
            ;;
    esac
done

# Add extra Caddy modules
for module in ${EXTRA_CADDY_MODULES}; do
    XCADDY_CMD="$XCADDY_CMD --with $module"
done

echo "Building Caddy with: $XCADDY_CMD"
eval $XCADDY_CMD
BUILD_SCRIPT

# ── Runtime Stage ──────────────────────────────────────────────────
FROM ${CADDY_IMAGE} AS runtime

ARG VERSION=1.0.0
ARG BUILD_DATE
ARG VCS_REVISION

# Metadata
LABEL org.opencontainers.image.title="Auto Ingress" \
      org.opencontainers.image.description="Generic Domain + Service + Auto HTTPS Gateway" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${VCS_REVISION}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.source="https://github.com/nodecoda/dc-platform" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="NodeCoda"

# Install utilities for health checks
ARG APK_MIRROR_HOST
RUN if [ -n "${APK_MIRROR_HOST}" ]; then \
      echo "https://${APK_MIRROR_HOST}/alpine/v$(cat /etc/alpine-release | cut -d. -f1,2)/main" > /etc/apk/repositories && \
      echo "https://${APK_MIRROR_HOST}/alpine/v$(cat /etc/alpine-release | cut -d. -f1,2)/community" >> /etc/apk/repositories; \
    fi && \
    apk add --no-cache curl jq

# Copy custom Caddy binary
COPY --from=builder /usr/bin/caddy /usr/bin/caddy

# Create non-root user
RUN addgroup -g 1000 -S caddy && \
    adduser -u 1000 -S -s /bin/sh -G caddy caddy && \
    mkdir -p /data /config /etc/caddy /var/log/caddy && \
    chown -R caddy:caddy /data /config /etc/caddy /var/log/caddy

# Copy entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set capabilities for binding to privileged ports
RUN setcap cap_net_bind_service=+ep /usr/bin/caddy

# Switch to non-root user
USER caddy

# Expose ports
EXPOSE 80 443 443/udp 2019

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:2019/health || exit 1

# Volumes for persistent data
VOLUME ["/data", "/config"]

# Entrypoint
ENTRYPOINT ["/entrypoint.sh"]
CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
