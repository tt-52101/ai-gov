# RED-4：模型访问越权

**日期**：2026-07-31
**风险级别**：严重（Critical）
**结论**：存在 3 个高危漏洞 + 2 个中危缺陷，ModelGrant 整体防护存在多处可绕过路径。

---

## 攻击路径 1：CheckAccess DENY 优先与级联正确性

**文件**：`modelgrant/checker.go`

### DENY 优先逻辑（正确）

第 53-78 行：DENY 在第一遍循环独立评估（53-63 行），命中即拒绝；ALLOW 在第二遍循环评估（66-69 行）。逻辑正确，与文档所述 "DENY 优先于 ALLOW（A-CON-04）" 一致。

### 级联逻辑（存在缺陷）

第 171-203 行 `loadGrantsForCascade`：

```go
// 第 175-178 行：级联层级声明
cascadeLevels := [][2]string{
    {PrincipalKey, principal.ID},
    {PrincipalPerson, principal.ID},
    {PrincipalParty, principal.ID},
}

// 第 184 行：仅加载与主体类型精确匹配的层级
if typ == principal.Type {
```

代码注释声称级联顺序为 "Key > Person > Party > 全局默认"（第 37-38 行），但实际实现中 `if typ == principal.Type` 守卫导致**每次只加载一个层级**（加上全局默认）。例如：Key 主体只会查 Key 级规则 + 全局默认，不会加载 Person 级或 Party 级规则。

**影响**：若 Party 级有 ALLOW 规则，但 Key 级无任何规则，使用该 Key 的请求将被**默认拒绝**（无匹配 ALLOW）。这与预期的级联行为——Key 继承 Person 和 Party 的 ALLOW 但可被更高级的 DENY 覆盖——不符。

### DefaultIntegrator 固定使用 Party 级别（进一步退化）

`store_integration.go` 第 196-215 行 `DefaultIntegrator.CheckModelAccess`：

```go
// 第 201-204 行：始终使用 "party" 类型
principal := modelgrant.Principal{
    Type: "party",
    ID:   call.PartyID,
}
```

这意味着在实际调用路径中，**Key 级和 Person 级的 ModelGrant 规则从未被评估**。即使管理员为特定 Key 或 Person 配置了精细化规则，也不会生效。

**风险评级**：中危。级联未覆盖所有层级，且实际调用路径进一步收窄为仅 Party 级。

---

## 攻击路径 2：Pipeline 中 ModelGrant 步骤执行保障

**文件**：`pipeline.go`、`http.go`、`pipeline_handler.go`

### 条件执行（第 291-298 行）

```go
// 第 291 行：ModelGrant 仅在非 nil 时执行
if p.ModelGrant != nil && result.Auth != nil && modelName != "" {
```

### 静默跳过路径

`ModelGrant` 在以下情况为 nil，导致步骤静默跳过：

| 条件 | 代码位置 | 说明 |
|------|----------|------|
| `s.govDeps.Integrator == nil` | `http.go` 第 341-342 行 | `pipelineModelGrant()` 直接返回 nil |
| `s.govDeps.Pipeline != nil`（全量注入模式）| `http.go` 第 255-257 行 | 跳过 `buildPipeline()`，依赖外部注入的 Pipeline 实例 |
| `modelName == ""` | `pipeline.go` 第 291 行 | 无法从请求中提取模型名称则跳过 |

**风险评级**：高危。ModelGrant 依赖 Integrator 注入；若部署时遗漏该依赖，所有请求将绕过模型授权检查而无任何告警。

---

## 攻击路径 3：`/v1/chat/completions` 绕过 ModelGrant

**文件**：`pipeline_handler.go`、`http.go`

### 绕过路径

`/v1/chat/completions` 路由（`http.go` 第 166 行）始终进入 `pipelineChatHandler`，但存在以下降级路径绕过 ModelGrant：

#### 绕过方式 1：PipelineEnabled=false（第 58 行）

```go
if !s.config.PipelineEnabled || s.pipeline == nil {
    s.fallbackChatCompletions(w, r, project, key, req)
    return
}
```

若配置中 `PipelineEnabled` 为 false，直接走降级路径。

#### 绕过方式 2：流式请求（第 64-69 行）

```go
if req.Stream {
    slog.DebugContext(r.Context(), "Pipeline: 流式请求降级到原有路径", ...)
    s.fallbackChatCompletions(w, r, project, key, req)
    return
}
```

所有流式请求（`stream: true`）绕过 Pipeline 及 ModelGrant 检查。

#### 绕过方式 3：Pipeline 执行失败（第 84-93 行）

```go
if pipeErr != nil {
    s.fallbackChatCompletions(w, r, project, key, req)
    return
}
```

若管线任一步骤失败（包括鉴权失败、安全钩子阻断、ModelGrant 拒绝……），请求降级到旧路径。**这里存在逻辑矛盾**：若 ModelGrant 拒绝导致 `pipeErr`，降级后 `fallbackChatCompletions` **不执行 ModelGrant 检查**，可能放行请求。

#### 降级路径 `fallbackChatCompletions`（第 124-209 行）

该函数复制了原始 `handleChatCompletions`（`http.go` 第 705-810 行）的完整逻辑。原始处理器**不含任何 ModelGrant 检查**——仅做 API Key 鉴权后直接执行路由和上游调用。

### 综合影响

| 攻击向量 | 前提条件 | 影响 |
|----------|----------|------|
| 发送 `stream: true` 请求 | 无 | 完全绕过 ModelGrant |
| ModelGrant 拒绝后重试 | 无（首次被拒后触发降级）| 降级路径放行 |
| 禁用 Pipeline | 配置变更 | 完全绕过 |
| Integrator 未注入 | 部署遗漏 | 完全绕过 |

**风险评级**：严重。ModelGrant 是整个模型访问治理的核心防线，但其对 `/v1/chat/completions` 存在多条完全绕过路径。流式请求（`stream:true`）即可实现零条件绕过。

---

## 攻击路径 4：ModelGrant handler 创建/删除的 ABAC 鉴权

**文件**：`gov_handlers_fund.go`、`gov_handlers.go`

### ABAC 鉴权已有配置

| 端点 | 方法 | ABAC Action | 代码位置 |
|------|------|-------------|----------|
| `/v1/gov/model-grants` | POST | `routing.model_grant.write` | 第 994 行 |
| `/v1/gov/model-grants` | GET | `routing.model_grant.read` | 第 1021 行 |
| `/v1/gov/model-grants/{id}` | GET | `routing.model_grant.read` | 第 1048 行 |
| `/v1/gov/model-grants/{id}` | DELETE | `routing.model_grant.write` | 第 1061 行 |

ABAC 鉴权存在，但 `requireGovItemAuth` 中的 `lookupResourceParty` 对 `model_grant` 类型存在缺陷。

### lookupResourceParty 缺陷（gov_handlers.go 第 319-320 行）

```go
case "model_grant":
    mapping = &partyQuery{table: "model_grants", idColumn: "id", col: "party_id"}
```

该函数查询 `model_grants` 表中的 `party_id` 列以获取资源归属信息，用于 ABAC 的 `scope_party_id` 角色绑定过滤。

但 `model_grants` 表（SQL schema `ai-gov-fusion-v3.2.sql` 第 529-541 行）**不包含 `party_id` 列**：

```sql
CREATE TABLE model_grants (
    id              TEXT PRIMARY KEY,
    principal_type  TEXT NOT NULL,
    principal_id    TEXT NOT NULL,
    model_id        TEXT,
    model_tag       TEXT,
    effect          TEXT NOT NULL,
    priority        INTEGER DEFAULT 0,
    quota_limit     INTEGER,
    conditions      JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

查询不存在的列将导致 `lookupResourceParty` 返回空字符串。ABAC 引擎将 `PartyID` 为空等同为 "不做 scope 过滤"（`engine.go` 第 61-63 行注释），这意味着**基于 scope_party_id 的角色绑定对 ModelGrant 资源完全失效**。一个绑定到某个 Party 的角色本应只对所属 Party 的 ModelGrant 生效，实际上将对所有 ModelGrant 生效。

**风险评级**：中危。ABAC 的 scope_party_id 隔离机制在 ModelGrant 资源上失效，可能导致跨组织越权访问（IDOR）。

---

## 攻击路径 5：quota_limit 双层预算绕过

**文件**：`modelgrant/checker.go`、`store_integration.go`

### 第一层：ModelGrant 级配额（部分失效）

`CheckQuotaLimit`（`checker.go` 第 91-122 行）委托给 `findQuotaGrant`（第 207-219 行）：

```go
// 第 209-210 行：仅精确匹配 principal_type + principal_id + model_id
err := c.DB.Where("principal_type = ? AND principal_id = ? AND model_id = ?",
    principal.Type, principal.ID, modelID).First(&mg).Error
```

该查询存在以下问题：

1. **仅匹配 `model_id`**：若授权规则使用 `model_tag` 而非 `model_id`，或为全局默认规则（`model_id` 和 `model_tag` 均为空），`findQuotaGrant` 无法定位配额配置，直接返回 `ErrNoModelGrantFound`（第 213-214 行），**配额不生效**。

2. **`CheckQuotaLimit` 未在 Pipeline 中集成**：Pipeline 第 4 步 `ModelGrant` 调用 `CheckAccess`，第 7 步 `BudgetCheck` 调用 `CheckBudgetCap`（账户级），但**没有步骤调用 `CheckQuotaLimit`**（ModelGrant 级配额）。`CheckQuotaLimit` 仅存在于 `checker.go` 中作为独立方法，未接入任何调用路径。

3. **`ConsumeQuota` 无乐观锁**（`checker.go` 第 133-166 行）：

```go
// 第 149-152 行：注释声称"使用乐观锁确保并发安全"
// 但实际使用简单 UpdateColumn，无 version 检查
result := c.DB.Model(&ModelGrant{}).
    Where("id = ?", mg.ID).
    UpdateColumn("quota_consumed", newConsumed)
```

并发请求可能同时通过 `CheckQuotaLimit` 检查（都看到旧值），然后各自写入消耗，导致实际消费超过配额上限。

### 第二层：Account 级预算帽（未实现）

`DefaultIntegrator.CheckBudgetCap`（`store_integration.go` 第 252-265 行）：

```go
func (d *DefaultIntegrator) CheckBudgetCap(ctx context.Context, _ *gorm.DB, call *StartCallContext, cost *EstimatedCallCost) error {
    if d.FundStore == nil {
        return nil
    }
    // 账户解析与预算帽检查——通过 fund 包的服务层执行。
    // 注意：此处需要在事务内完成，具体实现由 fund.Service.CheckBudgetCap 提供。
    _ = call
    _ = cost
    slog.DebugContext(ctx, "预算帽检查——待 fund.Service 集成", ...)
    return nil  // ← 硬编码返回 nil，完全不检查
}
```

该方法为空实现（stub），永远返回 nil。Account 级预算帽实际**未上线**。

### 综合预算绕过路径

| 层级 | 状态 | 绕过方式 |
|------|------|----------|
| ModelGrant 级配额 | 方法存在但未集成到 Pipeline | 所有请求绕过 |
| Account 级预算帽 | 空实现（stub） | 所有请求绕过 |
| 配额并发竞态 | 无乐观锁 | 并发请求超限消费 |

**风险评级**：严重。双层预算体系完全失效——第一层未集成，第二层未实现。任何已鉴权的请求可以无限制消费。

---

## 攻击路径 6：直接 SQL 插入 `model_grants` 表绕过 API

**文件**：`modelgrant/grant.go`、`schema/ai-gov-fusion-v3.2.sql`

### API 层防护

创建 ModelGrant 的唯一途径是通过 `/v1/gov/model-grants` 的 POST handler（`gov_handlers_fund.go` 第 993-1019 行），该 handler 要求 ABAC 鉴权（`routing.model_grant.write`）。

### 数据库层防护（缺失）

`model_grants` 表（SQL 第 529-541 行）**没有**：
- 行级安全策略（RLS）
- 数据库级触发器验证
- 数据库用户权限分离
- 审计列（`created_by`、`updated_by`）

任何拥有数据库直连权限的角色（DBA、运维）可通过以下 SQL 直接授予自己模型访问权限：

```sql
INSERT INTO model_grants (id, principal_type, principal_id, model_id, effect)
VALUES ('mg-bypass', 'party', 'target_party_id', NULL, 'allow');
```

其中 `model_id = NULL` 表示**全局 ALLOW**（`matchGrant` 在 `grant.go` 第 104-105 行：`model_id` 和 `model_tag` 均为空时返回 true）。

### 攻击场景

1. 内部人员（有 DB 访问权限但无 ABAC 授权）直接 SQL 插入
2. SQL 注入漏洞（若存在）利用 `model_grants` 表
3. 备份恢复过程中引入恶意记录
4. CI/CD 迁移脚本被篡改

**风险评级**：中危。API 层有 ABAC 但 DB 层无任何防护，依赖运维安全而非纵深防御。

---

## 汇总与修复建议

### 漏洞总结

| 编号 | 攻击路径 | 风险 | 核心问题 |
|------|----------|------|----------|
| V-4.1 | `/v1/chat/completions` 绕过 | 严重 | 流式请求、Pipeline 失败降级、配置关闭均可绕过 ModelGrant |
| V-4.2 | 双层预算全线失效 | 严重 | ModelGrant 配额未集成到 Pipeline；Account 预算帽为 stub |
| V-4.3 | 级联逻辑缺陷 | 高危 | `loadGrantsForCascade` 只加载单层级；实际调用仅用 Party 级 |
| V-4.4 | ABAC scope_party_id 对 ModelGrant 失效 | 中危 | `model_grants` 表缺少 `party_id` 列 |
| V-4.5 | 配额并发竞态 | 中危 | `ConsumeQuota` 声称乐观锁但未实现 version 检查 |
| V-4.6 | 直接 SQL 插入绕过 | 中危 | DB 层无 RLS、无触发器、无审计列 |

### 修复优先级

1. **P0**：修复 V-4.1（流式绕过）。在 `fallbackChatCompletions` 中插入 ModelGrant 检查，或移除降级逻辑改为直接返回 403。

2. **P0**：修复 V-4.2（预算失效）。将 `CheckQuotaLimit` 集成到 Pipeline 第 4 步之后、第 7 步之前；实现 `CheckBudgetCap` 方法。

3. **P1**：修复 V-4.3（级联）。移除 `loadGrantsForCascade` 中的 `if typ == principal.Type` 守卫，或改为收集所有 `principal.ID` 匹配且 `principal.Type` 在级联列表中的规则。

4. **P1**：修复 V-4.4（ABAC scope）。在 `model_grants` 表中添加 `party_id` 列并用触发器或应用层自动填充。

5. **P2**：修复 V-4.5（并发竞态）。在 `ConsumeQuota` 的 WHERE 子句中添加 `version` 字段检查。

6. **P2**：修复 V-4.6（DB 防护）。添加 `created_by`/`updated_by` 审计列，考虑数据库级 RLS。

### 关键代码位置速查

| 文件 | 行号 | 内容 |
|------|------|------|
| `modelgrant/checker.go` | 53-78 | DENY 优先逻辑（正确） |
| `modelgrant/checker.go` | 171-203 | `loadGrantsForCascade`（级联缺陷） |
| `modelgrant/checker.go` | 133-166 | `ConsumeQuota`（无乐观锁） |
| `modelgrant/checker.go` | 207-219 | `findQuotaGrant`（仅匹配 model_id） |
| `modelgrant/grant.go` | 89-106 | `matchGrant`（model_id/model_tag 为空即全局） |
| `pipeline.go` | 291-298 | ModelGrant 条件执行 |
| `pipeline_handler.go` | 58-69 | 降级条件：PipelineEnabled/流式 |
| `pipeline_handler.go` | 84-93 | 降级条件：Pipeline 失败 |
| `http.go` | 166 | `/v1/chat/completions` 路由注册 |
| `http.go` | 340-348 | `pipelineModelGrant`（Integrator nil 即跳过） |
| `http.go` | 705-810 | `handleChatCompletions`（旧路径，无 ModelGrant） |
| `store_integration.go` | 196-215 | `CheckModelAccess`（固定 Party 级） |
| `store_integration.go` | 252-265 | `CheckBudgetCap`（空实现 stub） |
| `gov_handlers.go` | 297-347 | `lookupResourceParty`（model_grant 缺少 party_id 列） |
| `gov_handlers_fund.go` | 990-1075 | ModelGrant CRUD handlers（ABAC 存在但 scope 失效） |
| `schema/ai-gov-fusion-v3.2.sql` | 529-541 | `model_grants` 表定义（无 party_id、无 version） |
