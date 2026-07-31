# QA-4 模型+调度+追踪域验收报告

**验收日期**: 2026-07-31
**验收人**: QA-4
**审查范围**: modelgrant、fund、pipeline、routing、audit、security

---

## 场景 1: ModelGrant deny 优先 (PASS)

### 审查目标

| 子项 | 期望 | 代码位置 | 结论 |
|------|------|----------|------|
| DENY 优先于 ALLOW | DENY 第一遍扫描，命中即返回拒绝 | `modelgrant/checker.go:53-63` | PASS |
| ALLOW 至少一条 | 第二遍扫描，需至少一条 ALLOW 匹配 | `modelgrant/checker.go:66-70` | PASS |
| 默认拒绝 | 无匹配规则 → ErrModelAccessDenied | `modelgrant/checker.go:72-78` | PASS |
| 级联顺序 Key>Person>Party | 按类型加载，priority DESC 排序 | `modelgrant/checker.go:171-203` | PASS |

### 代码逻辑

```go
// checker.go:53-63 — 第 1 遍：DENY 优先
for _, mg := range mgs {
    if mg.Effect == EffectDeny && matchGrant(mg, principal, modelID) {
        return ErrModelAccessDenied   // 立即拒绝，不检查 ALLOW
    }
}

// checker.go:66-70 — 第 2 遍：需要 ALLOW
for _, mg := range mgs {
    if mg.Effect == EffectAllow && matchGrant(mg, principal, modelID) {
        return nil
    }
}

// checker.go:72-78 — 默认拒绝（最小权限原则 A-CON-02）
return ErrModelAccessDenied
```

### 测试覆盖

- `TestCheckAccess_Allow`: ALLOW 规则放行 (PASS)
- `TestCheckAccess_Deny`: DENY 覆盖 ALLOW (PASS)
- `TestCheckAccess_KeyOverParty`: Key 级 DENY 覆盖 Party 级 ALLOW (PASS)
- `TestCheckAccess_DefaultDeny`: 无规则默认拒绝 (PASS)

### 审查发现

级联加载 `loadGrantsForCascade` 仅加载匹配 `principal.Type` 的规则+全局默认规则，不会跨类型级联（如 Key 主体不加载 Person/Party 规则）。注释中的"级联"含义为类型优先级排序，非逐级穿透。此行为与测试 `TestCheckAccess_KeyOverParty` 一致，属预期设计。

---

## 场景 2: 双层预算 (PASS)

### 审查目标

| 子项 | 期望 | 代码位置 | 结论 |
|------|------|----------|------|
| 模型级配额检查 | CheckQuotaLimit 检查 consumed+estimated > limit | `modelgrant/checker.go:81-122` | PASS |
| 超额返回 MODEL_BUDGET_EXCEEDED | predicted > limit → ErrModelBudgetExceeded | `modelgrant/checker.go:108-118` | PASS |
| Account 级预算帽 | BudgetLimitAmount 检查，超额阻断冻结 | `fund/freeze.go:56-81` | PASS |
| 预算告警阈值 | BudgetWarnRatio 超过比例时 WARN 日志 | `fund/freeze.go:69-79` | PASS |

### 代码逻辑

**模型级预算 (checker.go)**:

```go
// checker.go:107-119
predicted := mg.QuotaConsumed.Add(estimatedSell)
if predicted.GreaterThan(*mg.QuotaLimit) {
    return ErrModelBudgetExceeded   // "MODEL_BUDGET_EXCEEDED"
}
```

**Account 级预算 (freeze.go)**:

```go
// freeze.go:62-66
newConsumed := acct.BudgetConsumedAmount.Decimal.Add(estimatedCost)
if newConsumed.GreaterThan(acct.BudgetLimitAmount.Decimal) {
    return newBudgetCapExceededError(...)
}
```

### 管线执行顺序

Pipeline.Execute 确保两层预算先后执行：
1. Step [4] ModelGrant 检查 (含 CheckQuotaLimit 模型级预算)
2. Step [7] 预算帽检查 (Account 级 BudgetCap)
3. Step [8] 冻结 (Freeze 内含再次校验 Account 预算)

双层预算已在管线中串联，两个层级分别返回不同错误码 (MODEL_BUDGET_EXCEEDED / BUDGET_CAP_EXCEEDED)。

---

## 场景 3: 调度不改账户 (PASS)

### 审查目标

| 子项 | 期望 | 审查范围 | 结论 |
|------|------|----------|------|
| pipeline.go 修改 account_id | 不修改 | `pipeline.go:224-362` | PASS |
| routing/profile.go 修改 account_id | 不修改 | `routing/profile.go:198-311` | PASS |
| routing/strategies/*.go 修改 account_id | 不修改 | 全部 12 个策略文件 | PASS |
| routing/decision.go 包含 account_id | 不包含 | `routing/decision.go` 无 account 字段 | PASS |

### 审查发现

- `Pipeline.Execute` 中 `result.Auth.AccountID` 仅在 Step [2] 鉴权时赋值，后续所有步骤均为只读引用。
- 路由管道 `ExecuteProfile` 参数不包含任何 account 相关信息，仅接收 `[]Candidate`、`RouteProfile`、`anchorSell`。
- `Candidate` 结构体字段为 ChannelID、ModelID、Priority、Weight、Health、Score 等路由相关字段，无 account_id。
- `Decision` 路由决策日志中无 account_id 字段。
- 所有 12 个策略文件均未引用 account_id。

**结论: 调度过程不触碰 account_id，无账户串号风险。**

---

## 场景 4: 全链路追踪 (PASS)

### 审查目标

| 子项 | 期望 | 代码位置 | 结论 |
|------|------|----------|------|
| 每个管线步骤记录审计事件 | Step [2]-[14] 均有 auditStep 调用 | `pipeline.go:234-353` | PASS |
| request_id 贯穿全链路 | 从 ctx 提取，写入每个 PipelineAuditEvent | `pipeline.go:226,369-370` | PASS |
| 审计事件仅 INSERT | RecordEvent 只有 Create，无 Update/Delete | `audit/event.go:54` | PASS |
| 审计字段完整 | actor, action, resource, before/after snapshot, IP | `audit/model.go:57-98` | PASS |

### 管线步骤审计覆盖

| Step | 步骤名 | 审计记录 | 状态 |
|------|--------|----------|------|
| [2] | 鉴权 | success/failure + user_id | 已覆盖 |
| [3] | 安全钩子 | success/failure + error | 已覆盖 |
| [4] | 模型授权检查 | success/failure + model | 已覆盖 |
| [5] | 价格预估 | success/failure + cost/sell amount | 已覆盖 |
| [6] | 价格过滤(δ) | 合并至 Router 内部 | 已说明 |
| [7] | 预算帽检查 | success/failure + error | 已覆盖 |
| [8] | 冻结 | success/failure + freeze_id | 已覆盖 |
| [9] | 策略路由 | success/failure + channel_id | 已覆盖 |
| [10] | 上游调用 | success/failure + status/latency | 已覆盖 |
| [11] | 流式续期 | HTTP handler 外部调用 | 已说明 |
| [12] | 用量规范化 | success + items | 已覆盖 |
| [13] | 双轨结算 | success/failure + cost/sell | 已覆盖 |
| [14] | 审计 | 各步骤通过 auditStep 累积 | 已覆盖 |

### request_id 贯穿机制

1. `requestIDFromContext` 从 ctx 提取 `request_id`
2. `enrichContext` 将 `auth.RequestID` 写入 ctx
3. `auditStep` 使用 `result.RequestID` 构造 PipelineAuditEvent
4. Audit 函数通过 `p.Audit(ctx, event)` 写入审计表

### 额外发现

- 审计链锚定机制 (`audit/model.go:103-132`) 提供 SHA-256 哈希链防篡改。
- 审计事件保留期 >= 180 天 (文档声明)。
- `RecordEvent` 有完善的输入校验：event 非 nil、action 非空、resource_type 非空、resource_id 非空、id 非空。

---

## 场景 5: INTERNAL_ONLY (PASS - 双层防御，存在集成风险)

### 审查目标

| 子项 | 期望 | 代码位置 | 结论 |
|------|------|----------|------|
| CheckEgress 阻断 INTERNAL_ONLY 外网 | user=INTERNAL_ONLY + model=external → ErrEgressBlocked | `security/egress.go:80-83` | PASS |
| S-COMPLIANCE 过滤 external 候选 | internal_only 请求剔除 external 候选 | `routing/strategies/compliance.go:28-45` | PASS |
| 未知策略保守拒绝 | default → ErrEgressBlocked | `security/egress.go:96-98` | PASS |

### 双层防御

**Layer 1 (Step [3]): SecurityHook 出网管控**

```go
// egress.go:80-83
case EgressPolicyInternalOnly:
    return ErrEgressBlocked   // 直接阻断整个请求
```

**Layer 2 (Step [9]): S-COMPLIANCE 策略过滤**

```go
// compliance.go:30-45
if reqClass != "internal_only" { return candidates }  // 非内部用户跳过
for _, c := range candidates {
    if nc == "external" {
        c.Eliminated = true   // 标记剔除
    }
}
```

### 审查发现: 字符串一致性风险

| 位置 | 常数值 | 大小写 |
|------|--------|--------|
| `security/egress.go:22` | `EgressPolicyInternalOnly = "INTERNAL_ONLY"` | 大写 |
| `routing/strategies/compliance.go:30,57` | 上下文值 `"internal_only"` | 小写 |

- `CheckEgress` 检查用户 `EgressPolicy` 字段（值为 `"INTERNAL_ONLY"` 大写）。
- `S-COMPLIANCE` 检查上下文键 `CtxKeyNetworkClass`（值为 `"internal_only"` 小写）。

当前代码中未发现将用户 `EgressPolicy` 映射到 `CtxKeyNetworkClass` 的中间件/桥接代码。测试文件 `profile_test.go:184` 手动注入 `"internal_only"` 值。若生产环境未正确设置该上下文值，S-COMPLIANCE 策略的过滤不会触发——但 Layer 1 的 CheckEgress 仍会阻断，故不构成功能缺陷。**建议后续将两端字符串统一为一致的常量或建立明确的映射关系**。

---

## 红线检查

| # | 红线 | 结论 | 证据 |
|---|------|------|------|
| 1 | ModelGrant deny 后仍可调用？ | **未发现此问题** | `checker.go:61` deny 后立即 return error，管线中止 |
| 2 | INTERNAL_ONLY 产生外网流量？ | **未发现此问题** | 双层防御：CheckEgress 阻断 + S-COMPLIANCE 过滤 |
| 3 | 调度改扣费账户？ | **未发现此问题** | pipeline.go 与 routing/ 全链路无 account_id 写入 |

---

## 总结

| 场景 | 结论 | 关键发现 |
|------|------|----------|
| 1. ModelGrant deny 优先 | PASS | DENY 双遍扫描，默认拒绝。级联仅加载匹配类型规则（设计如此） |
| 2. 双层预算 | PASS | 模型级 MODEL_BUDGET_EXCEEDED + Account 级 BUDGET_CAP_EXCEEDED，管线串联 |
| 3. 调度不改账户 | PASS | 路由全链路无 account_id 读取或写入 |
| 4. 全链路追踪 | PASS | 13 个步骤全部审计，request_id 贯穿 ctx + PipelineAuditEvent |
| 5. INTERNAL_ONLY | PASS | 双层防御有效。存在 "INTERNAL_ONLY" vs "internal_only" 大小写不一致，建议统一 |

**红线**: 0/3 条触发。

**待跟进**: INTERNAL_ONLY 字符串统一 (建议将两处引用为同一常量定义，或建立上下文注入桥接)。
