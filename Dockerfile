# Auto Ingress - Generic Domain + Service + Auto HTTPS Gateway
# Supports multiple DNS providers via build args

ARG CADDY_VERSION=2.10.2
ARG ALPINE_VERSION=3.20

# ── Forge base image args ──────────────────────────────────────────
ARG CADDY_BUILDER_IMAGE=caddy:${CADDY_VERSION}-builder-alpine
ARG CADDY_IMAGE=caddy:${CADDY_VERSION}-alpine
ARG ALPINE_IMAGE=alpine:${ALPINE_VERSION}
ARG GOLANG_IMAGE=golang:1.26-alpine

# ── Forge mirror args ──────────────────────────────────────────────
ARG APK_MIRROR_HOST=
ARG GOPROXY=https://proxy.golang.org,direct

# ── Caddy Builder Stage ────────────────────────────────────────────
FROM ${CADDY_BUILDER_IMAGE} AS caddy-builder

ARG CADDY_VERSION
ARG DNS_PLUGINS="tencentcloud"
ARG EXTRA_CADDY_MODULES=""
ARG GOPROXY
ARG GOSUMDB=sum.golang.org

ENV GOMAXPROCS=2 \
    GOMEMLIMIT=1GiB \
    GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB}

RUN <<BUILD_SCRIPT
#!/bin/sh
set -e
XCADDY_CMD="xcaddy build v${CADDY_VERSION}"
for plugin in ${DNS_PLUGINS}; do
    case "$plugin" in
        tencentcloud) XCADDY_CMD="$XCADDY_CMD --with github.com/caddy-dns/tencentcloud" ;;
        aliyun)       XCADDY_CMD="$XCADDY_CMD --with github.com/caddy-dns/alidns" ;;
        cloudflare)   XCADDY_CMD="$XCADDY_CMD --with github.com/caddy-dns/cloudflare" ;;
        route53)      XCADDY_CMD="$XCADDY_CMD --with github.com/caddy-dns/route53" ;;
        digitalocean) XCADDY_CMD="$XCADDY_CMD --with github.com/caddy-dns/digitalocean" ;;
        googlecloud)  XCADDY_CMD="$XCADDY_CMD --with github.com/caddy-dns/googlecloud" ;;
        *) echo "Warning: Unknown DNS plugin: $plugin" ;;
    esac
done
for module in ${EXTRA_CADDY_MODULES}; do
    XCADDY_CMD="$XCADDY_CMD --with $module"
done
echo "Building Caddy with: $XCADDY_CMD"
eval $XCADDY_CMD
BUILD_SCRIPT

# ── Go Builder Stage ───────────────────────────────────────────────
FROM ${GOLANG_IMAGE} AS go-builder

ARG GOPROXY
ARG VERSION=1.0.0
ARG VCS_REVISION

ENV GOPROXY=${GOPROXY} \
    CGO_ENABLED=0

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w -X github.com/nodecoda/auto-ingress/cmd.version=${VERSION} -X github.com/nodecoda/auto-ingress/cmd.commit=${VCS_REVISION}" \
    -o /auto-ingress .

# ── Runtime Stage ──────────────────────────────────────────────────
FROM ${CADDY_IMAGE} AS runtime

ARG VERSION=1.0.0
ARG BUILD_DATE
ARG VCS_REVISION

LABEL org.opencontainers.image.title="Auto Ingress" \
      org.opencontainers.image.description="Generic Domain + Service + Auto HTTPS Gateway" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${VCS_REVISION}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.source="https://github.com/nodecoda/dc-platform" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="NodeCoda"

ARG APK_MIRROR_HOST
RUN if [ -n "${APK_MIRROR_HOST}" ]; then \
      echo "https://${APK_MIRROR_HOST}/alpine/v$(cat /etc/alpine-release | cut -d. -f1,2)/main" > /etc/apk/repositories && \
      echo "https://${APK_MIRROR_HOST}/alpine/v$(cat /etc/alpine-release | cut -d. -f1,2)/community" >> /etc/apk/repositories; \
    fi && \
    apk add --no-cache curl jq docker-cli

COPY --from=caddy-builder /usr/bin/caddy /usr/bin/caddy
COPY --from=go-builder /auto-ingress /usr/local/bin/auto-ingress
COPY compose.yml /etc/auto-ingress/compose.yml
COPY examples/dns-tencentcloud.env /etc/auto-ingress/edge-dns.env.example
COPY examples/auto-ingress.yaml.example /etc/auto-ingress/auto-ingress.yaml.example

RUN addgroup -g 1000 -S caddy && \
    adduser -u 1000 -S -s /bin/sh -G caddy caddy && \
    mkdir -p /data /config /etc/caddy /var/log/caddy && \
    chown -R caddy:caddy /data /config /etc/caddy /var/log/caddy && \
    setcap cap_net_bind_service=+ep /usr/bin/caddy

USER caddy
EXPOSE 80 443 443/udp 2019

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -sf http://localhost:2019/config/ || exit 1

VOLUME ["/data", "/config"]

ENTRYPOINT ["auto-ingress"]
CMD ["up", "--watch"]
