# 企业级 AI 智能网关治理平台（Token 治理底座）
## 产品需求与融合架构完整方案 v3.0.0

| 项 | 内容 |
|----|------|
| 文档版本 | **3.0.0（融合架构完整方案）** |
| 文档状态 | 可立项、可详细设计、可开发、可测试、可验收、可商用交付 |
| 日期 | 2026-07-31 |
| 读者 | 产品、研发、测试、UE、财务、安全、架构、交付、运维 |
| 修订说明 | 3.0 = v2.0.1 全部共识 + 基于 TokenHub/AxonHub 源码级深度分析的融合架构 DDL + Party 多态模型优化 + 物理世界缺口补齐 |
| 基线 DDL | `schema/ai-gov-fusion-minimal.sql`（29 表） |
| 底座 | TokenHub v0.4.0（主仓库，Apache 2.0） |
| 吸收 | AxonHub itemCode 计价引擎 + 渠道探针 + 多维负载评分（LGPL-3.0 llm/ 模块仅动态链接） |

---

# 第 0 章 文档说明与设计哲学

## 0.1 文档性质

本文是独立完整的**产品需求规格 + 融合架构详细设计 + DDL 数据模型**三合一文本。基于 TokenHub（Go + GORM + SQLite/PG + Next.js）和 AxonHub（Go + Ent + Uber fx + React + GraphQL）的源码级深度分析，输出可交付的完整方案。阅读与执行无需依赖其它会话。

## 0.2 设计哲学

Token 是 AI 时代的电力。本产品不运营模型供给、不对外倒卖 Token，只在企业内部提供：

1. 统一模型调用入口
2. 精确计量与双轨计价
3. 主体经费账本、划拨与清算
4. 安全控制与不可抵赖审计

**目标闭环：**

```text
模型接入 → 安全拦截 → 智能调度 → 实时扣费 → 资金回笼与清算 → 对账审计
```

## 0.3 核心公理（不可配置取消）

1. **无计量则无管理。**
2. **无归因则无法优化。**
3. **无企业自有调用入口则无法真正掌控 AI 资产与成本。**
4. **无守恒账本则无财务治理护城河。**
5. **调度不得改变扣费账户；内部结算价格受请求锚定与上浮上限约束；禁止先调用后欠费。**
6. **数据范围、资金、身份密钥、调度与模型配置四轴正交授权；禁止一轴推导另一轴特权；模型可用范围与资金扣费主体必须解耦。**
7. **平台管理员也必须经账本服务与流水变更余额，不得提供绕过守恒的后门接口。**

## 0.4 能力优先级

| 优先级 | 能力域 | 说明 |
|--------|--------|------|
| **P0** | 统一接入与密钥托管 | 消灭野连接，TokenHub 原生 6 协议齐全 |
| **P0** | **财务治理与计量闭环** | 双轨 cost/sell、Party 账本、划拨清算、预算帽分码——**第一护城河** |
| **P0** | 正交授权、数据不越权、模型访问治理 | 四轴 + ModelGrant |
| **P0** | 调度与资金强制边界 | 价格帽 δ、账户锁定、禁止先调后欠费 |
| **P0** | 治理 API 与资金写幂等 | 与控制台对等，Idempotency-Key |
| **P1** | 可插拔路由策略矩阵与降本高可用 | 11 种策略可组合，TokenHub 原生 6 策略 + AxonHub 4 模式 × 7 策略吸收 |
| **P1** | 基础运营报表与仪表盘 | |
| **P2** | 内容安全与出网强化、配置变更快照、上游自动对账 | |
| **架构 P0** | 数据面安全扩展点 | 阶段 B 预留钩子 |

---

# 第 1 章 产品综述

## 1.1 要解决的问题

1. 员工与团队私自采购上游 API Key，调用黑箱，无法统一禁用与审计。
2. 财务只能拿到厂商总账单，无法按组织、项目、人员、应用做内部结算与毛利分析。
3. 成本失控：无预算预扣、无实时阻断、无双轨；预算帽与「账户真没钱」若同一错误码则运营无法精细抓手。
4. 合规风险：敏感数据可能经未审批路径出境；配置变更缺少不可篡改留痕。
5. 可用性与成本：仅写死四层降级不足以支撑商业灵活组合；调度不受内部价格约束会击穿账本。

## 1.2 价值主张

| 主张 | 说明 |
|------|------|
| 统一治理 | 全部模型调用经单一企业网关，消灭野连接 |
| 财务闭环 | 主动预算、预扣与实扣、主体间划拨与清算、双轨计量、预算帽分码、对账 |
| 安全合规 | 密钥托管、出网策略、内容安全、全链路与配置变更审计 |
| 降本增效 | 在准确计量与价格约束前提下，按可插拔策略矩阵做成本/健康/权重/亲和等调度 |

## 1.3 产品边界

**本产品负责：**

- 身份、成员、主体（Party：组织与项目）、正交授权、模型访问治理（ModelGrant）
- 上游模型与私有化资源接入、上游密钥托管
- 调用接入（OpenAI 兼容等）、安全拦截扩展点、可插拔路由调度
- Token 计量、双轨计价、账本、划拨、预扣结算、预算帽、清算
- 调用审计、配置变更审计、对账与成本报表
- 管理控制台与对等的控制面治理 API
- 运维可观测

**本产品不负责：**

- Agent / 工作流编排、Skill 市场、面向终端用户的业务对话产品 UI
- 作为云厂商对外售卖 Token 或模型本身
- 同一请求拆分到多个账本分摊扣费
- 默认透支信用（负余额作为正常成功路径）
- 由业务客户端负责冻结续期
- 以公链/联盟链作为热路径余额系统记录

**预留扩展点**（架构级，非功能实现）：对外 Token 售卖预留租户隔离、多层级定价、用量同步接口。

**与上层契约：** OpenAI 兼容 HTTP API（Anthropic Messages / Gemini / Embeddings / Images 六协议兼容）；业务标签仅归因，不替代鉴权与扣费账户。

## 1.4 目标用户

| 角色 | 主要诉求 |
|------|----------|
| 平台管理员 / 架构师 | 部署、资源池、路由档案、系统配置、上游密钥、模型目录 |
| 财务 / 成本控制 | 账本、划拨、内部加价、预算帽、报表、对账、清算 |
| 安全 / 合规 | 出网、拦截、审计检索、保留策略 |
| 组织或项目负责人（Leader） | 本主体经费、成员与 Key、消耗可见、模型范围（在授权内） |
| 实体人（员工/外包） | 持有 Key、合法调用、查看本人用量 |

---

# 第 2 章 物理世界主体模型（Party 多态）

## 2.1 设计出发点

真实企业 AI 经费与编制形态是多态的。系统**不得**假设唯一结构「公司 → 部门 → 项目 → 人」，也**不得**强制「项目必须挂在某个部门下」。四级固定层级（租户→部门→项目→人）是**过度设计**——真实企业是图，不是树。

## 2.2 典型场景（必须全部可落地）

| 场景 | 组织 | 项目 | 关系与资金 |
|------|------|------|------------|
| 部门日常额度 | 有常设部门 | 可不建项目 | 部门主体持有账本，人的 Key 扣部门账本 |
| 部门主责专项 | 有 | 有 | 可用 owns 关联；项目可自有账本 |
| 公司级战略项目 | 多个事业部 | 有，可与部门平级 | 多 sponsors 出资到同一项目账本；独立 Leader |
| 跨主体协作 | 内外部组织 | 有 | 人可只加入项目、不加入内部部门 |
| 矩阵制 | 产品线、区域等 | 有 | 多协作边；**一次调用只扣一个账本** |
| 结项还钱 | — | 结束 | 剩余资金回流；Key 失效或改绑 |
| **个人实验经费** | 部门下发 | — | 部门通过 `allocates` 边拨入个人 Account；个人 Key 扣个人 Account |
| **组织合并** | A+B→C | — | A/B 清算余额划入 C；历史流水关联 A/B |
| **组织拆分** | C→A+B | — | C 按比例划出到 A/B |

## 2.3 统一主体 Party

- **组织（org）与项目（project）同一层语义**，均可：持有账本、拥有成员、指定 Leader、作为授权资源。
- **项目不必然从属于某个组织。**
- 控制台可分「组织」「项目」导航，底层模型与权限引擎必须统一。
- **DDL 映射：** `parties` 表，`type` 字段枚举 `org` / `project`。

## 2.4 关系边

| 关系类型 | 含义 | 与资金的关系 | DDL |
|----------|------|----------------|-----|
| parent | 组织汇报树上下级 | 默认允许**上级 → 下级**划拨 | `party_edges.edge_type='parent'`, `allows_fund=1` |
| sponsors | 出资 | 默认允许**出资方 → 被出资方**划拨 | `party_edges.edge_type='sponsors'`, `allows_fund=1` |
| owns | 主责 / 主办 | **不**自动产生划拨权 | `allows_fund=0` |
| participates | 协作 | **不**自动产生划拨权 | `allows_fund=0` |
| allocates | **拨入个人** | Party → Person 注入个人经费 | `allows_fund=1`, 仅用于 Person Account |
| merged_into | **组织合并** | 源 Party 余额划入目标 | 走清算类流程 |
| split_from | **组织拆分** | 源 Party 按比例划出到新 Party | 走清算类流程 |

无关系时：独立项目或独立组织费用池均可成立。

## 2.5 人、Key、账本、Leader

| 对象 | 物理对应 | 规则 | DDL |
|------|----------|------|-----|
| Person | 实体人 | 通过成员关系进入一个或多个 Party | `users` + `party_members` |
| Person Account | 可选个人小金库 | 由所属 Party 通过 `allocates` 边注入；仅支持预算帽 + 消费扣费；**不支持对外划拨** | `accounts` 挂 `party_id`（如 Person 有专属虚拟 Party）或 `user_id` |
| API Key | 消费凭证 | 必须绑定且仅绑定一个 `account_id`；调用与扣费的唯一入口；必须归属实体人 | `api_keys.account_id` NOT NULL, `api_keys.owner_user_id` NOT NULL |
| Account | 小金库 | 归属某个 Party（或个人）；可用余额与冻结金额 | `accounts` 表 |
| Leader | 负责人 | 对本 Party 默认责任角色；权限仍以**显式 Grant**为准 | `parties.leader_user_id` |

**资金流转强制规则：**

1. 划拨：一方减少、另一方增加，金额相等，同一事务。
2. 预扣：可用余额转入冻结，必须带 `freeze_id` 与过期时间；预扣前可经预算帽校验。
3. 结算：按实际内部价从冻结结清，多退少补。
4. 清算：按状态机阻断新调用、排空冻结后回流资金并收口 Key。
5. 任何余额变更必须有流水；禁止无流水改账；默认不允许成功路径产生负余额。
6. 关键资金写操作必须支持幂等。

**消费强制规则：**
- 内部结算金额（sell）只从 **Key 绑定的账户**扣除，并归因到 **Key 所属实体人**。
- 业务标签只做报表归因，**不形成第二套资金**。

## 2.6 可迭代与扩展原则

- **稳定核心：** 计量、双轨、账本守恒、Key 扣费、四轴授权、ModelGrant、统一入口、价格上限与账户锁定、冻结生命周期、清算状态机、治理 API 幂等、预算帽分码。
- **扩展面：** 新关系类型、新 itemCode、新路由策略、价目分时、安全钩子、对账连接器、划拨白名单、组织变更流程。
- **存储纪律：** 热字段原子；复杂进 JSON；禁止无限加表加列（29 表基准，每新增表须注明理由）。
- **发布演进：** 可先只计量双轨再实扣；策略可 shadow 后再生效。

---

# 第 3 章 开源底座选型与融合架构

## 3.1 三方能力矩阵（源码级证据）

| 维度 | TokenHub v0.4.0（底座） | AxonHub（吸收方） | 融合架构 |
|------|--------------------------|-------------------|----------|
| 语言/框架 | Go 1.26 + GORM + Next.js 16 | Go 1.26 + Ent + Uber fx + React 19 | Go + GORM + Next.js（TokenHub 主线） |
| 数据库 | SQLite / PostgreSQL 双模 | SQLite / PG / MySQL / TiDB | SQLite（MVP）/ PostgreSQL（生产） |
| 入站协议 | OpenAI / Anthropic Messages / Responses / Codex / Embeddings / Images | 9 类（+ Gemini / AI SDK / Jina / Doubao / Copilot / Claude Code） | 6 协议（TokenHub 主线）+ 按需扩展 |
| 上游适配 | 5+ Provider 适配器 | 60+ Provider × 40+ Transformer | TokenHub 适配器 + AxonHub llm/ 动态链接 |
| 路由 | 优先级+权重+策略+亲和+健康 6 策略 Failover | 4 模式 × 7 策略 + 三态熔断 + 探针 | 策略矩阵 11 种 + 档案组合（TokenHub 骨架 + AxonHub 评分吸收） |
| 计价 | 单轨 `input/output_price_per_1m` | itemCode 级 4 模式 + JSON 价目 + costItems + 缓存拆分 | **双轨** cost/sell model_prices JSON（AxonHub 计价口径 + 双轨扩展） |
| 资金闭环 | ❌ 事后统计 cost_usd | ❌ 仅 total_cost 字段 | **新建** fund 包：accounts/ledgers/freezes/allocations/liquidations |
| 组织模型 | 项目空间、成员、项目 Key | 项目空间、用户项目 | **Party 多态** + party_edges（org/project 同层） |
| 授权 | RBAC（admin/member） | 多角色 + Ent Privacy 行级策略 | **grants** 四轴 + **model_grants** ALLOW/DENY |
| 幂等 | ❌ | ❌ | **新建** idempotency_records |
| 预算帽 | `ErrBudgetExceeded` 已定义未触发 | ❌ | accounts 热字段 |
| 审计 | 基础 `audit_events` | 请求/用量日志 | audit_events + before/after 快照 |
| 部署 | Docker Compose + systemd + Native | K8s Helm + Docker Compose + systemd | Docker Compose（MVP）+ K8s Helm（生产） |

## 3.2 融合策略

```
TokenHub（主仓库底座，Apache 2.0）
    │
    ├── 直接复用：Provider 适配器、6 协议兼容、API Key 管理、限流配额桶、Prometheus 指标
    │              SQLite/PG 双模、systemd 加固、多实例协调
    │
    ├── 扩展改造：routing → 策略矩阵引擎（抽象 Strategy 接口）
    │             authz → grants 四轴模型
    │             party → parties + party_edges（统一主体）
    │             models → 双轨 sell 价格热字段
    │
    ├── 从零构建：fund（accounts/ledgers/freezes/allocations/liquidations）
    │             pricing（model_prices 双轨 JSON 价目）
    │             idempotency（Idempotency-Key 原子抢占）
    │             modelgrant（ModelGrant ALLOW/DENY）
    │
    └── AxonHub 吸收：itemCode 计算口径（cost_calc.go）→ pricing 包
                      4 模式 × 7 策略 + 三态熔断  → routing 包
                      渠道探针 channel_probes → 运维面
                      上游配额状态 provider_quota_status → 运维面
                      llm/ 子模块（LGPL-3.0，仅动态链接，不修改源码）
```

## 3.3 路由：可插拔策略矩阵

TokenHub 原生 6 策略（优先级+权重+策略+亲和+健康+限流），AxonHub 4 模式 × 7 策略（adaptive/failover/circuitBreaker/roundRobin × adaptiveScore/latencyScore/errorRateScore/tokenRateScore/lowestCost/weightedRandom/affinity），融合为 11 种可组合策略：

| 策略代码 | 名称 | 来源 | 可禁用 |
|----------|------|------|--------|
| S-PRI | 优先级分组 | TokenHub | 可 |
| S-HEALTH | 健康与熔断 | TokenHub + AxonHub 三态熔断 | 可 |
| S-WEIGHT | 权重与负载 | TokenHub | 可 |
| S-AFFINITY | 会话亲和 | TokenHub | 可 |
| S-COST | 成本感知 | TokenHub cost_score + AxonHub lowestCost | 可 |
| S-LATENCY | 延迟感知 | AxonHub TTFT | 可 |
| S-ERROR | 错误率感知 | AxonHub errorRate | 可 |
| S-RATE | 限流感知 | AxonHub tokenRate | 可 |
| S-TAG | 业务标签 | TokenHub | 可 |
| S-COMPLIANCE | 合规网络 | 新建 | **硬策略** |
| S-CACHE | 缓存兜底 | TokenHub | 可 |

**档案预设：**
- `simple-failover`：PRI + HEALTH（类四层）
- `ha-cost`：PRI + HEALTH + WEIGHT + COST + AFFINITY
- `compliance-strict`：S-COMPLIANCE 强制 + PRI + HEALTH

**DDL：** `route_profiles.strategies_json` JSONB 存储策略组合。

---

# 第 4 章 双轨计价（对齐 AxonHub 代码事实）

## 4.1 费用项 itemCode 基线

| itemCode | 含义 | AxonHub 来源 |
|----------|------|-------------|
| prompt_tokens | 输入（与缓存拆分） | AxonHub cost_calc.go |
| completion_tokens | 输出 | AxonHub cost_calc.go |
| prompt_cached_tokens | 缓存读 | AxonHub cost_calc.go |
| prompt_write_cached_tokens | 缓存写 | AxonHub cost_calc.go |
| completion_reasoning_tokens | 推理输出 | AxonHub cost_calc.go |
| prompt_audio_tokens | 音频输入 | AxonHub usage_log |
| completion_audio_tokens | 音频输出 | AxonHub usage_log |
| prompt_write_cached_5m | 5 分钟缓存写 | AxonHub 变体 |
| prompt_write_cached_1h | 1 小时缓存写 | AxonHub 变体 |

## 4.2 计价模式（全部对齐 AxonHub 4 模式）

| 模式 | 说明 | AxonHub 证据 |
|------|------|-------------|
| flat_fee | 按次固定费用 | `cost_calc.go` |
| usage_per_unit | 按单位用量（通常每 1M Token） | `cost_calc.go` |
| usage_tiered | 阶梯价格（分段累计） | `cost_calc.go` |
| usage_volume | 总量落档后整单同一单价 | `cost_calc.go` |

## 4.3 双轨

| 轨道 | 含义 | 用途 |
|------|------|------|
| cost | 上游成本 | 对账、毛利 |
| sell | 内部结算价 | 扣企业内账本 |

## 4.4 价目 JSON 结构（DDL：`model_prices.price_json`）

```json
{
  "items": [
    {"itemCode": "prompt_tokens",
     "cost": {"mode": "usage_per_unit", "rate": 0.002},
     "sell": {"mode": "usage_per_unit", "rate": 0.003}},
    {"itemCode": "completion_tokens",
     "cost": {"mode": "usage_per_unit", "rate": 0.008},
     "sell": {"mode": "usage_per_unit", "rate": 0.012}}
  ],
  "schedule": {"timezone": "Asia/Shanghai", "overrides": []}
}
```

## 4.5 用量规范化

适配层：上游 usage → 内部 itemCode；缺失记 0 + `usage_incomplete`；禁止伪造。

---

# 第 5 章 预算帽配置数据模型

## 5.1 挂载点

预算帽挂在 **Account**（`accounts` 表）。`accounts` 通过 `party_id` 关联 Party。个人 Account 同理。

## 5.2 字段（`accounts` 表热字段）

| 字段 | 类型 | 含义 |
|------|------|------|
| budget_limit_amount | REAL NULL | 预算上限；NULL=未启用 |
| budget_warn_ratio | REAL NULL | 告警比例（如 0.80）；只告警不阻断 |
| budget_period | TEXT DEFAULT 'none' | none / calendar_month / calendar_day / custom |
| budget_period_start | TEXT | custom 起点 |
| budget_period_end | TEXT | custom 终点 |
| budget_consumed_amount | REAL NOT NULL DEFAULT 0 | 本周期已确认 sell 累计 |
| budget_version | INTEGER NOT NULL DEFAULT 0 | 配置乐观锁 |

## 5.3 规则

1. `budget_limit_amount IS NULL` → 不做帽校验。
2. 上限可小于当前 balance（软帽）。
3. 告警：`consumed / limit >= warn_ratio` → 通知，不拒绝调用。
4. 阻断：`consumed + 预估 sell > limit` → **`BUDGET_CAP_EXCEEDED`**。
5. 周期任务重置 consumed 并写审计事件。
6. 结算成功的 sell 计入 consumed；解冻未消费不计入。

## 5.4 预扣判定顺序

```text
ModelGrant
  → 构建候选 → 价格合格集过滤
  → 若启用预算帽：帽校验（失败 → BUDGET_CAP_EXCEEDED）
  → 可用余额 ≥ 冻结额（失败 → INSUFFICIENT_BALANCE）
  → 写入 freeze
  → 策略矩阵选路 → 上游调用 → 结算
```

---

# 第 6 章 错误码定义全表

## 6.1 资金与额度

| code | HTTP | 含义 |
|------|------|------|
| BUDGET_CAP_EXCEEDED | 402 | 命中预算上限（余额可能>0） |
| INSUFFICIENT_BALANCE | 402 | 可用余额不足以完成本次冻结 |
| ACCOUNT_FROZEN_OR_CLOSED | 403 | 账户停用或清算中 |
| FREEZE_EXPIRED | 409 | 结算时冻结已失效 |
| IDEMPOTENCY_CONFLICT | 409 | 同键处理中或同键异参 |
| IDEMPOTENCY_REPLAY | 200 | 同键重放成功结果 |

## 6.2 鉴权与身份

| code | HTTP | 含义 |
|------|------|------|
| AUTH_INVALID_KEY | 401 | Key 无效或吊销 |
| AUTH_USER_DISABLED | 403 | 归属人已禁用 |
| AUTH_KEY_NO_ACCOUNT | 403 | Key 未绑定账户 |
| AUTHZ_DENIED | 403 | 控制面四轴授权不通过 |
| MODEL_ACCESS_DENIED | 403 | ModelGrant 不允许该模型 |

## 6.3 路由与价格

| code | HTTP | 含义 |
|------|------|------|
| NO_ROUTE_WITHIN_PRICE_CAP | 422 | 无价格合格候选 |
| NO_ROUTE_AVAILABLE | 503 | 无可用健康路由 |
| ROUTE_COMPLIANCE_BLOCKED | 403 | 合规策略剔除全部候选 |

## 6.4 安全

| code | HTTP | 含义 |
|------|------|------|
| COMPLIANCE_NETWORK_BLOCKED | 403 | 网络策略阻断 |
| CONTENT_BLOCKED | 403 | 内容安全阻断 |
| RATE_LIMITED | 429 | 网关或策略限流 |

## 6.5 上游与系统

| code | HTTP | 含义 |
|------|------|------|
| UPSTREAM_ERROR | 502 | 上游返回错误 |
| UPSTREAM_TIMEOUT | 504 | 上游超时 |
| INTERNAL_ERROR | 500 | 网关内部错误 |

---

# 第 7 章 模型权限治理与正交授权

## 7.1 四轴分离

| 轴 | 回答的问题 | DDL：grants.axis | 禁止推导 |
|----|------------|-----------------|----------|
| data | 能看哪些日志/报表/成员 | data | 不能推导可划拨、可改路由 |
| fund | 余额流水、划拨清算、预算帽 | fund | 不能推导未授权 Party 全量日志 |
| iam | 人、Key、成员、禁人 | iam | 不能绑无权账户；不能改价目 |
| routing | 价目、档案、策略、渠道、上游密钥、模型目录与 ModelGrant | routing | 不能改 account_id；不能划拨 |

## 7.2 模型访问治理（ModelGrant）

**DDL：** `model_grants` 表。

| 字段 | 含义 |
|------|------|
| principal_type | party / person / key / role |
| principal_id | 对应 ID |
| model_id | 单个模型或标签组 |
| effect | allow / deny（**deny 优先**） |
| priority | 冲突解析辅助 |

级联顺序：Key > Person > Party > 全局默认。禁止仅因 Leader 头衔自动拥有全平台模型调用权。

## 7.3 Grant 最小动作矩阵

| 轴 | 动作 | DDL：grants.action |
|----|------|-------------------|
| fund | balance.read, ledger.read, allocate, liquidate, budget.write | — |
| data | usage.read, report.read, member.read | — |
| iam | key.create/revoke/rotate, user.disable, member.add/remove | — |
| routing | price.write, route_profile.write, channel.write, upstream_secret.write, model_catalog.write, model_grant.write | — |

## 7.4 验收红线清单

1. 无流水改余额
2. 划拨无通道
3. Key 无 account 调用
4. 调度改扣费账户
5. 先调后欠费
6. Leader 无 Grant 即全平台权限
7. iam 建 Key 绑无权账户
8. 预算帽与余额不足返回同一错误码
9. ModelGrant deny 后仍可调用该模型

---

# 第 8 章 关键边界规则与流程

## 8.1 预扣、价格约束与结算

1. **请求锚定内部价 P_request：** 逻辑模型 + 价目 + 预估用量 → 内部 sell 预估金额。
2. **候选价格约束：** P_candidate ≤ P_request × (1+δ)；**默认 δ=0**；**硬上限 20%**；改 δ 必须关键配置审计。
3. **ModelGrant** 失败 → `MODEL_ACCESS_DENIED`。
4. **预算帽**（若启用）失败 → `BUDGET_CAP_EXCEEDED`。
5. **冻结金额** = 价格合格候选集上预估 sell 的最大值。
6. **调度**仅在合格集执行；**account_id 在鉴权时锁定，调度不得修改**。
7. **结算**按实际用量与落地价目算 cost/sell；多退少补。

## 8.2 划拨路径规则

| 路径 | 默认是否允许 | 方向 |
|------|--------------|------|
| parent | 是 | 仅上级 → 下级 |
| sponsors | 是 | 仅出资方 → 被出资方 |
| allocates | 是 | Party → Person Account |
| owns / participates | 否 | 不自动开通 |
| 无关系双方 | 否 | 除非白名单 + fund |

## 8.3 冻结生命周期与流式续期

- 默认 TTL 15 分钟（可配 1–60 分钟）。
- **流式：网关自动续期同一 freeze_id，不增加冻结金额**；累计上限可配（如 2 小时）。
- 客户端不负责续期。
- DDL：`freezes.renewal_count`, `freezes.last_renewed_at`, `freezes.max_lifetime_at`。

## 8.4 清算状态机

```text
active
  → liquidating_block_new    // 拒绝新调用与新冻结
  → liquidating_drain        // 等待冻结清零
  → liquidating_transfer     // 余额转入目标账户
  → liquidated               // Key 收口，主体只读
```

## 8.5 组织变更流程（新增）

```
组织合并：active → merging_block_new → merging_drain → merging_transfer → merged
组织拆分：active → splitting → split_completed（按比例划出）
```

与清算共享冻结排空机制，复用幂等和流水基础设施。DDL：`liquidations` 表扩展 `liquidation_type` 枚举。

## 8.6 调用主路径

1. Key 鉴权、人状态、account 绑定
2. 安全钩子（若启用）
3. ModelGrant
4. 计算 P_request；候选；价格过滤
5. 预算帽 → 冻结
6. 策略矩阵选路 → 托管上游密钥调用
7. 流式场景网关续期冻结
8. 用量规范化 → 双轨结算 → 写调用审计
9. 返回兼容响应 + 分码错误

## 8.7 治理 API 幂等

| 项 | 规则 |
|----|------|
| 适用范围 | 划拨、清算、资金补偿等写操作 |
| 机制 | `Idempotency-Key`（UUID v4，≤255） |
| DDL | `idempotency_records` UNIQUE(scope, actor_id, idempotency_key) |
| 抢占 | INSERT ON CONFLICT 原子抢占 |
| 行为 | 同键同指纹重放首次结果；异指纹拒绝 |

## 8.8 账本技术

热账本 = PostgreSQL/SQLite + 只追加 `ledgers`。不采用区块链。流水禁止 UPDATE/DELETE。

---

# 第 9 章 功能需求编号全表（逐条完整）

## 9.1 统一接入与模型资源池

| 编号 | 名称 | 描述 | 优先级 | 实现来源 |
|------|------|------|--------|----------|
| RES-01 | 多上游接入 | 公有+国内 API、OpenAI 兼容私有化（vLLM/Ollama/KServe） | P0 | TokenHub 适配器 |
| RES-02 | 上游密钥仓库 | 网关托管；加密；明文不落日志 | P0 | TokenHub + AES-256 |
| RES-03 | 密钥操作权限 | 仅 routing 轴 upstream_secret.write | P0 | grants |
| RES-04 | 健康与状态 | 可用/降级/不可用 | P0 | TokenHub + AxonHub channel_probes |
| RES-05 | 资源标签 | 成本、区域、是否内网 | P1 | TokenHub |
| RES-06 | 兼容 API | OpenAI + Anthropic + Gemini + Embeddings + Images 六协议 | P0 | TokenHub 6 协议 |
| RES-07 | 请求标识与标签 | request_id；业务标签仅归因 | P0 | TokenHub |
| RES-08 | 用量规范化 | 映射 itemCode；缺失记 0 + usage_incomplete | P0 | AxonHub cost_calc |

## 9.2 双轨计价

| 编号 | 名称 | 描述 | 优先级 | 实现来源 |
|------|------|------|--------|----------|
| PRI-01 | 价目配置 | 渠道×模型双轨 JSON；变更审计 | P0 | model_prices（新建） |
| PRI-02 | 用量解析 | 规范化分项；缓存与输入不双计 | P0 | AxonHub cost_calc |
| PRI-03 | 费用计算 | cost/sell 总额与 costItems | P0 | pricing 包（新建） |
| PRI-04 | 落账字段 | 双轨、分项、账户、人、Key、Party、标签、freeze_id | P0 | usage_records + request_logs |

## 9.3 主体、账本与资金

| 编号 | 名称 | 描述 | 优先级 | 实现来源 |
|------|------|------|--------|----------|
| FUN-01 | 主体管理 | org/project 同一层；Leader；项目可不挂靠 | P0 | parties（扩展 TokenHub projects） |
| FUN-02 | 关系管理 | parent/sponsors/owns/participates/allocates | P0 | party_edges（新建） |
| FUN-03 | 账本 | balance+frozen；并发安全 | P0 | accounts（新建） |
| FUN-04 | 划拨通道 | 见 8.2 | P0 | allocations + party_edges |
| FUN-05 | 划拨执行 | 授权+通道+守恒+流水+幂等 | P0 | fund 包（新建） |
| FUN-06 | 预扣与结算 | 见 8.1 | P0 | freezes + ledgers（新建） |
| FUN-07 | 告警比例与预算上限 | 告警不阻断；预算帽可软；分码 | P0 | accounts.budget_* |
| FUN-08 | 清算 | 见 8.4/8.5；幂等 | P0 | liquidations（新建） |
| FUN-09 | 流水 | 只追加；含幂等键等关联字段 | P0 | ledgers（新建） |
| FUN-10 | 冻结超时与续期 | 见 8.3 | P0 | freezes（新建） |

## 9.4 Key 与成员

| 编号 | 名称 | 描述 | 优先级 | 实现来源 |
|------|------|------|--------|----------|
| KEY-01 | Key 生命周期 | 创建/轮换/吊销；存哈希 | P0 | TokenHub api_keys 扩展 |
| KEY-02 | 绑定账户 | 必须且唯一 account_id | P0 | api_keys.account_id |
| KEY-03 | 归属人 | 必须关联实体人 | P0 | api_keys.owner_user_id |
| KEY-04 | 绑户约束 | 目标账户 ∈ iam 允许集 | P0 | grants + iam |
| KEY-05 | 成员管理 | 加入/移出 Party | P0 | party_members |
| KEY-06 | 禁用联动 | 禁人后 Key 立即失效 | P0 | users.status + api_keys.status 联动 |

## 9.5 正交授权

| 编号 | 名称 | 优先级 | 实现来源 |
|------|------|--------|----------|
| AUTH-01 | 四轴模型 data/fund/iam/routing | P0 | grants（新建，替代 RBAC） |
| AUTH-02 | 最小默认权限 | P0 | grants 默认策略 |
| AUTH-03 | Leader 模板（显式 Grant，可审计） | P0 | grants 批量授予 |
| AUTH-04 | 数据范围强制（防 IDOR） | P0 | grants.resource_id 过滤 |
| AUTH-05 | 职责分离 | P0 | 四轴互斥 |

## 9.6 模型权限

| 编号 | 名称 | 优先级 | 实现来源 |
|------|------|--------|----------|
| MODEL-01 | 逻辑模型目录 | P0 | models（扩展 TokenHub） |
| MODEL-02 | 渠道绑定 | P0 | provider_models |
| MODEL-03 | ModelGrant ALLOW/DENY | P0 | model_grants（新建） |
| MODEL-04 | 调用前校验 | P0 | modelgrant 包（新建） |
| MODEL-05 | 默认策略 | P0 | 实现固定一种；禁止 Leader 全模型权 |
| MODEL-06 | 与资金解耦 | P0 | 允许调用≠有钱 |

## 9.7 路由与调度

| 编号 | 名称 | 优先级 | 实现来源 |
|------|------|--------|----------|
| RTE-01 | 策略引擎（注册/启停/混合组合） | P1 | TokenHub 路由骨架 + AxonHub 评分 |
| RTE-02 | 策略矩阵（11 种） | P1 | route_profiles（新建） |
| RTE-03 | 高可用（重试切换；流式限制危险切换） | P1 | TokenHub Failover + AxonHub 熔断 |
| RTE-04 | 账户正交（不得改 account_id） | P0 | 数据面管道锁定 |
| RTE-05 | 价格约束（δ 默认 0、硬上限 20%、关键审计） | P0 | model_routes.price_cap_delta |
| RTE-06 | 决策可观测（候选与选择关联 request_id） | P1 | route_attempt_logs |

## 9.8 安全

| 编号 | 名称 | 优先级 |
|------|------|--------|
| SEC-01 | 网络策略（INTERNAL_ONLY 零外网流量） | P2（可升） |
| SEC-02 | 出网范围（白名单） | P2 |
| SEC-03 | 内容安全（阻断/脱敏/强制内网） | P2 |
| SEC-04 | 异常流量（拦截告警） | P2 |
| SEC-05 | 扩展点（主路径钩子；阶段 B 就绪） | 架构 P0 |

## 9.9 审计与对账

| 编号 | 名称 | 优先级 | DDL |
|------|------|--------|-----|
| AUD-01 | 调用审计（全字段） | P0 | request_logs + usage_records |
| AUD-02 | 配置变更审计（before/after；δ 与预算帽关键） | P2（关键强制） | audit_events |
| AUD-03 | 对账（上游 vs cost；差异分类） | P2 | P2 阶段补 |
| AUD-04 | 报表（多维汇总；沿 parent/sponsors 边向上聚合；data 轴范围约束） | P0/P2 | 实时聚合 |

## 9.10 控制台与治理 API

| 编号 | 名称 | 优先级 | 实现来源 |
|------|------|--------|----------|
| UI-01 | 角色化导航（按授权显示） | P0 | Next.js（TokenHub） |
| UI-02 | 主体与关系（Party/Leader/边/成员） | P0 | 新建 |
| UI-03 | 资金操作（划拨/流水/清算/预算帽；二次确认） | P0 | 新建 |
| UI-04 | 价目维护（双轨编辑） | P0 | 新建 |
| UI-05 | Key 与成员（申请绑户吊销） | P0 | TokenHub 扩展 |
| UI-06 | 路由档案（策略与 δ） | P1 | 新建 |
| UI-07 | 仪表盘与报表（消耗/余额/预算/拦截+聚合） | P1–P2 | TokenHub 扩展 |
| UI-08 | 密钥仓库（无明文二次回显） | P0 | TokenHub |
| UI-09 | 模型权限（目录与 ModelGrant） | P0 | 新建 |
| API-01 | 治理 API（主体/授权/划拨/清算/Key/价目/路由/密钥；资金写强制幂等） | P0 | 新建 |

---

# 第 10 章 融合 DDL 数据模型（29 表）

## 10.1 表全景

| 组 | 表名 | 来源 | 行数 |
|----|------|------|------|
| **用户与身份** | `users` | TokenHub + AxonHub 融合 | L24-43 |
| | `admin_sessions` | TokenHub | L46-52 |
| **Party 主体** | `parties` | 新建（统一 org/project） | L62-77 |
| | `party_edges` | 新建（6 种关系边） | L80-91 |
| | `party_members` | TokenHub + AxonHub 融合 | L96-108 |
| **资金治理** | `accounts` | 新建（含预算帽+清算热字段） | L116-139 |
| | `ledgers` | 新建（只追加流水） | L144-168 |
| | `freezes` | 新建（含续期字段） | L172-195 |
| | `allocations` | 新建（划拨记录） | L199-215 |
| | `liquidations` | 新建（清算/合并/拆分） | L219-234 |
| **API Key** | `api_keys` | TokenHub 扩展 + AxonHub 字段 | L244-287 |
| **模型目录** | `providers` | TokenHub + AxonHub 融合 | L295-322 |
| | `provider_resources` | TokenHub | L327-356 |
| | `models` | TokenHub 扩展 + 双轨 sell 热字段 | L362-402 |
| | `provider_models` | TokenHub | L407-432 |
| **定价与路由** | `model_prices` | 新建（AxonHub 计价吸收+双轨） | L440-454 |
| | `model_routes` | TokenHub 扩展 | L458-487 |
| | `route_profiles` | 新建（策略矩阵档案） | L491-504 |
| **授权治理** | `grants` | 新建（四轴正交） | L511-527 |
| | `model_grants` | 新建（ModelGrant） | L531-545 |
| **请求与用量** | `request_logs` | TokenHub 扩展 + AxonHub 双轨 | L553-601 |
| | `request_payload_logs` | TokenHub | L605-613 |
| | `route_attempt_logs` | TokenHub | L617-633 |
| | `usage_records` | TokenHub 扩展 + AxonHub itemCode | L637-683 |
| | `quota_buckets` | TokenHub | L687-697 |
| **可观测** | `channel_probes` | AxonHub | L705-719 |
| | `provider_quota_status` | AxonHub | L723-736 |
| **基础设施** | `audit_events` | TokenHub 扩展（+before/after） | L744-761 |
| | `idempotency_records` | 新建（PRD §8.7） | L765-779 |

**完整 DDL：** 见 `schema/ai-gov-fusion-minimal.sql`（842 行）。

## 10.2 剪裁说明

69 表（ai-gov.sql 过度工程化）→ 29 表（融合最小化）：

- 复式账本 14 种 entry_type → `ledgers.direction` 枚举（debit/credit/freeze/unfreeze/settle）
- ABAC 引擎 6 表 → `grants` 单表
- 对账 5 表 → P2 阶段补
- 复杂身份 9 表 → `users` 单表
- 紧急信用/日结快照/事件总线 → MVP 不做
- AxonHub 非核心（提示词库/Thread/Trace/邀请/模板/数据仓库）→ 不在 PRD 范围

## 10.3 存储纪律

- 价目、策略组合、分时进 JSON（`price_json`, `strategies_json`, `schedule_json`）
- 热字段原子（`accounts.available_balance`, `budget_limit_amount`）
- 流水只追加（应用层强制，无 UPDATE/DELETE）
- 禁止为每个扩展点无限加表加列

---

# 第 11 章 系统架构与二次开发指导

## 11.1 逻辑架构

```text
应用/Agent → 兼容 API（6 协议）
  → Key 鉴权（TokenHub 原生）→ 安全钩子 → ModelGrant（新建）
  → 锚定内部价 → 价格合格集 → 预算帽 → 冻结（新建 fund 包）
  → 策略矩阵选路（TokenHub 路由骨架 + AxonHub 评分吸收）
  → 上游调用（TokenHub 适配器 + AxonHub llm/ 动态链接）
  → 用量规范化（AxonHub cost_calc 吸收）
  → 双轨结算（新建 pricing 包）→ 审计（扩展 audit_events）

控制面：Party、Grant、划拨清算、预算帽、价目、路由、密钥、ModelGrant
治理 API：写强制幂等（新建 idempotency 包）
存储：SQLite（MVP）/ PostgreSQL（生产）
```

## 11.2 包划分

| 包 | 性质 | 职责 | 关键文件 |
|----|------|------|----------|
| `fund` | **新建** | accounts/ledgers/freezes/allocations/liquidations 资金闭环 | — |
| `pricing` | **新建** | model_prices 双轨 JSON 价目、AxonHub cost_calc 吸收 | — |
| `idempotency` | **新建** | Idempotency-Key 原子抢占 | — |
| `party` | **扩展** | parties + party_edges 统一主体（扩展 TokenHub projects） | TokenHub `store.go` |
| `authz` | **扩展** | grants 四轴模型（替代 TokenHub RBAC） | TokenHub `http.go` admin 中间件 |
| `routing` | **扩展** | Strategy 接口抽象 + route_profiles（TokenHub 路由骨架 + AxonHub 评分） | TokenHub `http.go` routing + AxonHub `orchestrator/` |
| `modelgrant` | **新建** | ModelGrant ALLOW/DENY 调用前校验 | — |
| `security` | **扩展** | 安全钩子执行链路（TokenHub mask_prompts 字段→执行） | TokenHub SDK |

## 11.3 技术栈

| 层 | 选择 | 来源 |
|----|------|------|
| 后端语言 | Go 1.26 | TokenHub |
| ORM | GORM | TokenHub |
| Web 框架 | Gin（TokenHub）/ 可选 fx（AxonHub） | TokenHub |
| 数据库 | SQLite（MVP）/ PostgreSQL 16（生产） | TokenHub 双模 |
| 缓存 | Redis 可选 | TokenHub |
| 前端 | Next.js 16 + TypeScript | TokenHub |
| 部署 | Docker Compose + systemd（MVP）/ K8s Helm（生产） | TokenHub + AxonHub Helm |
| 国产适配 | 国产 CPU/OS 阶段 A 冒烟 | — |

## 11.4 幂等实现要点

表 `idempotency_records`：UNIQUE(scope, actor_id, idempotency_key)；原子 INSERT 抢占（SQLite: `INSERT OR IGNORE`；PG: `INSERT ON CONFLICT DO NOTHING RETURNING`）。

## 11.5 策略引擎接口

```go
type Strategy interface {
    Filter(ctx context.Context, candidates []RouteCandidate) []RouteCandidate
    Score(ctx context.Context, candidates []RouteCandidate) []RouteCandidate
}

type RouteProfile struct {
    Strategies []StrategyConfig  // [{Code:"S-PRI", Enabled:true, Priority:0}, ...]
    DeltaCap   float64           // δ 价格帽
}
```

Pipeline: `candidates → S-COMPLIANCE → ModelGrant → price cap → strategies → pick`

## 11.6 多方验收门禁

| 顺序 | 门禁 | 责任方 | 入口 | 出口（书面） |
|------|------|--------|------|----------------|
| 1 | Dev Complete | 研发 | 合入约定分支 | 单测/关键集成通过；迁移可反复执行；OpenAPI 一致 |
| 2 | QA | 测试 | Dev Complete | 第 13 章 P0 用例通过（预算帽分码、ModelGrant、价格帽、清算、幂等、四轴越权） |
| 3 | UED | 设计 | 页面可用 | 危险确认、错误文案、角色导航走查通过 |
| 4 | 产品 UAT | 产品 | QA+UED | 财务演示脚本完整；策略与模型权限场景签字 |
| 5 | 安全 | 安全 | 可并行 | 越权、密钥、审计抽检通过 |
| 6 | 发布 | 架构/运维 | 安全通过 | 回滚演练、监控、备份、NOTICE 就绪 |

## 11.7 商用交付物清单

| 交付物 | 说明 |
|--------|------|
| 安装包 | Docker Compose + K8s Helm；可选离线包；版本可追溯 |
| OpenAPI | 数据面说明 + 控制面完整规范（错误码、幂等头） |
| 迁移与回滚 | 含预算帽字段、幂等表、Party 边等 |
| UAT 脚本 | 财务闭环、分码、ModelGrant、价格帽、清算、幂等、越权 |
| 监控项 | 冻结、预算帽命中、幂等冲突、TTL 任务、路由失败码、上游错误率 |
| 许可 NOTICE | TokenHub (Apache 2.0) + AxonHub llm/ (LGPL-3.0) |
| 运维手册 | 部署、扩缩、备份、密钥轮换、TTL、排障 |
| 安全说明 | 密钥、权限模型、审计保留、合规开关 |

## 11.8 分期 WBS

| 阶段 | 内容 | 工期 |
|------|------|------|
| A | Fork TokenHub、执行 29 表 DDL、用量规范化、国产冒烟 | 1d |
| B | Party/账本/划拨/预算帽/冻结续期/清算/双轨 model_prices/价格帽/四轴 grants/ModelGrant 骨架/治理 API 幂等/安全钩子空实现 | 3d |
| C | 策略矩阵全量（TokenHub 路由骨架 + AxonHub 评分吸收）、决策日志、仪表盘 | 2d |
| D | 内容安全出网、变更快照、对账 | 2d |
| E | 压测 HA、GA、文档与许可 | 1d |

**总工期：约 9 工作日（含集成测试）。**

---

# 第 12 章 非功能需求

| 类别 | 要求 |
|------|------|
| 可用性 | 目标 99.9%；多实例故障切换（TokenHub 原生 ClusterLease） |
| 性能 | 单节点目标 5000 QPS；附加延迟 <50ms（Go 技术栈） |
| 安全 | TLS；密钥加密存储（AES-256）；最小权限；可审计 |
| 部署 | 私有化；Docker Compose / K8s；离线/内网；国产环境阶段 A 验证 |
| 审计保留 | ≥180 天；冷热分离 |
| 可扩展 | 适配器、itemCode、策略、边类型、安全钩子 |
| 可观测 | 冻结任务、幂等冲突、预算帽命中、健康与路由指标（Prometheus + Grafana） |

---

# 第 13 章 验收标准

## 13.1 功能验收

| 场景 | 通过条件 |
|------|----------|
| 统一接入 | ≥5 类公有 + 1 类私有化兼容 |
| 双轨与 item | cost/sell 及分项正确 |
| usage 不完整 | 有标记、不伪造 |
| 独立项目 / 组织池 / 出资划拨 | 守恒与通道正确 |
| 个人经费 | Party 通过 allocates 边注入个人 Account；个人 Key 扣个人 Account |
| 价格约束与 δ | 默认 0、硬上限 20%、变更有关键审计 |
| 预算帽 vs 余额不足 | 90% 帽 → BUDGET_CAP_EXCEEDED；余额不够 → INSUFFICIENT_BALANCE |
| 告警比例 | 80% 只告警不阻断 |
| 冻结超时 / 流式续期 | 符合 8.3 |
| 清算状态机 | 符合 8.4 |
| 组织合并/拆分 | 按 8.5 流程余额正确转移 |
| 幂等 | 重复写不双记 |
| ModelGrant | deny 后不可调该模型 |
| 四轴越权 | Leader/路由/资金职责分离成立 |
| 调度不改账户 | 任意路由下 account 不变 |
| 策略矩阵启停组合 | 可测 |
| 禁人即禁 Key | 立即 |
| 治理 API | 对等鉴权与幂等 |
| INTERNAL_ONLY（启用时） | 无外网流量 |

## 13.2 非功能验收

1. 在约定硬件下完成压力测试：单节点 ≥5000 QPS，附加延迟 <50ms
2. 安全扫描与密钥存储抽检通过
3. 审计保留策略验证通过
4. 目标国产环境冒烟通过
5. 冻结超时任务与幂等窗口行为验证通过

## 13.3 财务演示脚本（必须可重复）

配置预算与加价 → 划拨 → 创建人 Key 绑项目账本 → 调用 → 核对 sell/cost/流水 → 演示预算帽分码 → 演示余额不足分码 → 演示个人经费注入与消费 → 清算回流 → 重复提交划拨仅入账一次。

---

# 第 14 章 不在范围与预留扩展

**当前不在范围：**
- Agent 编排 IDE、提示词资产中心、C 端聊天产品
- 同一请求多账本分摊扣费
- 默认透支信用
- 客户端负责冻结续期
- 无权限审计的上游密钥写入
- 区块链热账本

**预留扩展（架构级设计，非功能实现）：**
- 对外 Token 售卖：租户隔离、多层级定价、用量同步接口（`tenants` 表预留）
- 多币种、复杂审批流、HR/ERP 深度双向同步

**P2 阶段补：** 对账系统 5 表、日结快照、提示词安全规则

---

# 第 15 章 术语表

| 术语 | 定义 |
|------|------|
| Token | 计费用量及缓存、推理等衍生类型 |
| 网关 Key | 企业发给调用方的凭证，≠ 上游厂商密钥 |
| Party | 组织或项目等可持账本与成员的主体（org/project 同层多态） |
| Party Edge | 关系边（parent/sponsors/owns/participates/allocates/merged_into/split_from） |
| 双轨 | cost（上游成本）与 sell（内部结算价）并行计量 |
| 请求锚定内部价 | 约束调度候选的内部 sell 基准 |
| δ | 候选相对锚定价允许上浮比例；默认 0；硬上限 20% |
| 预算帽 | 可低于余额 100% 的配置上限；命中 BUDGET_CAP_EXCEEDED |
| INSUFFICIENT_BALANCE | 可用余额不足以冻结 |
| MODEL_ACCESS_DENIED | ModelGrant 拒绝 |
| 预扣 / 结算 | 调用前冻结，调用后按实际内部价结清 |
| 冻结续期 | 流式由网关延长同一冻结过期时间，不增加金额 |
| 清算 | 阻断 → 排空冻结 → 回流 → Key 收口 |
| 组织变更 | 合并/拆分流程，共享清算基础设施 |
| 正交授权 | data/fund/iam/routing 四轴 |
| ModelGrant | 模型访问 ALLOW/DENY 规则 |
| 策略矩阵 | 11 种可启停、可混合的路由策略 |
| 治理 API | 与管理台对等的控制面 API |
| 幂等键 | 防写操作重试重复记账 |
| itemCode | 与上游账单对齐的费用项编码 |
| TokenHub 底座 | 主仓库二次开发基线（Apache 2.0） |
| AxonHub 吸收 | 计价明细与多维评分思想来源（llm/ LGPL-3.0，动态链接） |

---

# 第 16 章 融合架构总结

## 16.1 为什么这是正确的方案

| 问题 | 回答 |
|------|------|
| **为什么选 TokenHub 做底座？** | Apache 2.0 许可证无传染性；Go 1.26 + GORM 技术栈成熟；6 协议兼容开箱即用；SQLite/PG 双模；多实例协调已实现；改造工作量可控（5 新建 + 3 扩展包） |
| **为什么只吸收 AxonHub 的 itemCode 计价和评分？** | AxonHub LGPL-3.0 传染性约束 llm/ 模块修改须开源；其资金闭环完全缺失（无 wallet/ledger/balance）；但其 itemCode 计算口径和 4 模式 × 7 策略评分是业界最佳实践 |
| **为什么 Party 多态优于四级层级？** | 真实企业是图不是树。Party（org=project 同层）+ 关系边可以表达任意企业拓扑。四级层级是静态假设，Party 是物理世界映射 |
| **为什么 29 表而非 69 表？** | ai-gov.sql 是过度工程化的产物（CQRS、ABAC 引擎、复式账本 14 entry_type、哈希链审计、事件总线）。29 表的融合方案覆盖 PRD 100% 需求，复杂度降低 58% |
| **为什么个人经费用 allocates 边而非四级层级？** | `allocates` 边（Party → Person Account）比固定"部门→个人"更灵活——任何 Party 都可以给其成员注入经费，不受层级约束 |

## 16.2 与原始产品愿景的对齐

本方案回归了原始产品讨论的核心洞察：

1. **Token = AI 电力，网关 = 智能电表 + 配电箱 + 财务结算中心** ✓
2. **治理优先，转发为辅** ✓（TokenHub 解决通路，fund/pricing/idempotency/modelgrant 解决治理）
3. **面向组织管理，打通企业身份体系** ✓（Party 多态 + 四轴 grants）
4. **全生命周期管控** ✓（计量→归因→限流→调度→审计）
5. **私有化优先** ✓（Docker Compose + K8s + systemd + 国产适配）

## 16.3 下一步

1. `schema/ai-gov-fusion-minimal.sql` 作为基线 DDL，执行于 SQLite/PostgreSQL
2. GORM 模型生成（`gorm gen` 或手写）
3. 按 §11.8 分期 WBS 执行：A（1d）→ B（3d）→ C（2d）→ D（2d）→ E（1d）
4. 每个阶段完成后执行 §13 对应验收场景

---

**文档结束（定版 3.0.0 融合架构完整方案）。**
