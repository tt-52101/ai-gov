# 落地推进包：任务计划 · 数据模型 DDL · 二次开发完整计划

基于定版 PRD 2.0.1，目标是在 **TokenHub** 底座上融合 **AxonHub** 计价与策略能力，尽快完成商用可交付版本。

---

# 一、总体里程碑与节奏

| 里程碑 | 目标周期（建议） | 出口标准 |
|--------|------------------|----------|
| **M0 基线就绪** | 第 1–2 周 | TokenHub fork 可编译部署；PG 替代/并行 SQLite；冒烟通 |
| **M1 财务闭环** | 第 3–6 周 | Party/账本/划拨/预算帽/冻结/清算/双轨/价格帽/四轴/ModelGrant 骨架/治理 API 幂等可演示 |
| **M2 策略矩阵** | 第 7–8 周 | 策略启停组合、决策日志、基础仪表盘 |
| **M3 安全与对账** | 第 9–10 周 | 出网/内容钩子实现、变更快照、对账作业 |
| **M4 GA** | 第 11–12 周 | 压测、国产验证、UAT 签字、安装包与文档 |

并行原则：**M1 为主干**；策略接口骨架可在 M1 后半与 M2 重叠；安全钩子空实现在 M1 结束前挂上。

---

# 二、详细任务计划（WBS）

## 阶段 A / M0：基线（约 2 周）

| ID | 任务 | 产出 | 依赖 | 责任建议 |
|----|------|------|------|----------|
| A1 | Fork TokenHub，建立内源仓库与 CI | 仓库、流水线 | — | 架构 |
| A2 | 统一 PostgreSQL 为开发/联调主库（保留 SQLite 仅本地可选） | 配置与文档 | A1 | 后端 |
| A3 | 梳理数据面调用链（Auth→Route→Adapter→Usage）插桩点 | 管道图 + 中间件挂点清单 | A1 | 后端 |
| A4 | 用量规范化接口草案（itemCode 映射） | 接口 + 单测骨架 | A3 | 后端 |
| A5 | 国产环境编译冒烟（目标 CPU/OS 各 1 套） | 冒烟报告 | A1 | 运维/后端 |
| A6 | OpenAPI/错误码枚举初稿（与 PRD 第 6 章对齐） | 错误码常量包 | — | 后端 |

**出口：** 兼容 API 仍可调通；PG 迁移空跑成功；插桩点评审通过。

---

## 阶段 B / M1：财务与权限主干（约 4 周）

### B1 数据与领域（第 3 周）

| ID | 任务 | 产出 |
|----|------|------|
| B1.1 | 执行 DDL（见第三节）与 GORM/Ent 模型 | migration + model |
| B1.2 | Party / Edge / Membership CRUD | 控制面 API + 单测 |
| B1.3 | Account / Ledger / 乐观锁或行锁封装 | FundRepository |
| B1.4 | Freeze 生命周期 + TTL 扫描任务 | 任务 + 流水类型 |
| B1.5 | 预算帽字段读写与判定函数 | BudgetService |

### B2 资金与幂等（第 3–4 周）

| ID | 任务 | 产出 |
|----|------|------|
| B2.1 | 划拨：通道校验 + 守恒事务 | Allocate API |
| B2.2 | 清算状态机 | Liquidate API |
| B2.3 | 幂等表与中间件（Idempotency-Key） | 中间件 + 单测（并发同键） |
| B2.4 | Key.account_id 绑定与鉴权改造 | 数据面 Auth 改造 |
| B2.5 | 数据面：预估 sell → 预算帽 → 冻结 →（暂固定路由）→ 结算 | 主路径打通 |

### B3 计价（第 4–5 周，可与 B2 并行）

| ID | 任务 | 产出 |
|----|------|------|
| B3.1 | ModelPrice JSON 存储与校验 | PRI API |
| B3.2 | CalculateCost/Sell（对齐 AxonHub item 口径 + 双轨） | pricing 包 |
| B3.3 | 价格合格集过滤 + δ（默认 0，硬上限 20%） | 接入数据面 |
| B3.4 | 调用日志写入 cost/sell/cost_items | 落库 |

### B4 授权与模型权限（第 5–6 周）

| ID | 任务 | 产出 |
|----|------|------|
| B4.1 | Grant 四轴判定中间件 | authz 包 |
| B4.2 | ModelGrant 存储与调用前校验 | MODEL_ACCESS_DENIED |
| B4.3 | Leader 模板发 Grant | 控制面 |
| B4.4 | 数据范围过滤（usage/report） | 防 IDOR |
| B4.5 | 治理 API 与管理台：主体/资金/预算帽/Key/价目 | UI + API |
| B4.6 | 安全钩子 no-op 挂入管道 | SEC-05 |

**M1 出口（必须演示）：**

- 独立项目建账 → 划拨 → 人 Key 调用 → 双轨落账  
- 预算帽 90% → `BUDGET_CAP_EXCEEDED`；余额不足 → `INSUFFICIENT_BALANCE`  
- 清算状态机跑通；同幂等键划拨不双记  
- ModelGrant deny 不可调；无 fund 不能划拨  

---

## 阶段 C / M2：策略矩阵（约 2 周）

| ID | 任务 | 产出 |
|----|------|------|
| C1 | Strategy 接口与 Profile 配置 | routing 包 |
| C2 | 实现 S-PRI / S-HEALTH / S-WEIGHT / S-AFFINITY / S-COST 等 | 可启停 |
| C3 | S-COMPLIANCE 硬过滤 | INTERNAL_ONLY 可测 |
| C4 | 决策日志关联 request_id | 可观测 |
| C5 | 档案 UI/API | 运维可配 |
| C6 | 基础仪表盘（调用量、余额、预算消耗、错误码分布） | 前端 |

**出口：** 档案切换行为可测；价格帽外候选永不调用。

---

## 阶段 D / M3：安全与对账（约 2 周）

| ID | 任务 | 产出 |
|----|------|------|
| D1 | 出网/网络策略实现 | SEC-01/02 |
| D2 | 内容安全适配器（可接外部引擎） | SEC-03 |
| D3 | 配置变更 before/after 快照 | AUD-02 |
| D4 | 上游账单导入与 cost 差异报告 | AUD-03 |
| D5 | 流式冻结续期压测与边界 | 稳定性 |

---

## 阶段 E / M4：GA（约 2 周）

| ID | 任务 | 产出 |
|----|------|------|
| E1 | 性能与可用性压测报告 | 报告 |
| E2 | 国产环境回归 | 报告 |
| E3 | 全量 UAT 与门禁签字 | 签字单 |
| E4 | 安装包、OpenAPI、运维手册、NOTICE | 交付物 |
| E5 | 生产发布与回滚演练 | 演练记录 |

---

# 三、数据模型 DDL（PostgreSQL，薄表）

> 原则：热字段原子；复杂进 JSON；不建能力维宽表。金额建议 `NUMERIC(20,8)` 或项目统一精度。

```sql
-- ========== 主体 ==========
CREATE TABLE parties (
  id              BIGSERIAL PRIMARY KEY,
  type            VARCHAR(16)  NOT NULL CHECK (type IN ('org', 'project')),
  name            VARCHAR(256) NOT NULL,
  status          VARCHAR(16)  NOT NULL DEFAULT 'active',
  leader_user_id  BIGINT,
  starts_at       TIMESTAMPTZ,
  ends_at         TIMESTAMPTZ,
  meta            JSONB,
  created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE party_edges (
  id             BIGSERIAL PRIMARY KEY,
  from_party_id  BIGINT NOT NULL REFERENCES parties(id),
  to_party_id    BIGINT NOT NULL REFERENCES parties(id),
  edge_type      VARCHAR(32) NOT NULL, -- parent|sponsors|owns|participates
  meta           JSONB,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (from_party_id, to_party_id, edge_type)
);

CREATE TABLE memberships (
  id         BIGSERIAL PRIMARY KEY,
  party_id   BIGINT NOT NULL REFERENCES parties(id),
  user_id    BIGINT NOT NULL,
  role       VARCHAR(64),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (party_id, user_id)
);

-- ========== 账本 / 预算帽 / 流水 / 冻结 ==========
CREATE TABLE accounts (
  id                      BIGSERIAL PRIMARY KEY,
  party_id                BIGINT NOT NULL REFERENCES parties(id) UNIQUE,
  balance                 NUMERIC(20,8) NOT NULL DEFAULT 0,
  frozen                  NUMERIC(20,8) NOT NULL DEFAULT 0,
  version                 BIGINT NOT NULL DEFAULT 0,
  budget_limit_amount     NUMERIC(20,8),
  budget_warn_ratio       NUMERIC(8,4),
  budget_period           VARCHAR(32),
  budget_period_start     TIMESTAMPTZ,
  budget_period_end       TIMESTAMPTZ,
  budget_consumed_amount  NUMERIC(20,8) NOT NULL DEFAULT 0,
  budget_version          BIGINT NOT NULL DEFAULT 0,
  status                  VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ledgers (
  id               BIGSERIAL PRIMARY KEY,
  account_id       BIGINT NOT NULL REFERENCES accounts(id),
  peer_account_id  BIGINT REFERENCES accounts(id),
  type             VARCHAR(32) NOT NULL, -- allocate_in/out, freeze, unfreeze, settle, liquidate, ...
  amount           NUMERIC(20,8) NOT NULL,
  balance_after    NUMERIC(20,8),
  frozen_after     NUMERIC(20,8),
  request_id       VARCHAR(64),
  freeze_id        VARCHAR(64),
  idempotency_key  VARCHAR(255),
  operator_id      BIGINT,
  remark           TEXT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_ledgers_account_created ON ledgers(account_id, created_at DESC);

CREATE TABLE freezes (
  id           VARCHAR(64) PRIMARY KEY, -- freeze_id
  account_id   BIGINT NOT NULL REFERENCES accounts(id),
  amount       NUMERIC(20,8) NOT NULL,
  status       VARCHAR(16) NOT NULL, -- open|settled|timeout_released|cancelled
  request_id   VARCHAR(64),
  expires_at   TIMESTAMPTZ NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_freezes_expires ON freezes(status, expires_at);

-- ========== 幂等 ==========
CREATE TABLE idempotency_records (
  id               BIGSERIAL PRIMARY KEY,
  scope            VARCHAR(64)  NOT NULL,
  actor_id         VARCHAR(128) NOT NULL,
  idempotency_key  VARCHAR(255) NOT NULL,
  request_hash     CHAR(64)     NOT NULL,
  status           VARCHAR(16)  NOT NULL, -- started|succeeded|failed
  response_code    INT,
  response_body    JSONB,
  resource_ref     VARCHAR(128),
  created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
  expires_at       TIMESTAMPTZ  NOT NULL,
  UNIQUE (scope, actor_id, idempotency_key)
);
CREATE INDEX idx_idem_expires ON idempotency_records(expires_at);

-- ========== 价目 ==========
CREATE TABLE model_prices (
  id          BIGSERIAL PRIMARY KEY,
  channel_id  BIGINT NOT NULL,
  model_id    VARCHAR(128) NOT NULL,
  price       JSONB NOT NULL, -- items[] cost/sell + schedule
  status      VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (channel_id, model_id)
);

-- ========== 授权 / 模型授权 ==========
CREATE TABLE grants (
  id            BIGSERIAL PRIMARY KEY,
  subject_type  VARCHAR(16) NOT NULL, -- person|role
  subject_id    VARCHAR(128) NOT NULL,
  axis          VARCHAR(16) NOT NULL, -- data|fund|iam|routing
  action        VARCHAR(64) NOT NULL,
  resource_type VARCHAR(32) NOT NULL DEFAULT 'party',
  resource_id   VARCHAR(128) NOT NULL, -- party_id or *
  effect        VARCHAR(8)  NOT NULL DEFAULT 'allow',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_grants_subject ON grants(subject_type, subject_id);

CREATE TABLE model_grants (
  id              BIGSERIAL PRIMARY KEY,
  principal_type  VARCHAR(16) NOT NULL, -- party|person|key|role
  principal_id    VARCHAR(128) NOT NULL,
  model_key       VARCHAR(128) NOT NULL, -- model_id or tag:xxx
  effect          VARCHAR(8)  NOT NULL, -- allow|deny
  priority        INT NOT NULL DEFAULT 0,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_model_grants_principal ON model_grants(principal_type, principal_id);

-- ========== 划拨白名单（可选） ==========
CREATE TABLE allocate_whitelist (
  id             BIGSERIAL PRIMARY KEY,
  from_party_id  BIGINT NOT NULL REFERENCES parties(id),
  to_party_id    BIGINT NOT NULL REFERENCES parties(id),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (from_party_id, to_party_id)
);

-- ========== Key 扩展（在现有 api_keys 表上增量） ==========
-- ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS account_id BIGINT REFERENCES accounts(id);
-- ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS owner_user_id BIGINT;

-- ========== 用量日志扩展 ==========
-- ALTER TABLE usage_logs / request_logs ADD COLUMN cost_amount NUMERIC(20,8);
-- ADD COLUMN sell_amount NUMERIC(20,8);
-- ADD COLUMN cost_items JSONB;
-- ADD COLUMN account_id BIGINT;
-- ADD COLUMN freeze_id VARCHAR(64);
-- ADD COLUMN party_id BIGINT;
```

**说明：** `users`、`api_keys`、`channels`、原路由/用量表优先 **ALTER 增量**，不复制平行大表。表名以 TokenHub 现网为准做映射。

---

# 四、二次开发推进计划（融合吸收）

## 4.1 仓库与分支策略

```text
main          —— 可发布
develop       —— 集成分支
feature/m1-*  —— 财务/权限
feature/m2-*  —— 策略
feature/m3-*  —— 安全对账
```

上游 TokenHub 定期 cherry-pick；治理表与包使用明确前缀/目录，减少冲突面。

## 4.2 代码结构（建议）

```text
backend/internal/
  fund/           # Account Ledger Freeze Allocate Liquidate Budget
  pricing/        # ModelPrice Calculate dual-track normalize usage
  idempotency/    # Claim Complete Replay
  party/          # Party Edge Membership
  authz/          # Grant 四轴
  modelgrant/     # ModelGrant 校验
  routing/        # Strategy Profile PriceCap filter
  security/       # Hooks
  gateway/        # 数据面管道编排（中间件链）
```

## 4.3 AxonHub 吸收清单（做什么 / 不做什么）

| 吸收 | 做法 |
|------|------|
| itemCode 与分项计算 | **重实现**于 `pricing`（按公开文档/口径），双轨 cost/sell |
| 价目 JSON 形态 | 对齐 items + schedule，存 `model_prices.price` |
| 缓存不双计 | 规范化与 Calculate 内固定公式 + 单测 |
| 多维评分思想 | 实现为 Strategy（S-COST/S-LATENCY/S-ERROR 等），非整仓移植 |
| Trace UI | M2/M3 增强决策与调用链，不必一期对等 AxonHub 全 Trace |

**不整仓替换 TokenHub**；不引入与资金无关的重依赖。

## 4.4 数据面管道改造顺序（关键路径）

```text
1. 现有：Auth(Key) → Route → Adapter → Log
2. 改为：
   Auth(Key+account+user)
   → SecurityHooks (noop → real)
   → ModelGrant
   → Estimate P_request (pricing)
   → BuildCandidates (现有路由)
   → PriceCapFilter (δ)
   → BudgetCapCheck
   → Freeze
   → StrategySelect (M1 可先 priority-only，M2 换矩阵)
   → Adapter
   → NormalizeUsage
   → Calculate cost/sell
   → Settle
   → Persist usage + decision
```

每步可独立单测；M1 结束前主路径必须实扣可关（feature flag：只记双轨不冻扣）。

## 4.5 控制面 API 优先清单（M1）

| 方法 | 路径示例 | 幂等 |
|------|----------|------|
| POST | /api/gov/parties | 否 |
| POST | /api/gov/party-edges | 否 |
| POST | /api/gov/accounts/{id}/allocate | **是** |
| POST | /api/gov/accounts/{id}/liquidate | **是** |
| PATCH | /api/gov/accounts/{id}/budget | 否（审计） |
| POST | /api/gov/keys | 否 |
| PUT | /api/gov/model-prices | 否 |
| POST | /api/gov/grants | 否 |
| POST | /api/gov/model-grants | 否 |
| GET | /api/gov/accounts/{id}/ledgers | 否 |

头：`Idempotency-Key`；错误码 body.`code` 与 PRD 第 6 章一致。

## 4.6 测试策略

| 类型 | 重点 |
|------|------|
| 单测 | Calculate 分项、预算帽边界、划拨守恒、幂等并发、δ 过滤 |
| 集成 | 主路径冻结结算、清算 drain、ModelGrant |
| 契约 | OpenAPI 与错误码 |
| UAT | PRD 第 13 章脚本 |
| 性能 | 冻结热点账户、TTL 任务、路由 P99 |

## 4.7 风险与缓解

| 风险 | 缓解 |
|------|------|
| 与 TokenHub 上游合并冲突 | 治理代码进独立 package；少改 http 核心，多挂中间件 |
| 计价口径与上游账单差 | usage_incomplete + 对账分类；适配器契约测试 |
| 热点账户锁争用 | 行锁粒度、预扣超时、监控等待时间 |
| 范围蔓延 | 严格按 M1→M2→M3；内容安全只接钩子 |

## 4.8 人力建议（示例）

| 角色 | 人数 | 重心 |
|------|------|------|
| 后端 | 2–3 | 资金管道、计价、授权 |
| 前端 | 1–2 | 主体/资金/价目/授权/档案 |
| 测试 | 1 | 用例与门禁 |
| 架构/运维 | 0.5–1 | 基线、发布、国产 |

---

# 五、第一周立刻可执行清单（启动）

1. 完成 A1–A3（fork、PG、管道插桩点评审）。  
2. 合入第三节 DDL（或分 migration：party/account → freeze/idem → price/grant）。  
3. 建立 `internal/fund` 与 `internal/pricing` 空包与接口定义。  
4. 错误码常量与 OpenAPI 骨架与 PRD 对齐。  
5. 排定 M1 演示日（建议第 6 周末）并冻结演示脚本。

---

# 六、成功标准（与 PRD 对齐的最短集）

- 财务演示脚本全绿（含预算帽/余额不足分码、清算、幂等）。  
- 价格帽外零上游调用。  
- ModelGrant deny 零调用。  
- 四轴越权用例全绿。  
- 交付物清单（安装包、OpenAPI、迁移、UAT、监控、NOTICE、手册）齐全。  
