# RED-2 资金攻击面审计报告

**审计日期**: 2026-07-31
**审计范围**: `ai-gov-fusion/backend/internal/server/fund/` 及关联包
**审计方法**: 静态代码审查，逐行审查每个攻击面

---

## 攻击面 1: 重复划拨 (Duplicate Allocation)

**目标**: `fund/service.go` Allocate()——无 Idempotency-Key 时是否拒绝？idempotency/claim.go 是否 INSERT ON CONFLICT 原子抢占？

### 代码证据

**`fund/service.go` 第 82 行**:
```go
if req.IdempotencyKey != "" && s.Idempotency != nil {
```
幂等键为**可选字段**。调用方不提供 `IdempotencyKey` 即可完全绕过幂等保护，重复执行同样的划拨请求。`AllocateRequest.IdempotencyKey` 是普通 `string` 类型，无必填标记。

**`idempotency/claim.go` 第 73-75 行** (IRON RULE 1: INSERT-first):
```go
if err := InsertRecord(db, rec); err == nil {
    return true, nil, nil
}
```
方案正确——INSERT 优先级，依赖数据库 UNIQUE 约束实现原子性，无 TOCTOU 竞态。

### 判定: HIGH (高危)

幂等键缺失时的重复划拨无任何防护。建议:
1. 对 POST/PUT 类 `Allocate` 请求强制要求 `Idempotency-Key` 头
2. 或在 `Allocate()` 无幂等键时返回 `422 Unprocessable Entity`，拒绝执行

---

## 攻击面 2: 并发竞态 (Concurrent Race)

**目标**: `fund/service.go`——两账户加锁顺序是否按 ID 排序防死锁？`sqlstore/pg.go`——是否使用 SELECT FOR UPDATE？

### 代码证据

**加锁顺序 (service.go 第 112-115 行)**:
```go
firstID, secondID := req.SrcAccountID, req.DstAccountID
if firstID > secondID {
    firstID, secondID = secondID, firstID
}
```
按账户 ID 字典序固定锁定顺序，**杜绝** AB-BA 死锁。PASS.

**SELECT FOR UPDATE (sqlstore/pg.go 第 91-103 行)**:
```go
gtx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
    Where("id = ?", id).First(&acct)
```
使用 GORM `clause.Locking{Strength: "UPDATE"}` 直接在 SQL 层添加 `SELECT ... FOR UPDATE` 行锁。PASS.

**乐观锁双重保护 (sqlstore/pg.go 第 109-112 行)**:
```go
Where("id = ? AND version = ?", id, version).
Updates(map[string]interface{}{
    "version": version + 1,
    ...
})
```
收支平衡更新同时检查 version，version 不匹配则拒绝写入。PASS.

**Settle() 竞态注意 (fund/freeze.go 第 214 行)**:
```go
freeze, err := s.Store.GetFreeze(ctx, req.FreezeID)
```
`GetFreeze` **不带行锁**（pg.go 第 181 行），仅使用普通 SELECT。账户后续被 FOR UPDATE 锁定（第 258 行），但 freeze 记录本身未锁定。如果两个并发 Settle 同用一个 freeze_id，第一个成功后会更新 freeze 状态为 "settled"，第二个在获取 freeze 状态时可能读到旧快照。

### 判定: MEDIUM (中危)

核心 Allocate/Freeze 路径的锁机制正确，但 Settle 对 freeze 记录未使用行锁，存在并发 Settle 同一 freeze 的窗口。建议对 `GetFreeze` 增加 FOR UPDATE 或在 `UpdateFreezeStatus` 中增加状态条件 (`WHERE status='open'`)，目前 `UpdateFreezeStatus` 仅按 `id` 过滤（pg.go 第 210 行）。

---

## 攻击面 3: 负金额划拨 (Negative Amount Allocation)

**目标**: `fund/service.go` Allocate()——amount <= 0 是否被拒绝？

### 代码证据

**`fund/service.go` 第 74-76 行**:
```go
if req.Amount.Decimal.LessThanOrEqual(decimal.Zero) {
    return nil, newAmountMustBePositiveError(req.Amount)
}
```
零金额和负金额均被 `LessThanOrEqual` 捕获，返回 `AMOUNT_MUST_BE_POSITIVE` 错误。PASS.

**`fund/freeze.go` 第 34-36 行——Freeze 相同保护**:
```go
if req.Amount.Decimal.LessThanOrEqual(decimal.Zero) {
    return nil, newAmountMustBePositiveError(req.Amount)
}
```
Freeze 同样拒绝零金额和负金额。PASS.

### 判定: PASS

负金额和零金额在 Allocate 和 Freeze 入口均被正确拒绝。测试覆盖缺失——service_test.go 中无针对零金额/负金额的单元测试。

---

## 攻击面 4: 自划拨 (Self-transfer)

**目标**: Allocate()——src == dst 是否被拒绝？

### 代码证据

**`fund/service.go` 第 77-79 行**:
```go
if req.SrcAccountID == req.DstAccountID {
    return nil, newSelfTransferError(req.SrcAccountID)
}
```
自划拨在入口被直接拒绝，返回 `SELF_TRANSFER` 错误。PASS.

**Liquidate 同样防护 (lifecycle.go 第 233-235 行)**:
```go
if req.TargetAccountID == req.AccountID {
    return newSelfTransferError(req.AccountID)
}
```
清算目标的自我转让也被拒绝。PASS.

### 判定: PASS

自划拨在 Allocate 和 Liquidate 两个入口均被正确拒绝。

---

## 攻击面 5: 绕过通道直接 SQL (Bypass Channel Validation)

**目标**: `fund/service.go`——是否有任何路径绕过 CanAllocate 校验？

### 代码证据

**`fund/service.go` 第 286-298 行——`validateChannel()`**:
```go
func (s *Service) validateChannel(ctx context.Context, channel string, srcAccountID, dstAccountID string, edgeID *string) error {
    switch channel {
    case ChannelParent, ChannelSponsors, ChannelAllocates, ChannelWhitelist:
        // Channel is syntactically valid. Full semantic validation (checking
        // party_edges and allocate_whitelist tables) requires party package
        // integration and is deferred to a later stage.
        return nil  // <-- 任何已知通道名称都直接放行
    default:
        return newAllocationChannelDeniedError(srcAccountID, dstAccountID, channel)
    }
}
```

**关键事实**: 
- `validateChannel()` 仅检查 channel 字符串是否属于四个已知常量 (`parent`, `sponsors`, `allocates`, `whitelist`)，**不查询数据库验证实际边是否存在**
- `party/service.go` 中的 `CanAllocate()` 方法（第 178 行）**已完成完整的边验证实现**——它通过 `FindEdge()` 查询 `party_edges` 表，检查 `AllowsFund` 和边类型，但 `validateChannel()` **从未调用它**
- 注释明确标注 "deferred to a later stage"，说明这是占位实现

### 判定: CRITICAL (严重)

任何调用方只需传入 `channel="allocates"` 即可完全绕过通道校验，在无实际 party 边关系的两个账户间划拨资金。攻击示例: 攻击者持有两个账户 A 和 B，发送 `POST /allocate {"src":"A","dst":"B","amount":1000,"channel":"allocates"}`，即使 A 和 B 之间没有 `allocates` 边，划拨也会成功执行。

**修复建议**: 在 `validateChannel()` 内部调用 `party.Service.CanAllocate()`，或将通道校验直接内联到 Allocate 事务中。

---

## 攻击面 6: 整数溢出 (Integer Overflow)

**目标**: decimal.Decimal 是否防御超大金额？

### 代码证据

**`fund/model.go` 第 11-16 行**:
```go
type Decimal struct {
    decimal.Decimal
}
```
`shopspring/decimal.Decimal` 内部使用 `math/big.Int` 表示数值，提供任意精度算术。不存在传统意义上的整数溢出。PASS.

**无最大金额限制**: 代码中无任何 `MAX_AMOUNT` 常量或边界检查。理论上可传入 `10^1000000` 级别的金额，虽不会导致计算错误，但可能引发:
- 内存膨胀（big.Int 按需分配）
- 数据库 NUMERIC(18,6) 列在 `pg.go` 中未映射——GORM 模型定义的 `type:numeric(18,6)`（model.go 第 174 行）是 GORM 的 AutoMigrate 提示，非运行时强制约束

### 判定: LOW (低危)

核心计算无溢出风险，但缺少明确的最大金额验证。建议添加合理的业务上限（如 `10^12` 即一万亿级别）。

---

## 攻击面 7: 冻结金额超过余额 (Freeze Amount Exceeds Balance)

**目标**: `fund/freeze.go`——freeze_amount > available 时是否拒绝？

### 代码证据

**`fund/freeze.go` 第 84-86 行**:
```go
if acct.AvailableBalance.Decimal.LessThan(req.Amount.Decimal) {
    return newInsufficientBalanceError(req.AccountID, acct.AvailableBalance, req.Amount)
}
```
在预算帽检查之后、余额更新之前，显式检查可用余额是否足够覆盖冻结金额。PASS.

**预算帽同样检查 (freeze.go 第 57-66 行)**:
```go
if acct.BudgetLimitAmount != nil {
    // ...
    if newConsumed.GreaterThan(acct.BudgetLimitAmount.Decimal) {
        return newBudgetCapExceededError(...)
    }
}
```
双重保护: 先检查预算帽，再检查可用余额。PASS.

### 判定: PASS

冻结金额超过可用余额或预算上限时均被正确拒绝。`service_test.go` 第 581-602 行已有单元测试覆盖 `TestFreeze_InsufficientBalance`。

---

## 综合评估

| 攻击面 | 状态 | 评级 |
|--------|------|------|
| 1. 重复划拨——幂等键可选 | **需修复** | HIGH |
| 1. 重复划拨——INSERT-first 原子抢占 | PASS | - |
| 2. 并发竞态——锁排序 | PASS | - |
| 2. 并发竞态——SELECT FOR UPDATE | PASS | - |
| 2. 并发竞态——Settle freeze 未锁定 | **需修复** | MEDIUM |
| 3. 负金额划拨 | PASS | - |
| 4. 自划拨 | PASS | - |
| 5. 绕过通道校验 (validateChannel 占位) | **需修复** | CRITICAL |
| 6. 整数溢出 | PASS | - |
| 7. 冻结金额超过余额 | PASS | - |

### 必须修复 (Release Blocker)

1. **CRITICAL** - `validateChannel()` 是占位实现，不验证实际的 party 边关系。必须在 Allocate 事务内调用 `party.Service.CanAllocate()` 或内联等效的边查询。
2. **HIGH** - `IdempotencyKey` 是可选的。`Allocate()` 在无 idempotency key 时不拒绝请求，允许重复划拨。建议强制要求。

### 建议修复 (下一迭代)

3. **MEDIUM** - `Settle()` 获取 freeze 记录时无行锁 (`GetFreeze` 是普通 SELECT)，并发 Settle 同一 freeze_id 存在竞态窗口。

---

## 审计文件清单

| 文件 | 绝对路径 |
|------|---------|
| Allocate/Freeze/Settle 核心逻辑 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\service.go` |
| Freeze 逻辑 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\freeze.go` |
| Lifecycle (Unfreeze/Liquidate) | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\lifecycle.go` |
| SQL Store (SELECT FOR UPDATE) | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\sqlstore\pg.go` |
| Idempotency Claim (INSERT-first) | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\idempotency\claim.go` |
| Idempotency Store | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\idempotency\store.go` |
| Idempotency Model | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\idempotency\model.go` |
| Fund Model (类型定义) | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\model.go` |
| Fund Errors | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\errors.go` |
| Store 接口 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\store.go` |
| Party CanAllocate (已实现但未调用) | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\party\service.go` |
| HTTP Handlers (Fund 域) | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\gov_handlers_fund.go` |
| Fund 测试 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\fund\service_test.go` |
| Idempotency 测试 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\idempotency\claim_test.go` |
