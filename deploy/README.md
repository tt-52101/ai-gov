# AI-GOV 部署指南

## K8s Helm 部署

### 前置条件

- Kubernetes 1.18+
- Helm 3.0+
- 可访问容器镜像仓库（默认：`ghcr.io/tt-52101/ai-gov`）

### 安装

```bash
# 创建命名空间并安装
helm install ai-gov ./charts/ai-gov -n ai-gov --create-namespace

# 使用自定义 values 文件安装
helm install ai-gov ./charts/ai-gov -n ai-gov --create-namespace -f my-values.yaml
```

### 升级

```bash
helm upgrade ai-gov ./charts/ai-gov -n ai-gov
```

### 卸载

```bash
helm uninstall ai-gov -n ai-gov
```

### 配置

所有可配置参数见 `charts/ai-gov/values.yaml`。常用配置项：

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `image.repository` | 镜像仓库地址 | `ghcr.io/tt-52101/ai-gov` |
| `image.tag` | 镜像标签 | Chart 版本号 |
| `replicaCount` | 副本数 | 1 |
| `config.database.host` | 数据库主机地址 | `""` |
| `config.database.password` | 数据库密码 | `change-me-db-password` |
| `config.gateway.adminToken` | 管理员令牌 | `""` |
| `config.gateway.secretKey` | 密钥 | `""` |
| `persistence.enabled` | 启用持久化存储 | true |
| `persistence.size` | 持久化存储大小 | 10Gi |
| `ingress.enabled` | 启用 Ingress | false |

### 架构说明

Pod 中包含两个容器：

- **api-server**：Go 后端服务，端口 8080，提供治理 API
- **ui-server**：Next.js 前端服务，端口 3000，提供管理控制台

两个容器使用同一镜像，通过不同的启动命令分别运行后端和前端服务。

### 健康检查

- 后端：`/healthz`（8080）
- 前端：`/api/health`（3000）

### 持久化存储

默认启用持久化存储，用于审计日志等数据存储。可通过 `persistence.enabled` 关闭。

## Docker Compose 部署

参考 `ai-gov-fusion/deploy/docker-compose.yml` 和 `ai-gov-fusion/deploy/docker-compose.postgres.yml`。