#!/bin/sh
set -e

# Auto Ingress Entrypoint
# Handles runtime configuration and validation

# ── Configuration Validation ────────────────────────────────────────
validate_config() {
    echo "Validating configuration..."
    
    # Check required environment variables
    required_vars="ACME_EMAIL SITE_DOMAINS"
    for var in $required_vars; do
        eval value=\$$var
        if [ -z "$value" ]; then
            echo "ERROR: Required environment variable $var is not set"
            exit 1
        fi
    done
    
    # Validate SITE_DOMAINS format
    echo "$SITE_DOMAINS" | tr ',' '\n' | while IFS=':' read -r domain service port; do
        if [ -z "$domain" ] || [ -z "$service" ] || [ -z "$port" ]; then
            echo "ERROR: Invalid SITE_DOMAINS format: $domain:$service:$port"
            echo "Expected: domain:service:port,domain2:service2:port2"
            exit 1
        fi
    done
    
    # Check DNS provider configuration
    if [ -n "$DNS_PROVIDER" ]; then
        case "$DNS_PROVIDER" in
            tencentcloud)
                if [ -z "$TENCENTCLOUD_SECRET_ID" ] || [ -z "$TENCENTCLOUD_SECRET_KEY" ]; then
                    echo "ERROR: Tencent Cloud DNS requires TENCENTCLOUD_SECRET_ID and TENCENTCLOUD_SECRET_KEY"
                    exit 1
                fi
                ;;
            aliyun)
                if [ -z "$ALIYUN_ACCESS_KEY_ID" ] || [ -z "$ALIYUN_ACCESS_KEY_SECRET" ]; then
                    echo "ERROR: Alibaba Cloud DNS requires ALIYUN_ACCESS_KEY_ID and ALIYUN_ACCESS_KEY_SECRET"
                    exit 1
                fi
                ;;
            cloudflare)
                if [ -z "$CLOUDFLARE_API_TOKEN" ]; then
                    echo "ERROR: Cloudflare DNS requires CLOUDFLARE_API_TOKEN"
                    exit 1
                fi
                ;;
            route53)
                if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
                    echo "ERROR: AWS Route53 requires AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY"
                    exit 1
                fi
                ;;
        esac
    fi
    
    echo "Configuration validated successfully"
}

# ── Generate Caddyfile ──────────────────────────────────────────────
generate_caddyfile() {
    echo "Generating Caddyfile..."
    
    # Default DNS provider
    DNS_PROVIDER=${DNS_PROVIDER:-tencentcloud}
    
    # Generate DNS configuration
    dns_config=""
    case "$DNS_PROVIDER" in
        tencentcloud)
            dns_config="acme_dns tencentcloud {
		secret_id {env.TENCENTCLOUD_SECRET_ID}
		secret_key {env.TENCENTCLOUD_SECRET_KEY}
	}"
            ;;
        aliyun)
            dns_config="acme_dns aliyun {
		access_key_id {env.ALIYUN_ACCESS_KEY_ID}
		access_key_secret {env.ALIYUN_ACCESS_KEY_SECRET}
	}"
            ;;
        cloudflare)
            dns_config="acme_dns cloudflare {
		api_token {env.CLOUDFLARE_API_TOKEN}
	}"
            ;;
        route53)
            dns_config="acme_dns route53 {
		access_key_id {env.AWS_ACCESS_KEY_ID}
		secret_access_key {env.AWS_SECRET_ACCESS_KEY}
		region {env.AWS_REGION}
	}"
            ;;
    esac
    
    # Generate site configurations
    site_configs=""
    IFS=',' read -ra sites <<< "$SITE_DOMAINS"
    for site in "${sites[@]}"; do
        IFS=':' read -r domain service port <<< "$site"
        site_configs="$site_configs
$domain {
	encode zstd gzip
	
	reverse_proxy $service:$port {
		health_uri /health
		health_interval 30s
		health_timeout 5s
	}
	
	log {
		output stdout
		format json
	}
}"
    done
    
    # Generate complete Caddyfile
    cat > /etc/caddy/Caddyfile << CADDYFILE
{
	email {$ACME_EMAIL}
	admin off
	$dns_config
}

$site_configs
CADDYFILE
    
    echo "Caddyfile generated at /etc/caddy/Caddyfile"
}

# ── Main ────────────────────────────────────────────────────────────
main() {
    echo "=== Auto Ingress Gateway ==="
    echo "Version: ${VERSION:-dev}"
    echo ""
    
    # Validate configuration
    validate_config
    
    # Generate Caddyfile if not exists
    if [ ! -f /etc/caddy/Caddyfile ]; then
        generate_caddyfile
    else
        echo "Using existing Caddyfile"
    fi
    
    # Execute command
    echo ""
    echo "Starting gateway..."
    exec "$@"
}

# Run main function
main "$@"
