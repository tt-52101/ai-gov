# DATA-1 财务守恒定理审计报告

**审计员**: DATA-1（数据安全专家）
**审计日期**: 2026-07-31
**审计范围**: F-CON-01 ~ F-CON-06（25 条守恒定理之财务守恒子集）
**审计原则**: 铁律——基于事实，代码行号证据

---

## 总览

| 定理 | 判定 | 关键文件 |
|------|------|----------|
| F-CON-01 余额守恒 | PASS | fund/service.go, fund/freeze.go, fund/lifecycle.go |
| F-CON-02 划拨守恒 | PASS | fund/service.go |
| F-CON-03 冻结守恒 | PASS | fund/freeze.go, fund/lifecycle.go |
| F-CON-04 禁止负余额 | PASS | fund/freeze.go, fund/service.go |
| F-CON-05 幂等写 | PASS | idempotency/claim.go, idempotency/store.go, idempotency/model.go |
| F-CON-06 管理员也不例外 | PASS（含设计观察） | server/gov_handlers.go, server/gov_handlers_fund.go |

**总体判定: 全部 6 条通过 (6/6 PASS)**

---

## F-CON-01 余额守恒

> **定理**: 所有余额变更必须经过 `InsertLedger` 写入流水表，禁止直接 UPDATE accounts 而不写 ledgers 的路径。

### 审计方法
逐一追踪所有修改 `available_balance` 或 `frozen_balance` 的代码路径，验证每个路径是否同时调用 `InsertLedger`。

### 证据

**路径 1: Allocate（划拨）**
- 文件: `ai-gov-fusion/backend/internal/server/fund/service.go`
- 余额写入: 第 169 行 `UpdateAccountBalances(tx, ctx, req.SrcAccountID, ...)`, 第 174 行 `UpdateAccountBalances(tx, ctx, req.DstAccountID, ...)`
- 流水写入: 第 211 行 `InsertLedger(tx, ctx, srcLedger)`, 第 226 行 `InsertLedger(tx, ctx, dstLedger)`
- 事务边界: 第 110 行 `s.Store.WithTx(ctx, ...)` 包裹全部操作

**路径 2: Freeze（冻结）**
- 文件: `ai-gov-fusion/backend/internal/server/fund/freeze.go`
- 余额写入: 第 97 行 `UpdateAccountBalances(tx, ctx, req.AccountID, availableAfter, frozenAfter, ...)`
- 流水写入: 第 132 行 `InsertLedger(tx, ctx, freezeLedger)`
- 事务边界: 第 44 行 `s.Store.WithTx(ctx, ...)`

**路径 3: Settle（结算）**
- 文件: `ai-gov-fusion/backend/internal/server/fund/freeze.go`
- 余额写入: 第 294 行 `UpdateAccountBalances(tx, ctx, freeze.AccountID, availableAfter, frozenAfter, ...)`
- 流水写入: 第 319 行 `InsertLedger(tx, ctx, settleLedger)`, 第 336 行 `InsertLedger(tx, ctx, refundLedger)`（存在退款时）
- 事务边界: 第 212 行 `s.Store.WithTx(ctx, ...)`
- 补充: 第 300 行 `UpdateAccountBudgetConsumed` 只修改 `budget_consumed_amount`，不修改余额字段

**路径 4: UnfreezeTimeout（超时解冻）**
- 文件: `ai-gov-fusion/backend/internal/server/fund/lifecycle.go`
- 余额写入: 第 133 行 `UpdateAccountBalances(tx, ctx, freeze.AccountID, availableAfter, frozenAfter, ...)`
- 流水写入: 第 149 行 `InsertLedger(tx, ctx, ledger)`
- 事务边界: 第 120 行 `s.Store.WithTx(ctx, ...)`

**路径 5: Liquidate refunding（清算资金转移）**
- 文件: `ai-gov-fusion/backend/internal/server/fund/lifecycle.go`
- 余额写入: 第 324 行 `UpdateAccountBalances(tx, ctx, req.AccountID, ...)`, 第 328 行 `UpdateAccountBalances(tx, ctx, targetID, ...)`
- 流水写入: 第 342 行 `InsertLedger(tx, ctx, srcLedger)`, 第 355 行 `InsertLedger(tx, ctx, dstLedger)`
- 事务边界: 第 209 行 `s.Store.WithTx(ctx, ...)`

**排除项**:
- `UpdateAccountStatus` (sqlstore/pg.go:127-143): 只修改 `status` 字段，不修改余额
- `UpdateAccountBudgetConsumed` (sqlstore/pg.go:146-158): 只修改 `budget_consumed_amount`，不修改余额
- 不存在任何直接调用 `gorm.DB.Updates` 或 `gorm.DB.Update` 修改 `accounts.available_balance` / `accounts.frozen_balance` 的代码路径

### 判定: **PASS**
全部 5 条余额变更路径均同时写入 ledgers 表，无绕过路径。`fund/store.go` 第 61-63 行明确声明 `InsertLedger` 为 append-only 日志，第 22-23 行注释规定"所有金额变更必须通过 Store 接口——禁止在 service 逻辑中直接调用 GORM"。

---

## F-CON-02 划拨守恒

> **定理**: `Allocate()` 在同一事务内保证 `src_delta + dst_delta = 0`。

### 证据

- 文件: `ai-gov-fusion/backend/internal/server/fund/service.go`
- 金额计算: 第 165 行 `srcAvailableAfter := srcAcct.AvailableBalance.Decimal.Sub(req.Amount.Decimal)`, 第 166 行 `dstAvailableAfter := dstAcct.AvailableBalance.Decimal.Add(req.Amount.Decimal)`
  - src_delta = -amount, dst_delta = +amount, 和为零
- 事务边界: 第 110 行 `s.Store.WithTx(ctx, ...)` 包裹从锁定账户到写入流水全部操作
- 源账户扣减: 第 169 行 `UpdateAccountBalances(tx, ctx, req.SrcAccountID, srcAvailableAfter, ...)`
- 目标账户增加: 第 174 行 `UpdateAccountBalances(tx, ctx, req.DstAccountID, dstAvailableAfter, ...)`
- 死锁防护: 第 112-115 行按 ID 排序锁定账户 (`firstID > secondID` 时交换锁定顺序)
- 余额充足性检查: 第 155-157 行 `srcAcct.AvailableBalance.Decimal.LessThan(req.Amount.Decimal)` 前置校验
- 测试验证: `fund/service_test.go` 第 368-400 行 `TestAllocate_Conservation` 显式断言 `totalBefore.Equal(totalAfter)`

### 判定: **PASS**
src 扣减 amount，dst 增加 amount，数学和为零。两者在同一事务内，通过乐观锁（version 字段）和 SELECT FOR UPDATE 行锁保证原子性。有专门的守恒测试用例。

---

## F-CON-03 冻结守恒

> **定理**: 冻结时 `available -= amount, frozen += amount` 在同一事务；解冻金额不得超过冻结金额。

### 证据

**Freeze（冻结）**:
- 文件: `ai-gov-fusion/backend/internal/server/fund/freeze.go`
- 计算: 第 93 行 `availableAfter := acct.AvailableBalance.Decimal.Sub(req.Amount.Decimal)`, 第 94 行 `frozenAfter := acct.FrozenBalance.Decimal.Add(req.Amount.Decimal)`
- 写入: 第 97 行 `UpdateAccountBalances(tx, ctx, req.AccountID, availableAfter, frozenAfter, ...)`——单次调用同时更新两个字段
- 事务边界: 第 44 行 `s.Store.WithTx(ctx, ...)`

**UnfreezeTimeout（超时解冻）**:
- 文件: `ai-gov-fusion/backend/internal/server/fund/lifecycle.go`
- 计算: 第 130 行 `availableAfter := acct.AvailableBalance.Decimal.Add(amount)`, 第 131 行 `frozenAfter := acct.FrozenBalance.Decimal.Sub(amount)`
- `amount` 来自 `freeze.Amount.Decimal`（第 129 行），即全部解冻 = 冻结金额
- 事务边界: 第 120 行 `s.Store.WithTx(ctx, ...)`

**Settle（结算）**:
- 文件: `ai-gov-fusion/backend/internal/server/fund/freeze.go`
- 解冻金额校验: 第 248-255 行 `if settleAmount.GreaterThan(frozenAmount)` 返回错误，确保 settleAmount <= frozenAmount
- 退款计算: 第 267 行 `refund := frozenAmount.Sub(settleAmount)`，refund >= 0
- 余额计算: 第 273 行 `availableAfter := acct.AvailableBalance.Decimal.Add(refund)`, 第 274 行 `frozenAfter := acct.FrozenBalance.Decimal.Sub(frozenAmount)`
- frozen_balance 减少 frozenAmount（全部释放），available 仅增加 refund（sell 部分被消费）
- 事务边界: 第 212 行 `s.Store.WithTx(ctx, ...)`

### 判定: **PASS**
- Freeze: available 减少 amount，frozen 增加 amount，同一 `UpdateAccountBalances` 调用内原子完成
- UnfreezeTimeout: 全部解冻金额 = 原始冻结金额，不增不减
- Settle: settleAmount <= frozenAmount 显式校验；frozen 减少 frozenAmount，available 仅增加 refund
- 三条路径均在单一事务内

---

## F-CON-04 禁止负余额

> **定理**: `Freeze()` 冻结前检查 `available >= freeze_amount`。

### 证据

**Freeze 前置检查**:
- 文件: `ai-gov-fusion/backend/internal/server/fund/freeze.go`
- 第 84-86 行:
  ```go
  if acct.AvailableBalance.Decimal.LessThan(req.Amount.Decimal) {
      return newInsufficientBalanceError(req.AccountID, acct.AvailableBalance, req.Amount)
  }
  ```
- 检查在 `UpdateAccountBalances`（第 97 行）之前执行
- 测试验证: `fund/service_test.go` 第 579-602 行 `TestFreeze_InsufficientBalance`

**Allocate 前置检查**:
- 文件: `ai-gov-fusion/backend/internal/server/fund/service.go`
- 第 155-157 行:
  ```go
  if srcAcct.AvailableBalance.Decimal.LessThan(req.Amount.Decimal) {
      return newInsufficientBalanceError(req.SrcAccountID, srcAcct.AvailableBalance, req.Amount)
  }
  ```

**Settle 防御性检查**:
- 文件: `ai-gov-fusion/backend/internal/server/fund/freeze.go`
- 第 276-291 行: `if availableAfter.LessThan(decimal.Zero)` 检查结算后余额不为负（防御 overdraw）

**边界条件**:
- service.go 第 74-76 行: `req.Amount <= 0` 拒绝（`newAmountMustBePositiveError`）
- freeze.go 第 34-36 行: 同样拒绝非正金额

### 判定: **PASS**
Freeze、Allocate、Settle 三条路径均有前置余额充足性检查。另有零/负金额边界校验。

---

## F-CON-05 幂等写

> **定理**: 使用 INSERT ON CONFLICT（或等效的 INSERT-first + UNIQUE 约束）原子抢占，禁止 SELECT-then-INSERT。

### 证据

**INSERT-first 策略**:
- 文件: `ai-gov-fusion/backend/internal/server/idempotency/claim.go`
- 第 73-87 行: `Claim()` 函数先执行 `InsertRecord(db, rec)`，依赖数据库 UNIQUE 约束实现原子性
- 第 47-64 行注释明确记录:
  - "Strategy (INSERT-first, IRON RULE 1)"
  - "NEVER use SELECT-then-INSERT: that path has a TOCTOU race"

**InsertRecord 实现**:
- 文件: `ai-gov-fusion/backend/internal/server/idempotency/store.go`
- 第 48-54 行: `InsertRecord` 是纯粹的 `db.Create(rec)`，无 SELECT 前置

**UNIQUE 约束定义**:
- 文件: `ai-gov-fusion/backend/internal/server/idempotency/model.go`
- 第 53 行: `Scope string ... gorm:"uniqueIndex:idx_idempotency_scope_actor_key,priority:1"`
- 第 57 行: `ActorID string ... gorm:"uniqueIndex:idx_idempotency_scope_actor_key,priority:2"`
- 第 61 行: `IdempotencyKey string ... gorm:"uniqueIndex:idx_idempotency_scope_actor_key,priority:3"`
- 复合唯一约束 `(scope, actor_id, idempotency_key)` 是竞争仲裁者

**冲突处理（INSERT 失败后）**:
- claim.go 第 84-87 行: `isDuplicateKeyError` 检测（PG: gorm.ErrDuplicatedKey, SQLite: UNIQUE constraint failed）
- 第 91 行: INSERT 失败后才 `GetRecord` 查询已有记录——这不是 SELECT-then-INSERT，这是 "INSERT 失败后 SELECT"，无 TOCTOU 风险
- 第 107-119 行: 过期记录通过 `reclaim()` 在子事务内原子 DELETE+INSERT 回收
- store.go 第 23-33 行: `isDuplicateKeyError` 跨 PostgreSQL / SQLite 驱动兼容

**并发测试**:
- 文件: `ai-gov-fusion/backend/internal/server/idempotency/claim_test.go`
- 第 200-266 行 `TestClaim_ConcurrentInsert`: 10 个并发 goroutine，断言 `claimedCount == 1`

**Fund Service 集成**:
- 文件: `ai-gov-fusion/backend/internal/server/fund/service.go`
- 第 82-107 行: Allocate 在事务外调用 `Idempotency.Claim`，Claim 失败时检索已有结果返回

### 判定: **PASS**
核心路径（Claim）使用 INSERT-first + UNIQUE 约束抢占，严禁 SELECT-then-INSERT。过期的回收在数据库子事务内原子 DELETE+INSERT。并发测试验证竞争正确性。Complete/Fail 在 Claim 胜出者的独占路径上执行，无并发冲突。

---

## F-CON-06 管理员也不例外

> **定理**: 禁止绕过 Store 接口直接操作 DB 的超级管理员路径。

### 证据

**Fund 域架构**:
- 文件: `ai-gov-fusion/backend/internal/server/fund/store.go`
- 第 20-27 行: `Store` 接口定义——要求"所有金额变更必须通过此接口"（第 22-23 行注释）
- 第 61-63 行: `InsertLedger` 注释为 append-only 日志
- `fund.Service` 只持有 `Store` 和 `IdempotencyChecker` 接口，无 `*gorm.DB` 引用

**HTTP Handler 层**:
- 文件: `ai-gov-fusion/backend/internal/server/gov_handlers_fund.go`
- 全部 handler 为占位实现（返回"待实现"），未执行任何数据库操作
- grep 确认: 该文件中无 `db.` / `DB.` / `gorm` 直接引用

**GovDependencies 结构**:
- 文件: `ai-gov-fusion/backend/internal/server/gov_handlers.go`
- 第 37-56 行: `GovDependencies` 结构体
- 第 39 行: `FundService *fund.Service`——正确的依赖注入
- 第 55 行: `DB *gorm.DB`——标注为"用于直接查询表"，当前未被任何 fund handler 使用

**设计观察（非当前缺陷）**:
`GovDependencies.DB` 字段（gov_handlers.go:55）暴露了裸 `*gorm.DB` 句柄。虽然当前无 fund handler 使用它，但存在未来被错误使用的设计风险。建议：
1. 将 `DB` 重命名为 `ReadOnlyDB` 并搭配只读事务中间件
2. 或直接移除该字段，由各领域 Store 接口提供所需的只读查询方法

### 判定: **PASS**
当前代码中不存在绕过 `fund.Store` 接口直接操作数据库的路径。所有余额变更必须经过 `fund.Service` -> `fund.Store` 调用链。`GovDependencies.DB` 暴露裸 GORM 句柄是设计风险而非当前违规，记录为观察项供后续关注。

---

## 附录: 审计文件清单

| 文件 | 用途 |
|------|------|
| `ai-gov-fusion/backend/internal/server/fund/service.go` | Allocate 实现、防护检查 |
| `ai-gov-fusion/backend/internal/server/fund/freeze.go` | Freeze / Settle 实现 |
| `ai-gov-fusion/backend/internal/server/fund/lifecycle.go` | UnfreezeTimeout / Liquidate 实现 |
| `ai-gov-fusion/backend/internal/server/fund/store.go` | Store 接口定义 |
| `ai-gov-fusion/backend/internal/server/fund/sqlstore/pg.go` | PostgreSQL/SQLite 持久化实现 |
| `ai-gov-fusion/backend/internal/server/fund/model.go` | 数据模型（Account, Ledger, Freeze 等） |
| `ai-gov-fusion/backend/internal/server/fund/errors.go` | 错误定义 |
| `ai-gov-fusion/backend/internal/server/fund/service_test.go` | 服务层单元测试 |
| `ai-gov-fusion/backend/internal/server/idempotency/claim.go` | 幂等键抢占逻辑 |
| `ai-gov-fusion/backend/internal/server/idempotency/store.go` | 幂等记录 CRUD |
| `ai-gov-fusion/backend/internal/server/idempotency/model.go` | 幂等记录数据模型（含 UNIQUE 约束） |
| `ai-gov-fusion/backend/internal/server/idempotency/claim_test.go` | 幂等并发测试 |
| `ai-gov-fusion/backend/internal/server/idempotency/middleware.go` | HTTP 幂等中间件 |
| `ai-gov-fusion/backend/internal/server/gov_handlers.go` | 治理 API 路由注册、GovDependencies |
| `ai-gov-fusion/backend/internal/server/gov_handlers_fund.go` | Fund 域 HTTP handler |
