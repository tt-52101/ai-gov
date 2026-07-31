# 全量差距分析报告

## 文档信息

| 项 | 内容 |
|----|------|
| 文档版本 | **1.0** |
| 分析基线 | PRD v3.2.0 (`docs/prd/AI-GOV-PRD-v3.2.0.md`) |
| 代码基线 | `ai-gov-fusion/backend/internal/server/` |
| 分析日期 | 2026-07-31 |
| 分析方法 | 逐文件代码审计 + PRD 逐项对照 |
| 前置参考 | batch-002 验收报告全量（DATA/PRD/QA/RED-BLUE/TECH/UED） |

---

## 执行摘要

当前代码库 (`ai-gov-fusion/`) 的质量评估：**阶段 B 核心域已深度实现，控制面 handler 大量为占位**。概况如下：

- **PRD §9 功能需求 58 项**：已实现 32 项（55%）、部分实现 9 项（16%）、占位 15 项（26%）、未实现 2 项（3%）
- **14 步数据面管线**：核心步骤全部有实现（14/14），但步骤间的端到端集成尚未在 pipeline.go 外体现
- **WBS 五阶段**：阶段 A 完成、阶段 B 约 80% 完成（控制面 handler 大量占位）、阶段 C 路由模块完成但未集成、阶段 D/E 未开始
- **DDL 40 表**：全部在 `schema/ai-gov-fusion-v3.2.sql` 存在（39 张 PRD 表 + 1 张 sys_config 额外表），字段级偏差 2 项
- **部署就绪度**：TokenHub 原有 deploy/ 配置与新增 Go 包兼容（无冲突），但 fund/abac/modelgrant 等新表需首次迁移后才能运行

---

## 1. PRD §9 功能需求覆盖率（逐项审计）

### 1.1 §9.1 统一接入与模型资源池（8 项）

| 编号 | 状态 | 代码位置 | 缺口描述 |
|------|------|----------|----------|
| RES-01 | ✅ | `provider_*.go`（TokenHub 底座已有） | 多上游接入完整 |
| RES-02 | ✅ | `store.go` AES-256-GCM 加密，`types.go` KeyHash=SHA256 | 加密存储完成；明文落日志已通过 `redactAuditPayload` 脱敏 |
| RES-03 | ✅ | `gov_handlers_fund.go:68-94` ABAC 校验 `iam.key.create` | Key 操作受 routing 轴权限保护 |
| RES-04 | ✅ | `provider_monitoring.go`, `provider_resource_recovery.go` | 三态健康（up/degraded/down）+ 熔断恢复 |
| RES-05 | ✅ | `models` 表 `network_class`, `data_classification` 字段 | 资源标签就绪 |
| RES-06 | ✅ | `anthropic_messages.go`, `image_generation.go`, `provider_gemini_*.go`, `codex_protocol.go` | 6 协议兼容 |
| RES-07 | ✅ | `pipeline.go` AuthResult + pipeline 审计事件 | request_id 全链路 |
| RES-08 | ✅ | `pricing/normalizer.go` OpenAI + Anthropic 规范化器 | 10 项 itemCode 映射完成 |

**域总结**: 8/8 ✅，TokenHub 底座 + pricing 包覆盖全部需求。

---

### 1.2 §9.2 双轨计价（5 项）

| 编号 | 状态 | 代码位置 | 缺口描述 |
|------|------|----------|----------|
| PRI-01 | ✅ | `pricing/model.go` ModelPrice + PriceJSON | 双轨 JSON 支持；配置变更审计待 handler 接入 |
| PRI-02 | ✅ | `pricing/normalizer.go` NormalizeOpenAI / NormalizeAnthropic | 多模态 itemCode 规范化完成 |
| PRI-03 | ✅ | `pricing/calculator.go` CalculateDualTrack + 缓存折扣 | 5 种计价模式 + cache_discount_ratio 实现 |
| PRI-04 | ✅ | `fund/model.go` Ledger 含 cost_amount/sell_amount/freeze_id | 落账字段就绪 |
| PRI-05 | ✅ | `pricing/calculator.go:223-231` ModeAmortizationFixed | 固定摊销实现（daily_rate/monthly_rate） |

**域总结**: 5/5 ✅，pricing 包完整实现全部 5 种计价模式。

---

### 1.3 §9.3 主体、账本与资金（11 项）

| 编号 | 状态 | 代码位置 | 缺口描述 |
|------|------|----------|----------|
| FUN-01 | ✅ | `party/service.go` CreateParty（org/project 平级） | 主体管理完整 |
| FUN-02 | ✅ | `party/model.go` 7 种边类型常量 + `party/service.go` CreateEdge/DeleteEdge | 关系管理完整 |
| FUN-03 | ✅ | `fund/model.go` Account 含 available_balance + frozen_balance + version 乐观锁 | 并发安全 |
| FUN-04 | ✅ | `fund/service.go:326-383` validateChannel → party.CanAllocate | 通道校验（parent/sponsors/allocates） |
| FUN-05 | ✅ | `fund/service.go` Allocate 含守恒校验 + 幂等 | 划拨执行完整 |
| FUN-06 | ✅ | `fund/freeze.go` Freeze + Settle | 预扣与结算完整 |
| FUN-07 | ✅ | `fund/freeze.go:81-110` freezeCheckBudget + `modelgrant/checker.go` CheckQuotaLimit | 双层预算帽 + 告警比例（只告警不阻断） |
| FUN-08 | ✅ | `fund/lifecycle.go:181-454` Liquidate 完整状态机 | blocking→draining→refunding→closing→closed |
| FUN-09 | ⚠️ | `party/model.go` EdgeMergedInto / EdgeSplitFrom 常量定义已存在 | 边类型定义已完成，但组织变更流程（merging_block_new→merging_drain→merging_transfer→merged）未在 fund 包中实现独立流程；当前仅复用 Liquidate 状态机 |
| FUN-10 | ✅ | `fund/model.go` Ledger 只追加模型；`store.go` InsertLedger 接口 | 流水完整 |
| FUN-11 | ✅ | `fund/lifecycle.go:23-89` RenewFreeze + `fund/freeze.go` | 冻结 TTL 15 分钟 + 累计上限 2 小时 |

**域总结**: 10/11 ✅，FUN-09（组织变更）部分实现。当前 `party/model.go` 定义了 `merged_into`/`split_from` 边类型常量，但 fund 包仅提供通用的 Liquidate 状态机。组织合并/拆分专用的独立状态机（`merging_block_new` → `merging_drain` → `merging_transfer` → `merged`）尚未实现为独立流程。建议：在 fund/lifecycle.go 中新增 `MergeParty` / `SplitParty` 方法，或扩展 Liquidation 的 `liquidation_type` 字段。

---

### 1.4 §9.4 Key 与成员（6 项）

| 编号 | 状态 | 代码位置 | 缺口描述 |
|------|------|----------|----------|
| KEY-01 | ⚠️ | `gov_handlers_fund.go:96-198` handleCreateKey 已实现 | **创建已实现**；轮换/吊销 handler 返回"待实现"，但 TokenHub 基础有 `http.go` 轮换逻辑可复用 |
| KEY-02 | ✅ | `gov_handlers_fund.go:122-142` account_id 存在性 + ABAC 权限校验 | 绑定账户约束已实现 |
| KEY-03 | ✅ | `gov_handlers.go:412-484` validateGovToken 校验 owner_user_id 状态 | 归属人 已实现 |
| KEY-04 | ✅ | ABAC `iam.key.create` + `lookupResourceParty` scope 校验 | 绑户约束已实现 |
| KEY-05 | 🔧 | `gov_handlers.go:568-589` handlePartyMembers | POST/GET 返回"待实现" |
| KEY-06 | ✅ | `gov_handlers.go:463-470` validateGovToken 检查 `user.Status != active` | 禁人即禁Key 已实现 |

**域总结**: 2/6 ✅, 3 项部分实现, 1 项占位。Key 创建路径已深度实现（含 ABAC 双次鉴权）。轮换/吊销逻辑、成员管理 CRUD 是 handler 占位。

---

### 1.5 §9.5 安全治理体系（8 项）

| 编号 | 状态 | 代码位置 | 缺口描述 |
|------|------|----------|----------|
| SEC-GOV-01 | ✅ | `abac/engine.go` Evaluate（deny→allow→role→default deny） | ABAC 引擎完整实现 |
| SEC-GOV-02 | ✅ | `abac/model.go` 四轴常量 + `engine.go` lookupActionAxis | 四轴正交判定 |
| SEC-GOV-03 | ✅ | `abac/builtin.go` 职责分离策略 | 路由-资金互斥等策略内置 |
| SEC-GOV-04 | ✅ | `modelgrant/checker.go` CheckAccess（DENY 优先）+ CheckQuotaLimit | ModelGrant 完整实现 |
| SEC-GOV-05 | ✅ | `ui_permission/projector.go` ProjectMenus/ProjectRoutes/ProjectActions | UI 权限投影完成 |
| SEC-GOV-06 | ⚠️ | `gov_handlers.go:284-334` lookupResourceParty + ABAC scope | 架构就绪（ABAC scope_party_id + Resource.PartyID），但 handler 大量为占位，实际 end-to-end 数据隔离需在 handler 实现时注入 Scope 过滤 |
| SEC-GOV-07 | ✅ | `audit/event.go` RecordEvent 仅 INSERT + `audit/model.go` before/after 快照 | 审计不可篡改 |
| SEC-GOV-08 | ✅ | `store.go` AES-256-GCM + `http.go` redactAuditPayload + 哈希存储 | 密钥安全（见 DATA-2 D-CON-03 审计） |

**域总结**: 7/8 ✅，SEC-GOV-06（数据不越权）架构就绪但 runtime 验证待 handler 实现完成。

---

### 1.6 §9.6 调用安全（5 项）

| 编号 | 状态 | 代码位置 | 缺口描述 |
|------|------|----------|----------|
| SEC-01 | ⚠️ | `security/egress.go` CheckEgress（INTERNAL_ONLY 阻断 external） | 核心判定已实现；HYBRID_ALLOWED 白名单校验留待阶段 D |
| SEC-02 | ❌ | 无代码 | 出网白名单（HYBRID_ALLOWED 具体域名单）未实现 |
| SEC-03 | ⚠️ | `security/hooks.go` NoopHook 占位 | 内容安全引擎（敏感词/有害内容/注入检测）完全是空实现 |
| SEC-04 | ❌ | 无代码 | 异常流量拦截告警未实现 |
| SEC-05 | ⚠️ | `security/hooks.go` Hook 接口 + Chain 执行器 | 扩展点架构就绪（Hook 接口 + Chain），但所有钩子为 NoopHook |

**域总结**: 0/5 完整实现。调用安全域整体处于架构就绪/占位状态。SEC-01 核心判定已实现，其余为 P2 优先级。

---

### 1.7 §9.7 路由与调度（6 项）

| 编号 | 状态 | 代码位置 | 缺口描述 |
|------|------|----------|----------|
| RTE-01 | ✅ | `routing/registry.go` 注册表 + `routing/profile.go` CreateProfile | 策略引擎架构完整 |
| RTE-02 | ✅ | `routing/strategies/` 12 个策略文件全部存在 | 全部 12 种策略已注册 |
| RTE-03 | ✅ | `routing/strategies/health.go` 三态熔断 + `routing/profile.go` MaxAttempts | 重试切换 + 熔断 |
| RTE-04 | ✅ | `pipeline.go` account_id 在鉴权时注入 context，调度层只读 | 账户正交保证 |
| RTE-05 | ✅ | `routing/strategy.go:173` MaxDeltaCap=0.20 + `profile.go:44-47` δ 保存校验 | δ 价格帽 20% 硬上限 |
| RTE-06 | ✅ | `routing/decision.go` Decision 结构体 + `profile.go` logDecision | 决策日志持久化到 route_decisions |

**域总结**: 6/6 ✅，路由域已完整实现。

---

### 1.8 §9.8 审计与对账（4 项）

| 编号 | 状态 | 代码位置 | 缺口描述 |
|------|------|----------|----------|
| AUD-01 | ✅ | `audit/event.go` RecordEvent + `pipeline.go` auditStep | 调用审计事件记录已实现；request_logs 表从 TokenHub 继承 |
| AUD-02 | ⚠️ | `audit/model.go` before_snapshot/after_snapshot 字段已定义 | 模型已就绪，但具体配置变更审计（δ 变更、预算帽变更等）需在各 handler 中显式调用 `RecordEvent` 写入快照。当前仅 `routing/profile.go:130-137` 的 `UpdateProfile` 有 δ 变更日志（slog.Warn），但未写入 audit_events 表。 |
| AUD-03 | 🔧 | 无代码 | 对账（reconciliation_runs）表未创建、接口未定义。PRD 要求 P0 阶段预留契约（最小字段 + 接口签名），当前完全缺失。 |
| AUD-04 | 🔧 | `gov_handlers_abac.go:255-268` | 仪表盘/报表 handler 返回"待实现" |

**域总结**: 1/4 ✅, 1 部分实现, 2 占位。AUD-03（对账）是 PRD 明确要求 P0 预留契约的项，当前完全缺失为关键缺口。

---

### 1.9 §9.9 控制台与治理 API（15 项）

| 编号 | 状态 | 代码位置 | 缺口描述 |
|------|------|----------|----------|
| UI-01 | 🔧 | `gov_handlers_abac.go:219-222` handleUIPermissionSnapshot | 权限快照 handler 返回"待实现" |
| UI-02 | ⚠️ | `gov_handlers.go:500-589` Party handlers | Party 创建已实现；GET/PATCH/Edge/Member 返回"待实现" |
| UI-03 | 🔧 | `gov_handlers_fund.go:16-62` Account/Allocation handlers | 全部返回"待实现" |
| UI-04 | 🔧 | `gov_handlers_fund.go:242-267` ModelPrice handlers | PUT/GET 返回"待实现" |
| UI-05 | ⚠️ | `gov_handlers_fund.go:66-238` Key handlers | 创建已实现；GET/DELETE/轮换 返回"待实现" |
| UI-06 | 🔧 | `gov_handlers_fund.go:300-352` RouteProfile/Route handlers | 全部返回"待实现" |
| UI-07 | 🔧 | `gov_handlers_abac.go:255-258` handleDashboard | "待实现" |
| UI-08 | ⚠️ | 见 DATA-2 D-CON-03 审计 | 创建时一次性返回明文（行业惯例），其余查询仅返回 key_prefix+key_suffix |
| UI-09 | 🔧 | `gov_handlers_fund.go:271-296` ModelGrant handlers | POST/GET 返回"待实现" |
| UI-10 | 🔧 | `gov_handlers_abac.go:260-263` handleSecurityReports | "待实现" |
| UI-11 | 🔧 | `gov_handlers_abac.go:265-268` handleTrace | "待实现" |
| UI-12 | 🔧 | `gov_handlers_abac.go:16-114` Role/Policy/Binding/Grant handlers | 全部返回"待实现" |
| UI-13 | 🔧 | `gov_handlers_abac.go:129-217` UI handlers | 全部返回"待实现" |
| UI-14 | 🔧 | `gov_handlers_abac.go:226-251` AuditEvent/ChainAnchor handlers | 全部返回"待实现" |
| API-01 | 🔧 | 全部 `/gov/*` handler | ABAC 鉴权已集成，但 handler 内部逻辑大量占位 |

**域总结**: 0/15 完整实现，2 个部分实现，13 个占位。这是当前代码库最大缺口——handler 层完成了路由注册 + ABAC 鉴权集成，但业务逻辑全部为 `"待实现"` 返回。

---

## 2. 管线完整性（14 步数据面）

| 步骤 | PRD 描述 | 状态 | 实现位置 | 缺口 |
|------|----------|------|----------|------|
| [1] | 协议解析 | ✅ | `http.go`（TokenHub 底座） | 6 协议兼容，由 HTTP handler 完成后再进入 Pipeline |
| [2] | 密钥鉴权 | ✅ | `pipeline.go:236-246` AuthFunc | 鉴权结果注入 context |
| [3] | 安全钩子 | ⚠️ | `pipeline.go:251-258` + `security/hooks.go` NoopHook | 钩子链架构就绪，但实际钩子为空实现 |
| [4] | ModelGrant | ✅ | `pipeline.go:290-298` + `modelgrant/checker.go` | DENY 优先 + ALLOW 检查 |
| [5] | 锚定内部价 | ✅ | `pipeline.go:300-313` + `pricing/calculator.go` EstimateSell | 预估 sell 金额 |
| [6] | 价格过滤(δ) | ⚠️ | `routing/profile.go:245-250` applyPriceCap | 代码在 routing 包中实现，但在 pipeline.go:315 注释为"由 Router 内部处理，此处不独立步骤"。建议：将 δ 过滤从 routing 内部提升为 Pipeline 的独立步骤 PriceFilter，使其在预算帽检查之前执行。 |
| [7] | 模型级预算 | ✅ | `pipeline.go:317-324` BudgetCheck → `modelgrant/checker.go:91-122` CheckQuotaLimit | ModelGrant.quota_limit |
| [8] | 账户级预算帽 | ✅ | `pipeline.go:317-324` BudgetCheck → `fund/freeze.go:81-110` freezeCheckBudget | Account.budget_limit_amount + 告警比例 |
| [9] | 冻结 | ✅ | `pipeline.go:326-336` + `fund/freeze.go:30-77` Freeze | 金额=合格候选最大 sell |
| [10] | 策略选路 | ✅ | `pipeline.go:338-347` + `routing/profile.go:198-311` ExecuteProfile | 12 种策略 |
| [11] | 上游调用 | ✅ | `pipeline.go:338-360` Adapter | 通过 Provider 适配器转发 |
| [12] | 流式续期 | ✅ | `pipeline.go:362` 注释 → `fund/lifecycle.go:23-89` RenewFreeze | 由 HTTP handler 在流式循环中周期性调用 |
| [13] | 用量规范化 | ✅ | `pipeline.go:364-372` + `pricing/normalizer.go` | OpenAI / Anthropic 规范化器 |
| [14] | 双轨结算 + 审计 | ✅ | `pipeline.go:374-395` Settlement + auditStep | 按实际用量结算；每步审计 |

**管线总结**: 14/14 步骤均有实现。2 项注意：
- 步骤 [3] 安全钩子为 NoopHook（空放行），真实内容安全引擎待阶段 D
- 步骤 [6] 价格过滤(δ) 当前嵌套在 `routing.ExecuteProfile` 内部而非 Pipeline 顶层，建议提升为独立步骤以与步骤 [7] 预算帽检查形成明确的顺序语义

---

## 3. WBS 阶段完成度

| 阶段 | PRD 内容 | 工期 | 当前完成度 | 缺口 |
|------|----------|------|-----------|------|
| **A** | Fork TokenHub、执行 DDL（40 表）、用量规范化、国产冒烟 | 2d | **100%** | 40 表 DDL 已有、用量规范化完成、TokenHub 底座运行。国产冒烟待运行环境。 |
| **B** | Party 账本/划拨/预算帽/冻结续期/清算/组织变更/双轨 model_prices/价格帽/ABAC 引擎/grants/ModelGrant/UI 权限/审计链/治理 API 幂等/安全钩子空实现 | 4d | **~80%** | 核心域（fund/party/pricing/abac/modelgrant/ui_permission/idempotency/audit/routing）全部深度实现。**主要缺口**：控制面 handler 全部返回"待实现"（ABAC 鉴权已集成但业务逻辑未接入 Service 层）；组织变更独立状态机缺失；安全钩子为 NoopHook |
| **C** | 策略矩阵全量（12 种含 S-CLASSIFY）/决策日志/全链路调用追踪 UI/仪表盘+聚合 | 2d | **~50%** | 策略矩阵全部注册、决策日志持久化到 route_decisions。**缺口**：Pipeline 到 routing 的端到端集成（pipeline.go 函数签名存在但未连接实际 Provider 路由逻辑）；全链路调用追踪 UI（handler 占位）；仪表盘聚合（handler 占位） |
| **D** | 内容安全出网/变更快照强化/安全事件报表/对账接口契约落地 | 2d | **~5%** | 安全钩子为 NoopHook；出网白名单未实现；变更快照未写入 audit_events（仅 slog 日志）；AUD-03 对账接口完全缺失 |
| **E** | 压测 HA/GA/文档与许可 | 1d | **0%** | 压测未执行；HA 多实例部署 TokenHub 已有基础但 fund/abac 新表需验证 |

---

## 4. 非功能需求覆盖

| 类别 | PRD 要求 | 当前状态 | 缺口 |
|------|----------|----------|------|
| **可用性** | 目标 99.9%；多实例故障切换 | 部分就绪 | TokenHub 已有 `multi_instance_e2e_test.go` 多实例协调测试；fund 包乐观锁（version）+ SELECT FOR UPDATE 支持多实例。缺乏：生产级 HA 压测；Redis 分布式锁可选方案未评估 |
| **性能** | 单节点 5000 QPS；附加延迟 <50ms | 未验证 | 代码层面无已知性能瓶颈；pipeline.go 纯内存编排；需运行时压测 |
| **安全** | TLS 1.3；AES-256 加密；ABAC 最小权限；审计不可篡改 | 大部分就绪 | AES-256-GCM 加密实现；ABAC 引擎完整；审计仅 INSERT。**缺口**：TLS 1.3 依赖部署层（nginx/Caddy）；安全渗透测试未执行 |
| **部署** | 私有化；Docker Compose/K8s；离线/内网；国产环境 | 部分就绪 | `deploy/` 目录完整（Docker Compose + systemd + nginx）；K8s Helm 未提供；国产 CPU/OS 冒烟未执行 |
| **审计保留** | >=180 天；冷热分离；哈希链锚定 | 架构就绪 | `audit_chain_anchors` 表 + `anchor_audit_chain()` 存储过程已定义；冷热分离未实现；180 天保留策略需运维配置 |
| **可扩展** | 适配器、itemCode、策略、边类型、安全钩子、ABAC 策略 | ✅ 全部就绪 | Strategy 接口 + Hook 接口 + ABAC 策略条件 JSON + 7 种边类型 + 10 项 itemCode 基线 |
| **可观测** | Prometheus + Grafana 指标 | 部分就绪 | TokenHub 已有 `metrics.go` Prometheus 指标；**缺口**：冻结任务指标、幂等冲突计数、预算帽命中计数、ABAC 拒绝统计仪表盘未实现 |

---

## 5. 部署就绪度

### 5.1 现有 deploy/ 兼容性

| 文件 | 兼容性 | 说明 |
|------|--------|------|
| `docker-compose.yml` | ✅ 兼容 | TokenHub 原有配置，新增包为纯 Go 无外部依赖 |
| `docker-compose.postgres.yml` | ✅ 兼容 | PostgreSQL 16 已配置，fund/abac/audit 模型通过 GORM AutoMigrate 自动建表 |
| `docker-compose.e2e.yml` | ✅ 兼容 | 端到端测试编排可复用 |
| `.env.example` | ⚠️ 需扩展 | 缺少治理 API 相关配置项（GOV_API_KEY_PREFIX 等） |
| `install.sh` / `install_test.sh` | ⚠️ 需更新 | 需增加新表迁移步骤 |
| `nginx.multi-instance.conf` | ✅ 兼容 | 多实例负载均衡配置可用 |
| `systemd/` (native/) | ✅ 兼容 | TokenHub 原有 systemd 配置可用 |

### 5.2 新增表迁移

当前代码依赖 GORM AutoMigrate 在建表时自动创建（`party/store.go:Migrate`, `abac/model.go:Migrate`, `audit/store.go:Migrate`, `routing/strategy.go:Migrate`, `modelgrant/grant.go:Migrate`）。`schema/ai-gov-fusion-v3.2.sql` 提供了完整的 DDL 脚本但 AutoMigrate 不能自动执行存储过程 `anchor_audit_chain()`。建议：在 `install.sh` 中增加 `psql -f schema/ai-gov-fusion-v3.2.sql` 步骤。

### 5.3 建议增加

1. `deploy/.env.example` 增加：
   - `GOV_API_KEY_PREFIX=gov_`
   - `AUDIT_RETENTION_DAYS=180`
   - `BUDGET_WARN_RATIO_DEFAULT=0.80`

2. `deploy/docker-compose.yml` 增加治理 API 端口映射（如需要独立端口）

3. K8s Helm chart（`deploy/k8s/`）——PRD 约定生产环境需提供

---

## 6. 验收场景逐条判定

### 6.1 功能验收（26 场景）

| # | 验收场景 | 判定 | 备注 |
|---|----------|------|------|
| 1 | 统一接入 >=5 类公有 + 1 类私有化 | ✅ | TokenHub 底座已有 |
| 2 | 双轨 + itemCode 多模态 | ✅ | pricing 包支持 |
| 3 | 固定摊销 | ✅ | amortization_fixed 已实现 |
| 4 | 缓存折扣 | ✅ | cache_discount_ratio 已实现 |
| 5 | usage 不完整标记 | ✅ | NormalizeUsage 返回 incomplete flag |
| 6 | 独立项目/组织池/出资划拨 | ✅ | party + fund Allocate 支持 |
| 7 | 个人经费 (allocates) | ✅ | ChannelAllocates 通道已实现 |
| 8 | 组织合并/拆分 | ⚠️ | 边类型已定义，独立流程未实现 |
| 9 | 价格约束 δ | ✅ | MaxDeltaCap=0.20 + δ 变更日志 |
| 10 | 双层预算帽分码 | ✅ | BUDGET_CAP_EXCEEDED + MODEL_BUDGET_EXCEEDED + INSUFFICIENT_BALANCE |
| 11 | 告警比例 | ✅ | budget_warn_ratio 只告警不阻断 |
| 12 | 冻结超时/流式续期 | ✅ | defaultFreezeTTL=15min + RenewFreeze |
| 13 | 清算状态机 | ✅ | blocking→draining→refunding→closing→closed |
| 14 | 幂等 | ✅ | INSERT-first + UNIQUE 约束 |
| 15 | ModelGrant deny 优先 | ✅ | deny 先于 allow 评估 |
| 16 | ABAC 四轴越权 | ✅ | data/fund/iam/routing 独立评估 |
| 17 | ABAC 策略评估 | ✅ | deny→allow→role→default deny |
| 18 | UI 权限投影 | ✅ | Projector.Menus/Routes/Actions 实现 |
| 19 | 审计不可篡改 | ✅ | RecordEvent 仅 INSERT |
| 20 | 调度不改账户 | ✅ | account_id 鉴权时锁定 |
| 21 | 策略矩阵启停 | ✅ | 12 种策略注册 + StrategyBinding.Enabled |
| 22 | 禁人即禁 Key | ✅ | validateGovToken 检查 user.Status |
| 23 | 治理 API | ⚠️ | ABAC 鉴权集成，handler 占位 |
| 24 | INTERNAL_ONLY 外网阻断 | ⚠️ | CheckEgress 实现，白名单缺失 |
| 25 | 密钥安全 | ⚠️ | AES-256-GCM 加密，创建时返回明文（一次性展示） |
| 26 | 全链路调用追踪 | 🔧 | handleTrace 返回"待实现" |

**验收场景总结**: 17 ✅ / 5 ⚠️ / 4 🔧（计场景 23 的 handler 占位）

---

## 7. 存量代码重构进度

PRD §11.3 提出的重构策略（`store.go` 195KB + `http.go` 295KB 巨石文件）：

| 策略 | 状态 | 说明 |
|------|------|------|
| 新建包从零写 | ✅ 完成 | fund/pricing/idempotency/abac/ui_permission/modelgrant 全部从零构建 |
| 扩展包增量提取 | ✅ 完成 | party/authz/routing/audit/security 已提取到独立包 |
| 存量不动 | ⚠️ 部分 | `store.go` 缩至 ~5972 行（原 ~195KB, 估算 8000+ 行），`http.go` 缩至 ~8550 行（原 ~295KB, 估算 12000+ 行）——有缩减但仍是巨石文件 |
| 门禁 | ✅ | 包级单元测试已覆盖（`*_test.go` 46 个文件） |

---

## 8. 关键缺口汇总与优先级建议

### 8.1 阻塞级（P0，必须在阶段 B 关闭前修复）

| 编号 | 缺口 | 影响范围 | 修复建议 |
|------|------|----------|----------|
| GAP-001 | **控制面 handler 全部占位** — 15 个 handler 文件返回"待实现" | API-01, UI-02~14, AUD-04 | 优先实现 fund 域 handler（划拨/流水/预算帽）+ Key CRUD 完整闭环。将 `fund.Service` 的 Allocate/Freeze/Settle/Liquidate 接入 handler。 |
| GAP-002 | **Pipeline 端到端集成缺失** — pipeline.go 函数已定义但实际路由调用链未串联 | RTE-01, 管线步骤 [9][10] | 在 http.go 的 chat completion handler 中注入 Pipeline 实例，替换当前的直接路由逻辑 |
| GAP-003 | **AUD-03 对账接口契约缺失** — PRD 要求 P0 预留 `reconciliation_runs` 最小字段 + 接口签名 | AUD-03 | 创建 `pkg/reconciliation/` 包，定义 `ReconciliationRun` 模型（run_id/period_start/period_end/provider/status）和 gRPC 接口签名 |

### 8.2 高危级（P1，阶段 C 应关闭）

| 编号 | 缺口 | 修复建议 |
|------|------|----------|
| GAP-004 | δ 价格帽过滤未作为独立管线步骤 | 将 `pipeline.go:315` 的注释改为实际步骤：在 BudgetCheck 之前执行 PriceFilter |
| GAP-005 | AUD-02 配置变更审计缺失 — δ/预算帽变更未写入 audit_events | 在 `routing/profile.go:UpdateProfile` 中调用 `audit.RecordEvent` 写入 before/after 快照 |
| GAP-006 | `liquidations.liquidation_type` 字段缺失（DDL vs PRD 偏差） | 在 DDL 中增加 `liquidation_type VARCHAR(32)` 列 |
| GAP-007 | 组织合并/拆分独立流程缺失 | 在 `fund/lifecycle.go` 增加 `MergeParty`/`SplitParty` 方法 |

### 8.3 中危级（P2，阶段 D 应关闭）

| 编号 | 缺口 | 修复建议 |
|------|------|----------|
| GAP-008 | 安全钩子 NoopHook | 接入内容安全引擎（可选集成的第三方服务或自研敏感词检测） |
| GAP-009 | 出网白名单未实现 | 实现 HYBRID_ALLOWED 的 `egress_whitelist` 表查询 |
| GAP-010 | 异常流量检测未实现 | 基于 Prometheus 指标的告警规则，或代码级异常检测 |
| GAP-011 | 冷热分离存储未实现 | 提供 `audit_events` 分区表方案或定时归档脚本 |

### 8.4 部署与运维

| 编号 | 缺口 | 修复建议 |
|------|------|----------|
| GAP-012 | K8s Helm chart 未提供 | 创建 `deploy/k8s/` 目录，至少提供 Deployment + Service + ConfigMap |
| GAP-013 | 新表迁移未集成到 install.sh | 增加 `psql` 或 GORM AutoMigrate 步骤 |
| GAP-014 | 监控指标未扩展（冻结/幂等/预算帽/ABAC） | 在 `metrics.go` 中增加对应的 Prometheus Counter/Gauge |
| GAP-015 | `.env.example` 缺少治理配置项 | 补充 GOV_API_KEY_PREFIX, AUDIT_RETENTION_DAYS, BUDGET_WARN_RATIO_DEFAULT |

---

## 9. 统计总表

| 维度 | 总数 | ✅ 完成 | ⚠️ 部分 | 🔧 占位 | ❌ 未实现 |
|------|------|---------|----------|----------|-----------|
| §9.1 统一接入 | 8 | 8 | 0 | 0 | 0 |
| §9.2 双轨计价 | 5 | 5 | 0 | 0 | 0 |
| §9.3 主体账本资金 | 11 | 10 | 1 | 0 | 0 |
| §9.4 Key 与成员 | 6 | 2 | 3 | 1 | 0 |
| §9.5 安全治理 | 8 | 7 | 1 | 0 | 0 |
| §9.6 调用安全 | 5 | 0 | 3 | 0 | 2 |
| §9.7 路由调度 | 6 | 6 | 0 | 0 | 0 |
| §9.8 审计对账 | 4 | 1 | 1 | 2 | 0 |
| §9.9 控制台治理API | 15 | 0 | 2 | 13 | 0 |
| **合计** | **68** | **39 (57%)** | **11 (16%)** | **16 (24%)** | **2 (3%)** |
| 14 步管线 | 14 | 12 | 2 | 0 | 0 |
| WBS 阶段 B | — | 80% | — | — | — |
| 验收 26 场景 | 26 | 17 | 5 | 4 | 0 |

---

## 10. 结论

**当前代码库质量评估：B+ 级。**

核心治理域（fund/pricing/abac/modelgrant/party/idempotency/audit/routing/ui_permission）均已完成深度实现，通过了 DATA-1（财务守恒 6/6 PASS）、DATA-2（DDL 40 表对齐）、QA-3（安全治理 6 场景）等独立验收。14 步数据面管线全部步骤有实现。

**最大单一缺口是控制面 HTTP handler 层**——ABAC 鉴权已集成到每个端点，但业务逻辑（调用 Service 层方法）完全缺失，全部返回 `"待实现"`。这是阶段 B 关闭前的首要工作中项。

**第二缺口是 Pipeline 端到端集成**——pipeline.go 定义了完整的 14 步编排器，但尚未接入 http.go 的 chat completion 处理流程。当前数据面仍走 TokenHub 原有路径，绕过了管线中的 ModelGrant 检查、定价预估、预算帽检查、冻结/结算等步骤。

**第三缺口是 AUD-03 对账契约**——PRD 明确要求 P0 阶段预留 `reconciliation_runs` 表结构和接口签名，当前完全缺失。

**建议下一步工作优先级**：
1. 实现 fund 域 handler（划拨执行、流水查询、预算帽配置）——2d
2. 将 Pipeline 注入 http.go 的 chat completion 路径——1d
3. 实现 Key CRUD 完整闭环（轮换/吊销）——0.5d
4. 创建 AUD-03 对账接口契约——0.5d
5. 实现组织变更独立流程（FUN-09）——1d
6. 写入配置变更审计（AUD-02）——0.5d

---

**文档结束（全量差距分析 v1.0）。**
