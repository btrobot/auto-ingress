# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 2.x     | ✅ Active support  |
| 1.x     | ⚠️ Security fixes only |

## Reporting a Vulnerability

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please report them via:

1. **GitHub Security Advisories** (preferred): Use the "Security" tab → "Report a vulnerability"
2. **Email**: Send to the project maintainers

### What to include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Acknowledgment**: within 48 hours
- **Initial assessment**: within 1 week
- **Fix or mitigation**: depends on severity, typically within 2 weeks for critical issues

## Security Design

Auto Ingress follows these security principles:

- **Non-root execution**: Gateway container runs as unprivileged user
- **Read-only filesystem**: Container root filesystem is read-only
- **Minimal capabilities**: Only `NET_BIND_SERVICE` is granted
- **No privilege escalation**: `no-new-privileges` security option enabled
- **Resource limits**: Memory and PID limits enforced
- **DNS credentials isolation**: Stored separately from configuration
- **Docker socket read-only**: When auto-discovery is enabled, socket is mounted read-only

## Best Practices for Deployment

1. Store DNS credentials in a separate file with `chmod 600`
2. Bind the admin API to `127.0.0.1` only
3. Use digest-pinned images (`@sha256:...`) in production
4. Regularly update the Caddy base image
5. Monitor certificate expiry with `auto-ingress cert list`
