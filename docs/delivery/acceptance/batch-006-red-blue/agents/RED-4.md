# RED-4 单兵作战记录

| 属性 | 值 |
|------|-----|
| Agent ID | RED-4 |
| 审计域 | 模型访问越权（ModelGrant 防御体系完整度） |
| 执行时间 | 2026-07-31 |
| 审计文件 | `modelgrant/checker.go`, `modelgrant/grant.go`, `store_integration.go`, `pipeline.go`, `pipeline_handler.go`, `http.go`, `gov_handlers.go`, `gov_handlers_fund.go`, `schema/ai-gov-fusion-v3.2.sql` |
| 攻击路径 | 6 |
| 严重 | 2 |
| 高危 | 1 |
| 中危 | 3 |

---

## 审计方法

六条攻击路径逐一测试：(1) CheckAccess DENY 优先与级联正确性；(2) Pipeline 中 ModelGrant 步骤执行保障；(3) `/v1/chat/completions` 绕过路径；(4) ModelGrant handler CRUD 的 ABAC 鉴权；(5) quota_limit 双层预算；(6) 直接 SQL 插入绕过。

---

## 发现详情

### V-4.1 (严重) `/v1/chat/completions` 流式请求完全绕过 ModelGrant

三条绕过路径：
1. **stream=true**：`pipeline_handler.go` L64-69 流式请求直接走 `fallbackChatCompletions`，无 ModelGrant 检查
2. **PipelineEnabled=false**：L58 配置关闭即降级
3. **Pipeline 失败降级**：L84-93 任何步骤失败（含 ModelGrant 拒绝）均降级到旧路径，旧路径无 ModelGrant

`fallbackChatCompletions`（L124-209）复制了 `handleChatCompletions`（http.go L705-810），原始处理器不含任何 ModelGrant 检查。

### V-4.2 (严重) 双层预算体系完全失效

- **ModelGrant 级配额**：`CheckQuotaLimit`（checker.go L91-122）已实现但未集成到 Pipeline——无任何步骤调用它
- **Account 级预算帽**：`CheckBudgetCap`（store_integration.go L252-265）为空实现 stub，永远 `return nil`
- **并发竞态**：`ConsumeQuota`（checker.go L133-166）声称乐观锁但 WHERE 子句无 version 检查

### V-4.3 (高危) 级联逻辑退化——仅单层级

`loadGrantsForCascade`（checker.go L171-203）声明级联顺序 Key > Person > Party > 全局，但 `if typ == principal.Type` 守卫（L184）导致每次只加载一个层级。

更严重：`DefaultIntegrator.CheckModelAccess`（store_integration.go L196-215）固定 `principal.Type = "party"`，Key 级和 Person 级规则从未被评估。

### V-4.4 (中危) ModelGrant ABAC scope_party_id 失效

`lookupResourceParty`（gov_handlers.go L319-320）查询 `model_grants.party_id`，但该表（schema L529-541）不含此列。scope 过滤完全失效。

### V-4.5 (中危) 配额并发竞态

`ConsumeQuota` 的 UPDATE 不含 version 检查，并发请求可超限消费。

### V-4.6 (中危) 直接 SQL 插入绕过

`model_grants` 表无 RLS、无触发器、无 `created_by`/`updated_by` 审计列。任何有 DB 直连权限的角色可通过 INSERT 直接授予模型访问。

---

## 关键代码位置

| 文件 | 行号 | 内容 |
|------|------|------|
| `pipeline_handler.go` | 58-69 | stream/PipelineEnabled 降级 |
| `pipeline_handler.go` | 84-93 | Pipeline 失败降级（含被拒后重试绕过） |
| `http.go` | 705-810 | handleChatCompletions 旧路径（无 ModelGrant） |
| `store_integration.go` | 252-265 | CheckBudgetCap 空实现 stub |
| `checker.go` | 91-122 | CheckQuotaLimit 未集成到 Pipeline |
| `checker.go` | 133-166 | ConsumeQuota 无乐观锁 |
| `checker.go` | 171-203 | loadGrantsForCascade 单层级退化 |
| `store_integration.go` | 196-215 | CheckModelAccess 固定 Party 级 |
| `gov_handlers.go` | 319-320 | model_grants 表缺 party_id 列 |
