#!/bin/sh
set -e

# Auto Ingress Entrypoint
# 支持两种模式: 环境变量配置 或 Docker标签自动发现

# ── 配置验证 ──────────────────────────────────────────────────
validate_config() {
    echo "Validating configuration..."
    
    # 检查必需的环境变量
    for var in ACME_EMAIL; do
        eval value=\$$var
        if [ -z "$value" ]; then
            echo "ERROR: Required environment variable $var is not set"
            exit 1
        fi
    done
    
    # 检查 DNS提供商配置
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

# ── Docker 标签发现 ──────────────────────────────────────────
discover_from_docker() {
    echo "Discovering services from Docker..."
    
    # 发现带 auto-ingress.enable=true 标签的容器
    SERVICES=""
    
    # 使用 docker ps 获取所有带标签的容器
    docker ps --format '{{.ID}}' | while read -r container_id; do
        # 检查容器是否有 auto-ingress.enable 标签
        ENABLED=$(docker inspect "$container_id" --format '{{index .Config.Labels "auto-ingress.enable"}}' 2>/dev/null || echo "")
        
        if [ "$ENABLED" = "true" ]; then
            # 获取域名
            DOMAINS=$(docker inspect "$container_id" --format '{{index .Config.Labels "auto-ingress.domain"}}' 2>/dev/null || echo "")
            # 获取端口
            PORT=$(docker inspect "$container_id" --format '{{index .Config.Labels "auto-ingress.port"}}' 2>/dev/null || echo "80")
            # 获取服务名
            SERVICE_NAME=$(docker inspect "$container_id" --format '{{.Name}}' 2>/dev/null | sed 's/^\///')
            
            if [ -n "$DOMAINS" ]; then
                # 将域名转换为逗号分隔
                DOMAIN_LIST=$(echo "$DOMAINS" | tr ' ' ',')
                echo "Found service: $SERVICE_NAME -> $DOMAIN_LIST:$PORT"
                
                # 添加到服务列表
                if [ -n "$SERVICES" ]; then
                    SERVICES="$SERVICES,$DOMAIN_LIST:$SERVICE_NAME:$PORT"
                else
                    SERVICES="$DOMAIN_LIST:$SERVICE_NAME:$PORT"
                fi
            fi
        fi
    done
    
    # 如果没有发现服务，使用环境变量
    if [ -z "$SERVICES" ] && [ -n "$SITE_DOMAINS" ]; then
        echo "No services discovered from Docker, using SITE_DOMAINS"
        SERVICES="$SITE_DOMAINS"
    fi
    
    echo "Discovered services: $SERVICES"
}

# ── 生成 Caddyfile ──────────────────────────────────────────
generate_caddyfile() {
    echo "Generating Caddyfile..."
    
    # 默认 DNS 提供商
    DNS_PROVIDER=${DNS_PROVIDER:-tencentcloud}
    
    # 生成 DNS 配置
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
    
    # 使用发现的服务或环境变量
    SERVICES_TO_USE="${SERVICES:-$SITE_DOMAINS}"
    
    # 生成站点配置
    site_configs=""
    OLD_IFS="$IFS"
    IFS=','
    for site in $SERVICES_TO_USE; do
        IFS=':' read -r domain service port << SPLIT
$site
SPLIT
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
    IFS="$OLD_IFS"
    
    # 生成完整的 Caddyfile
    cat > /etc/caddy/Caddyfile << CADDYFILE
{
	email $ACME_EMAIL
	admin localhost:2019
	$dns_config
}

$site_configs
CADDYFILE
    
    echo "Caddyfile generated at /etc/caddy/Caddyfile"
}

# ── 主流程 ──────────────────────────────────────────────────
main() {
    echo "=== Auto Ingress Gateway ==="
    echo "Version: ${VERSION:-dev}"
    echo "Mode: ${AUTO_DISCOVER:-env}"
    echo ""
    
    # 验证配置
    validate_config
    
    # 如果启用自动发现，则从 Docker 标签发现服务
    if [ "${AUTO_DISCOVER:-false}" = "true" ]; then
        discover_from_docker
    fi
    
    # 生成 Caddyfile
    generate_caddyfile
    
    # 执行命令
    echo ""
    echo "Starting gateway..."
    exec "$@"
}

# 运行主函数
main "$@"
