# PRD-1 财务闭环演示脚本审计报告

> 审计基准：PRD §13.3 财务演示脚本  
> 审计时间：2026-07-31  
> 审计结论：**11 步骤中 8 步可执行（含 2 步有实质缺口），3 步阻塞性缺口**

---

## 审计总览

| 步骤 | 名称 | 判定 | 关键缺口 |
|------|------|------|----------|
| 1 | 配置预算与加价 | 可执行 | 无独立 markup 字段，以独立 sell tier 替代 |
| 2 | 划拨 | 部分可执行 | channel 语义校验为占位实现，未集成 party_edges |
| 3 | 创建人 Key 绑项目账本 | **不可执行** | POST /gov/keys 返回占位，无 account_id 绑定逻辑 |
| 4 | 调用（管线） | 部分可执行 | 14 步骨架完整，但 [6] 价格过滤缺独立实现、[11] 流式续期缺集成 |
| 5 | 核对 sell/cost/流水 | 可执行 | Ledger 有 cost_amount/sell_amount；AuditEvent 无专用字段 |
| 6 | 双层预算帽分码 | 可执行 | BUDGET_CAP_EXCEEDED vs MODEL_BUDGET_EXCEEDED 独立编码 |
| 7 | 余额不足分码 | 可执行 | INSUFFICIENT_BALANCE 独立错误码 |
| 8 | 个人经费注入 | 部分可执行 | EdgeAllocates 边类型存在，但 channel 语义校验为占位 |
| 9 | 清算回流 | 可执行 | 5 阶段状态机完整，余额转入目标账户 |
| 10 | 组织合并 | 部分可执行 | EdgeMergedInto 边类型存在，但无独立合并逻辑，依赖清算流 |
| 11 | 幂等划拨 | 可执行 | INSERT-first 去重，同 key 重复提交仅入账一次 |

---

## 逐步骤详审

### 步骤 1：配置预算与加价

**审查文件：**
- `pricing/model.go`（L155-L268）
- `fund/model.go`（L168-L190）

**审查结论：可执行**

| 审查点 | 代码位置 | 状态 | 说明 |
|--------|----------|------|------|
| PriceJSON 是否支持 sell 独立定价 | `pricing/model.go` L164-L172 | 通过 | `PriceItem` 同时包含 `Cost PricingTier` 和 `Sell PricingTier`，sell 轨道独立可配 |
| 是否有显式 markup/加价率字段 | `pricing/model.go` L164-L172 | 缺口 | 无独立的 markup/margin 字段。加价通过 sell tier 独立于 cost tier 设置实现（隐式加价），但无法直接表达"在 cost 基础上 +20%" |
| Account 是否有 budget_limit_amount | `fund/model.go` L177 | 通过 | `BudgetLimitAmount *Decimal` 字段存在，支持 NULL（不限制） |
| 是否支持 budget_warn_ratio | `fund/model.go` L178 | 通过 | `BudgetWarnRatio *Decimal` 存在，用于预警阈值 |

**缺口说明：** sell 定价是独立的 tier（可配置不同 mode/rate/tiers），但缺少显式 `markup_percent` 或 `margin` 字段。若演示脚本需要"cost 基础上浮动 X%"，当前只能手动计算 sell 值后写入。定价模型自身是功能完备的。

---

### 步骤 2：划拨（Allocate）

**审查文件：**
- `fund/service.go`（L72-L279）
- `fund/model.go`（L130-L141, L246-L264）
- `fund/errors.go`（L79-L104）

**审查结论：部分可执行（channel 语义校验为占位）**

| 审查点 | 代码位置 | 状态 | 说明 |
|--------|----------|------|------|
| 是否支持 parent/sponsors/allocates 通道 | `fund/model.go` L135-L140 | 通过 | `ChannelParent`, `ChannelSponsors`, `ChannelAllocates`, `ChannelWhitelist` 四个常量 |
| 是否在同一事务内双边记账 | `fund/service.go` L109-L265 | 通过 | 双账户 FOR UPDATE 锁（L112-138），同一 WithTx 内 dedit(src) + credit(dst) 各一条 Ledger 流水（L199-227） |
| 死锁防护 | `fund/service.go` L111-113 | 通过 | 按 account_id 排序后加锁（firstID > secondID 交换） |
| channel 语义校验（核心缺口） | `fund/service.go` L286-298 | **缺口** | `validateChannel()` 对 4 种 channel 一概返回 nil（L288-294），注释明确标注"Full semantic validation...is deferred to a later stage" |
| 防自转 | `fund/service.go` L77-79 | 通过 | `SrcAccountID == DstAccountID` 时返回 SELF_TRANSFER 错误 |

**缺口说明：** 通道权限的语义校验（验证 party_edges 表是否存在对应 parent/sponsors/allocates 边，验证 whitelist 表是否有授权记录）是占位实现。演示脚本中的"上级拨付"、"赞助"、"个人经费注入"等操作在代码层面不做边关系校验即可通过，但这不是生产就绪行为。

---

### 步骤 3：创建人 Key 绑项目账本

**审查文件：**
- `gov_handlers_fund.go`（L61-L72）
- `gov_handlers.go`（L112-L113）

**审查结论：不可执行（阻塞性缺口）**

| 审查点 | 代码位置 | 状态 | 说明 |
|--------|----------|------|------|
| POST /gov/keys 路由注册 | `gov_handlers.go` L112 | 通过 | 路由已注册 |
| 是否强制绑定 account_id | `gov_handlers_fund.go` L61-L66 | **缺口** | handler 直接返回 `{"message": "Key 创建——待实现"}`，无任何请求体解析、无 account_id 校验、无 account 存在性验证 |
| 是否校验 iam 轴权限 | `gov_handlers_fund.go` L64 | 部分 | 调用了 `requireGovAuth(w, r, "iam.key.create")`，但 Key 创建逻辑完全缺失 |
| 是否写入 API Key 记录 | 无 | **缺口** | 无 Key 模型、无 Key 创建服务、无 Key ↔ Account 绑定关系 |

**缺口说明：** 这是三个阻塞性缺口之一。演示脚本第三步"创建人 Key 绑项目账本"完全无法执行——handler 只返回占位 JSON，无任何业务逻辑。需要补充：1) Key 数据模型（含 account_id 外键）；2) Key 创建 service；3) account 存在性 + 状态校验；4) iam 轴 ABAC 权限校验。

---

### 步骤 4：调用（14 步管线）

**审查文件：**
- `pipeline.go`（L224-L362）

**审查结论：部分可执行（骨架完整，但步骤 [6] 和 [11] 缺独立实现）**

管线 14 步骤覆盖分析：

| 步骤编号 | 步骤名称 | 代码行 | 状态 |
|----------|----------|--------|------|
| [1] | 协议解析 | 注释 L179 | 由 HTTP handler 完成，不进 Pipeline |
| [2] | 密钥鉴权 | L233-L243 | 通过（AuthFunc 注入，有审计日志） |
| [3] | 安全钩子 | L246-L253 | 通过（SecurityHookFunc 注入） |
| [4] | ModelGrant 检查 | L256-L264 | 通过（ModelGrantCheckFunc 注入） |
| [5] | 价格预估 | L267-L279 | 通过（PricingFunc 注入） |
| [6] | 价格过滤(δ) | L281 | **缺口**——注释"由 Router 内部处理，此处不独立步骤" |
| [7] | 预算帽检查 | L283-L290 | 通过（BudgetCheck 注入） |
| [8] | 冻结 | L293-L302 | 通过（Freeze 注入） |
| [9] | 策略路由 | L305-L313 | 通过（Router + RouteSelectFunc 注入） |
| [10] | 上游调用 | L315-L326 | 通过（Adapter + UpstreamCallFunc 注入） |
| [11] | 流式续期 | L328 | **缺口**——注释"由 HTTP handler 在流式写入循环中周期性调用，此处不执行" |
| [12] | 用量规范化 | L331-L338 | 通过（Normalizer + UsageNormalizeFunc 注入） |
| [13] | 双轨结算 | L341-L353 | 通过（Settlement 注入，记录 cost_amount + sell_amount） |
| [14] | 审计持久化 | L355-L361 | 通过（各步骤 auditStep 累积，汇总全链路耗时） |

**缺口说明：**
1. 步骤 [6] 价格过滤(δ)：Pipeline 中有 `PriceFilter` 字段（L192），但 Execute 中未调用，注释标记为 Router 内部处理。若演示需要独立演示 δ 过滤逻辑，需要补齐独立调用。
2. 步骤 [11] 流式续期：Pipeline 有 `StreamRenewal` 字段（L201），但 Execute 中标记为"不执行"。流式响应的 freeze 续期逻辑需在 HTTP handler 层集成 `fund.Service.RenewFreeze()`。
3. 所有步骤函数均通过字段注入——Pipeline.Execute 本身是完整骨架，但实际行为取决于注入的函数实现。若注入函数为 nil，对应步骤被跳过（silent skip）。

---

### 步骤 5：核对 sell/cost/流水

**审查文件：**
- `audit/model.go`（L49-L165）
- `audit/event.go`（L26-L59）
- `fund/model.go`（L198-L215）

**审查结论：可执行**

| 审查点 | 代码位置 | 状态 | 说明 |
|--------|----------|------|------|
| Ledger 是否有 cost_amount 字段 | `fund/model.go` L205 | 通过 | `CostAmount *Decimal`，允许 NULL |
| Ledger 是否有 sell_amount 字段 | `fund/model.go` L206 | 通过 | `SellAmount *Decimal`，允许 NULL |
| Settle 时是否写入 cost/sell 到 Ledger | `fund/freeze.go` L306-L314 | 通过 | settle ledger 的 CostAmount 和 SellAmount 均被设置（L313-L314） |
| AuditEvent 是否有 cost/sell 字段 | `audit/model.go` L57-L98 | 缺口 | AuditEvent 无专用 cost_amount/sell_amount 字段。成本信息通过 BeforeSnapshot/AfterSnapshot JSON（L82-L88）或 Message（L80）间接承载 |
| PipelineAuditEvent 是否支持双轨 | `pipeline.go` L115-L130 | 间接 | `Detail map[string]any` 可承载任意数据，步骤 [13] 结算审计中传入了 cost_amount/sell_amount（pipeline.go L349-L351） |

**缺口说明：** AuditEvent（管理面审计）不直接携带 cost/sell 金额字段，管理操作的财务影响需从快照 JSON 中解析。但数据面的 Ledger 流水和 PipelineAuditEvent 均记录了完整的双轨金额，对演示脚本"核对 sell/cost/流水"是可执行的。

---

### 步骤 6：双层预算帽分码

**审查文件：**
- `fund/freeze.go`（L57-L67）
- `fund/errors.go`（L97-L103）
- `modelgrant/checker.go`（L15-L21）

**审查结论：可执行**

| 审查点 | 代码位置 | 状态 | 说明 |
|--------|----------|------|------|
| 账户级预算帽错误码 | `fund/errors.go` L97-L103 | 通过 | `newBudgetCapExceededError()` → Code: `"BUDGET_CAP_EXCEEDED"` |
| 模型级预算超限错误码 | `modelgrant/checker.go` L18 | 通过 | `ErrModelBudgetExceeded` → message: `"MODEL_BUDGET_EXCEEDED"` |
| 两者是否不同 | — | 通过 | `BUDGET_CAP_EXCEEDED` vs `MODEL_BUDGET_EXCEEDED`，独立编码，可区分 |
| 拦截顺序 | `pipeline.go` L283-290 | 通过 | Pipeline [7] 预算帽检查（账户级）在 [4] ModelGrant 检查（含模型级配额）之后——正确的双层拦截顺序 |

---

### 步骤 7：余额不足分码

**审查文件：**
- `fund/errors.go`（L37-L38, L79-L86）

**审查结论：可执行**

| 审查点 | 代码位置 | 状态 | 说明 |
|--------|----------|------|------|
| 是否独立错误码 | `fund/errors.go` L80-L86 | 通过 | `newInsufficientBalanceError()` → Code: `"INSUFFICIENT_BALANCE"` |
| 错误信息是否含诊断信息 | `fund/errors.go` L82-83 | 通过 | 消息含 account_id、available 余额、requested 金额 |
| 在 Freeze 中是否触发 | `fund/freeze.go` L84-86 | 通过 | 余额不足时返回 INSUFFICIENT_BALANCE |
| 在 Allocate 中是否触发 | `fund/service.go` L155-157 | 通过 | 划拨时余额不足同样返回 INSUFFICIENT_BALANCE |

---

### 步骤 8：个人经费注入（EdgeAllocates）

**审查文件：**
- `party/model.go`（L50-L53, L78-L94）
- `fund/service.go`（L286-L298）
- `fund/model.go`（L138）

**审查结论：部分可执行（边类型存在，但 channel 语义校验为占位）**

| 审查点 | 代码位置 | 状态 | 说明 |
|--------|----------|------|------|
| EdgeAllocates 边类型是否存在 | `party/model.go` L50-L53 | 通过 | `EdgeAllocates = "allocates"`，注释明确"Fund transfer IS allowed (party→person account)" |
| fundAutoEdges 是否包含 allocates | `party/model.go` L90-L94 | 通过 | `fundAutoEdges` 包含 `EdgeAllocates: true` |
| Allocate 是否接受 allocates channel | `fund/service.go` L288-294 | 占位 | `validateChannel()` 接受 ChannelAllocates 但无实际边关系校验 |
| channel 语义校验是否验证 party_edges | `fund/service.go` L286-298 | **缺口** | 同步骤 2，语义校验为占位 |

**缺口说明：** 数据模型层面 EdgeAllocates 完备——边类型已定义、默认 fund 权限已开通。但分配运行时的实际权限校验与其他 channel 一样为占位实现。演示脚本中的"给个人注入经费"操作在代码层面可执行（不校验边关系的 Allocate 调用），但生产就绪需要补齐 party_edges 交叉验证。

---

### 步骤 9：清算回流（Liquidate）

**审查文件：**
- `fund/lifecycle.go`（L207-L438）
- `fund/model.go`（L95-L104, L155-L162, L266-L284）

**审查结论：可执行**

| 审查点 | 代码位置 | 状态 | 说明 |
|--------|----------|------|------|
| 状态数量 | `fund/lifecycle.go` L189-L205 | 通过 | 5 阶段：blocking → draining → refunding → closing → closed（注释完整标注 PRD S8.4） |
| 状态迁移映射 | `fund/lifecycle.go` L422-437 | 通过 | `advanceLiquidationStage()` 定义完整迁移表 |
| 余额转入目标账户 | `fund/lifecycle.go` L300-L365 | 通过 | refunding 阶段：`remainingBalance = acct.AvailableBalance` → 转入 `targetAcct.AvailableBalance.Add(remainingBalance)`（L320-330） |
| 双边记账 | `fund/lifecycle.go` L332-357 | 通过 | src 方向 AllocateOut + dst 方向 AllocateIn，各一条 Ledger |
| 目标账户校验 | `fund/lifecycle.go` L237-247, L307-316 | 通过 | 启动时和目标转账时均校验目标账户存在且为 active |
| 防自清算 | `fund/lifecycle.go` L233-235 | 通过 | `TargetAccountID == AccountID` 时报 SELF_TRANSFER |

---

### 步骤 10：组织合并（EdgeMergedInto）

**审查文件：**
- `party/model.go`（L56-L58, L78-L87）

**审查结论：部分可执行（边类型存在，无独立合并服务）**

| 审查点 | 代码位置 | 状态 | 说明 |
|--------|----------|------|------|
| EdgeMergedInto 边类型是否存在 | `party/model.go` L56-L58 | 通过 | `EdgeMergedInto = "merged_into"`，注释"Fund transfer follows liquidation flow, not normal allocation" |
| 是否在 validEdgeTypes 中 | `party/model.go` L78-L87 | 通过 | 7 种合法边类型之一 |
| 是否有独立合并服务 | 无 | **缺口** | 无 MergeParties() 或类似的合并操作。合并依赖于手动执行清算流：对源 Party 的 Account 调用 Liquidate() → 余额转入目标账户 |
| 合并后余额是否正确转移 | `fund/lifecycle.go` L318-L330 | 间接 | 通过清算流转账实现，但不处理非余额资产（如活跃 freeze）的迁移 |

**缺口说明：** EdgeMergedInto 是数据模型的正确预留，但没有配套的合并业务服务。当前演示脚本可通过"清算 src account → 余额转入 dst account"模拟合并效果。生产就绪需要：1) MergeParties() 服务（合并 party 记录、迁移成员、标记旧 party 为 liquidated）；2) 活跃 freeze 的处理策略。

---

### 步骤 11：幂等划拨

**审查文件：**
- `idempotency/claim.go`（L73-L136）
- `idempotency/model.go`（L12-L26）
- `fund/service.go`（L81-L107, L245-L249）

**审查结论：可执行**

| 审查点 | 代码位置 | 状态 | 说明 |
|--------|----------|------|------|
| 重复提交是否仅入账一次 | `fund/service.go` L91-L103 | 通过 | 已 claim 的 key → Retrieve 出已存储结果 → 直接返回原始 AllocateResult |
| 去重策略 | `idempotency/claim.go` L73-76 | 通过 | INSERT-first：UNIQUE 约束保证原子性，无 TOCTOU 竞争 |
| 同 key 不同 body 冲突检测 | `idempotency/claim.go` L127-135 | 通过 | 不同 RequestHash → ErrIdempotencyConflict |
| 同 key 同 body 重放处理 | `idempotency/claim.go` L140-167 | 通过 | handleReplay() 按 StatusStarted/Succeeded/Failed 三级分类处理 |
| 结果存储在同一事务内 | `fund/service.go` L246-249 | 通过 | `StoreIdempotency(tx, ctx, key, result)` 在 Allocate 事务内执行 |

---

## 缺陷汇总

### 阻塞性缺陷（演示脚本无法走通）

| 编号 | 步骤 | 缺陷 | 影响 |
|------|------|------|------|
| B-1 | 步骤 3 | POST /gov/keys 返回占位 JSON，无 Key 创建逻辑 | 无法创建 API Key，后续"人 Key 绑项目账本"完全阻塞 |
| B-2 | 步骤 3 | 无 Key ↔ Account 绑定模型 | Key 无法关联到账本，调用时无法确定扣费账户 |
| B-3 | 步骤 2/8 | channel 语义校验为占位（所有通道放行） | parent/sponsors/allocates 权限控制不生效，虽然"可执行"但不符合安全设计意图 |

### 功能缺口（可走通但行为不完整）

| 编号 | 步骤 | 缺陷 | 影响 |
|------|------|------|------|
| G-1 | 步骤 1 | 无独立 markup/margin 字段 | 加价只能通过手动设置 sell tier 实现，无法表达"在 cost 基础上 +X%" |
| G-2 | 步骤 4 | 管线步骤 [6] 价格过滤(δ) 无独立执行 | δ 过滤逻辑由 Router "内部处理"，无法独立演示 |
| G-3 | 步骤 4 | 管线步骤 [11] 流式续期 无集成 | `StreamRenewal` 字段注入但未在 Execute 中调用 |
| G-4 | 步骤 5 | AuditEvent 无专用 cost/sell 金额字段 | 管理面审计不直接体现财务金额，需从快照 JSON 解析 |
| G-5 | 步骤 10 | 无独立 MergeParties() 服务 | 组织合并只能通过"清算 + 转账"模拟 |
| G-6 | 步骤 3 | /gov/accounts、/gov/allocations 等 handler 均为占位 | 账本查询、划拨记录查询等辅助操作为占位实现 |

---

## 演示可行性评级

| 步骤 | 演示可行性 | 备注 |
|------|-----------|------|
| 1. 配置预算与加价 | 可演示 | 通过 pricing API 写入双轨价目 + account 设置 budget_limit |
| 2. 划拨 | 可演示 | channel 校验为占位，但不阻塞演示（直接传入 channel=parent 即可） |
| 3. 创建人 Key | **不可演示** | handler 为占位，需先实现 |
| 4. 调用 | 可演示（需注入函数） | 管线骨架完整，需提供具体函数实现（Auth/Pricing/Freeze/Settle 等） |
| 5. 核对 sell/cost | 可演示 | Ledger 表双轨字段完备 |
| 6. 双层预算帽 | 可演示 | 两种错误码独立，检查逻辑完备 |
| 7. 余额不足 | 可演示 | 错误码独立 |
| 8. 个人经费注入 | 可演示 | channel=allocates 放行 |
| 9. 清算回流 | 可演示 | 5 阶段状态机完整 |
| 10. 组织合并 | 可演示（模拟） | 通过清算流变通实现 |
| 11. 幂等划拨 | 可演示 | INSERT-first 去重完备 |

**总体结论：11 步中 8 步可直接演示（含 2 步需变通），1 步阻塞（步骤 3），2 步需补齐集成后方可演示（步骤 4 的 [6]/[11]）。最小可行演示需要先实现步骤 3 的 Key 创建逻辑。**
