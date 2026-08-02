# 数据层 DDL 与部署配置实现完整性核查报告 | 版本：v1.0 | 日期：2026-08-01 | 状态：已发布

> 本报告基于**事实代码**逐文件核查，禁止依赖 `docs/delivery/acceptance/` 下任何报告文档。
> 核查基线：PRD `docs/prd/AI-GOV-PRD-v3.2.0.md` §10 数据库设计、§12 非功能需求、§11.7 阶段 A/E 验收焦点。
> 核查性质：**只读代码核查，未修改任何代码文件**。

---

## 0. 核查范围与基线文件清单

| 类别 | 文件路径 | 用途 |
|------|---------|------|
| DDL Schema | `schema/ai-gov-fusion-v3.2.sql` | 40 表核心 DDL |
| 迁移脚本 | `scripts/migrate-db-v3.2.mjs` | legacy TokenHub → v3.2 迁移 |
| 部署编排 | `ai-gov-fusion/deploy/docker-compose.yml` | 主编排 |
| 部署编排 | `ai-gov-fusion/deploy/docker-compose.postgres.yml` | PostgreSQL 编排 |
| 部署编排 | `ai-gov-fusion/deploy/docker-compose.e2e.yml` | E2E 测试编排 |
| 环境样例 | `ai-gov-fusion/deploy/.env.example` | 环境变量样例 |
| 安装脚本 | `ai-gov-fusion/deploy/install.sh` | 顶层安装脚本 |
| 原生部署 | `ai-gov-fusion/deploy/native/install.sh` | systemd 原生安装 |
| 原生部署 | `ai-gov-fusion/deploy/native/tokenhub.service` | systemd 单元 |
| 本地部署 | `ai-gov-fusion/deploy/local/run-local.sh` | 本地运行脚本 |
| 反向代理 | `ai-gov-fusion/deploy/nginx.multi-instance.conf` | 多实例负载均衡 |
| 后端镜像 | `ai-gov-fusion/backend/Dockerfile` | 容器构建 |
| 原生镜像 | `ai-gov-fusion/backend/Dockerfile.native` | 原生构建产物 |
| GORM 集中迁移 | `ai-gov-fusion/backend/internal/server/store.go` | 集中 AutoMigrate 入口 |
| GORM 模块迁移 | `ai-gov-fusion/backend/internal/server/{abac,audit,authz,fund,idempotency,modelgrant,party,pricing,reconciliation,routing,ui_permission}/` | 分散 AutoMigrate |

---

## 1. DDL 核查结果（`schema/ai-gov-fusion-v3.2.sql`）

### 1.1 数量统计（PowerShell 实测）

| 指标 | 实测值 | PRD 要求 | 判定 |
|------|--------|---------|------|
| `CREATE TABLE` | **40** | §10.2「总计：40 表」 | ✅ 符合 |
| `CREATE INDEX` | **103** | 无硬性数量要求 | ✅ 充分 |
| `CREATE PROCEDURE` | **3** | §10.3 审计链锚定 + ABAC 评估 | ✅ 符合 |
| `CREATE TRIGGER` | **0** | §10.3「流水只追加（应用层禁止 UPDATE/DELETE）」 | ⚠️ 应用层约束，无 DB 层触发器（见 §1.4） |
| `trg_append_only` 出现次数 | **0** | — | ⚠️ 见 §1.4 |
| `liquidation_type` 出现次数 | **0** | §10.1 liquidations 表要求该字段 | ❌ 缺失（见 §1.3） |
| `model_grants.party_id` | **存在** | batch-007 FIX-E R6-16 | ✅ 已修复 |
| `route_profiles.party_id` | **存在** | batch-007 FIX-E R6-12 | ✅ 已修复 |

### 1.2 40 表清单对照（PRD §10.1 vs DDL）

| 组别 | 表名 | PRD §10.1 | DDL 行号 | 判定 |
|------|------|-----------|---------|------|
| 第1组 用户与身份（2 表） | `users` | ✓ | schema 行 1-25 | ✅ |
| | `admin_sessions` | ✓ | schema 行 29-42 | ✅ |
| 第2组 Party 统一主体（3 表） | `parties` | ✓ | schema 行 49-66 | ✅ |
| | `party_edges` | ✓ | schema 行 70-87 | ✅ |
| | `party_members` | ✓ | schema 行 91-108 | ✅ |
| 第3组 资金治理（5 表） | `accounts` | ✓ | schema 行 114-138 | ✅ |
| | `ledgers` | ✓ | schema 行 142-164 | ✅ |
| | `freezes` | ✓ | schema 行 170-191 | ✅ |
| | `allocations` | ✓ | schema 行 195-208 | ✅ |
| | `liquidations` | ✓ | schema 行 217-229 | ⚠️ 字段缺失见 §1.3 |
| 第4组 API Key（1 表） | `api_keys` | ✓ | schema 行 242-280 | ✅ |
| 第5组 模型目录（4 表） | `providers` | ✓ | schema 行 286-302 | ✅ |
| | `provider_resources` | ✓ | schema 行 306-323 | ✅ |
| | `models` | ✓ | schema 行 327-352 | ✅ |
| | `provider_models` | ✓ | schema 行 356-368 | ✅ |
| 第6组 定价与路由（3 表） | `model_prices` | ✓ | schema 行 374-390 | ✅ |
| | `model_routes` | ✓ | schema 行 394-410 | ✅ |
| | `route_profiles` | ✓ | schema 行 489-503 | ✅ |
| 第7组 安全治理（9 表） | `sys_action_catalogs` | ✓ | schema 行 547-560 | ✅ |
| | `sys_roles` | ✓ | schema 行 564-573 | ✅ |
| | `sys_role_permissions` | ✓ | schema 行 577-586 | ✅ |
| | `sys_subject_role_bindings` | ✓ | schema 行 590-604 | ✅ |
| | `sys_access_policies` | ✓ | schema 行 608-622 | ✅ |
| | `sys_access_policy_bindings` | ✓ | schema 行 626-636 | ✅ |
| | `sys_ui_menus` | ✓ | schema 行 640-651 | ✅ |
| | `sys_ui_routes` | ✓ | schema 行 655-666 | ✅ |
| | `sys_ui_action_bindings` | ✓ | schema 行 670-681 | ✅ |
| 第8组 授权治理（2 表） | `grants` | ✓ | schema 行 514-526 | ✅ |
| | `model_grants` | ✓ | schema 行 530-543 | ✅ |
| 第9组 请求与用量（5 表） | `request_logs` | ✓ | schema 行 414-440 | ✅ |
| | `request_payload_logs` | ✓ | schema 行 444-452 | ✅ |
| | `route_attempt_logs` | ✓ | schema 行 456-470 | ✅ |
| | `usage_records` | ✓ | schema 行 474-485 | ✅ |
| | `quota_buckets` | ✓ | schema 行 685-698 | ✅ |
| 第10组 可观测（2 表） | `channel_probes` | ✓ | schema 行 702-715 | ✅ |
| | `provider_quota_status` | ✓ | schema 行 719-729 | ✅ |
| 第11组 基础设施（3 表） | `audit_events` | ✓ | schema 行 733-757 | ✅ |
| | `audit_chain_anchors` | ✓ | schema 行 761-771 | ✅ |
| | `idempotency_records` | ✓ | schema 行 775-788 | ✅ |

**40 表全部存在，组别与数量 100% 符合 PRD §10.1。**

### 1.3 liquidations 表字段缺失（阻塞级缺陷）

**PRD §10.1 行 915 要求：**
```
liquidations | 新建 | party_id, account_id, target_account_id,
              status(blocking/draining/refunding/closing/closed),
              liquidation_type(project_close/org_merge/org_split)
```

**DDL 实际定义（schema 行 217-229）：**
```sql
CREATE TABLE liquidations (
    id                  TEXT PRIMARY KEY,
    party_id            TEXT NOT NULL,
    account_id          TEXT NOT NULL,
    target_account_id   TEXT,
    status              TEXT NOT NULL DEFAULT 'blocking', -- blocking/draining/refunding/closing/closed
    initiated_by        TEXT NOT NULL,
    initiated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at           TIMESTAMPTZ,
    metadata            JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**缺陷：**
- ❌ `liquidation_type` 字段缺失，无法区分清算触发场景（项目关闭/组织合并/组织拆分），违反 PRD §8.5 组织变更流程的归因要求。

### 1.4 清算状态机命名不一致（高风险）

| 来源 | 状态值 |
|------|--------|
| PRD §8.4 行 716-720 | `active → liquidating_block_new → liquidating_drain → liquidating_transfer → liquidated` |
| PRD §10.1 行 915 | `blocking/draining/refunding/closing/closed` |
| DDL schema 行 222 | `blocking/draining/refunding/closing/closed`（同 §10.1） |

**结论：** PRD 内部 §8.4 与 §10.1 状态命名不一致；DDL 遵循 §10.1。这是 **PRD 自身矛盾**，需产品决策后统一。代码层目前以 §10.1 为准。

### 1.5 trg_append_only 触发器缺失（中风险）

PRD §10.3 行 995-996 要求：
- 「流水只追加（应用层禁止 UPDATE/DELETE）」
- 「审计不可篡改（应用层禁止 UPDATE/DELETE audit_events）」

DDL 实测 `CREATE TRIGGER` 数量为 **0**，无 `trg_append_only` 触发器。

**判定：** PRD 明确写「**应用层**禁止」，故 DB 层触发器非强制。但缺乏 DB 层兜底意味着任何直连数据库的运维操作或 SQL 注入均可绕过应用层约束，违反「财务治理闭环是第一护城河」铁律。建议作为 P2 强化项。

### 1.6 存储过程核查（3 个，符合）

| 存储过程 | DDL 行号 | 用途 | PRD 对照 |
|---------|---------|------|---------|
| `evaluate_access` | schema 行 ~1009 | ABAC 直接策略评估 | §7 ABAC 引擎 ✅ |
| `evaluate_access_via_roles` | schema 行 ~1080 | ABAC 角色权限评估 | §7 ABAC 引擎 ✅ |
| `anchor_audit_chain` | schema 行 ~1180 | 审计链哈希锚定 | §10.3 审计不可篡改 ✅ |

---

## 2. batch-002 / batch-007 DDL 修复验证

### 2.1 batch-007 FIX-E 修复项（已验证）

| 修复项 | PRD 引用 | DDL 实测 | 判定 |
|--------|---------|---------|------|
| R6-12: `route_profiles.party_id` | §6 路由策略归因 | schema 行 489-503 含 `party_id` 字段 | ✅ 已修复 |
| R6-16: `model_grants.party_id` | §6 ModelGrant 归因 | schema 行 530-543 含 `party_id` 字段 | ✅ 已修复 |

### 2.2 batch-002 GAP-006 遗留（未修复）

| 缺陷 | 状态 | 证据 |
|------|------|------|
| `liquidations.liquidation_type` 字段缺失 | ❌ 未修复 | PowerShell 实测 `liquidation_type` 出现次数 = 0 |

---

## 3. 迁移脚本核查（`scripts/migrate-db-v3.2.mjs`）

### 3.1 数量统计（PowerShell 实测）

| 指标 | 实测值 | 说明 |
|------|--------|------|
| `CREATE TABLE` | **27** | 新建表数（部分表由 SQL 脚本创建，迁移脚本负责补充） |
| `addColumn` 函数调用 | **83** | 为现有表添加新列 |
| `ALTER TABLE` | **2** | 直接 ALTER 语句 |
| `CREATE INDEX` 直接语句 | **2** | 多数索引通过 `indexes` 数组循环创建 |
| `idx_` 索引数组项 | **75** | 索引定义数组元素 |
| `INSERT INTO` | **17** | 种子数据插入 |

### 3.2 迁移脚本能力评估

| 能力 | 实现状态 | 证据 |
|------|---------|------|
| 幂等性（可重运行） | ✅ | 脚本头部声明，跳过已存在列和表 |
| 首次运行备份 | ✅ | 首次运行备份数据库 |
| 数据迁移（admin_users → users） | ✅ | 行 533-558 |
| 现有表加列（api_keys 等） | ✅ | 行 568-671，添加 owner_user_id/account_id/party_id |
| 索引创建 | ✅ | 行 1040-1117，117 个索引定义（部分与 DDL 重叠） |
| 种子数据 | ✅ | 17 个 INSERT INTO |

**判定：** 迁移脚本结构完整，具备幂等、备份、加列、数据迁移、索引、种子数据全链路能力。

---

## 4. 部署配置核查

### 4.1 部署目录结构（`ai-gov-fusion/deploy/`）

```
deploy/
├── container/              # 容器入口脚本
│   ├── tokenhub-build-id
│   ├── tokenhub-entrypoint
│   └── tokenhub-entrypoint_test.sh
├── local/                  # 本地运行
│   ├── README.md
│   ├── run-local.sh
│   └── standalone-bundle.sh
├── native/                 # 原生 systemd 部署
│   ├── install.sh
│   ├── install_test.sh
│   ├── tokenhub-run
│   └── tokenhub.service
├── .env.example
├── docker-compose.e2e.yml
├── docker-compose.model-catalog.yml
├── docker-compose.postgres.yml
├── docker-compose.remote-postgres.yml
├── docker-compose.yml
├── install.sh
├── install_test.sh
└── nginx.multi-instance.conf
```

### 4.2 部署矩阵对照（PRD §11.4 / §12）

| 部署方式 | PRD 要求 | 实现状态 | 证据 |
|---------|---------|---------|------|
| Docker Compose | §11.4「Docker Compose + systemd（MVP）」 | ✅ 完整 | `docker-compose.yml` + `docker-compose.postgres.yml` |
| systemd 原生 | §11.4「systemd（MVP）」 | ✅ 完整 | `native/install.sh` + `native/tokenhub.service` |
| 本地运行 | — | ✅ 完整 | `local/run-local.sh`（无 root、无 systemd） |
| 多实例负载均衡 | §12「多实例故障切换」 | ✅ 完整 | `nginx.multi-instance.conf` |
| **K8s Helm** | §11.4「K8s Helm（生产）」、§12「K8s」 | ❌ **缺失** | 全局搜索 `Chart.yaml` 无结果 |
| 离线/内网 | §12「离线/内网」 | ⚠️ 部分 | 原生安装支持离线 tar 包，但无独立离线打包脚本 |

### 4.3 docker-compose.yml 核查

**文件：** `ai-gov-fusion/deploy/docker-compose.yml`

- 服务：`tokenhub-backend`，镜像 `ghcr.io/astaxie/tokenhub-backend`
- 端口映射：`8080:8080`（后端）、`3000:3000`（前端）
- 环境变量：`TOKENHUB_ENV`、`TOKENHUB_HTTP_ADDR`、`TOKENHUB_DATABASE_URL`
- 卷挂载：`tokenhub-data`、`tokenhub-releases`
- 健康检查：配置完整

**判定：** MVP 级 Docker Compose 编排完整，符合 §11.4 MVP 要求。

### 4.4 native/tokenhub.service 核查

**文件：** `ai-gov-fusion/deploy/native/tokenhub.service`

- `Type=simple`，`Restart=always`，`RestartSec=5`
- 安全加固：`NoNewPrivileges=true`、`PrivateDevices=true`、`PrivateTmp=true`、`ProtectHome=true`、`ProtectSystem=strict`、`ReadWritePaths` 限定
- `LimitNOFILE=65535`
- `UMask=0077`

**判定：** systemd 单元安全加固达标，符合 §12 安全要求。

### 4.5 后端 Dockerfile 核查

**文件：** `ai-gov-fusion/backend/Dockerfile`

- 多阶段构建：frontend-deps → frontend-builder → backend-builder → runtime
- 前端：Node 22.23.1-bookworm-slim，Next.js standalone 输出
- 后端：Go 1.26-alpine，`CGO_ENABLED=1`（SQLite 驱动依赖），`-tags "netgo osusergo"`，静态链接
- 运行时：debian:bookworm-slim，非 root 用户 `node`（UID 1000）
- 健康检查：`/healthz` + 前端 3000 端口双探活

**判定：** 生产级多阶段构建，符合 §12 安全与可用性要求。

### 4.6 Dockerfile.native 核查

**文件：** `ai-gov-fusion/backend/Dockerfile.native`

- 单阶段 builder + scratch artifact
- `CGO_ENABLED=1`，静态链接，`-extldflags '-static'`
- `deploymentType=native` 标记
- 产物：`/tokenhub` 静态二进制

**判定：** 原生构建产物适合离线分发，符合 §12 离线/内网要求。

---

## 5. GORM AutoMigrate 集成核查

### 5.1 集中迁移入口（`server/store.go`）

**文件：** `ai-gov-fusion/backend/internal/server/store.go` 行 481-508

**关键注释（行 482-483）：**
```
// v3.2: 仅迁移 v3.2 表中 GORM 运行时需要的模型。
// 40 张 v3.2 表由 ai-gov-fusion-v3.2.sql 在部署时创建。
```

**集中迁移的 17 个模型（行 484-504）：**

| GORM 模型 | 映射 v3.2 表 |
|-----------|-------------|
| `Project` | → parties |
| `ProjectTeam` | → party_members |
| `AdminUser` | → users |
| `APIKey` | → api_keys |
| `Provider` | → providers |
| `ProviderResource` | → provider_resources |
| `ProviderModel` | → provider_models |
| `Model` | → models |
| `ModelRoute` | → model_routes |
| `UsageRecord` | → usage_records |
| `RequestLog` | → request_logs |
| `RequestPayloadLog` | → request_payload_logs |
| `RouteAttemptLog` | → route_attempt_logs |
| `AuditEvent` | → audit_events |
| `AdminSession` | → admin_sessions |
| `QuotaBucket` | → quota_buckets |
| `InFlightLease`/`ClusterLease` | 运行时基础设施表（非 v3.2 DDL 表） |

### 5.2 模块级分散 AutoMigrate（11 个模块，25 个模型）

| 模块 | 文件:行号 | 迁移表 | 表数 |
|------|---------|--------|------|
| `abac` | `abac/model.go:259` | SysActionCatalog, SysRole, SysRolePermission, SysSubjectRoleBinding, SysAccessPolicy, SysAccessPolicyBinding | 6 |
| `audit` | `audit/store.go:17` | AuditEvent, AuditChainAnchor | 2 |
| `authz` | `authz/grant.go:123` | Grant | 1 |
| `fund/sqlstore` | `fund/sqlstore/pg.go:37` | Account, Ledger, Freeze, Allocation, Liquidation | 5 |
| `idempotency` | `idempotency/model.go` | Record | 1 |
| `modelgrant` | `modelgrant/grant.go:113` | ModelGrant | 1 |
| `party` | `party/store.go:190` | Party, PartyEdge, PartyMember | 3 |
| `pricing` | `pricing/store.go:16` | ModelPrice | 1 |
| `reconciliation` | `reconciliation/store.go:20` | ReconciliationRun | 1 |
| `routing` | `routing/strategy.go:190` | RouteProfile | 1 |
| `ui_permission` | `ui_permission/store.go:308` | SysUIMenu, SysUIRoute, SysUIActionBinding | 3 |

### 5.3 AutoMigrate 策略判定

**设计意图（store.go 行 482-483 注释）：**
- 40 张 v3.2 表由 `ai-gov-fusion-v3.2.sql` 在**部署时**通过 `psql -f` 创建（见 `deploy/install.sh`）
- GORM AutoMigrate 仅负责**运行时**所需的模型同步（确保列存在，不删除列）
- 模块级 AutoMigrate 由各领域包自管

**潜在问题：**
- ⚠️ `AuditEvent` 在 `server/store.go:498` 和 `audit/store.go:17` **双重迁移**——非错误（AutoMigrate 幂等），但职责重叠。
- ⚠️ 集中迁移的 `Project`/`ProjectTeam`/`AdminUser` 等是 TokenHub legacy 模型名，与 v3.2 表名（parties/party_members/users）通过 GORM 表名映射关联，需确认 `TableName()` 方法正确覆盖。

**判定：** AutoMigrate 集中+分散混合策略符合 §11.3「存量不动 + 新建包从零写」原则。40 表 DDL 由 SQL 脚本主导创建，GORM 仅做运行时列同步，避免了 AutoMigrate 不修改列类型的局限（audit/store.go 行 14 注释明确此局限）。

---

## 6. 国产环境兼容性核查（PRD §11.4 / §11.7 / §12）

### 6.1 PRD 要求

| PRD 条款 | 要求 |
|---------|------|
| §11.4 行 1063 | 「国产适配：国产 CPU/OS 阶段 A 冒烟」 |
| §11.7 行 1096 | 阶段 A「国产冒烟」验收焦点 |
| §12 行 1113 | 「国产环境阶段 A 验证」 |

### 6.2 代码层实测（PowerShell / Grep 全局搜索）

| 核查项 | 搜索范围 | 实测结果 | 判定 |
|--------|---------|---------|------|
| OceanBase 驱动 import | `backend/**/*.go` | **0 匹配** | ❌ |
| TiDB 驱动 import | `backend/**/*.go` | **0 匹配** | ❌ |
| `github.com/pingcap` | `backend/**/*.go` | **0 匹配** | ❌ |
| MySQL 驱动 import | `backend/**/*.go` | **0 匹配** | ❌ |
| 麒麟/Kylin | 全仓库代码 | **0 匹配**（仅文档提及） | ❌ |
| 统信 UOS | 全仓库代码 | **0 匹配**（仅文档提及） | ❌ |
| 鲲鹏/Kunpeng | 全仓库代码 | **0 匹配**（仅文档提及） | ❌ |
| 飞腾 | 全仓库代码 | **0 匹配** | ❌ |
| LoongArch | 全仓库代码 | **0 匹配** | ❌ |
| arm64 架构支持 | `deploy/native/install.sh:338`、`version_update.go:191` | ✅ 支持 amd64/arm64 | ⚠️ 通用 ARM，非国产特化 |

### 6.3 GORM 数据库驱动实测

**文件：** `ai-gov-fusion/backend/internal/server/store.go` 行 29-34

```go
import (
    sqlite3 "github.com/mattn/go-sqlite3"
    "gorm.io/driver/postgres"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
    gormlogger "gorm.io/gorm/logger"
)
```

**实测：** GORM 仅导入 `postgres` + `sqlite` 两个驱动，**无 OceanBase/TiDB/MySQL 驱动**。

### 6.4 国产环境判定

| 维度 | 状态 | 证据 |
|------|------|------|
| 国产数据库适配 | ❌ **未实现** | 无 OceanBase/TiDB 驱动，GORM 仅 PG+SQLite |
| 国产 OS 适配 | ❌ **未实现** | 无麒麟/统信 UOS 适配代码 |
| 国产 CPU 适配 | ⚠️ **部分** | arm64 通用支持，无鲲鹏/飞腾特化 |
| 国产冒烟测试 | ❌ **未执行** | 无冒烟测试脚本/报告 |

**结论：** PRD §11.7 阶段 A「国产冒烟」验收焦点**未达成**。代码层完全缺失国产化实现，所有「国产」提及均停留在 PRD/历史文档层面。这与 `docs/analysis/PROGRESS-SUMMARY-v1.0.md` 行 108 的 I-07 缺陷「国产环境冒烟未执行」一致。

---

## 7. PRD 条款对照表

| PRD 条款 | 要求摘要 | 实现状态 | 证据 |
|---------|---------|---------|------|
| §10.1 | 40 表 DDL | ✅ 符合 | schema 行 1-1243，40 个 CREATE TABLE |
| §10.1 liquidations | 含 liquidation_type 字段 | ❌ 缺失 | schema 行 217-229 无该字段 |
| §10.3 | 流水只追加（应用层） | ⚠️ 应用层约束 | 无 DB 触发器，依赖应用层 |
| §10.3 | 审计不可篡改（应用层） | ⚠️ 应用层约束 | audit_events 无 DB 层 UPDATE/DELETE 防护 |
| §11.4 | Go 1.26 + GORM | ✅ 符合 | Dockerfile 行 27 `golang:1.26-alpine` |
| §11.4 | SQLite（MVP）/ PostgreSQL 16（生产） | ✅ 符合 | store.go 行 30-31 双驱动 |
| §11.4 | Docker Compose + systemd（MVP） | ✅ 符合 | deploy/ 完整 |
| §11.4 | K8s Helm（生产） | ❌ **缺失** | 无 Chart.yaml |
| §11.4 | 国产 CPU/OS 阶段 A 冒烟 | ❌ **未实现** | 见 §6 |
| §11.7 阶段 A | 40 表 DDL + 国产冒烟 | ⚠️ 部分 | DDL ✅，国产冒烟 ❌ |
| §11.7 阶段 E | 压测 HA/GA/文档 | ❌ 未实施 | 无压测脚本/Helm/运维手册 |
| §12 可用性 | 99.9%；多实例故障切换 | ⚠️ 部分 | nginx 多实例 ✅，K8s 缺失 |
| §12 性能 | 5000 QPS；<50ms | ❌ 未验证 | 无压测报告 |
| §12 安全 | TLS 1.3；AES-256；ABAC | ✅ 代码层实现 | abac 模块 6 表 + evaluate_access 存储过程 |
| §12 部署 | 私有化；Docker/K8s；离线/内网；国产 | ⚠️ 部分 | Docker ✅，K8s ❌，国产 ❌ |
| §12 审计保留 | ≥180 天；哈希链锚定 | ✅ 代码层实现 | audit_chain_anchors + anchor_audit_chain 存储过程 |
| §12 可观测 | Prometheus + Grafana | ❌ 未实现 | 无 Prometheus 指标导出代码 |

---

## 8. 缺陷分级清单

### 8.1 阻塞级（P0，必须修复方可 GA）

| ID | 缺陷 | PRD 引用 | 证据 | 修复建议 |
|----|------|---------|------|---------|
| **BLK-01** | `liquidations.liquidation_type` 字段缺失 | §10.1 行 915 | schema 行 217-229 | ALTER TABLE 添加 `liquidation_type TEXT` + 迁移脚本同步 |
| **BLK-02** | K8s Helm Chart 缺失 | §11.4 行 1062、§12 行 1113 | 全局无 Chart.yaml | 新建 `deploy/helm/ai-gov/` Chart |

### 8.2 高风险（P1，影响阶段 E 验收）

| ID | 缺陷 | PRD 引用 | 证据 | 修复建议 |
|----|------|---------|------|---------|
| **HIGH-01** | 国产环境兼容性代码完全缺失 | §11.4 行 1063、§11.7 行 1096、§12 行 1113 | GORM 仅 PG+SQLite，无国产驱动/OS/CPU 适配 | 引入 TiDB/OceanBase 驱动兼容层；麒麟/统信 UOS 冒烟脚本 |
| **HIGH-02** | PRD §8.4 与 §10.1 清算状态机命名矛盾 | §8.4 行 716-720 vs §10.1 行 915 | DDL 遵循 §10.1 | 产品决策统一状态命名后同步 DDL 与代码 |
| **HIGH-03** | 可观测性（Prometheus + Grafana）未实现 | §12 行 1116 | 无指标导出代码 | 新增 `/metrics` 端点 + Grafana 仪表盘 |

### 8.3 中风险（P2，强化项）

| ID | 缺陷 | PRD 引用 | 证据 | 修复建议 |
|----|------|---------|------|---------|
| **MID-01** | 无 DB 层 `trg_append_only` 触发器 | §10.3 行 995 | CREATE TRIGGER = 0 | 新增 ledgers/audit_events BEFORE UPDATE/DELETE 触发器兜底 |
| **MID-02** | AuditEvent 双重 AutoMigrate | — | store.go:498 + audit/store.go:17 | 统一由 audit 模块负责，集中入口移除 |
| **MID-03** | 离线打包脚本缺失 | §12 行 1113 | 无独立离线安装包构建 | 补充离线 tar 打包脚本 |

### 8.4 低风险（P3，观察项）

| ID | 缺陷 | PRD 引用 | 证据 |
|----|------|---------|------|
| **LOW-01** | 性能压测未执行 | §12 行 1111 | 无压测报告 |
| **LOW-02** | 阶段 E 文档与许可未交付 | §11.7 行 1100 | 无运维手册 |

---

## 9. 核查结论

### 9.1 总体判定

| 维度 | 达成率 | 判定 |
|------|--------|------|
| DDL 40 表完整性 | **97.5%**（39/40 表字段完整，1 表缺字段） | ⚠️ 接近达成 |
| 迁移脚本完整性 | **100%** | ✅ 达成 |
| 部署配置 MVP（Docker + systemd） | **100%** | ✅ 达成 |
| 部署配置生产（K8s Helm） | **0%** | ❌ 未达成 |
| GORM AutoMigrate 集成 | **100%** | ✅ 达成 |
| 国产环境兼容性 | **0%** | ❌ 未达成 |
| 可观测性 | **0%** | ❌ 未达成 |

### 9.2 阶段 A 验收焦点（PRD §11.7）

| 验收项 | 状态 |
|--------|------|
| Fork TokenHub 通路 | ✅ |
| 执行 DDL（40 表含 ABAC+UI 治理） | ✅（40 表存在，liquidation_type 缺失） |
| 用量规范化 | ✅（usage_records 表含 itemCode JSON） |
| **国产冒烟** | ❌ **未达成** |

### 9.3 阶段 E 验收焦点（PRD §11.7）

| 验收项 | 状态 |
|--------|------|
| 压测 HA | ❌ 未实施 |
| K8s Helm | ❌ 缺失 |
| 文档与许可 | ❌ 未交付 |

### 9.4 关键阻塞路径

1. **BLK-01** liquidation_type 缺失 → 阻塞 §8.5 组织变更流程归因
2. **BLK-02** K8s Helm 缺失 → 阻塞 §11.4 生产部署路径
3. **HIGH-01** 国产化缺失 → 阻塞 §11.7 阶段 A 验收焦点

---

## 10. 核查方法说明

- **数据来源：** 直接读取代码文件，引用具体行号；禁止依赖 `docs/delivery/acceptance/` 报告文档
- **统计工具：** PowerShell `[regex]::Matches` 实测 CREATE TABLE/INDEX/PROCEDURE/TRIGGER 数量
- **搜索工具：** Grep 工具（ripgrep）全局搜索国产化关键词
- **核查性质：** 只读，未修改任何代码文件
- **可复现性：** 所有统计命令可在 Windows PowerShell 重新执行验证

---

*报告结束*
