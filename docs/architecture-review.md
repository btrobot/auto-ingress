# Auto Ingress 架构审查报告

> **审查日期**: 2026-08-07
> **审查目标**: 将 auto-ingress 从 Docker 镜像脚本改造为独立 CLI 基础工具
> **当前版本**: 1.0.0
> **审查范围**: 架构设计、用户界面、技术选型、迁移路径

---

## 一、现状评估

### 1.1 当前架构概览

```
┌─────────────────────────────────────────────────┐
│               当前形态                           │
│                                                  │
│   用户 ──> env 文件 ──> compose.yml              │
│                           │                      │
│                           ▼                      │
│                     Docker 容器                   │
│                     ┌─────────────┐              │
│                     │ entrypoint  │              │
│                     │    .sh      │──> Caddyfile  │
│                     └─────────────┘      │       │
│                           ▲              ▼       │
│                     Docker Socket    Caddy       │
│                     (自动发现)      (反向代理)    │
└─────────────────────────────────────────────────┘
```

**本质**: 一个 Docker entrypoint 脚本 + Caddy 镜像定制，不是独立工具。

### 1.2 当前优势

| 优势 | 说明 |
|------|------|
| **零编译依赖** | 纯 Shell，任何环境可修改 |
| **安全默认值** | 非 root、只读 FS、最小权限 |
| **Caddy 选型正确** | 自动 HTTPS 是 Caddy 的核心竞争力 |
| **DNS-01 支持** | 6 家 DNS 提供商，不依赖 80 端口可达 |
| **构建时插件注入** | xcaddy 方案优雅，单 Dockerfile 支持多插件 |

### 1.3 当前核心问题

```
问题严重度: ██████████ = 严重, ██████ = 中等, ███ = 轻微

██████████ 无独立 CLI — 用户必须理解 Docker/Compose/env 三层才能使用
██████████ 配置即环境变量 — 无结构化、无类型校验、无 schema
██████████ 启动时一次性扫描 — 自动发现无运行时刷新能力
████████   Shell 硬编码 — Caddyfile 生成逻辑与 Shell 耦合，不可测试
████████   无配置预览 — 用户看不到"即将生效的配置"
██████     无多环境支持 — dev/staging/prod 靠手动管理多份 env 文件
██████     无证书生命周期视图 — 只有 Caddy 日志，无结构化状态
██████     无运行时管理 — 停止/重启/状态/日志全靠 docker compose
███        无插件管理 — DNS 插件在构建时固定，运行时不可变
███        模板与逻辑分离不彻底 — Caddyfile.tmpl 未被使用，逻辑在 entrypoint.sh
```

---

## 二、目标架构设计

### 2.1 设计原则

```
P1  独立可安装    一个二进制，不需要理解 Docker 内部就能使用
P2  声明式配置    YAML 配置文件，类型安全，可版本控制
P3  渐进式复杂度  5 分钟快速开始 → 按需深入高级功能
P4  Caddy 不变    继续使用 Caddy 作为代理引擎，不造轮子
P5  Docker 兼容   工具管理 Docker 容器，而不是替代 Docker
P6  可观测       状态、证书、日志一目了然
```

### 2.2 目标形态

```
┌─────────────────────────────────────────────────────────────┐
│                     auto-ingress CLI                        │
│                     (Go 单二进制)                            │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  init    │  │   add    │  │   up     │  │  status  │   │
│  │  配置向导 │  │ 添加站点 │  │ 启动网关 │  │ 查看状态 │   │
│  └──────────┘  └──────────┘  └────┬─────┘  └──────────┘   │
│                                    │                        │
│        ┌───────────────────────────┼──────────────┐        │
│        ▼                           ▼              ▼        │
│  ┌──────────┐  ┌──────────────────────┐  ┌──────────┐     │
│  │ 配置引擎  │  │   Docker 管理器      │  │ 证书管理 │     │
│  │          │  │                      │  │          │     │
│  │ YAML解析  │  │ 容器生命周期         │  │ ACME状态 │     │
│  │ 校验     │  │ 标签发现             │  │ 续期监控 │     │
│  │ 模板渲染  │  │ 网络管理             │  │ 手动续期 │     │
│  └──────────┘  └──────────────────────┘  └──────────┘     │
│        │               │                     │             │
│        └───────────────┼─────────────────────┘             │
│                        ▼                                    │
│                 ┌──────────────┐                            │
│                 │  Caddy 容器   │                            │
│                 │  (管理的对象) │                            │
│                 └──────────────┘                            │
└─────────────────────────────────────────────────────────────┘
```

### 2.3 关键架构决策

| 决策 | 选择 | 理由 |
|------|------|------|
| **实现语言** | Go | Caddy 本身是 Go 生态；xcaddy 是 Go 工具；单二进制分发；Docker SDK 官方支持 |
| **配置格式** | YAML | 结构化、类型安全、社区通用、支持注释 |
| **Caddy 交互方式** | Docker API 管理 Caddy 容器 | 保持容器化部署，不侵入 Caddy 内部 |
| **配置传递** | 生成 Caddyfile → 挂载到容器 | 利用 Caddy 原生热重载，无需额外集成 |
| **服务发现** | Docker 事件流监听 | 实时响应容器启停，不依赖轮询 |
| **分发方式** | 单二进制 + Docker 镜像双轨 | CLI 本地使用，Docker 镜像用于服务器部署 |

---

## 三、CLI 命令体系设计

### 3.1 命令树

```
auto-ingress
│
├── init                      # 初始化配置（交互式向导）
│   ├── --from-env            # 从现有 .env 文件导入
│   └── --dns-provider        # 指定 DNS 提供商
│
├── site                      # 站点管理
│   ├── add <domain> <service:port>   # 添加站点
│   ├── remove <domain>               # 移除站点
│   ├── list                          # 列出所有站点
│   └── set <domain> <key> <value>    # 修改站点属性
│
├── config                    # 配置管理
│   ├── show                  # 显示当前配置
│   ├── validate              # 校验配置
│   ├── diff                  # 显示配置变更
│   └── edit                  # 打开编辑器
│
├── cert                      # 证书管理
│   ├── list                  # 列出所有证书
│   ├── status <domain>       # 查看证书状态
│   ├── renew <domain>        # 手动续期
│   └── export <domain>       # 导出证书
│
├── up                        # 启动/更新网关
│   ├── --dry-run             # 预览变更，不执行
│   └── --watch               # 监听模式（自动发现）
│
├── down                      # 停止网关
├── restart                   # 重启网关
│
├── status                    # 网关状态概览
│   ├── --json                # JSON 输出
│   └── --watch               # 实时刷新
│
├── logs                      # 查看日志
│   ├── --follow              # 实时跟踪
│   ├── --since               # 时间过滤
│   └── --domain <domain>     # 按域名过滤
│
├── doctor                    # 诊断检查
│   ├── --fix                 # 自动修复
│   └── --all                 # 包含可选检查
│
└── version                   # 版本信息
```

### 3.2 交互场景示例

#### 场景一：5 分钟快速开始

```bash
# 安装
brew install auto-ingress
# 或
curl -fsSL https://get.auto-ingress.dev | sh

# 初始化（交互式）
$ auto-ingress init
? DNS provider: Tencent Cloud
? ACME email: ops@example.com
? Secret ID: ****
? Secret Key: ****
✓ Configuration saved to ./auto-ingress.yaml

# 添加站点
$ auto-ingress site add example.com web:3000
✓ Site added: example.com → web:3000

# 启动
$ auto-ingress up
✓ Gateway started
  HTTP:  0.0.0.0:80
  HTTPS: 0.0.0.0:443
  Admin: 127.0.0.1:2019
  Certificate: pending (ACME in progress...)
```

#### 场景二：配置预览

```bash
$ auto-ingress up --dry-run
Configuration diff:
  + example.com → web:3000 (new)
  ~ api.example.com → api:8080 (health_uri changed)
  - old.example.com (removed)

Generated Caddyfile:
┌──────────────────────────────────────┐
│ {                                    │
│     email ops@example.com            │
│     acme_dns tencentcloud { ... }    │
│ }                                    │
│                                      │
│ example.com {                        │
│     encode zstd gzip                 │
│     reverse_proxy web:3000 { ... }   │
│ }                                    │
└──────────────────────────────────────┘

Apply changes? [y/N]:
```

#### 场景三：证书状态一览

```bash
$ auto-ingress cert list
DOMAIN              STATUS     EXPIRES      DAYS LEFT
example.com         valid      2026-11-05   90
api.example.com     valid      2026-11-05   90
*.example.com       pending    —            (requesting)
old.example.com     expired    2026-07-01   -37 ⚠
```

#### 场景四：运行时状态

```bash
$ auto-ingress status
╔══════════════════════════════════════════════════╗
║  Auto Ingress Gateway                            ║
╠══════════════════════════════════════════════════╣
║  Status:    ● running                            ║
║  Uptime:    3d 12h 41m                           ║
║  Version:   1.2.0                                ║
║  Caddy:     2.10.2                               ║
╠══════════════════════════════════════════════════╣
║  Sites (3 active)                                ║
║  ├── example.com      → web:3000     ● healthy  ║
║  ├── api.example.com  → api:8080     ● healthy  ║
║  └── admin.example.com → admin:3001  ● healthy  ║
╠══════════════════════════════════════════════════╣
║  Certificates                                    ║
║  ├── example.com       valid   90d remaining     ║
║  ├── api.example.com   valid   90d remaining     ║
║  └── admin.example.com valid   90d remaining     ║
╠══════════════════════════════════════════════════╣
║  Resources                                       ║
║  Memory:    42MB / 256MB (16%)                   ║
║  Requests:  12,847 total / 23 avg/min            ║
╚══════════════════════════════════════════════════╝
```

#### 场景五：诊断检查

```bash
$ auto-ingress doctor
Checking configuration...
  ✓ auto-ingress.yaml exists and valid
  ✓ DNS credentials configured
  ✓ All sites have valid format

Checking Docker...
  ✓ Docker daemon running (v24.0.7)
  ✓ Docker Compose available (v2.21.0)
  ✓ Network "app-network" exists

Checking gateway...
  ✓ Container running
  ✓ Health check passing
  ✓ Admin API accessible

Checking certificates...
  ✓ example.com — valid (90 days remaining)
  ⚠ api.example.com — expires in 15 days, renewal scheduled
  ✗ old.example.com — expired, no renewal possible (DNS unreachable)

Checking connectivity...
  ✓ example.com → web:3000 (HTTP 200, 12ms)
  ✓ api.example.com → api:8080 (HTTP 200, 8ms)
  ✗ admin.example.com → admin:3001 (timeout after 5s)

Summary: 10 passed, 1 warning, 2 errors
Run `auto-ingress doctor --fix` to attempt automatic fixes.
```

---

## 四、配置文件格式设计

### 4.1 配置文件 `auto-ingress.yaml`

```yaml
# Auto Ingress Configuration
# 位置: ./auto-ingress.yaml 或 ~/.config/auto-ingress/config.yaml
# 文档: https://auto-ingress.dev/docs/config

version: "1"

# ─── 网关设置 ────────────────────────────────────────────
gateway:
  # ACME 证书注册邮箱
  email: ops@example.com

  # 端口绑定
  http_bind: "0.0.0.0:80"
  https_bind: "0.0.0.0:443"
  admin_bind: "127.0.0.1:2019"

  # 资源限制
  memory_limit: 256m
  pids_limit: 256

# ─── DNS 提供商 ──────────────────────────────────────────
dns:
  provider: tencentcloud
  # 凭据从环境变量读取（安全）或通过 --dns-env 指定文件
  # env_file: /etc/auto-ingress/dns.env

# ─── 站点配置 ────────────────────────────────────────────
sites:
  - domain: example.com
    upstream: web:3000
    health:
      uri: /health
      interval: 30s
      timeout: 5s
    headers:
      X-Custom-Header: "value"

  - domain: api.example.com
    upstream: api:8080

  - domain: "*.example.com"
    upstream: web:3000

# ─── 安全 ────────────────────────────────────────────────
security:
  hsts: true
  x_frame_options: SAMEORIGIN
  content_type_nosniff: true
  referrer_policy: strict-origin-when-cross-origin
  hide_server_header: true

# ─── 自动发现（可选）─────────────────────────────────────
discovery:
  enabled: false
  # Docker 标签前缀
  label_prefix: auto-ingress
  # 刷新间隔
  poll_interval: 30s
  # 网络（自动连接发现的容器到此网络）
  network: app-network

# ─── 日志 ────────────────────────────────────────────────
logging:
  access:
    enabled: true
    format: json
    output: stdout
  level: info

# ─── 高级 ────────────────────────────────────────────────
advanced:
  caddy_image: caddy:2.10.2-alpine
  caddy_data_dir: /var/lib/auto-ingress/data
  docker_network: auto-ingress-upstream
```

### 4.2 DNS 凭据管理

DNS 凭据**不写入配置文件**，通过以下方式提供（优先级从高到低）：

```
1. 环境变量          AUTO_INGRESS_DNS_* 或 provider 原生变量
2. --dns-env 文件    auto-ingress up --dns-env ./dns.env
3. 系统密钥环        auto-ingress dns set-credentials (存储到 OS keyring)
4. 交互式输入        启动时提示输入（不持久化）
```

### 4.3 配置优先级

```
命令行标志 > 环境变量 > 项目配置 (./auto-ingress.yaml)
                       > 用户配置 (~/.config/auto-ingress/config.yaml)
                       > 系统配置 (/etc/auto-ingress/config.yaml)
```

---

## 五、技术架构

### 5.1 模块划分

```
auto-ingress/
│
├── cmd/                          # CLI 入口 (cobra)
│   └── root.go
│       ├── init.go               # init 命令
│       ├── site.go               # site 子命令
│       ├── config.go             # config 子命令
│       ├── cert.go               # cert 子命令
│       ├── up.go                 # up 命令
│       ├── down.go               # down 命令
│       ├── status.go             # status 命令
│       ├── logs.go               # logs 命令
│       └── doctor.go             # doctor 命令
│
├── internal/                     # 内部包（不对外暴露）
│   ├── config/                   # 配置引擎
│   │   ├── schema.go             # YAML schema 定义
│   │   ├── loader.go             # 多源配置加载 + 合并
│   │   ├── validator.go          # 配置校验规则
│   │   ├── diff.go               # 配置变更计算
│   │   └── migrate.go            # env 文件迁移器
│   │
│   ├── caddy/                    # Caddy 集成
│   │   ├── generator.go          # Caddyfile 生成（替代 entrypoint.sh）
│   │   ├── template.go           # Go template 渲染
│   │   ├── client.go             # Caddy Admin API 客户端
│   │   └── reload.go             # 热重载逻辑
│   │
│   ├── docker/                   # Docker 管理
│   │   ├── manager.go            # 容器生命周期管理
│   │   ├── discovery.go          # 标签服务发现
│   │   ├── watcher.go            # Docker 事件监听（运行时刷新）
│   │   └── network.go            # 网络管理
│   │
│   ├── acme/                     # 证书管理
│   │   ├── store.go              # 证书存储（读取 /data）
│   │   ├── status.go             # 证书状态查询
│   │   └── notify.go             # 到期提醒
│   │
│   ├── dns/                      # DNS 提供商
│   │   ├── provider.go           # Provider 接口
│   │   ├── tencentcloud.go       # 腾讯云实现
│   │   ├── aliyun.go             # 阿里云实现
│   │   ├── cloudflare.go         # Cloudflare 实现
│   │   └── route53.go            # AWS Route53 实现
│   │
│   └── doctor/                   # 诊断引擎
│       ├── checks.go             # 检查项定义
│       └── runner.go             # 检查执行器
│
├── pkg/                          # 可复用公共包
│   ├── output/                   # 终端输出格式化
│   │   ├── table.go              # 表格渲染
│   │   ├── color.go              # 颜色支持
│   │   └── json.go               # JSON 输出
│   └── version/                  # 版本信息
│       └── version.go
│
├── docker/                       # Docker 镜像构建
│   ├── Dockerfile                # 容器化部署用 Dockerfile
│   └── Dockerfile.cli            # CLI 构建用 Dockerfile
│
├── templates/                    # Go templates
│   ├── Caddyfile.gotmpl          # Caddyfile 模板
│   └── compose.gotmpl            # 生成的 compose.yml 模板
│
├── testdata/                     # 测试数据
│   ├── configs/                  # 配置样例
│   └── caddyfiles/               # 期望的 Caddyfile 输出
│
├── go.mod
├── go.sum
├── Makefile
├── .goreleaser.yml               # GoReleaser 多平台构建配置
└── README.md
```

### 5.2 核心接口设计

```go
// config/schema.go — 配置结构
type Config struct {
    Version   string           `yaml:"version"`
    Gateway   GatewayConfig    `yaml:"gateway"`
    DNS       DNSConfig        `yaml:"dns"`
    Sites     []SiteConfig     `yaml:"sites"`
    Security  SecurityConfig   `yaml:"security"`
    Discovery DiscoveryConfig  `yaml:"discovery"`
    Logging   LoggingConfig    `yaml:"logging"`
    Advanced  AdvancedConfig   `yaml:"advanced"`
}

type SiteConfig struct {
    Domain   string            `yaml:"domain"`
    Upstream string            `yaml:"upstream"`
    Health   *HealthConfig     `yaml:"health,omitempty"`
    Headers  map[string]string `yaml:"headers,omitempty"`
}

// caddy/generator.go — Caddyfile 生成器
type Generator interface {
    Generate(cfg *Config) ([]byte, error)
    Diff(old, new *Config) (string, error)
}

// docker/manager.go — 容器管理
type Manager interface {
    Up(ctx context.Context, cfg *Config) error
    Down(ctx context.Context) error
    Status(ctx context.Context) (*GatewayStatus, error)
    Logs(ctx context.Context, opts LogOptions) (io.ReadCloser, error)
}

// docker/discovery.go — 服务发现
type Discoverer interface {
    // Start begins watching Docker events for container changes
    Start(ctx context.Context) (<-chan []SiteConfig, error)
    Stop()
}

// acme/store.go — 证书状态
type CertStore interface {
    List(ctx context.Context) ([]CertInfo, error)
    Status(ctx context.Context, domain string) (*CertInfo, error)
    Renew(ctx context.Context, domain string) error
}
```

### 5.3 技术选型

| 组件 | 选择 | 版本 | 理由 |
|------|------|------|------|
| **语言** | Go | 1.22+ | Caddy 生态、单二进制、Docker SDK 官方支持 |
| **CLI 框架** | cobra | v1.8 | Go CLI 事实标准，子命令支持完善 |
| **配置** | YAML (gopkg.in/yaml.v3) | v3 | 结构化、注释友好、Go 原生支持 |
| **Docker SDK** | docker/client | v25+ | 官方 SDK，事件流 API 支持 |
| **模板** | text/template | stdlib | Caddyfile 生成，Go 原生 |
| **终端 UI** | lipgloss + tablewriter | — | 美观的终端表格和状态展示 |
| **密钥管理** | 99designs/keyring | — | OS keyring 跨平台支持 |
| **构建发布** | GoReleaser | v2 | 多平台构建 + Homebrew + Docker |
| **测试** | testify + testcontainers | — | 单元 + 集成测试 |

### 5.4 数据流

```
用户输入
   │
   ▼
┌─────────────┐     ┌───────────────┐     ┌──────────────┐
│ CLI (cobra)  │────>│ Config Loader │────>│ Validator    │
│              │     │               │     │              │
│ site add     │     │ YAML 解析     │     │ 格式校验     │
│ up --dry-run │     │ 多源合并      │     │ 依赖检查     │
│ cert list    │     │ 环境变量覆盖   │     │ DNS 验证     │
└─────────────┘     └───────────────┘     └──────┬───────┘
                                                  │
                                                  ▼
                                          ┌──────────────┐
                                          │ Caddyfile    │
                                          │ Generator    │
                                          │ (Go template)│
                                          └──────┬───────┘
                                                  │
                              ┌────────────────────┼─────────────────────┐
                              ▼                    ▼                     ▼
                    ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
                    │ Docker       │     │ Caddy Admin  │     │ Cert Store   │
                    │ Manager      │     │ API Client   │     │              │
                    │              │     │              │     │ 读取 /data   │
                    │ 创建/管理    │     │ 配置热重载    │     │ 解析证书     │
                    │ Caddy 容器   │     │ 健康检查     │     │ 到期计算     │
                    └──────────────┘     └──────────────┘     └──────────────┘
```

---

## 六、关键改进详解

### 6.1 运行时服务发现 (Docker Watcher)

**现状**: 启动时一次性扫描，运行中容器变化不感知。

**改进**: 使用 Docker Events API 实时监听。

```go
// docker/watcher.go
func (w *Watcher) Start(ctx context.Context) (<-chan []SiteConfig, error) {
    eventsChan := make(chan []SiteConfig)

    go func() {
        defer close(eventsChan)

        // 监听容器 start/stop/die 事件
        filters := filters.NewArgs(
            filters.Arg("type", "container"),
            filters.Arg("event", "start"),
            filters.Arg("event", "stop"),
            filters.Arg("event", "die"),
        )

        events, errs := w.client.Events(ctx, types.EventsOptions{
            Filters: filters,
        })

        // 防抖: 多个容器短时间内变化时合并为一次刷新
        debounce := time.NewTimer(0)
        <-debounce.C // 消耗初始触发

        for {
            select {
            case <-ctx.Done():
                return
            case <-events:
                debounce.Reset(2 * time.Second)
            case <-debounce.C:
                sites := w.discover(ctx)
                eventsChan <- sites
            case err := <-errs:
                log.Printf("watcher error: %v", err)
                return
            }
        }
    }()

    return eventsChan, nil
}
```

### 6.2 配置校验体系

```go
// config/validator.go
func (v *Validator) Validate(cfg *Config) []ValidationError {
    var errs []ValidationError

    // 1. 格式校验
    for i, site := range cfg.Sites {
        if !isValidDomain(site.Domain) {
            errs = append(errs, ValidationError{
                Path:    fmt.Sprintf("sites[%d].domain", i),
                Message: "invalid domain format",
                Value:   site.Domain,
            })
        }
        if !isValidUpstream(site.Upstream) {
            errs = append(errs, ValidationError{
                Path:    fmt.Sprintf("sites[%d].upstream", i),
                Message: "upstream must be host:port",
                Value:   site.Upstream,
            })
        }
    }

    // 2. 依赖校验
    if cfg.Discovery.Enabled && cfg.Discovery.Network == "" {
        errs = append(errs, ValidationError{
            Path:    "discovery.network",
            Message: "network is required when discovery is enabled",
        })
    }

    // 3. DNS 连通性校验（可选）
    if v.checkDNS {
        for _, site := range cfg.Sites {
            if !canResolveDNS(site.Domain) {
                errs = append(errs, ValidationError{
                    Path:    fmt.Sprintf("sites[%s]", site.Domain),
                    Message: "domain does not resolve publicly",
                })
            }
        }
    }

    return errs
}
```

### 6.3 证书生命周期管理

```go
// acme/store.go
type CertInfo struct {
    Domain    string
    Status    CertStatus // pending | valid | expired | error
    Issuer    string     // Let's Encrypt
    NotBefore time.Time
    NotAfter  time.Time
    DaysLeft  int
    AutoRenew bool
    Error     string
}

func (s *Store) List(ctx context.Context) ([]CertInfo, error) {
    // Caddy 存储证书在 /data/caddy/certificates/
    // 解析目录结构 + 读取证书元数据
    entries, err := os.ReadDir(s.dataDir + "/certificates")
    // ... 解析每个证书，计算状态
}
```

### 6.4 配置迁移工具

```bash
# 从现有 .env 文件迁移
$ auto-ingress init --from-env /etc/auto-ingress/gateway.env

Reading /etc/auto-ingress/gateway.env...
Detected:
  ACME_EMAIL     = ops@example.com
  DNS_PROVIDER   = tencentcloud
  SITE_DOMAINS   = example.com:web:3000,api.example.com:api:8080
  UPSTREAM_NETWORK = app-network

Generated auto-ingress.yaml:
  gateway.email: ops@example.com
  dns.provider: tencentcloud
  sites: 2 entries

Save to ./auto-ingress.yaml? [Y/n]:
```

---

## 七、分阶段实施计划

### Phase 1: 核心 CLI (2-3 周)

**目标**: 独立可运行的 CLI 工具，替代手动编辑 env + docker compose

```
交付物:
├── config/schema.go          配置 YAML schema
├── config/loader.go          配置加载和校验
├── caddy/generator.go        Caddyfile 生成（从 Shell 迁移到 Go）
├── caddy/template.go         Go template 模板
├── cmd/init.go               init 命令
├── cmd/site.go               site add/remove/list
├── cmd/config.go             config show/validate
├── cmd/up.go                 up 命令（生成配置 + docker compose up）
├── cmd/down.go               down 命令
└── cmd/version.go            version 命令

验证标准:
✓ auto-ingress init 能生成有效配置
✓ auto-ingress site add 能添加站点
✓ auto-ingress up 能启动 Caddy 容器并获得 HTTPS
✓ auto-ingress config validate 能检测配置错误
✓ 所有单元测试通过
```

### Phase 2: 可观测性 (1-2 周)

**目标**: 状态、证书、日志可视化

```
交付物:
├── cmd/status.go             status 命令（表格展示）
├── cmd/cert.go               cert list/status
├── cmd/logs.go               logs 命令
├── caddy/client.go           Caddy Admin API 客户端
├── acme/store.go             证书状态解析
└── pkg/output/               终端格式化

验证标准:
✓ auto-ingress status 显示完整的网关状态
✓ auto-ingress cert list 显示所有证书和到期时间
✓ auto-ingress logs --follow 实时跟踪日志
```

### Phase 3: 运行时增强 (1-2 周)

**目标**: 自动发现、热重载、诊断

```
交付物:
├── docker/manager.go         容器管理器
├── docker/discovery.go       标签服务发现
├── docker/watcher.go         Docker 事件监听
├── cmd/up.go --watch         监听模式
├── cmd/doctor.go             诊断命令
└── config/migrate.go         env 文件迁移

验证标准:
✓ auto-ingress up --watch 监听容器变化并自动更新
✓ auto-ingress doctor 检测常见问题
✓ auto-ingress init --from-env 迁移现有配置
```

### Phase 4: 发布与分发 (1 周)

**目标**: 可安装、可更新

```
交付物:
├── .goreleaser.yml           多平台构建配置
├── Dockerfile                容器化部署版
├── Homebrew formula          macOS/Linux 安装
├── install.sh                一键安装脚本
└── README.md / docs/         文档

验证标准:
✓ brew install auto-ingress 可用
✓ curl -fsSL https://get.auto-ingress.dev | sh 可用
✓ docker run auto-ingress 可用
✓ GitHub Release 包含 linux/darwin × amd64/arm64 共 4 个二进制
```

### 实施甘特图

```
Week    1    2    3    4    5    6    7
        ├────┼────┼────┼────┼────┼────┤
Phase 1 ████████████████
Phase 2                 ██████████
Phase 3                       ██████████
Phase 4                                █████
        ├────┼────┼────┼────┼────┼────┤
        Go项目  配置  Caddy  状态  发现  诊断  发布
        骨架   引擎  生成   证书  watcher     分发
```

---

## 八、Docker 镜像兼容方案

改造后的 CLI 工具同时提供 Docker 镜像，用于无 Go 环境的服务器：

### 8.1 Docker 镜像（保持向后兼容）

```dockerfile
# Dockerfile — 容器化部署版
FROM caddy:2.10.2-alpine AS runtime

# 复制 Go 编译的 CLI
COPY auto-ingress /usr/local/bin/auto-ingress

# 保留 entrypoint.sh 作为兼容层
COPY entrypoint.sh /entrypoint.sh
COPY templates/ /etc/auto-ingress/templates/

# 用户可选择:
# 1. 新方式: CMD ["auto-ingress", "up", "--foreground"]
# 2. 旧方式: CMD ["caddy", "run", ...] (entrypoint.sh 仍可用)
```

### 8.2 双轨部署

```bash
# 方式一: CLI 直接管理（推荐）
auto-ingress up
# → 内部: 生成 Caddyfile → docker run caddy → 管理生命周期

# 方式二: Docker Compose（向后兼容）
docker compose up -d
# → 内部: entrypoint.sh 仍然工作，兼容现有 env 配置

# 方式三: CLI 管理 Compose 项目
auto-ingress up --compose
# → 内部: 生成 compose.yml → docker compose up -d
```

---

## 九、风险评估

| 风险 | 概率 | 影响 | 缓解策略 |
|------|------|------|---------|
| Go 重写引入 Bug | 中 | 高 | 保留 Shell 脚本作为回归测试基准；Caddyfile 输出 diff 测试 |
| Docker API 版本兼容 | 低 | 中 | 使用 Docker SDK 的版本协商；测试多版本 Docker |
| Caddy 升级破坏兼容 | 低 | 高 | Caddyfile 模板化后便于适配；pin 版本 + CI 矩阵测试 |
| 用户迁移阻力 | 中 | 中 | 提供 `init --from-env` 自动迁移；保留 env 模式兼容 |
| 跨平台构建复杂 | 低 | 低 | GoReleaser 处理；CGO_ENABLED=0 避免 C 依赖 |
| DNS 插件维护 | 中 | 中 | DNS 操作由 Caddy 处理，CLI 不直接调用 DNS API |

---

## 十、决策记录

### ADR-001: 选择 Go 而非 Rust/Python

**背景**: 需要选择实现语言。
**决策**: Go。
**理由**:
- Caddy 本身是 Go 项目，xcaddy 是 Go 工具
- Docker SDK for Go 官方维护，API 覆盖最全
- 单二进制静态编译，无运行时依赖
- 团队可能已有 Go 经验（Caddy 生态常见）
- GoReleaser 生态成熟

**替代方案**: Rust（性能过度、生态较小）、Python（运行时依赖、分发不便）。

### ADR-002: YAML 配置而非 TOML/HCL

**背景**: 需要结构化配置格式。
**决策**: YAML。
**理由**:
- 支持注释（TOML 也支持，但 HCL 不友好）
- 社区通用（Docker Compose、Kubernetes 均使用 YAML）
- Go 生态 yaml.v3 成熟
- 嵌套结构表达自然

### ADR-003: 管理 Caddy 容器而非嵌入 Caddy

**背景**: 是把 Caddy 作为库嵌入还是管理其容器？
**决策**: 管理 Caddy 容器。
**理由**:
- 保持 Caddy 官方镜像的完整性和更新路径
- 用户可独立升级 Caddy 版本
- 利用 Caddy 原生的容器安全配置（非 root、只读 FS）
- CLI 聚焦配置管理，不重复 Caddy 的工作

### ADR-004: 保留 entrypoint.sh 作为兼容层

**背景**: 改造后是否删除 Shell 脚本？
**决策**: 保留，标记为 legacy 兼容模式。
**理由**:
- 不破坏现有用户的部署
- Shell 脚本可作为 Go 重写的回归测试基准
- 无 Go 环境的用户仍可通过 Docker 使用

---

## 十一、总结

### 改造前后对比

| 维度 | 改造前 (Shell + env) | 改造后 (Go CLI) |
|------|---------------------|-----------------|
| **安装** | 需要理解 Docker/Compose/env | `brew install auto-ingress` |
| **添加站点** | 编辑 env 文件、重启容器 | `auto-ingress site add example.com web:3000` |
| **配置校验** | 运行时才发现错误 | `auto-ingress config validate` |
| **配置预览** | 无 | `auto-ingress up --dry-run` |
| **证书状态** | 翻日志 | `auto-ingress cert list` |
| **服务发现** | 启动时一次性扫描 | Docker 事件流实时监听 |
| **多环境** | 手动管理多份 env 文件 | 配置文件 + 环境变量覆盖 |
| **诊断** | 手动排查 | `auto-ingress doctor` |
| **可测试性** | Shell 无法单元测试 | Go test + Caddyfile diff 测试 |
| **分发** | 只有 Docker 镜像 | 二进制 + Homebrew + Docker |

### 核心价值主张

```
Before:  "你需要一个 Docker 专家来配置 HTTPS 反向代理"
After:   "一条命令，HTTPS 即可"
```

改造后的 auto-ingress 将从"Docker 镜像附带脚本"升级为一个**独立的、可安装的、有清晰用户界面的基础设施工具**，同时保持对现有 Docker 部署方式的完全兼容。
