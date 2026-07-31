# RED-3 数据泄露攻击面审计报告

**审计员**: RED-3  
**审计日期**: 2026-07-31  
**审计范围**: ai-gov-fusion/backend 核心安全模块  
**方法论**: 代码白盒审查——基于事实，不推测，不猜测  

---

## 摘要

| 严重度 | 数量 | 说明 |
|--------|------|------|
| CRITICAL | 1 | INTERNAL_ONLY 出网管控完全失效 |
| HIGH | 1 | 内部错误信息（含系统 ID）泄漏至客户端 |
| LOW | 1 | 结构化日志含内部系统 ID |
| PASS | 3 | 密钥泄漏、审计篡改、SQL 注入均通过 |

---

## 攻击面 1: INTERNAL_ONLY 绕过

**严重度**: CRITICAL

**审查文件**:
- `security/egress.go` (CheckEgress 函数)
- `routing/strategies/compliance.go` (ComplianceStrategy.Filter)
- `routing/profile.go` (路由管线集成)

### 发现

#### 事实 1: CheckEgress 是死代码——从未被调用

`CheckEgress` 函数在 `security/egress.go:73` 中定义，逻辑正确：
- `INTERNAL_ONLY` 用户请求 external 模型时返回 `ErrEgressBlocked`
- `HYBRID_ALLOWED` 用户当前放行（白名单校验标注为 P2 骨架）
- 内网模型全部放行

**然而，整个代码库中没有任何调用方**。全局搜索 `CheckEgress` 仅命中其定义位置 `security/egress.go:73`，无任何外部引用。该函数孤悬于代码库中，从未在执行路径中被触发。

#### 事实 2: 路由合规过滤器依赖从未设置的 context key

`ComplianceStrategy.Filter` (`routing/strategies/compliance.go:28-46`) 在路由管线中被注册为硬过滤器（`profile.go:243`，阶段 1），但其判定逻辑依赖从 context 读取 `CtxKeyNetworkClass` (key: `"network_class"`)：

```go
reqClass, _ := ctx.Value(CtxKeyNetworkClass).(string)
if reqClass != "internal_only" {
    return candidates  // 未设置标签时：全部放行，不做任何过滤
}
```

全局搜索 `CtxKeyNetworkClass` 和 `context.WithValue.*network_class`：**无任何代码向 context 注入此键值**。因此 `reqClass` 始终为空字符串，永远不会等于 `"internal_only"`，整个合规过滤逻辑形同虚设。

#### 事实 3: 安全钩子也未启用

`security/hooks.go` 中的 `Hook` 接口定义了 `OnRequest`/`OnResponse` 扩展点，注释明确提到 `INTERNAL_ONLY 强制拦截` 是计划接入的场景之一。但当前使用的是 `NoopHook`（空实现），所有请求直接放行。

### 利用路径

1. 攻击者创建或获取一个 `INTERNAL_ONLY` 策略的用户账户
2. 通过 TokenHub 网关发起对 external 模型（如 OpenAI GPT-4）的请求
3. `CheckEgress` 不会被调用（死代码）
4. `ComplianceStrategy.Filter` 不会过滤（context key 未设置）
5. 请求正常路由至外部模型，数据出境成功

### 影响

违反 D-CON-02（数据不出境定理）。INTERNAL_ONLY 标记的用户/数据可以不受限制地调用外部 AI 模型，所有内部敏感数据面临泄漏至境外云 API 的风险。

### 修复建议

1. **在请求进入路由管线之前设置 context key**：在认证/授权阶段加载用户的 `egress_policy`，若为 `INTERNAL_ONLY` 则调用 `context.WithValue(ctx, CtxKeyNetworkClass, "internal_only")`
2. **在网关层接入 `CheckEgress`**：在 TokenHub 的请求处理管线中（`http.go` 或路由分发逻辑）调用 `CheckEgress`，阻止不合规的请求进入路由阶段
3. **启用安全钩子**：将 `CheckEgress` 封装为一个 `security.Hook` 实现，注入到 `Chain` 中，在 `OnRequest` 阶段拦截

---

## 攻击面 2: 密钥明文泄漏

**严重度**: PASS（当前无风险）

**审查文件**:
- `gov_handlers_fund.go` (handleKeys, handleKeyItem)
- `gov_handlers.go` (路由注册)

### 发现

`/gov/keys` 和 `/gov/keys/{id}` 端点的 handler 函数（`handleKeys`、`handleKeyItem`）**均未实现**——全部返回占位 JSON 响应，例如 `{"message": "Key 列表——待实现"}` 和 `{"message": "Key 详情——待实现"}`。

**当前没有代码返回任何密钥数据（明文或密文）**。端点受 `requireGovAuth` 保护，需要有效的 Bearer Token 或 X-API-Key 以及 ABAC 权限 `iam.key.read`/`iam.key.create`/`iam.key.delete`。

### 审计日志中的密钥

审计事件模型 (`audit/model.go`) 定义了 `ActionKeyCreate` 和 `ActionKeyRevoke` 操作类型常量，但审计事件的 `before_snapshot`/`after_snapshot` 字段标注为 JSON 格式的快照。**当前无任何 handler 实际调用 `RecordEvent` 记录密钥操作**，所以审计日志中不存在密钥泄漏风险。

### 后续关注

当 `handleKeys`/`handleKeyItem` 实现时，需确保：
- GET 响应中不返回 `secret` 字段的明文值（仅返回 `key_id`、`masked_secret` 或哈希）
- 审计事件的 `after_snapshot` 必须脱敏（替换 secret 为 `[REDACTED]` 或仅记录 key_id）
- POST 创建密钥时，secret 仅在响应中返回一次，之后不可再获取

---

## 攻击面 3: 审计日志篡改

**严重度**: PASS

**审查文件**:
- `audit/event.go` (RecordEvent 函数)
- `audit/model.go` (AuditEvent 模型)
- `gov_handlers_abac.go` (handleAuditEvents, handleAuditEventItem)

### 发现

#### RecordEvent: 仅 INSERT，无 UPDATE/DELETE 路径

`RecordEvent` (`audit/event.go:36-59`) 内部只执行 `db.WithContext(ctx).Create(event)`，**没有 UPDATE 或 DELETE 语句**。代码注释明确标注铁律「AU-CON-01：审计事件一旦写入即不可变更或删除」。

函数执行严格校验（event 非空、action/resource_type/resource_id/id 必填），违反直接返回错误——不存在静默跳过或部分写入的情况。

#### API 端点: 无修改/删除能力

`/gov/audit-events/{id}` 端点 (`gov_handlers_abac.go:232-236`) 仅处理 GET 请求，需要 `data.audit.read` 权限。**没有 PUT、PATCH、DELETE 方法**。

`/gov/audit-events` 列表端点 (`gov_handlers_abac.go:227-230`) 也仅支持 GET。

#### 审计哈希链锚定

`audit/model.go` 定义了 `AuditChainAnchor` 模型（哈希链锚点），用于将连续审计事件锚定为 SHA-256 哈希链，提供防篡改验证能力。当前 handler `/gov/audit-chain-anchors` 为只读 + 占位实现。

### 结论

应用层不存在审计日志篡改的攻击路径。唯一的数据修改可能来自数据库层直接操作，但这是基础设施安全域的问题，不在本次代码审计范围内。

---

## 攻击面 4: SQL 注入

**严重度**: PASS

**审查文件**:
- `fund/sqlstore/pg.go` (全部数据访问代码)
- `audit/event.go` (SearchEvents, GetEvent)
- `authz/grant.go` (CreateGrant, GetGrant, DeleteGrant, Evaluate)

### 发现

**所有数据库查询均使用 GORM 参数化查询（`?` 占位符），未发现任何字符串拼接或 `fmt.Sprintf` 构建 SQL 语句的情况。**

关键检查点：

| 位置 | 查询方式 | 安全 |
|------|---------|------|
| `pg.go:80` GetAccount | `Where("id = ?", id)` | 安全 |
| `pg.go:95` GetAccountForUpdate | `Where("id = ?", id)` | 安全 |
| `pg.go:110` UpdateAccountBalances | `Where("id = ? AND version = ?", id, version)` | 安全 |
| `pg.go:149` UpdateAccountBudgetConsumed | `gorm.Expr("budget_consumed_amount + ?", delta)` | 安全 (GORM Expr 参数化) |
| `pg.go:243` ListExpiredFreezes | `Where("status = ? AND expires_at < ?", ...)` | 安全 |
| `pg.go:291` GetLiquidation | `Where("status NOT IN ?", []string{...})` | 安全 |
| `audit/event.go:84-111` SearchEvents | 全部使用 `Where("field = ?", value)` | 安全 |
| `authz/grant.go:98-99` Evaluate | `Where("principal_type = ? AND principal_id = ? AND ...", ...)` | 安全 |

**零实例**使用 `db.Raw()`、`db.Exec()` 或字符串拼接构建 SQL。`gorm.Expr` 使用时也通过参数化传入外部值。

---

## 攻击面 5: 日志敏感信息

**严重度**: LOW

**审查文件**:
- `fund/service.go` (Allocate, Freeze 日志)
- `fund/freeze.go` (Freeze, Settle 日志)
- `fund/lifecycle.go` (RenewFreeze, UnfreezeTimeout, Liquidate 日志)

### 发现

结构化日志 (slog) 中记录以下字段：

| 日志字段 | 出现位置 | 敏感性 |
|----------|---------|--------|
| `account_id` | Allocate, Freeze, Settle, Liquidate | 内部系统 ID |
| `freeze_id` | Freeze, Settle, RenewFreeze | 内部系统 ID |
| `allocation_id` | Allocate | 内部系统 ID |
| `idempotency_key` | Allocate | 业务幂等键（非密钥） |
| `request_id` | Freeze, Settle | 全链路追踪 ID |
| `amount` / `settle_amount` / `refund_amount` | Allocate, Freeze, Settle | 金额数值 |
| `channel` | Allocate | 转账渠道类型 |
| `party_id` | Liquidate | 内部组织 ID |
| `target_account_id` | Liquidate | 内部系统 ID |

**未发现任何密钥、密码、Token、API Key 或其他凭证类数据出现在日志中。** 所有日志字段均为内部系统标识符和业务数值。

### 风险

- LOW：内部系统 ID 虽然本身非敏感，但若日志系统被攻破，可辅助攻击者进行横向移动的关联分析（例如通过 `account_id` 串联资金流水）

### 建议

- 在非开发环境考虑对 `account_id` 和 `freeze_id` 做哈希截断（如 `SHA256(id)[:8]`），降低日志泄露时的关联风险

---

## 攻击面 6: 错误信息泄露

**严重度**: HIGH

**审查文件**:
- `types.go` (AsHTTPError, HTTPError)
- `fund/errors.go` (FundError 及其构造函数)
- `http.go` (writeError)

### 发现

#### 事实 1: AsHTTPError 将原始错误信息直接暴露

`types.go:79-88` 中 `AsHTTPError` 的实现：

```go
func AsHTTPError(err error) *HTTPError {
    if err == nil {
        return nil
    }
    var httpErr *HTTPError
    if errors.As(err, &httpErr) {
        return httpErr
    }
    // 未知错误类型——直接包装原始错误信息暴露给客户端
    return NewHTTPError(500, "internal_error", err.Error())
}
```

对于非 `*HTTPError` 类型的错误，其 `err.Error()` 内容直接作为 HTTP 响应的 `message` 字段返回给客户端。

#### 事实 2: FundError 消息包含内部系统 ID

`fund/errors.go` 中所有 `FundError` 的 `Message` 字段都包含内部系统标识符：

| 错误构造函数 | 消息示例 |
|-------------|---------|
| `newInsufficientBalanceError` | `"account {id} has {amount} available, requested {amount}"` |
| `newAccountFrozenError` | `"account {id} is {status}"` |
| `newFreezeExpiredError` | `"freeze {id} has expired"` |
| `newFreezeNotFoundError` | `"freeze {id} not found"` |
| `newAllocationChannelDeniedError` | `"channel {ch} not permitted from {src} to {dst}"` |
| `newLiquidationStageInvalidError` | `"account {id} cannot transition from {cur} to {target}"` |

**14 个 FundError 构造函数中，12 个直接暴露了 account_id、freeze_id 或 allocation_id。**

#### 事实 3: 写错误路径不区分内外

`http.go:8280-8295` 中 `writeError` 不判断错误类型是否为内部错误，所有错误均按统一格式返回。HTTP 500 错误携带原始 `err.Error()` 消息。

### 利用路径

1. 攻击者向 Fund API 发送精心构造的请求（如对不存在的 freeze_id 发起 settle）
2. 服务返回错误，其中包含内部 freeze_id 或 account_id
3. 攻击者收集这些 ID，用于信息枚举、暴力猜测或关联攻击
4. 结合其他漏洞（如 IDOR），可能访问未授权的资源

### 影响

- 内部系统标识符泄漏，辅助攻击者进行信息收集
- 数据库表结构或业务逻辑信息可能通过错误消息间接暴露

### 修复建议

1. **修改 AsHTTPError**：对于未知错误类型，返回通用消息而非 `err.Error()`：
   ```go
   return NewHTTPError(500, "internal_error", "内部服务错误，请联系管理员")
   ```
2. **引入错误脱敏层**：FundError 在序列化为 HTTP 响应时，使用 `Code` 字段作为客户端可见的错误码，`Message` 仅记录到服务端日志而非响应
3. **审计所有 handler 的错误返回路径**：确保 `writeError` 调用时传入的 error 不会包含数据库错误、SQL 语句、表名等底层信息

---

## 附录: 审查文件清单

| 文件 | 攻击面 | 绝对路径 |
|------|--------|---------|
| security/egress.go | AS-1 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\security\egress.go` |
| security/hooks.go | AS-1 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\security\hooks.go` |
| routing/strategies/compliance.go | AS-1 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\routing\strategies\compliance.go` |
| routing/profile.go | AS-1 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\routing\profile.go` |
| gov_handlers.go | AS-2, AS-3 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\gov_handlers.go` |
| gov_handlers_fund.go | AS-2 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\gov_handlers_fund.go` |
| gov_handlers_abac.go | AS-3 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\gov_handlers_abac.go` |
| audit/event.go | AS-3, AS-4 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\audit\event.go` |
| audit/model.go | AS-3 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\audit\model.go` |
| fund/sqlstore/pg.go | AS-4 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\sqlstore\pg.go` |
| fund/service.go | AS-5 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\service.go` |
| fund/freeze.go | AS-5 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\freeze.go` |
| fund/lifecycle.go | AS-5 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\lifecycle.go` |
| fund/errors.go | AS-6 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\errors.go` |
| http.go | AS-6 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\http.go` |
| types.go | AS-6 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\types.go` |
| authz/middleware.go | AS-1 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\authz\middleware.go` |
| authz/grant.go | AS-4 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\authz\grant.go` |

---

*报告结束。所有结论基于上述文件的代码审查，无可推测内容。*
