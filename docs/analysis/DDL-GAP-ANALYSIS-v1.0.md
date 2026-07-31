# AI-GOV 融合架构 DDL 深度差距分析与优化治理方案

> **基线：** `./schema/ai-gov.sql` (6553 行, PostgreSQL 16, 69 表, `ai_governance` schema)  
> **目标：** `./docs/prd/AI-GOV-PRD-v2.0.1.md`  
> **治理面：** TokenHub 底座 + AxonHub 高收益能力吸收  
> **目标：** 最短时间内完成融合系统产品交付  
> **日期：** 2026-07-31

---

## 一、现网数据库全景评估

### 1.1 69 表功能域分类

| 领域 | 表数量 | 关键表 | 评估 |
|------|--------|--------|------|
| **多租户** | 1 | `tenants` | ✅ |
| **身份/人员** | 7 | `identity_subjects`, `identity_issuer_configs`, `user_profiles`, `user_login_identities`, `user_password_credentials`, `user_sessions`, `user_contact_points` | ✅ 完备 |
| **组织/项目/成员** | 4 | `organizations`, `projects`, `organization_memberships`, `project_memberships` | ⚠️ 分离设计 |
| **授权 (ABAC)** | 6 | `sys_action_catalogs`, `sys_roles`, `sys_role_permissions`, `sys_subject_role_bindings`, `sys_access_policies`, `sys_access_policy_bindings` | ✅ 完备 |
| **API Key** | 4 | `user_api_keys`, `user_api_key_secret_versions`, `key_limit_policies`, `key_usage_counter_projections` | ✅ 完备 |
| **OAuth** | 1 | `oauth_provider_configs` | ✅ |
| **模型目录** | 5 | `model_providers`, `model_catalog_entries`, `model_channels`, `model_channel_health_events`, `provider_credential_refs` | ✅ |
| **模型授权 (ModelGrant)** | 1 | `model_access_policies` | ✅ 含 ALLOW/DENY + scope |
| **定价** | 1 | `credit_rate_versions` | ❌ **单轨** |
| **资金账户** | 2 | `funding_accounts`, `balance_projections` | ⚠️ 缺预算帽 |
| **用户额度** | 2 | `user_allocations`, `user_governance_projections` | ✅ |
| **复式账本** | 2 | `ledger_transactions`, `ledger_legs` | ✅ 14 种 entry_type |
| **额度预占** | 1 | `quota_reservations` | ⚠️ 缺流式续期 |
| **紧急信用** | 2 | `emergency_credit_grants`, `emergency_credit_grant_keys` | ✅ 超出 PRD |
| **路由** | 2 | `route_policies`, `route_policy_candidates` | ❌ 顺序 failover |
| **用量** | 3 | `usage_requests`, `usage_attempts`, `usage_events` | ⚠️ 缺双轨分项 |
| **安全防火墙** | 4 | `prompt_firewall_policies/rules/decisions`, `safety_events` | ✅ |
| **审计** | 2 | `audit_events` (哈希链), `audit_chain_anchors` | ✅ 超越 PRD |
| **审批** | 2 | `approval_requests`, `approval_decisions` | ✅ |
| **对账** | 5 | `reconciliation_runs/items/differences/resolutions`, `closing_snapshots` | ✅ |
| **用户日结** | 1 | `user_closing_snapshot_items` | ✅ |
| **完整性** | 1 | `data_integrity_findings` | ✅ |
| **事件** | 2 | `outbox_events`, `inbox_receipts` | ✅ |
| **基础设施** | 5 | `runtime_snapshots`, `projection_checkpoints`, `schema_migration_contracts`, `auth_challenges`, `auth_events` | ✅ |
| **UI/导航** | 3 | `sys_ui_menus`, `sys_ui_routes`, `sys_ui_action_bindings` | ✅ |

### 1.2 现有数据库设计亮点（直接复用）

1. **UUIDv7 主键 + tenant_id 隔离键**：全局不透明，分片友好，每表强制
2. **row_version 乐观锁**：防并发覆盖
3. **软删除规范**：`is_deleted + deleted_at + deleted_by + delete_reason` 四件套（30+ 表采用）
4. **追加式不可变触发器**：`trg_append_only`（审计/资金事实禁止 UPDATE）、`trg_no_physical_delete`
5. **CHECK 约束枚举**：所有受控字段有值域约束
6. **BRIN 索引**：`audit_events.occurred_at` 等时间序列高效扫描
7. **audit_event_id 全链路**：所有资金写携带审计链引用
8. **trace_id 全链路**：`Request→Attempt→Usage→Ledger→Audit` 贯通
9. **逻辑引用无 FK**：写入时应用层校验而非数据库外键，配合 `data_integrity_findings` 检测孤儿引用
10. **三层余额体系**：`funding_accounts`（账户配置）+ `ledger_legs`（复式分录事实）+ `balance_projections`（可重建投影，不可写）
11. **复式账本 14 种 entry_type**：ISSUE / ALLOCATE_TO_SCOPE / ALLOCATE_TO_USER / RECLAIM_FROM_USER / RESERVE / SUPPLEMENTAL_RESERVE / SETTLE / RELEASE / ADJUST / REVERSAL + 4 种 EMERGENCY_*
12. **对账体系完备**：`reconciliation_runs→items→differences→resolutions` 含 12 种差异类型

---

## 二、PRD 需求 vs 现网 DDL 逐项差距矩阵

### 2.1 核心差距总览（已确认）

| 序号 | PRD 需求 | 现网状态 | 差距等级 | 治理动作 |
|------|----------|----------|----------|----------|
| G-01 | **预算帽** §5 | ❌ 完全缺失 | **P0 阻断** | `funding_accounts` + `balance_projections` 加字段 |
| G-02 | **双轨计价 (cost/sell)** §4 | ❌ 单轨 (`credit_rate_versions`) | **P0 阻断** | 新建 `model_prices` 表 |
| G-03 | **冻结续期 (流式 TTL)** §8.3 | ⚠️ 有 `quota_reservations` 但缺续期 | **P1** | 加 `renewal_count`/`last_renewed_at`/`max_lifetime_expires_at` |
| G-04 | **治理 API 幂等记录表** §8.7 | ❌ 无独立表 | **P0 阻断** | 新建 `idempotency_records` |
| G-05 | **Party 统一边表** §2.4 | ❌ 无统一边表 | **P0 架构** | 新建 `party_edges` |
| G-06 | **策略矩阵引擎** §3.3 | ❌ 顺序路由 | **P1** | 新建 `strategy_profiles` |
| G-07 | **价格帽 δ** §8.1 | ❌ 缺失 | **P0 阻断** | `route_policies` 加字段 |
| G-08 | **cost_items / 用量分项** §4.1 | ❌ 单值 (`settled_credit_amount`) | **P0** | `usage_events` 加 JSONB 字段 |
| G-09 | **清算状态机** §8.4 | ⚠️ `funding_accounts.status=FROZEN/CLOSED` | **P1** | 加 `liquidation_stage` 等字段 |
| G-10 | **计费模式多样性** §4.2 | ❌ 仅 `usage_per_unit` | **P1** | 进 `model_prices.price_items_json` |

---

## 三、逐项详细差距分析与 DDL 优化方案

### 3.1 G-01: 预算帽字段（P0 阻断 — 最快速交付）

**PRD §5 要求**：`funding_accounts`(accounts) 需包含预算帽配置字段。

**现网状态**：`funding_accounts` 和 `balance_projections` **均无**预算帽相关字段。

**优化方案**：配置字段挂 `funding_accounts`，周期累计挂 `balance_projections`。

```sql
-- Phase 1: funding_accounts 加配置字段
ALTER TABLE ai_governance.funding_accounts
  ADD COLUMN budget_limit_amount      BIGINT,                -- NULL=未启用
  ADD COLUMN budget_warn_ratio        NUMERIC(5,4),          -- 如 0.8000，仅告警不阻断
  ADD COLUMN budget_period            VARCHAR(24) DEFAULT 'none',  -- none/calendar_month/calendar_day/custom
  ADD COLUMN budget_period_start      TIMESTAMPTZ,
  ADD COLUMN budget_period_end        TIMESTAMPTZ,
  ADD COLUMN budget_version           BIGINT NOT NULL DEFAULT 1;  -- 配置乐观锁

ALTER TABLE ai_governance.funding_accounts
  ADD CONSTRAINT ck_budget_limit CHECK (budget_limit_amount IS NULL OR budget_limit_amount >= 0),
  ADD CONSTRAINT ck_budget_warn  CHECK (budget_warn_ratio IS NULL OR (budget_warn_ratio >= 0 AND budget_warn_ratio <= 1)),
  ADD CONSTRAINT ck_budget_period CHECK (budget_period IN ('none','calendar_month','calendar_day','custom'));

-- Phase 2: balance_projections 加消费累计
ALTER TABLE ai_governance.balance_projections
  ADD COLUMN budget_consumed_credit BIGINT NOT NULL DEFAULT 0;
```

**效果**：预算帽与余额不足分码 — `BUDGET_CAP_EXCEEDED` vs `INSUFFICIENT_BALANCE`。

### 3.2 G-02 + G-10: 双轨计价模型价目表（P0 阻断 — 对齐 AxonHub）

**PRD §4 要求**：
- cost（上游成本）/ sell（内部结算价）双轨
- 5 种 itemCode：prompt_tokens, completion_tokens, prompt_cached_tokens, prompt_write_cached_tokens, completion_reasoning_tokens
- 4 种计价模式：flat_fee, usage_per_unit, usage_tiered, usage_volume
- 渠道 × 模型 JSON 价目 + schedule 分时

**现网状态**：`credit_rate_versions` 只有 `input_credit_per_million_tokens` + `output_credit_per_million_tokens`（单轨、单模式、无渠道绑定、无 itemCode）。

**优化方案**：新建 `model_prices` 表，`credit_rate_versions` 保留只读兼容过渡。

```sql
CREATE TABLE ai_governance.model_prices (
  id                  UUID NOT NULL,
  tenant_id           UUID NOT NULL,
  price_code          VARCHAR(96) NOT NULL,
  model_id            UUID NOT NULL,            -- → model_catalog_entries
  channel_id          UUID,                     -- NULL=默认价；非NULL=渠道特化
  -- 核心：双轨+itemCode+多模式 JSON（AxonHub 对齐）
  price_items_json    JSONB NOT NULL,           -- 见下方示例
  schedule_json       JSONB DEFAULT '{}',       -- {timezone, overrides:[{start,end,items}]}
  valid_from          TIMESTAMPTZ NOT NULL,
  valid_until         TIMESTAMPTZ,
  status              VARCHAR(24) NOT NULL DEFAULT 'DRAFT',
  revision            BIGINT NOT NULL DEFAULT 1,
  row_version         BIGINT NOT NULL DEFAULT 1,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  created_by          VARCHAR(128) NOT NULL,
  updated_by          VARCHAR(128) NOT NULL,
  operation_source    VARCHAR(32) NOT NULL DEFAULT 'ADMIN',
  operation_trace_id  VARCHAR(128),
  is_deleted          BOOL NOT NULL DEFAULT false,
  deleted_at          TIMESTAMPTZ,
  deleted_by          VARCHAR(128),
  delete_reason       VARCHAR(512),
  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX uq_model_price ON ai_governance.model_prices(
  tenant_id, price_code
) WHERE is_deleted = false;

CREATE INDEX ix_model_price_model ON ai_governance.model_prices(
  tenant_id, model_id, channel_id, status
) WHERE status = 'ACTIVE';
```

**price_items_json 示例**（对齐 PRD §4.4）：

```json
{
  "items": [
    {"itemCode": "prompt_tokens",
     "cost": {"mode": "usage_per_unit", "rate": 1500},
     "sell": {"mode": "usage_per_unit", "rate": 2000}},
    {"itemCode": "completion_tokens",
     "cost": {"mode": "usage_per_unit", "rate": 6000},
     "sell": {"mode": "usage_per_unit", "rate": 8000}},
    {"itemCode": "prompt_cached_tokens",
     "cost": {"mode": "usage_per_unit", "rate": 750},
     "sell": {"mode": "usage_per_unit", "rate": 1000}},
    {"itemCode": "prompt_write_cached_tokens",
     "cost": {"mode": "usage_per_unit", "rate": 3000},
     "sell": {"mode": "usage_per_unit", "rate": 4000}},
    {"itemCode": "completion_reasoning_tokens",
     "cost": {"mode": "usage_per_unit", "rate": 12000},
     "sell": {"mode": "usage_per_unit", "rate": 16000}}
  ],
  "schedule": {"timezone": "Asia/Shanghai", "overrides": []}
}
```

**rate 单位**：AI Credit per million tokens（P0 不表达人民币，对齐现有 `credit_rate_versions` 整数体系）。

### 3.3 G-03: 冻结/预占续期增强（P1 — 流式场景）

**PRD §8.3 要求**：流式场景网关自动续期 `expires_at`，不增加冻结金额，累计有效期上限可配置。

**现网状态**：`quota_reservations` 已存在且完备（reservation_id, request_id, key_id, user_allocation_id, initial_reserved_credit, total_reserved_credit, expires_at, status），**但缺少续期相关字段**。

**优化方案**：最小化扩展，只加续期字段。

```sql
-- quota_reservations 已存在，只需加 3 个字段
ALTER TABLE ai_governance.quota_reservations
  ADD COLUMN renewal_count           INT NOT NULL DEFAULT 0,     -- 续期次数
  ADD COLUMN last_renewed_at         TIMESTAMPTZ,                -- 最近续期时间
  ADD COLUMN max_lifetime_expires_at TIMESTAMPTZ;                -- 累计有效期硬上限（如 2h）

-- 后台扫描过期预占的索引
CREATE INDEX ix_reservation_expiry ON ai_governance.quota_reservations(
  tenant_id, status, expires_at
) WHERE status IN ('RESERVED', 'IN_FLIGHT');
```

**与 ledger 关系**：`quota_reservations` 管 TTL 生命周期，`ledger_transactions(RESERVE/SUPPLEMENTAL_RESERVE)` 管资金记账，双方通过 `reservation_id` 互引。

### 3.4 G-04: 治理 API 幂等记录表（P0 阻断）

**PRD §8.7 要求**：划拨、清算、补偿等资金写接口必须 `Idempotency-Key`，`INSERT ON CONFLICT` 原子抢占。

**现网状态**：`ledger_transactions.idempotency_key` 和 `usage_requests.idempotency_key` 存在，但**无独立 API 幂等表**。

**优化方案**：

```sql
CREATE TABLE ai_governance.idempotency_records (
  id                UUID NOT NULL,
  tenant_id         UUID NOT NULL,
  scope             VARCHAR(64) NOT NULL,        -- allocate/liquidate/compensate
  actor_id          VARCHAR(128) NOT NULL,
  idempotency_key   VARCHAR(256) NOT NULL,       -- Idempotency-Key 请求头
  request_hash      VARCHAR(71) NOT NULL,        -- SHA-256 请求体
  status            VARCHAR(16) NOT NULL DEFAULT 'started',  -- started/succeeded/failed
  response_json     JSONB,                       -- 首次成功响应
  resource_ref      VARCHAR(256),                -- 如 transaction_id
  expires_at        TIMESTAMPTZ NOT NULL,        -- 窗口 ≥24h
  created_at        TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (id)
);

-- 原子抢占唯一约束
CREATE UNIQUE INDEX uq_idempotency ON ai_governance.idempotency_records(
  scope, actor_id, idempotency_key
);

CREATE INDEX ix_idempotency_expiry ON ai_governance.idempotency_records(expires_at);

ALTER TABLE ai_governance.idempotency_records
  ADD CONSTRAINT ck_idempotency_status CHECK (status IN ('started','succeeded','failed'));
```

**使用模式**：`INSERT ON CONFLICT (scope, actor_id, idempotency_key) DO NOTHING RETURNING ...` — 同键同指纹 → 重放结果；同键异指纹 → `IDEMPOTENCY_CONFLICT`(409)。

### 3.5 G-05: Party 边关系表（P0 架构）

**PRD §2.4 要求**：统一边表支持 parent / sponsors / owns / participates 四种类型。

**现网状态**：
- `organizations.parent_organization_id` — 仅层级
- `projects.sponsor_organization_id` — 仅赞助
- **无跨类型边表**（org↔project 无直接关系记录）

**优化方案**：新建边表，不替代现有字段，作为统一查询面。

```sql
CREATE TABLE ai_governance.party_edges (
  id                          UUID NOT NULL,
  tenant_id                   UUID NOT NULL,
  edge_code                   VARCHAR(96) NOT NULL,
  source_type                 VARCHAR(24) NOT NULL,    -- ORGANIZATION / PROJECT
  source_id                   UUID NOT NULL,
  target_type                 VARCHAR(24) NOT NULL,    -- ORGANIZATION / PROJECT
  target_id                   UUID NOT NULL,
  relationship_type           VARCHAR(24) NOT NULL,    -- parent/sponsors/owns/participates
  allow_downstream_allocation BOOL NOT NULL DEFAULT false,  -- 划拨方向
  valid_from                  TIMESTAMPTZ NOT NULL,
  valid_until                 TIMESTAMPTZ,
  status                      VARCHAR(24) NOT NULL DEFAULT 'ACTIVE',
  revision                    BIGINT NOT NULL DEFAULT 1,
  row_version                 BIGINT NOT NULL DEFAULT 1,
  created_at                  TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  updated_at                  TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  created_by                  VARCHAR(128) NOT NULL,
  updated_by                  VARCHAR(128) NOT NULL,
  operation_source            VARCHAR(32) NOT NULL DEFAULT 'ADMIN',
  operation_trace_id          VARCHAR(128),
  is_deleted                  BOOL NOT NULL DEFAULT false,
  deleted_at                  TIMESTAMPTZ,
  deleted_by                  VARCHAR(128),
  delete_reason               VARCHAR(512),
  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX uq_party_edge ON ai_governance.party_edges(
  tenant_id, source_type, source_id, target_type, target_id, relationship_type
) WHERE is_deleted = false;

ALTER TABLE ai_governance.party_edges
  ADD CONSTRAINT ck_edge_no_self CHECK (
    NOT (source_type = target_type AND source_id = target_id)
  ),
  ADD CONSTRAINT ck_edge_relationship CHECK (
    relationship_type IN ('parent','sponsors','owns','participates')
  );
```

**默认划拨规则**（对齐 PRD §8.2）：
- parent → `allow_downstream_allocation=true`（上级→下级）
- sponsors → `allow_downstream_allocation=true`（出资方→被出资方）
- owns → `false`
- participates → `false`

**迁移**：`organizations.parent_organization_id` → parent 边；`projects.sponsor_organization_id` → sponsors 边。

### 3.6 G-06: 策略矩阵引擎（P1）

**PRD §3.3 要求**：11 种可插拔策略（S-PRI/S-HEALTH/S-WEIGHT/S-AFFINITY/S-COST/S-LATENCY/S-ERROR/S-RATE/S-TAG/S-COMPLIANCE/S-CACHE），可组合启停。

**现网状态**：`route_policies` + `route_policy_candidates` 是顺序 failover 模型。

**优化方案**：

```sql
CREATE TABLE ai_governance.strategy_profiles (
  id                UUID NOT NULL,
  tenant_id         UUID NOT NULL,
  profile_code      VARCHAR(64) NOT NULL,
  profile_name      VARCHAR(160) NOT NULL,
  strategies_json   JSONB NOT NULL DEFAULT '[]',
  -- [{"code":"S-COMPLIANCE","enabled":true,"priority":0},
  --  {"code":"S-PRI","enabled":true,"priority":10,"config":{"groups":[...]}},
  --  {"code":"S-COST","enabled":true,"priority":30}]
  scope_type        VARCHAR(24) NOT NULL DEFAULT 'TENANT',
  scope_id          UUID,
  status            VARCHAR(24) NOT NULL DEFAULT 'DRAFT',
  revision          BIGINT NOT NULL DEFAULT 1,
  row_version       BIGINT NOT NULL DEFAULT 1,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
  created_by        VARCHAR(128) NOT NULL,
  updated_by        VARCHAR(128) NOT NULL,
  operation_source  VARCHAR(32) NOT NULL DEFAULT 'ADMIN',
  operation_trace_id VARCHAR(128),
  is_deleted        BOOL NOT NULL DEFAULT false,
  deleted_at        TIMESTAMPTZ,
  deleted_by        VARCHAR(128),
  delete_reason     VARCHAR(512),
  PRIMARY KEY (id)
);

CREATE UNIQUE INDEX uq_strategy_profile ON ai_governance.strategy_profiles(
  tenant_id, profile_code
) WHERE is_deleted = false;
```

### 3.7 G-07: 价格帽 δ（P0 阻断）

**PRD §8.1 要求**：`route_policies` 含 δ 百分比，默认 0%，硬上限 20%。

```sql
ALTER TABLE ai_governance.route_policies
  ADD COLUMN price_cap_delta_bps INT NOT NULL DEFAULT 0;  -- 基点(万分比), 0=0%, 2000=20%

ALTER TABLE ai_governance.route_policies
  ADD CONSTRAINT ck_price_cap_delta CHECK (price_cap_delta_bps >= 0 AND price_cap_delta_bps <= 2000);
```

### 3.8 G-08: cost_items 用量分项（P0）

**PRD §4.5 要求**：`usage_events` 需含 itemCode 级别 cost/sell 明细。

```sql
ALTER TABLE ai_governance.usage_events
  ADD COLUMN cost_items_json       JSONB DEFAULT '{}',      -- {itemCode: tokens}
  ADD COLUMN cost_breakdown_json   JSONB DEFAULT '{}',      -- {itemCode: cost_credit}
  ADD COLUMN sell_breakdown_json   JSONB DEFAULT '{}',      -- {itemCode: sell_credit}
  ADD COLUMN upstream_cost_credit  BIGINT,                   -- 上游总成本
  ADD COLUMN internal_sell_credit  BIGINT,                   -- 内部总结算
  ADD COLUMN usage_incomplete      BOOL NOT NULL DEFAULT false;  -- 用量不完整标记
```

### 3.9 G-09: 清算状态机（P1）

**PRD §8.4 要求**：`active → liquidating_block_new → liquidating_drain → liquidating_transfer → liquidated`

```sql
ALTER TABLE ai_governance.funding_accounts
  ADD COLUMN liquidation_stage                VARCHAR(24),
  ADD COLUMN liquidation_target_account_id    UUID,
  ADD COLUMN liquidation_started_at           TIMESTAMPTZ,
  ADD COLUMN liquidation_completed_at         TIMESTAMPTZ,
  ADD COLUMN liquidation_drain_timeout_min    INT DEFAULT 60;

ALTER TABLE ai_governance.funding_accounts
  ADD CONSTRAINT ck_liquidation_stage CHECK (
    liquidation_stage IS NULL OR
    liquidation_stage IN ('block_new','drain','transfer','completed')
  );
```

---

## 四、治理优先级与最小交付路径

### 4.1 四阶段治理路线

| 阶段 | 工作项 | 估算工期 |
|------|--------|----------|
| **A (立即)** | ① `funding_accounts` + `balance_projections` 预算帽 (0.5d) | **1.5d** |
| | ② `idempotency_records` 表 (0.5d) | |
| | ③ `party_edges` 表 (0.5d) | |
| **B (核心)** | ④ `model_prices` 表 (2d) | **4d** |
| | ⑤ `usage_events` 双轨分项 (0.5d) | |
| | ⑥ `route_policies` δ 字段 (0.5d) | |
| | ⑦ `strategy_profiles` 表 (1d) | |
| **C (增强)** | ⑧ `quota_reservations` 续期 (0.5d) | **1.5d** |
| | ⑨ `funding_accounts` 清算状态 (0.5d) | |
| | ⑩ 数据迁移脚本 (0.5d) | |
| **D (交付)** | ⑪ 索引优化 + 压测验证 + 文档 (1d) | **1d** |

**总工期：约 8 工作日**

### 4.2 最速交付路径（3 天 MVP）

| 天 | 交付物 | 验收标准 |
|----|--------|----------|
| **DAY 1** | 预算帽字段 + 幂等表 + 冻结续期 | 财务演示脚本核心依赖就绪 |
| **DAY 2** | model_prices 双轨表 + usage_events 分项 + δ 字段 | 演示 cost/sell 双轨 + 价格帽过滤 |
| **DAY 3** | party_edges + strategy_profiles 骨架 | 组织模型 + 路由控制完整 |

### 4.3 变更清单汇总

**新建表（4 张）**：`model_prices`, `idempotency_records`, `party_edges`, `strategy_profiles`

**扩展表（5 张）**：`funding_accounts` (+6 列), `balance_projections` (+1 列), `quota_reservations` (+3 列), `usage_events` (+6 列), `route_policies` (+1 列)

**不动表（~60 张）**：所有其他表保持原样

---

## 五、关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 预算帽挂在哪 | `funding_accounts` (配置) + `balance_projections` (累计) | 配置与运行时分离 |
| `credit_rate_versions` 去留 | 保留只读；新建 `model_prices` | 旧表承担 Token→Credit 换算；新表承担双轨多模式价目 |
| 冻结续期 | 扩展现有 `quota_reservations`，不加新表 | 表已完备，加 3 列即可 |
| Party 边 | 新建 `party_edges` + 保留现有字段 | 渐进式，不破坏现有 org/project 表 |
| 策略矩阵 | JSONB in `strategy_profiles` | 策略频繁调整，JSONB 支持快速迭代和版本快照 |
| 双轨金额单位 | 整数 AI Credit（对齐现有体系） | P0 不表达人民币，避免浮点精度问题 |

---

## 六、验收红线对齐

| PRD §7.6 红线 | 数据库保障 |
|---------------|------------|
| 无流水改余额 | `ledger_legs` append-only + `trg_append_only` |
| 划拨无通道 | `party_edges.allow_downstream_allocation` |
| Key 无 account 调用 | `user_api_keys.user_allocation_id` NOT NULL |
| 调度改扣费账户 | `account_id` 在 `usage_requests` 创建时锁定，`route_policies` 不可写 |
| 先调后欠费 | 冻结 → 结算原子流程 + 价格帽 δ 上限 |
| Leader 无 Grant 万能 | `sys_access_policies` + `model_access_policies` 显式授权 |
| iam 建 Key 绑无权账户 | `user_api_keys.user_allocation_id` 写入时校验 iam 轴授权 |
| 预算帽与余额不足同码 | `BUDGET_CAP_EXCEEDED` vs `INSUFFICIENT_BALANCE` 应用层分码 |
| ModelGrant deny 仍可调用 | `model_access_policies.effect=DENY` 优先于 ALLOW |

---

> **结论：现网 ai-gov.sql 是高度成熟的生产级 schema（69 表），覆盖 PRD 约 75% 需求。核心差距精确锁定为 10 项，均通过 4 新表 + 5 扩表增量完成。最速 3 天可交付 MVP，完整治理 8 工作日。无需重构任何现有表结构，所有变更均为增量叠加。**
