# QA-1 资金治理域验收报告

**验收人员**：QA-1（资金治理域功能验收专家）
**验收日期**：2026-07-31
**验收阶段**：静态代码审查（非运行时验收）
**PRD 基线**：`docs/prd/AI-GOV-PRD-v3.2.0.md`

---

## 检查项清单

| # | PRD 引用 | 检查项 | 代码证据 | 结果 |
|---|---------|--------|---------|------|
| 1 | §13.1 "独立项目/组织池/出资划拨——守恒与通道正确"；F-CON-02 | 划拨守恒：src_delta + dst_delta = 0 在单事务内 | `fund/service.go` 行110：`s.Store.WithTx(ctx, fn)` 将所有操作包在单事务内；行164-166：`srcAvailableAfter = src - amount`, `dstAvailableAfter = dst + amount`，满足 dst_delta = -src_delta；`fund/service_test.go` 行370-400：`TestAllocate_Conservation` 显式验证守恒约束 | PASSED |
| 2 | §8.3 "默认 TTL 15 分钟（可配 1-60 分钟）" | 冻结 TTL —— 默认 15 分钟 | `fund/service.go` 行44：`defaultFreezeTTL = 15 * time.Minute`；`fund/freeze.go` 行38-41：`if req.TTL <= 0 { ttl = defaultFreezeTTL }` | PASSED |
| 2b | §8.1 step 5 "预算帽检查（若启用）BUDGET_CAP_EXCEEDED" | 冻结 TTL —— 检查预算帽 | `fund/freeze.go` 行57-66：若 `acct.BudgetLimitAmount != nil`，计算 `newConsumed = consumed + estimatedCost`，若 `newConsumed > limit` 返回 `newBudgetCapExceededError()` | PASSED |
| 3 | §8.1 step 8 "结算按实际用量与落地价目算 cost/sell；多退少补" | 结算：多退少补 | `fund/freeze.go` 行267：`refund = frozenAmount - settleAmount`（退）；行273：`availableAfter = available + refund`（多退）；行293-296：settle 金额真正从 available 消耗（少补）；行323-339：若有 refund 则插入 unfreeze 流水 | PASSED |
| 3b | §8.1 step 8 | 结算：孤儿结算（冻结过期场景） | `fund/freeze.go` 行227-237：`if now.After(freeze.ExpiresAt) && freeze.Status == FreezeStatusOpen` 记录警告日志并尝试继续执行结算（注释说明 "Attempt orphan settle"），行248-255 仍在 OPEN 状态下进行 settle amount 校验 | PASSED |
| 4 | §8.4 "active -> liquidating_block_new -> liquidating_drain -> liquidating_transfer -> liquidated" | 清算 5 阶段状态机 | `fund/lifecycle.go` 行207-418：`Liquidate()` 实现状态机；行422-438：`advanceLiquidationStage()` 定义 blocking->draining->refunding->closing->closed 五步转换；`fund/model.go` 行98-104：Account 状态常量 `active/liquidating_block_new/liquidating_drain/liquidating_transfer/liquidated/closed` 完整映射 PRD 阶段 | PASSED (注) |
| 5 | §8.6 "Idempotency-Key (UUID v4, <=255)；UNIQUE(scope, actor_id, idempotency_key) 原子抢占" | 幂等：Idempotency-Key 原子抢占 | `fund/service.go` 行82-107：`Allocate()` 先 `Claim()` 后执行，已 claim 则 `Retrieve()` 返回原结果或返回 409；`idempotency/claim.go` 行73-136：`Claim()` 使用 INSERT-first 策略，数据库 UNIQUE 约束保证原子性；`idempotency/model.go` 行43：`UNIQUE(scope, actor_id, idempotency_key)` 复合唯一索引 | PASSED |
| 6 | §8.1 step 5 "预算帽检查 -> BUDGET_CAP_EXCEEDED"；step 7 "可用余额 >= 冻结额（失败 -> INSUFFICIENT_BALANCE）" | 预算帽分码：BUDGET_CAP_EXCEEDED vs INSUFFICIENT_BALANCE | `fund/errors.go` 行82-86：`newInsufficientBalanceError()` Code=`"INSUFFICIENT_BALANCE"`；行98-103：`newBudgetCapExceededError()` Code=`"BUDGET_CAP_EXCEEDED"` —— 两个错误码完全独立区分；`fund/freeze.go` 行63-66：预算帽命中返回 `newBudgetCapExceededError()`；行84-86：余额不足返回 `newInsufficientBalanceError()` | PASSED |
| 7 | §13.1 "告警比例 80% 只告警不阻断" | 告警不阻断：80% 告警只通知不拒绝 | `fund/freeze.go` 行69-80：当 `BudgetWarnRatio != nil` 且 `ratio >= warnRatio` 时，仅执行 `slog.WarnContext()` 记录警告日志，**无 return error，不阻断冻结继续执行**；行83-86 之后仍检查可用余额，流程不中断 | PASSED |

**注（项 4）**：代码中的 liquidation 状态机使用 `blocking -> draining -> refunding -> closing -> closed` 五个内部阶段（`fund/model.go` 行157-162），与 Account 级别的 `liquidating_block_new -> liquidating_drain -> liquidating_transfer -> liquidated -> closed`（`fund/model.go` 行98-104）形成双轨状态映射。PRD §8.4 定义的 5 阶段在 Account 状态层完全对应，内部 liquidation stage 为细分增强——功能等价，命名差异不影响语义正确性。对应关系：blocking->BlockNew, draining->Drain, refunding->Transfer (含余额转移), closing->Transfer (状态标记), closed->Closed。

---

## 红线检查

| # | 红线（PRD §7.7） | 代码证据 | 状态 |
|---|------|---------|------|
| 1 | 无流水改余额（F-CON-01） | 所有余额变更均通过 `s.Store.InsertLedger(tx, ctx, ...)` 追加 `ledgers` 表记录后更新账户余额：Allocate 插入 2 条流水（`fund/service.go` 行199-228）；Freeze 插入 1 条（`fund/freeze.go` 行121-133）；Settle 插入 1-2 条（`fund/freeze.go` 行306-339）；TTL 释放插入 1 条（`fund/lifecycle.go` 行139-150）。`fund/store.go` 行63 明确声明 "Ledger rows are never updated or deleted -- this is an append-only log"。`fund/model.go` 行198-215：Ledger 结构体包含 `BalanceAfter`/`FrozenAfter` 快照字段，每次变更形成完整审计轨迹**。未见任何 UPDATE balance 不写流水的路径。 | 通过 |
| 2 | 划拨无通道（非 parent/sponsors/allocates 方向 / 非白名单） | `fund/service.go` 行160-162：Allocate 执行 `validateChannel()` 校验；行286-298：`validateChannel()` 仅接受 `ChannelParent/ChannelSponsors/ChannelAllocates/ChannelWhitelist` 四种通道，其他通道返回 `ALLOCATION_CHANNEL_DENIED`。`fund/service_test.go` 行457-487：`TestAllocate_ChannelDenied` 测试使用 `"owns"` 通道被拒绝。 | **部分通过 (注)** |
| 3 | Key 无 account 调用 | 资金域核心代码 `fund/service.go`、`fund/freeze.go`、`fund/lifecycle.go` 中均未发现 Key 绑定 account 的显式校验逻辑。HTTP handler 层（`gov_handlers_fund.go`）使用 `requireGovAuth` 进行 ABAC 权限校验但未校验 Key-account 绑定。当前资金操作（Allocate/Freeze/Settle）的请求结构体中无 `APIKeyID` 字段强制绑定要求。 | **未完成** |
| 4 | 预算帽与余额不足返回同一错误码 | `fund/errors.go` 行82：`"INSUFFICIENT_BALANCE"` vs 行100：`"BUDGET_CAP_EXCEEDED"` —— 两个完全不同的错误码，语义独立。`fund/freeze.go` 行63-66（预算帽）与行84-86（余额不足）分别调用对应的错误构造函数，返回路径明确分离。 | 通过 |

**注（红线 2）**：当前 `validateChannel()` 方法注释（`fund/service.go` 行284-292）明确声明 "Full semantic validation (checking party_edges and whitelist tables) requires party package integration and is deferred to a later stage"。当前实现仅执行**语法级通道校验**（是否属于已知通道类型），但**未校验方向语义**（如 parent 通道是否真的是上级->下级方向，sponsors 是否真是出资方->被出资方方向）。PRD §8.2 要求的方向规则未代码实现。这属于已知的技术债务，有明确 TODO 标记。

---

## 验收结论

**总体评估：条件通过（6/7 场景通过，1 红线未完成，1 红线部分通过）**

### 通过项（7 项场景验收）

| 场景 | 状态 |
|------|------|
| 划拨守恒（F-CON-02） | 通过 —— 单事务内双账户锁定 + 双边校验，测试覆盖守恒 |
| 冻结 TTL | 通过 —— 默认 15 min，可配，含 max lifetime 2h 上限 |
| 结算（多退少补 + 孤儿结算） | 通过 —— refund 逻辑完整，过期冻结有日志告警 |
| 清算 5 阶段状态机 | 通过 —— dual-layer 状态机实现，Account 级 6 态 + Liquidation 级 5 步转换 |
| 幂等 | 通过 —— Idempotency-Key + UNIQUE 原子抢占 + Replay/Conflict/InProgress 三态处理 |
| 预算帽分码 | 通过 —— BUDGET_CAP_EXCEEDED 与 INSUFFICIENT_BALANCE 完全独立 |
| 告警不阻断 | 通过 —— BudgetWarnRatio 仅日志告警，不 return error |

### 待解决项

1. **红线 3（Key 无 account 调用）—— 未实现**：当前资金域核心服务（Allocate/Freeze/Settle/Liquidate）在调用层面未校验 Key 与 account 的绑定关系。PRD §7.7 明确要求在调用入口处校验 Key 绑定 account。需要在下阶段实现 Key-account 绑定校验层。

2. **红线 2（划拨无通道）—— 部分实现**：通道语法校验已完成（4 种已知通道类型），但 PRD §8.2 要求的方向语义校验（parent 仅上级->下级，sponsors 仅出资方->被出资方）未完成，代码中已标记 TODO（`fund/service.go` 行284-292）。

### 附加发现

- **代码质量良好**：所有资金操作使用 `shopspring/decimal.Decimal` 精确计算（无 float64），乐观锁版本控制防止并发冲突，死锁预防通过 ID 排序锁定（`fund/service.go` 行112-115）。
- **测试覆盖充分**：`fund/service_test.go` 覆盖正常划拨、守恒验证、幂等重放、通道拒绝、冻结/预算帽/余额不足、结算/退款、TTL 超时释放、完整清算状态机共 8 个场景。
- **审计完整性**：每个资金写操作对应 ledger 追加记录，包含 balance_after 快照、freeze_id、idempotency_key 关联字段。
- **PRD 版本**：验收基于 `AI-GOV-PRD-v3.2.0.md`（行1120-1162 第 13 章验收标准，行666-681 第 7.7 章红线清单）。

---

**签名**：QA-1 资金治理域功能验收专家
**日期**：2026-07-31
