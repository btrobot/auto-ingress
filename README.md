# Auto Ingress

**Domain + Service + Auto HTTPS Gateway**

自动为域名申请 Let's Encrypt 证书，配置反向代理到后端服务。

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 功能特性

- ✅ **域名配置** - 支持多域名
- ✅ **服务映射** - domain → service:port
- ✅ **证书自动申请** - ACME DNS-01
- ✅ **证书自动续期** - 到期前 30 天
- ✅ **多 DNS 提供商** - 腾讯云/阿里云/Cloudflare/AWS
- ✅ **安全头配置** - HSTS, X-Frame-Options 等
- ✅ **健康检查** - 自动检测后端服务
- ✅ **热重载** - 配置变更无需重启
- ✅ **安全运行** - 非 root 用户、只读文件系统

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/your-org/auto-ingress.git
cd auto-ingress
```

### 2. 配置

```bash
# 创建配置目录
mkdir -p /etc/auto-ingress

# 复制示例配置
cp examples/generic.env /etc/auto-ingress/gateway.env
cp examples/dns-tencentcloud.env /etc/auto-ingress/dns.env

# 编辑配置
vim /etc/auto-ingress/gateway.env
vim /etc/auto-ingress/dns.env

# 设置权限
chmod 600 /etc/auto-ingress/dns.env
```

### 3. 启动

```bash
docker compose up -d
```

### 4. 验证

```bash
# 检查容器状态
docker compose ps

# 查看日志
docker compose logs -f

# 测试 HTTPS
curl -I https://example.com
```

## 配置说明

### 环境变量

| 变量 | 必需 | 说明 | 示例 |
|------|------|------|------|
| `GATEWAY_IMAGE` | ✅ | 镜像地址 (digest) | `registry/auto-ingress@sha256:...` |
| `COMPOSE_PROJECT_NAME` | ✅ | Compose 项目名 | `auto-ingress` |
| `GATEWAY_ENV_FILE` | ✅ | DNS 凭据文件 | `/etc/auto-ingress/dns.env` |
| `ACME_EMAIL` | ✅ | 证书邮箱 | `ops@example.com` |
| `SITE_DOMAINS` | ✅ | 域名和服务映射 | `example.com:web:3000` |
| `UPSTREAM_NETWORK` | ✅ | 上游网络 | `app-network` |
| `DNS_PROVIDER` | ❌ | DNS 提供商 | `tencentcloud` |
| `HEALTH_DOMAIN` | ❌ | 健康检查域名 | `example.com` |
| `HTTP_BIND` | ❌ | HTTP 绑定 | `0.0.0.0:80` |
| `HTTPS_BIND` | ❌ | HTTPS 绑定 | `0.0.0.0:443` |
| `ADMIN_BIND` | ❌ | Admin API 绑定 | `127.0.0.1:2019` |

### SITE_DOMAINS 格式

```
domain1:service1:port1,domain2:service2:port2
```

**示例:**
```bash
# 单个域名
SITE_DOMAINS=example.com:web-app:3000

# 多个域名
SITE_DOMAINS=example.com:web:3000,api.example.com:api:8080,admin.example.com:admin:3001

# 多个域名指向同一服务
SITE_DOMAINS=example.com:web:3000,www.example.com:web:3000
```

### DNS 提供商配置

#### 腾讯云

```bash
# dns.env
TENCENTCLOUD_SECRET_ID=your-secret-id
TENCENTCLOUD_SECRET_KEY=your-secret-key
```

#### 阿里云

```bash
# dns.env
ALIYUN_ACCESS_KEY_ID=your-access-key-id
ALIYUN_ACCESS_KEY_SECRET=your-access-key-secret
```

#### Cloudflare

```bash
# dns.env
CLOUDFLARE_API_TOKEN=your-api-token
```

#### AWS Route53

```bash
# dns.env
AWS_ACCESS_KEY_ID=your-access-key-id
AWS_SECRET_ACCESS_KEY=your-secret-access-key
AWS_REGION=us-east-1
```

## 构建

### 默认构建 (腾讯云)

```bash
docker build -t auto-ingress .
```

### 指定 DNS 插件

```bash
# 单个插件
docker build --build-arg DNS_PLUGINS=cloudflare -t auto-ingress .

# 多个插件
docker build --build-arg DNS_PLUGINS="tencentcloud cloudflare" -t auto-ingress .
```

### 使用 Makefile

```bash
make build                              # 默认构建
make build DNS_PLUGINS=cloudflare       # 指定插件
make build-all                          # 构建所有插件版本
make test                               # 运行测试
```

## 目录结构

```
auto-ingress/
├── Dockerfile              # 多阶段构建，支持多 DNS 插件
├── entrypoint.sh           # 入口脚本，配置验证和生成
├── compose.yml             # Docker Compose 配置
├── Makefile                # 构建和测试
├── README.md               # 文档
├── LICENSE                 # MIT 许可证
├── bin/
│   ├── generate-config     # 配置生成器
│   ├── test-config         # 配置测试
│   ├── test-dockerfile     # Dockerfile 测试
│   └── quick-start         # 快速开始
├── examples/
│   ├── generic.env         # 通用配置
│   ├── multi-site.env      # 多站点配置
│   └── dns-*.env           # DNS 提供商配置
└── product.json            # 版本信息
```

## 高级用法

### 自定义 Caddyfile

如果需要更复杂的配置，可以挂载自定义 Caddyfile：

```bash
# 生成自定义 Caddyfile
./bin/generate-config /etc/auto-ingress/gateway.env Caddyfile

# 挂载到容器
docker compose -f compose.yml -f compose.custom.yml up -d
```

### 热重载配置

```bash
# 重新生成配置
./bin/generate-config /etc/auto-ingress/gateway.env Caddyfile

# 重载 Caddy (无需重启)
docker compose exec gateway caddy reload --config /etc/caddy/Caddyfile
```

### 多实例部署

```bash
# 实例 1: 主站
COMPOSE_PROJECT_NAME=main-site \
  SITE_DOMAINS=example.com:web:3000 \
  docker compose up -d

# 实例 2: API
COMPOSE_PROJECT_NAME=api-gateway \
  SITE_DOMAINS=api.example.com:api:8080 \
  docker compose up -d
```

## 故障排除

### 证书申请失败

```bash
# 检查 DNS 配置
docker compose exec gateway env | grep -E "TENCENTCLOUD|ALIYUN|CLOUDFLARE|AWS"

# 检查 ACME 日志
docker compose logs gateway | grep -i acme

# 验证 DNS 记录
dig TXT _acme-challenge.example.com
```

### 健康检查失败

```bash
# 检查后端服务
docker compose exec gateway curl -v http://service:port/health

# 检查网络连接
docker network inspect app-network
```

### 配置验证失败

```bash
# 测试配置
./bin/test-config /etc/auto-ingress/gateway.env

# 查看生成的 Caddyfile
./bin/generate-config /etc/auto-ingress/gateway.env Caddyfile.test
cat Caddyfile.test
```

## 支持的 DNS 提供商

| 提供商 | 构建参数 | 环境变量 |
|--------|----------|----------|
| 腾讯云 | `tencentcloud` | `TENCENTCLOUD_SECRET_ID/KEY` |
| 阿里云 | `aliyun` | `ALIYUN_ACCESS_KEY_ID/SECRET` |
| Cloudflare | `cloudflare` | `CLOUDFLARE_API_TOKEN` |
| AWS Route53 | `route53` | `AWS_ACCESS_KEY_ID/SECRET` |
| DigitalOcean | `digitalocean` | `DO_AUTH_TOKEN` |
| Google Cloud | `googlecloud` | `GCE_SERVICE_ACCOUNT` |

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License - 详见 [LICENSE](LICENSE)
