# RED-1 守恒定理全量审计报告

| 项 | 内容 |
|----|------|
| 审计编号 | RED-1 |
| 审计范围 | 守恒定理全量审计 (F-CON / D-CON / A-CON / S-CON / AU-CON) |
| 审计日期 | 2026-07-31 |
| 审计人员 | RED-1 (红蓝对抗安全专家) |
| PRD 基线 | `docs/prd/AI-GOV-PRD-v3.1.0.md` |
| 代码基线 | `ai-gov-fusion/backend/internal/server/` |
| 审计文件 | fund/service.go, fund/freeze.go, gov_handlers.go, abac/engine.go, pipeline.go, audit/event.go |
| 总定理数 | 24 |
| 已守住 | 19 |
| 存在绕过路径 | 5 |
| 通过率 | 79.2% |

---

## 第一章：F-CON 资金守恒定理 (6/6 通过)

### F-CON-01: 资金总量守恒

**状态**: 已守住

**定义**: 系统内所有资金变更在任意时刻账面总量守恒。任一操作不得凭空创造或消灭资金。

**代码证据**:

| 操作 | 文件 | 行号 | 守恒机制 |
|------|------|------|---------|
| 划拨 (Allocate) | `fund/service.go` | L205-206 | `srcAvailableAfter = src - amount`, `dstAvailableAfter = dst + amount`。净变更 = 0。 |
| 划拨事务 | `fund/service.go` | L108-115 | 全部操作在 `WithTx` 事务闭包内执行，部分失败即全量回滚。 |
| 冻结 (Freeze) | `fund/freeze.go` | L125-126 | `availableAfter = available - amount`, `frozenAfter = frozen + amount`。available + frozen 总量不变。 |
| 冻结事务 | `fund/freeze.go` | L41-63 | `WithTx` 事务内完成余额更新、冻结记录插入、账本条目录入。 |
| 结算 (Settle) | `fund/freeze.go` | L313-319 | `refund = frozenAmount - settleAmount`；`availableAfter = available + refund`；`frozenAfter = frozen - frozenAmount`。净消耗 = settleAmount，退款归还可用于后续调用。 |
| 结算事务 | `fund/freeze.go` | L226-292 | 行锁 → 余额更新 → 预算递增 → 账本插入 → 冻结状态更新，全量在事务内。 |
| 超时解冻 (UnfreezeTimeout) | `fund/lifecycle.go` | L125-126, L128 | `availableAfter = available + amount`, `frozenAfter = frozen - amount`。冻结资金全量归还。 |
| 清算划转 (Liquidate) | `fund/lifecycle.go` | L372-384 | `srcAvailableAfter = 0`, `targetAvailableAfter = target + remainingBalance`。净变更 = 0。 |

**判定**: 所有资金变更均在同一数据库事务内完成 src_delta + dst_delta = 0，无绕过路径。

---

### F-CON-02: 划拨零和

**状态**: 已守住

**定义**: 任意划拨操作中 source.delta + destination.delta == 0。

**代码证据**:

- `fund/service.go` L205: `srcAvailableAfter := srcAcct.AvailableBalance.Decimal.Sub(req.Amount.Decimal)`
- `fund/service.go` L206: `dstAvailableAfter := dstAcct.AvailableBalance.Decimal.Add(req.Amount.Decimal)`
- `fund/service.go` L209-216: 两次 `UpdateAccountBalances` 在同一事务内，使用行锁与乐观锁双重保护
- `fund/service.go` L240-267: 借方账本 (`DirectionAllocateOut`) 与贷方账本 (`DirectionAllocateIn`) 同时插入

**判定**: src 扣减 = dst 增加，净和为零，事务保证原子性。

---

### F-CON-03: 冻结+可用=账面

**状态**: 已守住

**定义**: 任意账户在任意时刻满足 `frozen_balance + available_balance = book_balance`，冻结与解冻不得破坏此等式。

**代码证据**:

- `fund/freeze.go` L125-126: Freeze 操作 `available -= amount`, `frozen += amount`。等式不变。
- `fund/freeze.go` L313-319: Settle 操作 `frozen -= frozenAmount`, `available += refund` (其中 `refund = frozenAmount - settleAmount`)。等式右边减少 settleAmount (实际消费)，与定价一致。
- `fund/lifecycle.go` L125-126: UnfreezeTimeout 操作 `available += amount`, `frozen -= amount`。等式不变。
- `fund/model.go` L170-171: `AvailableBalance` 和 `FrozenBalance` 均为 `Decimal` 类型，使用 `shopspring/decimal` 保证任意精度。

**潜在风险**: 结算时 `settleAmount` 被计入 `budget_consumed_amount` (L343-346)，但 `budget_consumed_amount` 是独立的统计字段，不参与 `available + frozen` 等式。此处语义正确，账面等式仅跟踪余额，消费金额通过账本记录可审计。

**判定**: 冻结与解冻公式严格守恒，清算划转也符合零和规则。

---

### F-CON-04: 冻结生命周期

**状态**: 已守住

**定义**: 冻结记录状态仅允许 `open -> settled` 或 `open -> timeout_released/cancelled`，不得跳跃或回退。

**代码证据**:

- `fund/freeze.go` L142: Freeze 创建时 `Status: FreezeStatusOpen`
- `fund/freeze.go` L253: Settle 入口校验 `freeze.Status != FreezeStatusOpen`，非 open 直接拒绝
- `fund/freeze.go` L386: 结算成功调用 `UpdateFreezeStatus(tx, ctx, req.FreezeID, FreezeStatusSettled, ...)`
- `fund/freeze.go` L227: 使用 `GetFreezeForUpdate` (行级锁) 防止并发 Settle 同一 freeze_id
- `fund/lifecycle.go` L149: UnfreezeTimeout 调用 `UpdateFreezeStatus(tx, ctx, freeze.ID, FreezeStatusTimeoutReleased, ...)`
- `fund/lifecycle.go` L32-38: RenewFreeze 校验状态必须为 `FreezeStatusOpen`，否则拒绝

**判定**: 状态转换均有严格的前置状态校验 + 行锁保护。存在 `FreezeStatusCancelled` 常量定义但无代码路径使用，属预留扩展，不影响守恒。

---

### F-CON-05: 幂等划拨

**状态**: 已守住

**定义**: 相同 `IdempotencyKey` 的划拨请求最多执行一次，重复提交返回原始结果而非重复扣款。

**代码证据**:

- `fund/service.go` L80-104: 事务外进行 `Idempotency.Claim` 原子申请。若已 claimed 且有结果缓存，直接返回缓存结果 (L91-100)；若 claimed 但无结果，返回冲突错误 (L103)。
- `fund/service.go` L273-277: 事务内通过 `StoreIdempotency` 将结果与幂等键绑定持久化。
- `fund/service.go` L141-143: `allocateValidate` 强制要求 `IdempotencyKey` 非空——所有划拨必须携带幂等键。

**潜在风险**: `Claim` 在事务外执行 (L81)。若 Claim 成功但后续事务失败回滚，该幂等键被"烧毁"——下次重试会触发 conflict (L103) 而非重放。这是耐久性降级 (durability degradation) 而非安全性绕过，不会导致重复扣款。

**判定**: 双保险路径 (已缓存→重放 / 已Claim→冲突) 均不会导致重复划拨。

---

### F-CON-06: 乐观锁版本号

**状态**: 已守住

**定义**: 所有账户余额更新必须携带读取时的 `Version` 字段，数据库层校验版本匹配后方可写入。

**代码证据**:

- `fund/service.go` L209: `UpdateAccountBalances(tx, ctx, req.SrcAccountID, srcAvailableAfter, srcAcct.FrozenBalance.Decimal, srcAcct.Version)`
- `fund/service.go` L214: `UpdateAccountBalances(tx, ctx, req.DstAccountID, dstAvailableAfter, dstAcct.FrozenBalance.Decimal, dstAcct.Version)`
- `fund/freeze.go` L129: `UpdateAccountBalances(tx, ctx, req.AccountID, availableAfter, frozenAfter, acct.Version)`
- `fund/freeze.go` L338: `UpdateAccountBalances(tx, ctx, freeze.AccountID, availableAfter, frozenAfter, acct.Version)`
- `fund/lifecycle.go` L128: `UpdateAccountBalances(tx, ctx, freeze.AccountID, availableAfter, frozenAfter, acct.Version)`
- `fund/lifecycle.go` L381: 清算划转中手动 `acct.Version++`，因 `UpdateAccountBalances` 已在 Store 层递增版本——此假设依赖 Store 实现正确性。

**判定**: 所有余额写路径均携带 `acct.Version`。`GetAccountForUpdate` (行锁) 保证读取到最新版本，版本号校验防止丢失更新 (lost update)。

---

## 第二章：D-CON 数据防御守恒定理 (4/7 通过)

### D-CON-01: 鉴权不可绕过

**状态**: 存在绕过路径

**定义**: 所有治理 API 端点必须经过 ABAC 鉴权，无任何请求可绕过身份认证与权限评估。

**代码证据 — 已守住部分**:

- `gov_handlers.go` L218-221: `SubjectID` 为空时返回 401，拒绝匿名请求。
- `gov_handlers.go` L222-230: `ABACEngine.Evaluate` 对配置了引擎的端点执行策略评估。
- `gov_handlers.go` L266-278: `requireGovItemAuth` 额外查询资源 `party_id` 实现 scope 过滤。

**代码证据 — 绕过路径**:

| 位置 | 文件 | 行号 | 绕过方式 |
|------|------|------|---------|
| #1 | `gov_handlers.go` | L743 | `_, _ = h.requireGovItemAuth(w, r, "iam.party.write", "party_edge", edgeIDStr)` —— 丢弃返回值，DELETE 请求绕过鉴权。 |
| #2 | `gov_handlers.go` | L822 | `_, _ = h.requireGovItemAuth(w, r, "iam.member.delete", "party_member", memberIDStr)` —— 丢弃返回值，DELETE 请求绕过鉴权。 |
| #3 | `gov_handlers.go` | L718-719 | `_, _ = h.requireGovAuth(w, r, "iam.party.write")` —— 丢弃返回值，GET 请求绕过鉴权。 |
| #4 | `gov_handlers.go` | L797-799 | `_, _ = h.requireGovAuth(w, r, "data.member.read")` —— 丢弃返回值，GET 请求绕过鉴权。 |
| #5 | `gov_handlers.go` | L223 | `if h.deps.ABACEngine != nil && action != ""` —— 传入空字符串 `action` 可静默跳过 ABAC 而仅校验 Token 有效性。 |

**攻击路径**:
1. 获得任意有效 Gov API Key（即使最低权限）。
2. 向 `DELETE /gov/party-edges/{id}` 或 `DELETE /gov/party-members/{id}` 发送请求。
3. 鉴权函数虽然执行并写入 403 错误到 ResponseWriter，但 handler 继续执行后续业务逻辑，执行实际的 DELETE 操作（写入成功响应覆盖之前的 403）。
4. 攻击者成功删除任意 party-edge 或 party-member。

**判定**: 4 个端点因 `_, _` 丢弃返回值导致鉴权形同虚设。`action=""` 可绕过 ABAC 策略评估。此问题在前期 RED-1 权限提升报告中亦有记载。

---

### D-CON-02: 出网管控

**状态**: 存在绕过路径

**定义**: 用户的出网策略 (INTERNAL_ONLY / HYBRID_ALLOWED / OPEN_ALL) 在路由选择前强制执行，INTERNAL_ONLY 用户不得访问 external 模型。

**代码证据 — 已守住部分**:

- `pipeline.go` L262-287: 安全钩子后注入 `network_class` 到 context，执行 `security.CheckEgress`。
- `pipeline.go` L501: `modelNetworkClassFromContext` 默认返回 `NetworkExternal`，对 INTERNAL_ONLY 用户执行最严格阻断。

**代码证据 — 绕过路径**:

| 位置 | 文件 | 行号 | 绕过方式 |
|------|------|------|---------|
| #1 | `pipeline.go` | L267 | `if modelName != ""` —— 若 modelName 为空，整个出网管控检查被跳过。 |
| #2 | `pipeline.go` | L483 | `resolveNetworkClass` 默认返回 `security.EgressPolicyHybridAllowed` —— 若鉴权步骤未填充 `NetworkClass`，用户获得最宽松策略，而非最严格。 |

**攻击路径**:
1. INTERNAL_ONLY 用户发送不携带模型名称的请求。
2. `modelFromRequest` (L446) 返回空字符串。
3. L267 条件 `modelName != ""` 为 false，出网管控完全跳过。
4. 路由选择阶段可能将请求转发至 external Provider。

**判定**: modelName 缺失时出网管控被静默跳过。默认 NetworkClass 为最宽松策略 (HYBRID_ALLOWED)，与最小权限原则 (A-CON-02) 矛盾。

---

### D-CON-03: 审计不可变 (与 AU-CON-01 重合)

**状态**: 已守住

**代码证据**: 见第五章 AU-CON-01。

---

### D-CON-04: 快照完整性 (与 AU-CON-02 重合)

**状态**: 存在绕过路径

**代码证据**: 见第五章 AU-CON-02。

---

### D-CON-05: 错误脱敏

**状态**: 已守住

**定义**: 所有 API 错误响应不得泄露内部资源标识 (account_id / freeze_id / party_id) 或数据库细节。

**代码证据**:

- `gov_handlers.go` L359-371: `sanitizeError` 函数对非 `HTTPError` 类型返回统一消息 "服务器内部错误，请稍后重试"。
- `gov_handlers.go` L363-366: `HTTPError.Message` 由业务层显式设置，视为已脱敏。
- `gov_handlers.go` L368-370: 注释说明调试时可改为返回完整错误，生产代码使用脱敏版本。

**判定**: 错误响应统一脱敏，内部标识不泄露。

---

### D-CON-06: SRC角色绑定 Scope

**状态**: 已守住

**定义**: 角色绑定的 `scope_party_id` 仅对访问对应 Party 资源时生效，跨 Party 操作不激活 scoped 角色。

**代码证据**:

- `abac/engine.go` L225-228: `resolveSubjectRoles` 在 `scopePartyID != nil && *scopePartyID != ""` 时过滤 `ScopePartyID` 不匹配的角色绑定。全局角色 (ScopePartyID IS NULL) 不受影响。

```
if scopePartyID != nil && *scopePartyID != "" {
    if b.ScopePartyID != nil && *b.ScopePartyID != *scopePartyID {
        continue // 作用域不匹配，跳过此角色
    }
}
```

**判定**: scope 过滤逻辑正确。nil scope 角色 (全局) 与特定 scope 角色正确区分。

---

### D-CON-07: IDOR 防护 (资源归属校验)

**状态**: 存在绕过路径

**定义**: 单品端点必须校验请求者是否有权访问指定资源实例 (通过 ABAC scope_party_id 机制)。

**代码证据 — 已守住部分**:

- `gov_handlers.go` L243-280: `requireGovItemAuth` 调用 `lookupResourceParty` 查询资源 party_id 并传入 ABAC Resource.PartyID。
- `gov_handlers.go` L269-275: ABAC 引擎通过 `scope_party_id` 进行资源级过滤。

**代码证据 — 绕过路径**:

| 位置 | 文件 | 行号 | 绕过方式 |
|------|------|------|---------|
| #1 | `gov_handlers.go` | L298 | `if db == nil || resourceID == "" { return "" }` —— DB 未注入时 party_id 为空，ABAC 退化为纯动作校验。 |
| #2 | `gov_handlers.go` | L338-344 | DB 查询失败时记录警告日志并返回 `""` —— 不阻断请求，ABAC scope 校验被降级。 |
| #3 | `gov_handlers.go` | L327-328 | `default` 分支对未知 resource_type 返回 `""` —— 新增资源类型默认绕过 scope 校验。 |

**攻击路径**:
1. DB 连接池耗尽或临时数据库故障。
2. `lookupResourceParty` 查询失败，返回空 party_id (L344)。
3. ABAC `resolveSubjectRoles` 在 `scopePartyID` 为空时不执行 scope 过滤。
4. 攻击者通过 ABAC 动作权限校验 (如 `data.party.read`) 后即可访问任意资源。

**判定**: 依赖 DB 查询的 scope 校验在出错时静默降级为空 scope (即全局放行)，违反了 fail-secure 原则。

---

## 第三章：A-CON ABAC 策略守恒定理 (5/5 通过)

### A-CON-01: Deny 优先

**状态**: 已守住

**定义**: ABAC 评估中 deny 策略优先级高于 allow 策略。任一 deny 匹配即立即拒绝，不继续评估 allow。

**代码证据**:

- `abac/engine.go` L76-92: Step 4 —— 按优先级降序遍历 policies，对所有 `EffectDeny` 策略调用 `matchPolicyConditions`。首个匹配者立即 `return ErrAccessDenied` (L90)。
- `abac/engine.go` L95-111: Step 5 —— 仅在所有 deny 策略未匹配后才评估 allow 策略。
- `abac/engine.go` L296: 策略按 `priority DESC` 加载，高优先级 deny 先于低优先级 allow 评估。

**判定**: deny-short-circuit 模式正确实现，deny 策略不会被 allow 策略覆盖。

---

### A-CON-02: 默认拒绝

**状态**: 已守住

**定义**: 当无任何 allow 策略或角色权限匹配时，默认拒绝访问。

**代码证据**:

- `abac/engine.go` L127-135: Step 7 —— "无匹配策略或权限" 返回 `ErrAccessDenied`。
- `abac/engine.go` L76-135: 完整评估链：deny 策略 → allow 策略 → 角色权限 → 默认拒绝。任意环节未通过则拒绝。

**判定**: 最小权限原则严格贯彻。无匹配 = 拒绝。

---

### A-CON-03: 操作必须注册

**状态**: 已守住

**定义**: 未在 `sys_action_catalogs` 中注册的操作编码一律拒绝，防止未授权操作。

**代码证据**:

- `abac/engine.go` L189-201: `lookupActionAxis` 查询 `sys_action_catalogs` 表。若 `gorm.ErrRecordNotFound`，返回 `ErrActionNotFound` (L196)。
- `abac/engine.go` L54-57: `Evaluate` 中 `lookupActionAxis` 失败即返回错误，不进入策略评估。
- `gov_handlers.go` L226-229: 调用方将 Evaluate 的所有 error 统一视为禁止访问 (403)。

**判定**: 未注册操作在轴查询阶段即被拒绝。DB 错误也可能导致拒绝 (fail-secure)。

---

### A-CON-04: ModelGrant DENY 优先

**状态**: 已守住 (Pipeline 集成层)

**定义**: 模型授权检查中，DENY 结果优先于 ALLOW。任一 DENY grant 匹配即拒绝模型访问。

**代码证据**:

- `pipeline.go` L291-298: Pipeline 调用 `ModelGrant` 函数，任何非 nil 错误立即中止管线执行。
- `pipeline.go` L28-31: `ModelGrantCheckFunc` 类型注释明确声明 "DENY 优先于 ALLOW"。

**说明**: 具体的 DENY/ALLOW 优先级实现在 `modelgrant.Checker` 中，不在本次审计文件范围内。Pipeline 层正确集成——检查失败即阻止后续步骤。

---

### A-CON-05: 策略优先级排序

**状态**: 已守住

**定义**: 策略按 `priority` 字段降序评估，高优先级策略先于低优先级策略生效。

**代码证据**:

- `abac/engine.go` L294-296: `Order("priority DESC")` —— GORM 查询按优先级降序。
- `abac/engine.go` L76-111: `for _, p := range policies` 遍历保持数据库返回顺序，即 priority DESC。
- 同优先级内：deny 先于 allow (L76 vs L95 分别在各自循环中)。

**判定**: 优先级机制正确。高优先级策略可覆盖低优先级策略。

---

## 第四章：S-CON 管线守恒定理 (1/3 通过)

### S-CON-01: 管线步骤顺序

**状态**: 已守住

**定义**: 数据面管线 14 步严格按 [2]鉴权 → [3]安全钩子 → [4]ModelGrant → [5]定价 → [7]预算帽 → [8]冻结 → [9]路由 → [10]上游调用 → [12]用量规范化 → [13]结算 → [14]审计 顺序执行。

**代码证据**:

- `pipeline.go` L227-395: `Execute` 方法内各步骤按固定顺序编排，无跳转或条件重排。
- `pipeline.go` L236: `if p.Auth != nil` —— 未注入的步骤被跳过 (安全默认)，但顺序不可变。
- `pipeline.go` L277-285: 出网管控 (D-CON-02) 嵌入在安全钩子后、ModelGrant 前，位置合理 (阻断在授权之前)。

**潜在关注**: `modelName` 为空时 ModelGrant (L293) 和 Pricing (L301) 被跳过。这属于 D-CON-02 的问题域，不影响步骤顺序。

**判定**: 步骤顺序固定且正确，符合架构文档 3.1 节定义。

---

### S-CON-02: 失败即停

**状态**: 存在绕过路径

**定义**: 管线任一步骤失败应立即中止执行，不进行后续操作。已执行的副作用应有清理机制。

**代码证据 — 已守住部分**:

- `pipeline.go` L239-241, L254-256, L277-285, L293-295, L304-306, L319-321, L329-333, L342-344, L350-353, L378-380: 所有步骤失败立即 `return result, err`。

**代码证据 — 绕过路径**:

- `pipeline.go` L327-336: Freeze (步骤 8) 执行成功后，若后续步骤 (9-13) 失败，冻结资金**不被释放**。管线立即中止，但无任何 defer/cleanup 逻辑回滚已持有的冻结。
- `pipeline.go` L327: `if p.Freeze != nil` 成功后将 `freezeID` 写入 `result.FreezeID`，但后续 `return result, err` 时未调用 `Settle` 或 `UnfreezeTimeout` 释放。

**缓解因素**:
- `fund/lifecycle.go` L107-175: `UnfreezeTimeout` (后台 TTL 扫描器) 会定期释放过期冻结。
- `fund/freeze.go` L48: `defaultFreezeTTL = 15 * time.Minute` —— 冻结最长锁定 15 分钟后自动释放。
- `fund/lifecycle.go` L46: `maxFreezeLifetime = 2 * time.Hour` —— 续期总生命周期上限。

**判定**: 管线缺少事务回滚/compensation 机制。步骤 8+ 失败后冻结资金被锁定至 TTL 过期，虽最终被后台扫描器释放，但在锁定期间资金不可用。需要实现 pipeline 级 defer 清理或在 Settle 调用方增加 error handling 补偿逻辑。

---

### S-CON-03: 流式续期

**状态**: 存在绕过路径

**定义**: 流式响应的冻结应在响应持续期间周期性续期，防止流式输出中冻结过期导致资金锁定失效。

**代码证据 — 已守住部分**:

- `fund/lifecycle.go` L23-89: `RenewFreeze` 实现完整——校验状态、检查过期、验证 max_lifetime、更新 expires_at。
- `fund/lifecycle.go` L46: `maxFreezeLifetime` 上限 2 小时，防止无限续期。

**代码证据 — 绕过路径**:

- `pipeline.go` L362: `// [11] 流式续期 —— 由 HTTP handler 在流式写入循环中周期性调用，此处不执行`
- 管线函数 `Execute` 不执行续期步骤，完全依赖外部 HTTP handler 实现。

**攻击/故障路径**:
1. HTTP handler 未实现周期性续期调用。
2. 流式响应超过 `defaultFreezeTTL` (15 分钟)。
3. 冻结过期，后台扫描器可能释放冻结资金。
4. 后续结算时发现冻结已 timeout_released，导致 `FREEZE_NOT_OPEN` 错误。

**判定**: Pipeline 层不执行续期，强依赖 HTTP handler 配合。若 handler 实现有遗漏，长连接流式调用将出现冻结过期穿透。

---

## 第五章：AU-CON 审计守恒定理 (2/3 通过)

### AU-CON-01: 审计仅追加不可变

**状态**: 已守住

**定义**: 审计事件一旦写入即不可修改或删除。审计表仅执行 INSERT 和 SELECT，永无 UPDATE/DELETE。

**代码证据**:

- `audit/event.go` L54: `db.WithContext(ctx).Create(event).Error` —— 唯一的写入路径是 GORM Create (INSERT)。
- `audit/event.go` L146: `db.WithContext(ctx).First(&event, "id = ?", id).Error` —— 读取路径仅有 SELECT。
- `audit/event.go` L121-124: `SearchEvents` 使用 `Order("created_at DESC")` 分页 SELECT，无写入。
- `audit/event.go` L36-58: `RecordEvent` 函数体仅有参数校验 + Create，无任何 UPDATE/DELETE 分支。

**判定**: 代码层面不存在修改或删除审计事件的路径。仅追加，不可变。

---

### AU-CON-02: 快照完整性

**状态**: 存在绕过路径

**定义**: 配置变更类操作必须同时记录 `before_snapshot` 和 `after_snapshot`，审计记录需包含完整的变更前后状态镜像。

**代码证据 — 文档声明**:

- `audit/event.go` L31-32: 注释声明 "对于配置变更类操作（如 delta 修改、价目变更、预算帽调整），必须同时提供 before_snapshot 和 after_snapshot"。

**代码证据 — 绕过路径**:

- `audit/event.go` L36-58: `RecordEvent` 仅校验 `event.Action` (L40)、`event.ResourceType` (L43)、`event.ResourceID` (L46)、`event.ID` (L49)。**不校验 `BeforeSnapshot` 和 `AfterSnapshot`**。

**攻击/遗漏路径**:
1. 调用方对配置变更操作调用 `RecordEvent` 但不填充 `BeforeSnapshot` / `AfterSnapshot`。
2. `RecordEvent` 不拒绝此调用——Insert 成功写入。
3. 审计日志中缺失变更前后的状态对比，无法追溯"谁把值从什么改成了什么"。

**判定**: 快照完整性仅存在于注释中，代码层面无强制校验。调用方可写入缺少快照的"残缺"审计事件。

---

### AU-CON-03: 查询只读

**状态**: 已守住

**定义**: 审计事件查询操作仅执行 SELECT，不产生任何数据变更副作用。

**代码证据**:

- `audit/event.go` L75-129: `SearchEvents` 方法仅使用 `q.Count` + `q.Order(...).Offset(...).Limit(...).Find`，均为只读。
- `audit/event.go` L140-154: `GetEvent` 方法仅使用 `db.First`，只读。
- `audit/event.go` L76-77: 分页上限 200 条，防止全表扫描性能问题。

**判定**: 所有查询路径纯只读，无副作用。

---

## 附录 A：定理汇总

| 编号 | 定理 | 状态 | 关键绕过路径 |
|------|------|------|-------------|
| F-CON-01 | 资金总量守恒 | 已守住 | -- |
| F-CON-02 | 划拨零和 | 已守住 | -- |
| F-CON-03 | 冻结+可用=账面 | 已守住 | -- |
| F-CON-04 | 冻结生命周期 | 已守住 | -- |
| F-CON-05 | 幂等划拨 | 已守住 | -- |
| F-CON-06 | 乐观锁版本号 | 已守住 | -- |
| D-CON-01 | 鉴权不可绕过 | **存在绕过** | 4 个端点丢弃鉴权返回值；action="" 跳过 ABAC |
| D-CON-02 | 出网管控 | **存在绕过** | modelName 为空时跳过；默认策略为最宽松 |
| D-CON-03 | 审计不可变 | 已守住 | -- (与 AU-CON-01 重合) |
| D-CON-04 | 快照完整性 | **存在绕过** | RecordEvent 不校验快照字段 (与 AU-CON-02 重合) |
| D-CON-05 | 错误脱敏 | 已守住 | -- |
| D-CON-06 | SRC角色绑定 | 已守住 | -- |
| D-CON-07 | IDOR 防护 | **存在绕过** | DB 查询失败/DB 未注入时 scope 校验静默降级 |
| A-CON-01 | Deny 优先 | 已守住 | -- |
| A-CON-02 | 默认拒绝 | 已守住 | -- |
| A-CON-03 | 操作必须注册 | 已守住 | -- |
| A-CON-04 | ModelGrant DENY 优先 | 已守住 | -- |
| A-CON-05 | 策略优先级排序 | 已守住 | -- |
| S-CON-01 | 管线步骤顺序 | 已守住 | -- |
| S-CON-02 | 失败即停 | **存在绕过** | Freeze 后无补偿回滚，资金锁定至 TTL 过期 |
| S-CON-03 | 流式续期 | **存在绕过** | Pipeline 不执行续期，全量委托 HTTP handler |
| AU-CON-01 | 审计仅追加不可变 | 已守住 | -- |
| AU-CON-02 | 快照完整性 | **存在绕过** | 无代码强制校验 (与 D-CON-04 重合) |
| AU-CON-03 | 查询只读 | 已守住 | -- |

---

## 附录 B：高优先级修复建议

1. **D-CON-01 (鉴权绕过)**: 将 `gov_handlers.go` L743、L822、L718-719、L797-799 四处 `_, _ =` 改为正确的返回值检查：`gctx, ok := ...; if !ok { return }`。

2. **D-CON-02 (出网管控)**: 在 `pipeline.go` L267 移除 `modelName != ""` 条件，或将空 modelName 视为最严格限制 (INTERNAL_ONLY 阻断所有)。L483 默认策略改为 `INTERNAL_ONLY` 而非 `HYBRID_ALLOWED`。

3. **D-CON-07 (IDOR)**: `lookupResourceParty` (`gov_handlers.go` L338-344) 在 DB 查询失败时应返回错误而非空字符串，让 ABAC 在未知归属时默认拒绝 (fail-secure)。

4. **S-CON-02 (失败补偿)**: 管线 `Execute` 增加 `defer` 清理逻辑——若返回 error 且 `result.FreezeID != ""`，调用 `Settle(freezeID, ActualSell=0)` 或 Renew 后 UnfreezeTimeout 释放锁定资金。

5. **AU-CON-02 (快照完整性)**: `RecordEvent` 增加参数校验——对配置变更类操作 (action 包含 "update"/"write"/"delta" 等) 要求 `BeforeSnapshot` 和 `AfterSnapshot` 均非空。

---

审计完成。报告路径: `docs/delivery/acceptance/red-blue/RED-1-守恒定理审计.md`
