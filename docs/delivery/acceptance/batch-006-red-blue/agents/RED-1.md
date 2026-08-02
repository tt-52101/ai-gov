# RED-1 单兵作战记录

| 属性 | 值 |
|------|-----|
| Agent ID | RED-1 |
| 审计域 | 守恒定理全量审计 (F-CON / D-CON / A-CON / S-CON / AU-CON) |
| 执行时间 | 2026-07-31 |
| 审计文件 | `fund/service.go`, `fund/freeze.go`, `fund/lifecycle.go`, `gov_handlers.go`, `abac/engine.go`, `pipeline.go`, `audit/event.go`, `security/egress.go` |
| 定理总数 | 24 |
| 通过 | 18（去重后有效 18/22） |
| 绕过 | 5（去重 D-CON-03/04 = AU-CON-01/02 后为 4 独立绕过） |
| 通过率 | 79.2% |

---

## 审计方法

逐函数静态审查：识别每个定理对应的代码路径 → 检查是否有条件分支可跳过 → 验证失败模式是 fail-secure 还是 fail-open。

---

## 发现详情

### 已守住（18 定理）

**F-CON 资金守恒（6/6）**：所有资金变更在同一 DB 事务内完成 src_delta + dst_delta = 0。幂等键通过 INSERT ON CONFLICT 原子声明。乐观锁版本号在所有余额更新中携带。

**A-CON ABAC 策略（5/5）**：Deny 短路、默认拒绝、操作注册检查、priority DESC 排序全部正确。ModelGrant DENY 优先在 Pipeline 集成层正确。

**AU-CON 审计（2/3）**：audit_events 仅 INSERT+SELECT，无 UPDATE/DELETE。查询只读。

### 绕过路径（5 条）

1. **D-CON-01 鉴权不可绕过 — 4 个端点丢弃 `_, _`**：`gov_handlers.go` L743 (party_edge DELETE)、L822 (party_member DELETE)、L718-719 (parties GET)、L797-799 (party_members GET)。鉴权函数写入 403 但 handler 继续执行业务逻辑。

2. **D-CON-02 出网管控 — modelName 为空跳过 + 默认最宽松**：`pipeline.go` L267 `if modelName != ""` 条件跳过整个出网检查；L483 默认 `HYBRID_ALLOWED`（最宽松）而非 `INTERNAL_ONLY`（最严格）。

3. **D-CON-07 IDOR — DB 故障时 scope 静默降级**：`lookupResourceParty`（gov_handlers.go L338-344）DB 查询失败返回 `""`，ABAC scope 退化为全局放行（fail-open）。

4. **S-CON-02 失败即停 — 冻结后无补偿**：Pipeline Freeze 成功后，步骤 9-13 失败不回滚冻结，资金锁定 15 分钟至 TTL 过期。

5. **AU-CON-02 快照完整性 — 无代码强制**：`RecordEvent`（audit/event.go L36-58）不校验 `BeforeSnapshot`/`AfterSnapshot`，调用方可写入缺少快照的审计事件。

---

## 关键代码位置

| 文件 | 行号 | 内容 |
|------|------|------|
| `fund/service.go` | L205-206 | 划拨守恒公式 |
| `fund/freeze.go` | L125-126 | 冻结守恒公式 |
| `gov_handlers.go` | L743, L822 | `_, _` 丢弃鉴权返回值 |
| `gov_handlers.go` | L338-344 | DB 故障时 scope 降级 |
| `pipeline.go` | L267 | modelName 为空跳过出网管控 |
| `pipeline.go` | L483 | 默认 HYBRID_ALLOWED |
| `pipeline.go` | L327-336 | 冻结后无补偿回滚 |
| `audit/event.go` | L36-58 | 不校验快照字段 |
