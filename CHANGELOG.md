# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Rewritten from Shell scripts to Go CLI tool
- Configuration format changed from env files to YAML
- New CLI command interface (init, site, config, up, down, status, cert, logs, doctor)

### Added
- `auto-ingress init` — interactive configuration wizard
- `auto-ingress site add/remove/list` — site management
- `auto-ingress config show/validate` — configuration inspection
- `auto-ingress up --dry-run` — preview generated Caddyfile
- `auto-ingress cert list/status` — certificate lifecycle view
- `auto-ingress doctor` — diagnostic checks
- `--json` flag for machine-readable output
- YAML configuration with validation
- GoReleaser multi-platform binary distribution
- Comprehensive GitHub project governance (templates, CI, dependabot)

### Removed
- Shell-based entrypoint script (replaced by Go implementation)
- Environment variable configuration (replaced by YAML)

## [1.0.0] - 2024-01-01

### Added
- Initial release as Docker image
- Shell-based Caddyfile generation
- Multi DNS provider support (Tencent Cloud, Alibaba Cloud, Cloudflare, AWS Route53)
- Docker label auto-discovery
- Security hardening (non-root, read-only FS)
