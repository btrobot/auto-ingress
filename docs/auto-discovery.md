# Auto Ingress 自动发现

Auto Ingress 支持从 Docker 标签自动发现服务，无需手动配置 `SITE_DOMAINS`。

## 工作原理

```
┌─────────────────────────────────────────────────────────────┐
│  auto-ingress                                               │
│  ┌─────────────────┐    ┌─────────────────┐                │
│  │ Docker Socket   │───>│ 标签解析器      │                │
│  │ (只读)          │    │                 │                │
│  └─────────────────┘    └────────┬────────┘                │
│                                  │                          │
│                                  ▼                          │
│                         ┌─────────────────┐                │
│                         │ Caddyfile 生成  │                │
│                         └────────┬────────┘                │
│                                  │                          │
│                                  ▼                          │
│                         ┌─────────────────┐                │
│                         │ Caddy 热重载    │                │
│                         └─────────────────┘                │
└─────────────────────────────────────────────────────────────┘
```

## 使用方法

### 1. 启用自动发现

```bash
# 在 auto-ingress.env 中设置
AUTO_DISCOVER=true
```

### 2. 为应用服务添加标签

```yaml
# docker-compose.yml
services:
  my-app:
    image: my-app:latest
    labels:
      # 启用 auto-ingress
      - "auto-ingress.enable=true"
      # 域名 (可以多个)
      - "auto-ingress.domain=example.com"
      - "auto-ingress.domain=www.example.com"
      # 端口
      - "auto-ingress.port=3000"
```

### 3. 启动服务

```bash
# 启动应用
docker compose up -d

# 启动 auto-ingress (自动发现模式)
cd /path/to/auto-ingress
AUTO_DISCOVER=true docker compose up -d
```

## 标签说明

| 标签 | 必需 | 说明 | 示例 |
|------|------|------|------|
| `auto-ingress.enable` | ✅ | 是否启用 | `true` / `false` |
| `auto-ingress.domain` | ✅ | 域名 (可多个) | `example.com` |
| `auto-ingress.port` | ❌ | 端口 (默认 80) | `3000` |

## 示例

### 单域名单服务

```yaml
services:
  web:
    labels:
      - "auto-ingress.enable=true"
      - "auto-ingress.domain=example.com"
      - "auto-ingress.port=3000"
```

### 多域名单服务

```yaml
services:
  web:
    labels:
      - "auto-ingress.enable=true"
      - "auto-ingress.domain=example.com"
      - "auto-ingress.domain=www.example.com"
      - "auto-ingress.port=3000"
```

### 多服务不同域名

```yaml
services:
  web:
    labels:
      - "auto-ingress.enable=true"
      - "auto-ingress.domain=example.com"
      - "auto-ingress.port=3000"
  
  api:
    labels:
      - "auto-ingress.enable=true"
      - "auto-ingress.domain=api.example.com"
      - "auto-ingress.port=8080"
```

### 禁用服务

```yaml
services:
  database:
    labels:
      - "auto-ingress.enable=false"
```

## 工作流程

1. **启动**: auto-ingress 启动时读取 Docker socket
2. **发现**: 扫描所有带 `auto-ingress.enable=true` 标签的容器
3. **解析**: 提取域名和端口信息
4. **生成**: 自动生成 Caddyfile
5. **启动**: 启动 Caddy
6. **监听**: 监听 Docker 事件
7. **更新**: 当容器启动/停止时，重新生成 Caddyfile
8. **热重载**: 重载 Caddy 配置

## 优势

- ✅ **零配置**: 无需手动维护 `SITE_DOMAINS`
- ✅ **自动更新**: 服务启动/停止时自动更新
- ✅ **自描述**: 服务自己声明域名
- ✅ **减少错误**: 避免配置不一致

## 注意事项

### 网络要求

- auto-ingress 必须能访问目标容器
- 建议使用共享网络

### 安全

- Docker socket 以只读模式挂载
- 仅读取容器标签，不修改容器

### 性能

- 首次启动时扫描所有容器
- 后续通过 Docker 事件监听更新

## 故障排除

### 未发现服务

```bash
# 检查容器标签
docker inspect my-app --format '{{json .Config.Labels}}' | jq .

# 检查 auto-ingress 日志
docker compose logs gateway
```

### 域名未生效

```bash
# 检查生成的 Caddyfile
docker exec gateway cat /etc/caddy/Caddyfile

# 重载配置
docker exec gateway caddy reload --config /etc/caddy/Caddyfile
```
