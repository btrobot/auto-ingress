# Auto Ingress 代码库分析文档

> 自动生成于 2026-08-07 | 项目版本: 1.0.0 | 许可证: MIT

---

## 一、项目概述

**Auto Ingress** 是一个基于 [Caddy](https://caddyserver.com/) 的**自动 HTTPS 反向代理网关**。它的核心目标是：

> **输入域名 + 后端服务地址 → 自动获得一个带 Let's Encrypt HTTPS 证书的反向代理**

面向场景是中小型 Docker Compose 部署环境，尤其适合需要快速将多个域名/服务暴露为 HTTPS 端点的团队。

### 核心能力一览

| 能力 | 说明 |
|------|------|
| 域名 → 服务映射 | `example.com:web:3000` 格式声明式配置 |
| 自动 HTTPS 证书 | ACME DNS-01 challenge，到期前 30 天自动续期 |
| 多 DNS 提供商 | 腾讯云、阿里云、Cloudflare、AWS Route53、DigitalOcean、Google Cloud |
| Docker 标签自动发现 | 通过 `auto-ingress.*` 标签零配置发现后端服务 |
| 安全加固 | 非 root 用户、只读文件系统、最小权限、安全响应头 |
| 热重载 | 配置变更无需重启容器 |

---

## 二、总体架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Docker Compose                              │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                  auto-ingress gateway 容器                   │   │
│  │                                                              │   │
│  │  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │   │
│  │  │ entrypoint   │───>│ generate     │───>│  Caddyfile   │  │   │
│  │  │ .sh          │    │ Caddyfile    │    │  (生成产物)   │  │   │
│  │  └──────────────┘    └──────────────┘    └──────┬───────┘  │   │
│  │        │                                         │          │   │
│  │        │  (可选: 读取 Docker 标签)                ▼          │   │
│  │        │                                  ┌──────────┐     │   │
│  │        └───── Docker Socket (只读) ───────>│  Caddy   │     │   │
│  │                                            │  Server  │     │   │
│  │                                            └────┬─────┘     │   │
│  └─────────────────────────────────────────────────┼───────────┘   │
│                                                     │               │
│       ┌─────────────────────────────────────────────┼─────────┐    │
│       │              upstream Docker network         │         │    │
│       │                                              │         │    │
│       │   ┌─────────┐  ┌─────────┐  ┌─────────┐    │         │    │
│       │   │ web:3000 │  │ api:8080│  │ admin   │◄───┘         │    │
│       │   └─────────┘  └─────────┘  └─────────┘              │    │
│       └───────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐                      │
│  │ caddy_data│  │caddy_config│ │ caddy_etc  │  ← 持久化卷          │
│  │ /data     │  │ /config    │ │ /etc/caddy │                      │
│  └───────────┘  └───────────┘  └───────────┘                      │
└─────────────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    │   ACME DNS-01     │
                    │   (证书申请/续期)  │
                    └─────────┬─────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
   ┌──────────┐      ┌──────────────┐     ┌────────────┐
   │ 腾讯云DNS │      │ Cloudflare   │     │ AWS Route53│
   │ 阿里云DNS │      │              │     │ ...        │
   └──────────┘      └──────────────┘     └────────────┘
```

---

## 三、实现原理详解

### 3.1 容器启动流程

```
docker compose up
    │
    ▼
ENTRYPOINT: /entrypoint.sh
    │
    ├─ 1. validate_config()     验证环境变量完整性
    │     ├─ 检查 ACME_EMAIL
    │     └─ 按 DNS_PROVIDER 检查对应凭据
    │
    ├─ 2. discover_from_docker() (仅 AUTO_DISCOVER=true)
    │     ├─ 通过 Docker Socket 扫描所有容器
    │     ├─ 读取 auto-ingress.enable/domain/port 标签
    │     └─ 拼接为 SITE_DOMAINS 格式
    │
    ├─ 3. generate_caddyfile()  生成 Caddy 配置
    │     ├─ 解析 DNS_PROVIDER → 生成 DNS 配置块
    │     ├─ 解析 SITE_DOMAINS → 生成站点反向代理块
    │     └─ 写入 /etc/caddy/Caddyfile
    │
    └─ 4. exec caddy run        启动 Caddy 服务器
          └─ Caddy 读取 Caddyfile → 启动 HTTP/HTTPS 监听
```

### 3.2 配置生成机制

配置通过 shell 脚本在容器内动态生成，核心逻辑在 `entrypoint.sh` 的 `generate_caddyfile()` 函数中。

**输入**: 环境变量 `SITE_DOMAINS`（格式: `domain:service:port,domain:service:port,...`）

**输出**: Caddy 配置文件 `/etc/caddy/Caddyfile`

生成的 Caddyfile 结构如下：

```caddyfile
{
    email ops@example.com                    # ACME 账户
    admin localhost:2019                      # 管理 API
    acme_dns tencentcloud {                   # DNS-01 验证
        secret_id {env.TENCENTCLOUD_SECRET_ID}
        secret_key {env.TENCENTCLOUD_SECRET_KEY}
    }
}

example.com {
    encode zstd gzip                          # 响应压缩
    
    reverse_proxy web:3000 {                  # 反向代理
        health_uri /health                    # 健康检查
        health_interval 30s
        health_timeout 5s
    }
    
    log {                                     # 访问日志
        output stdout
        format json
    }
}
```

### 3.3 Docker 多阶段构建

Dockerfile 采用两阶段构建策略：

| 阶段 | 基础镜像 | 职责 |
|------|---------|------|
| **Builder** | `caddy:2.10.2-builder-alpine` | 使用 `xcaddy` 编译自定义 Caddy 二进制，注入 DNS 插件 |
| **Runtime** | `caddy:2.10.2-alpine` | 安装 curl/jq，创建非 root 用户，设置权限 |

**DNS 插件注入原理**: 通过构建参数 `DNS_PLUGINS` 控制，在 builder 阶段用 `xcaddy build` 编译 Caddy 时通过 `--with` 标志添加对应的 `github.com/caddy-dns/*` 插件。

### 3.4 Docker 标签自动发现

当 `AUTO_DISCOVER=true` 时，`entrypoint.sh` 的 `discover_from_docker()` 函数会：

1. 通过只读挂载的 Docker Socket 调用 `docker ps` 枚举容器
2. 对每个容器检查 `auto-ingress.enable` 标签是否为 `true`
3. 提取 `auto-ingress.domain` 和 `auto-ingress.port` 标签
4. 拼接为 `SITE_DOMAINS` 兼容格式

支持的容器标签:

| 标签 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `auto-ingress.enable` | 是 | — | `true` 启用发现 |
| `auto-ingress.domain` | 是 | — | 域名（可多次声明多域名） |
| `auto-ingress.port` | 否 | `80` | 后端服务端口 |

### 3.5 HTTPS 证书自动管理

Caddy 原生支持 ACME 证书自动申请和续期。本项目通过 DNS-01 challenge 方式工作：

1. Caddy 向 Let's Encrypt 请求证书
2. Let's Encrypt 要求在 DNS 中创建 `_acme-challenge.example.com` TXT 记录
3. Caddy 通过 DNS 插件（如 `caddy-dns/tencentcloud`）自动操作 DNS API 写入记录
4. Let's Encrypt 验证后签发证书
5. 证书存储在 `/data` 卷中，到期前 30 天自动续期

**DNS-01 相比 HTTP-01 的优势**: 无需 80 端口可达、可申请通配符证书、适合内网/NAT 环境。

---

## 四、目录结构与文件职责

```
auto-ingress/
├── Dockerfile                    # 多阶段 Docker 构建（Builder + Runtime）
├── entrypoint.sh                 # 容器入口：配置验证、服务发现、Caddyfile 生成
├── compose.yml                   # Docker Compose 编排定义
├── Makefile                      # 构建、测试、推送等开发命令
├── product.json                  # 项目元数据（名称、版本、功能列表、DNS 提供商）
├── build.yaml                    # 平台构建配置（镜像源、构建参数）
├── README.md                     # 用户文档
├── LICENSE                       # MIT 许可证
│
├── bin/                          # 开发/运维工具脚本
│   ├── generate-config           # 离线 Caddyfile 生成器（bash）
│   ├── quick-start               # 交互式快速开始向导
│   ├── test-config               # 配置文件校验脚本
│   └── test-dockerfile           # Dockerfile 语法检查脚本
│
├── templates/                    # 配置模板
│   └── Caddyfile.tmpl            # Caddyfile 占位符模板（供外部参考）
│
├── examples/                     # 配置示例
│   ├── generic.env               # 通用配置示例
│   ├── multi-site.env            # 多站点配置示例
│   ├── auto-discover.env         # 自动发现模式配置
│   ├── nodecoda.env              # 生产环境配置示例
│   ├── dns-tencentcloud.env      # 腾讯云 DNS 凭据模板
│   ├── dns-aliyun.env            # 阿里云 DNS 凭据模板
│   ├── dns-cloudflare.env        # Cloudflare DNS 凭据模板
│   ├── dns-aws.env               # AWS Route53 DNS 凭据模板
│   └── app-with-labels.yml       # Docker 标签示例应用
│
├── docs/                         # 项目文档
│   ├── auto-discovery.md         # 自动发现功能文档
│   └── codebase-analysis.md      # 本文档
│
├── .github/workflows/
│   └── build.yml                 # CI: 测试 → 构建矩阵 → 发布
│
└── .artifacts/                   # 构建产物（自动生成）
    └── build/1.0.0/evidence/     # 构建证据
```

---

## 五、技术栈

| 层次 | 技术 | 版本 | 用途 |
|------|------|------|------|
| 反向代理 / HTTPS | Caddy | 2.10.2 | HTTP/HTTPS 服务器、自动证书管理 |
| DNS 插件编译 | xcaddy | — | 编译带自定义 DNS 插件的 Caddy |
| 容器运行时 | Docker + Compose | ≥20.10 / ≥2.0 | 容器化部署 |
| 基础镜像 | Alpine Linux | 3.20 | 最小化运行时镜像 |
| 脚本语言 | POSIX Shell / Bash | — | 入口脚本、配置生成、工具脚本 |
| CI/CD | GitHub Actions | — | 自动构建、测试、发布 |

---

## 六、安全设计

### 容器安全

| 措施 | 实现方式 |
|------|---------|
| 非 root 运行 | 创建 `caddy` 用户 (uid=1000)，`USER caddy` |
| 只读文件系统 | `read_only: true` + tmpfs 挂载 `/tmp` |
| 最小权限 | `cap_drop: ALL` + 仅 `cap_add: NET_BIND_SERVICE` |
| 禁止提权 | `security_opt: no-new-privileges:true` |
| 资源限制 | 内存 256MB、PID 256、停止宽限期 30s |
| 日志轮转 | 最大 10MB × 5 文件 |
| Docker Socket 只读 | `docker.sock:ro` |

### 网络安全

- 响应头自动注入: HSTS、X-Content-Type-Options、X-Frame-Options、Referrer-Policy
- Admin API 绑定 `127.0.0.1:2019`（仅本地访问）
- DNS 凭据文件建议 `chmod 600`

---

## 七、CI/CD 流水线

```
push/PR to main
    │
    ├─ test job
    │   └─ 运行 test-config 校验示例配置
    │
    └─ build job (matrix: tencentcloud / aliyun / cloudflare / route53)
        ├─ Docker Buildx 构建
        ├─ GitHub Actions 缓存 (GHA cache)
        └─ 非 PR 时推送镜像
            │
            └─ release job (仅 tag: v*)
                └─ 创建 GitHub Release + 上传 product.json
```

---

## 八、配置体系

### 两层配置分离

```
/etc/auto-ingress/
├── gateway.env      # 网关配置（域名、邮箱、网络、端口）
└── dns.env          # DNS 凭据（敏感，chmod 600）
```

通过 `GATEWAY_ENV_FILE` 在 compose.yml 中引用 `dns.env`，实现敏感信息与业务配置分离。

### 配置变量分类

**必需变量**:
- `GATEWAY_IMAGE` — 镜像地址（digest 限定）
- `COMPOSE_PROJECT_NAME` — Compose 项目名
- `GATEWAY_ENV_FILE` — DNS 凭据文件路径
- `ACME_EMAIL` — Let's Encrypt 注册邮箱
- `SITE_DOMAINS` — 域名→服务映射
- `UPSTREAM_NETWORK` — 上游 Docker 网络

**可选变量**:
- `DNS_PROVIDER` — DNS 提供商（默认 `tencentcloud`）
- `AUTO_DISCOVER` — 启用 Docker 标签自动发现
- `HTTP_BIND` / `HTTPS_BIND` / `ADMIN_BIND` — 端口绑定
- `MEM_LIMIT` / `PIDS_LIMIT` — 资源限制
- `HEALTH_DOMAIN` — 健康检查域名

---

## 九、设计亮点与潜在改进

### 设计亮点

1. **零代码实现** — 纯 Shell + Caddy 组合，无编译语言依赖，维护门槛极低
2. **构建时插件注入** — 通过 `xcaddy` + build arg 实现同一 Dockerfile 支持任意 DNS 插件组合
3. **双模式配置** — 声明式环境变量 + Docker 标签自动发现，兼顾简单和灵活
4. **安全默认值** — 非 root、只读 FS、最小权限、资源限制，开箱即安全
5. **DNS-01 验证** — 避免 80 端口可达性要求，适用 NAT/防火墙环境

### 潜在改进方向

1. **自动发现动态刷新** — 当前 `discover_from_docker()` 仅在启动时执行一次，可考虑加入 Docker 事件监听实现运行时动态更新
2. **配置模板化** — Caddyfile 生成逻辑硬编码在 Shell 中，可引入 `envsubst` 或模板引擎提升可维护性
3. **健康检查增强** — 可增加对后端服务就绪状态的主动探测和告警
4. **TLS 配置定制** — 当前使用 Caddy 默认 TLS 设置，可增加 TLS 版本和密码套件的配置项
5. **日志结构化** — 可增加访问日志的自定义字段和日志级别控制

---

## 十、快速理解路径

推荐阅读顺序:

1. **README.md** — 了解功能和使用方式
2. **examples/generic.env** — 理解配置变量
3. **compose.yml** — 理解容器编排和安全策略
4. **Dockerfile** — 理解镜像构建和插件注入
5. **entrypoint.sh** — 理解核心运行逻辑
6. **bin/generate-config** — 理解 Caddyfile 生成逻辑
7. **examples/app-with-labels.yml** — 理解自动发现模式
