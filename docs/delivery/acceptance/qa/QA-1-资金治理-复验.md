# QA-1 资金治理 -- batch-002 FIX-A 复验报告

- **复验日期**：2026-07-31
- **复验范围**：batch-002 缺陷收口中的 FIX-A（validateChannel / IdempotencyKey / Settle行锁）
- **结论**：全部通过，三项缺陷均已关闭。

---

## 检查项 1：validateChannel 是否接入 party.CanAllocate()？IdempotencyKey 是否必填？

**文件**：`ai-gov-fusion/backend/internal/server/fund/service.go`

| 子项 | 代码位置 | 结论 | 详细 |
|---|---|---|---|
| validateChannel 调用 party.CanAllocate() | 第 358 行 | **已修复** | 生产路径通过 `s.PartyService.CanAllocate(ctx, srcPartyID, dstPartyID)` 查询 party_edges 表进行边关系语义校验；PartyService 为 nil 时降级为仅校验 channel 名称常量（兼容测试环境）。 |
| IdempotencyKey 必填 | 第 141-143 行 | **已修复** | `allocateValidate` 函数中若 `req.IdempotencyKey == ""` 直接返回 `newIdempotencyKeyRequiredError()`，所有划拨操作必须提供幂等键。 |

---

## 检查项 2：Settle 是否调用 GetFreezeForUpdate？

**文件**：`ai-gov-fusion/backend/internal/server/fund/freeze.go`

| 子项 | 代码位置 | 结论 | 详细 |
|---|---|---|---|
| Settle 使用行锁获取 freeze | 第 228 行 | **已修复** | `Settle` 方法在事务内通过 `s.Store.GetFreezeForUpdate(tx, ctx, req.FreezeID)` 获取冻结记录，注释明确标注为"防止并发 Settle 同一 freeze_id（RED-2 竞态修复）"。 |

---

## 检查项 3：GetFreezeForUpdate 是否有 SELECT FOR UPDATE？

**文件**：`ai-gov-fusion/backend/internal/server/fund/sqlstore/pg.go`

| 子项 | 代码位置 | 结论 | 详细 |
|---|---|---|---|
| GetFreezeForUpdate 使用行锁 | 第 193-204 行 | **已修复** | 通过 GORM 的 `clause.Locking{Strength: "UPDATE"}` 子句生成 `SELECT ... FOR UPDATE` SQL，注释明确标注为"防止并发结算同一 freeze_id 产生竞态窗口（RED-2 安全修复）"。 |

---

## 总结

FIX-A 涉及的三项 RED-2 安全修复全部落实到位：

1. 划拨通道从仅校验名称常量升级为通过 `party.CanAllocate()` 查询 party_edges 表进行边关系语义校验。
2. `IdempotencyKey` 在 Allocate 入口参数校验中强制要求非空，杜绝无幂等键的划拨。
3. Settle 操作在数据库事务内使用 `SELECT FOR UPDATE` 行锁获取 freeze 记录，消除并发结算竞态窗口。

**batch-002 FIX-A 收口通过，无遗留问题。**
