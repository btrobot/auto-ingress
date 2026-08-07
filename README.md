# Auto Ingress

**Automatic HTTPS reverse proxy gateway — one command, HTTPS ready.**

A CLI tool that manages a Caddy-based reverse proxy with automatic Let's Encrypt certificate provisioning via DNS-01 challenges.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Install

```bash
# macOS / Linux (Homebrew)
brew install nodecoda/taps/auto-ingress

# Binary download
curl -fsSL https://github.com/nodecoda/auto-ingress/releases/latest/download/auto-ingress_linux_amd64.tar.gz | tar xz
sudo mv auto-ingress /usr/local/bin/

# From source
go install github.com/nodecoda/auto-ingress@latest
```

## Quick Start

```bash
# 1. Create configuration
auto-ingress init

# 2. Add sites
auto-ingress site add example.com web:3000
auto-ingress site add api.example.com api:8080

# 3. Start gateway
auto-ingress up
```

## Commands

```
auto-ingress init                    Interactive configuration wizard
auto-ingress site add <domain> <upstream>
auto-ingress site remove <domain>
auto-ingress site list               List all configured sites
auto-ingress config show             Display configuration
auto-ingress config validate         Validate configuration
auto-ingress up                      Start the gateway
auto-ingress up --dry-run            Preview generated Caddyfile
auto-ingress down                    Stop the gateway
auto-ingress restart                 Restart the gateway
auto-ingress status                  Show gateway status
auto-ingress cert list               List certificates and expiry
auto-ingress cert status <domain>    Certificate details
auto-ingress logs -f                 Follow gateway logs
auto-ingress doctor                  Run diagnostic checks
auto-ingress version                 Version info
```

All commands support `--json` for JSON output and `--config` to specify a config file path.

## Configuration

Configuration lives in `auto-ingress.yaml`:

```yaml
version: "1"

gateway:
  email: ops@example.com
  http_bind: "0.0.0.0:80"
  https_bind: "0.0.0.0:443"
  admin_bind: "127.0.0.1:2019"
  memory_limit: 256m
  pids_limit: 256

dns:
  provider: cloudflare

sites:
  - domain: example.com
    upstream: web:3000
  - domain: api.example.com
    upstream: api:8080

security:
  hsts: true
  x_frame_options: SAMEORIGIN
  content_type_nosniff: true
  referrer_policy: strict-origin-when-cross-origin
  hide_server_header: true

discovery:
  enabled: false
  network: app-network

logging:
  access:
    enabled: true
    format: json
    output: stdout
  level: info
```

### DNS Providers

Set credentials as environment variables before running `auto-ingress up`:

| Provider | Environment Variables |
|----------|----------------------|
| Tencent Cloud | `TENCENTCLOUD_SECRET_ID`, `TENCENTCLOUD_SECRET_KEY` |
| Alibaba Cloud | `ALIYUN_ACCESS_KEY_ID`, `ALIYUN_ACCESS_KEY_SECRET` |
| Cloudflare | `CLOUDFLARE_API_TOKEN` |
| AWS Route53 | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION` |
| DigitalOcean | `DO_AUTH_TOKEN` |
| Google Cloud | `GCE_SERVICE_ACCOUNT` |

Or reference a DNS env file:

```yaml
dns:
  provider: cloudflare
  env_file: /etc/auto-ingress/dns.env
```

## Docker Label Auto-Discovery

When `discovery.enabled: true`, the gateway automatically discovers services from Docker container labels:

```yaml
# docker-compose.yml of your app
services:
  web:
    image: my-app:latest
    labels:
      auto-ingress.enable: "true"
      auto-ingress.domain: "example.com"
      auto-ingress.port: "3000"
```

## How It Works

```
auto-ingress up
    │
    ├─ 1. Load and validate auto-ingress.yaml
    ├─ 2. Generate Caddyfile (Go template)
    ├─ 3. Launch Caddy container with:
    │     - Generated Caddyfile mounted
    │     - DNS credentials passed as env vars
    │     - Security hardening (non-root, read-only FS, minimal caps)
    └─ 4. Caddy auto-provisions HTTPS certificates via ACME DNS-01
```

## Project Structure

```
auto-ingress/
├── main.go                          Entry point
├── cmd/                             CLI commands (cobra)
│   ├── root.go, init.go, site.go
│   ├── up.go, down.go, status.go
│   ├── cert.go, config.go, logs.go
│   ├── doctor.go, version.go
├── internal/
│   ├── config/                      Config loading and validation
│   │   ├── schema.go                Types and defaults
│   │   ├── loader.go                YAML loading + env overrides
│   │   └── validator.go             Validation rules
│   ├── caddy/                       Caddyfile generation
│   │   ├── generator.go             Go template renderer
│   │   └── templates/               Caddyfile.gotmpl
│   ├── docker/                      Docker container management
│   │   └── manager.go               Container lifecycle via CLI
│   └── acme/                        Certificate store
│       └── store.go                 Read certs from Caddy data
├── pkg/output/                      Terminal formatting
├── examples/                        Config examples
├── docs/                            Documentation
├── Makefile                         Build commands
├── .goreleaser.yml                  Release automation
├── Dockerfile.release               Docker image for releases
├── product.json                     Project metadata
└── compose.yml                      Docker Compose (container mode)
```

## Build

```bash
make build              # Build binary
make test               # Run tests
make lint               # Run linter
make install            # Install to $GOPATH/bin
```

## License

MIT License — see [LICENSE](LICENSE)
