# E2E 前端 + Pipeline 验证报告

> 版本：v1.0
> 日期：2026-07-31
> 范围：前端 8 模块页面完整性、ABAC 动态菜单、Pipeline 接入、前端 API 调用一致性

---

## 任务1：8 模块页面完整性检查

**检查方法**：遍历 `ai-gov-fusion/frontend/app/(console)/gov/` 下每个业务目录，验证是否包含 `page.tsx`、`loading.tsx`、`error.tsx`。

**路径**：`D:/ai-work/grok/a-gov/ai-gov-fusion/frontend/app/(console)/gov/`

| 模块目录 | page.tsx | loading.tsx | error.tsx | 结果 |
|----------|----------|-------------|-----------|------|
| dashboard | 有 | 有 | 有 | PASS |
| parties | 有 | 有 | 有 | PASS |
| fund | 有 | 有 | 有 | PASS |
| pricing | 有 | 有 | 有 | PASS |
| routes | 有 | 有 | 有 | PASS |
| abac | 有 | 有 | 有 | PASS |
| ui-permissions | 有 | 有 | 有 | PASS |
| audit | 有 | 有 | 有 | PASS |

**结论**：全部 8 个模块均包含必需的三文件。PASS。

---

## 任务2：ABAC 动态菜单

**检查文件**：`D:/ai-work/grok/a-gov/ai-gov-fusion/frontend/app/(console)/gov/layout.tsx`

### 2.1 API 调用验证

**通过**。第 75 行确认调用 `GET /gov/ui-permissions/snapshot`：

```typescript
const res = await fetch("/gov/ui-permissions/snapshot");
```

后端路由 `gov_handlers.go:164` 已注册：

```go
mux.HandleFunc("/gov/ui-permissions/snapshot", wrapGovHandler(h.handleUIPermissionSnapshot))
```

前后端路径一致。

### 2.2 菜单过滤逻辑

**通过**。第 100-104 行实现过滤逻辑：

```typescript
const filtered = allNavItems.filter((item) => {
  const v = visibleMap.get(item.code);
  return v !== false; // undefined 或 true 均视为可见
});
```

- `visible === false` 的菜单项被过滤隐藏。
- `visible === true` 或未在投影中出现（`undefined`）的菜单项保留显示。
- 符合安全原则：未配置的菜单默认可见，不会误隐藏。

### 2.3 失败回退

**通过**。三个回退路径均已实现：

1. **HTTP 非 OK 响应**（第 77-81 行）：`res.ok === false` 时，`setVisibleItems(allNavItems)` 显示全部菜单。
2. **后端返回空菜单列表**（第 88-91 行）：`menus.length === 0` 时，`setVisibleItems(allNavItems)` 显示全部菜单。
3. **网络异常**（第 107-109 行）：catch 块中 `setVisibleItems(allNavItems)` 显示全部菜单。

降级策略正确：API 失败时宁可全量显示（用户可进入但受 ABAC 控制），也不锁死导航。

**结论**：三项子检查全部 PASS。

---

## 任务3：Pipeline 接入验证

**检查文件**：
- `D:/ai-work/grok/a-gov/ai-gov-fusion/backend/internal/server/http.go`
- `D:/ai-work/grok/a-gov/ai-gov-fusion/backend/internal/server/pipeline_handler.go`
- `D:/ai-work/grok/a-gov/ai-gov-fusion/backend/internal/server/pipeline.go`

### 3.1 `/v1/chat/completions` 路由

**通过**。`http.go` 第 163 行：

```go
s.mux.HandleFunc("/v1/chat/completions", s.gatewayInFlight(s.pipelineChatHandler))
```

`/v1/chat/completions` 路由到 `pipelineChatHandler`，并通过 `gatewayInFlight` 包装保持并发控制。

### 3.2 `pipelineChatHandler` 调用 `pipeline.Execute()`

**通过**。`pipeline_handler.go` 第 84 行：

```go
result, pipeErr := s.pipeline.Execute(r.Context(), r)
```

`pipelineChatHandler` 完整实现了 9 步执行流程：
1. 校验 HTTP 方法（仅 POST）
2. 密钥鉴权
3. 解析请求体
4. 判断 Pipeline 是否启用（含流式降级）
5. 设置 context（request_id、model_name）
6. 懒初始化 Pipeline
7. 调用 `pipeline.Execute()`
8. 返回 OpenAI 兼容响应
9. 任一步骤失败时降级到 `fallbackChatCompletions`

降级策略正确：
- `PipelineEnabled === false` 或 `pipeline === nil` 时降级（第 58-61 行）
- 流式请求降级（第 64-70 行）
- `pipeline.Execute()` 失败降级（第 85-93 行）
- 上游响应为空降级（第 97-105 行）

### 3.3 Pipeline 14 步骤字段完整性

**通过**。`pipeline.go` 中 `Pipeline` 结构体字段覆盖全部 14 个步骤（步骤 [1] 协议解析由 HTTP handler 完成）：

| 步骤 | 字段名 | 字段类型 | buildPipeline 注入 |
|------|--------|----------|-------------------|
| [2] 密钥鉴权 | `Auth` | `AuthFunc` | `s.pipelineAuthFunc()` |
| [3] 安全钩子 | `SecurityHook` | `SecurityHookFunc` | `s.pipelineSecurityHook()` |
| [4] ModelGrant 检查 | `ModelGrant` | `ModelGrantCheckFunc` | `s.pipelineModelGrant()` |
| [5] 价格预估 | `Pricing` | `PricingFunc` | `s.pipelinePricing()` |
| [6] 价格过滤(δ) | `PriceFilter` | `func` | 未注入（Router 内部处理） |
| [7] 预算帽检查 | `BudgetCheck` | `func` | `s.pipelineBudgetCheck()` |
| [8] 冻结 | `Freeze` | `func` | `s.pipelineFreeze()` |
| [9] 策略路由 | `Router` | `RouteSelectFunc` | `s.pipelineRouter()` |
| [10] 上游调用 | `Adapter` | `UpstreamCallFunc` | `s.pipelineAdapter()` |
| [11] 流式续期 | `StreamRenewal` | `func` | 未注入（HTTP handler 层调用） |
| [12] 用量规范化 | `Normalizer` | `UsageNormalizeFunc` | `s.pipelineNormalizer()` |
| [13] 双轨结算 | `Settlement` | `func` | `nil`（待集成） |
| [14] 审计持久化 | `Audit` | `AuditRecordFunc` | `nil`（待集成） |

**注意**：
- `Settlement`（双轨结算）和 `Audit`（审计持久化）在 `buildPipeline()` 中为 `nil`，标注为"待集成"。
- 这两个字段在 `Pipeline` 结构体中已定义，`Execute()` 中有 nil 检查保护（不会 panic），只是当前未注入实际实现。
- `PriceFilter`（价格过滤）和 `StreamRenewal`（流式续期）虽未在 `buildPipeline` 注入，但结构体字段已定义，且分别在 Router 内部和 HTTP handler 层处理。

**结论**：三项子检查全部 PASS。

---

## 任务4：前端 API 调用一致性抽查

### 4.1 fund/page.tsx

**API_BASE**：`"/gov"`

| 前端调用 | HTTP 方法 | 后端路由 | 匹配 |
|----------|-----------|----------|------|
| `/gov/accounts` | GET | `handleAccounts` | PASS |
| `/gov/accounts/{id}/ledgers` | GET | `handleAccountItem` → `handleGetAccountLedgers` | PASS |
| `/gov/accounts/{id}/allocate` | POST | `handleAccountItem` → `handleAllocate` | PASS |
| `/gov/accounts/{id}/liquidate` | POST | `handleAccountItem` → `handleLiquidate` | PASS |

**结论**：4 个 API 调用全部与后端路由匹配。PASS。

### 4.2 parties/page.tsx

**API_BASE**：`"/gov"`

| 前端调用 | HTTP 方法 | 后端路由 | 匹配 |
|----------|-----------|----------|------|
| `/gov/parties` | GET | `handleParties` | PASS |
| `/gov/parties` | POST | `handleParties` | PASS |
| `/gov/parties/{id}` | DELETE | `handlePartyItem` | PASS |
| `/gov/party-edges` | GET/POST | `handlePartyEdges` | PASS |
| `/gov/party-edges/{id}` | DELETE | `handlePartyEdgeItem` | PASS |
| `/gov/party-members` | GET/POST | `handlePartyMembers` | PASS |
| `/gov/party-members/{id}` | DELETE | `handlePartyMemberItem` | PASS |

**结论**：7 个 API 调用全部与后端路由匹配。PASS。

### 4.3 abac/page.tsx -- 发现问题

**API_BASE**：`"/gov"`

| 前端调用 | HTTP 方法 | 后端路由 | 匹配 |
|----------|-----------|----------|------|
| `/gov/roles` | GET | `handleRoles` | PASS |
| `/gov/roles` | POST | `handleRoles` | PASS |
| `/gov/roles/{id}` | DELETE | `handleRoleItem` | PASS |
| `/gov/policies` | GET | `handlePolicies` | PASS |
| `/gov/policies` | POST | `handlePolicies` | PASS |
| `/gov/policies/{id}` | DELETE | `handlePolicyItem` | PASS |
| `/gov/subject-role-bindings` | GET | `handleSubjectRoleBindings` | PASS |
| `/gov/subject-role-bindings` | POST | `handleSubjectRoleBindings` | PASS |
| `/gov/subject-role-bindings/{id}` | DELETE | `handleSubjectRoleBindingItem` | PASS |
| `/gov/policies/evaluate` | POST | **不匹配** | **FAIL** |

**问题详情**：

前端 `abac/page.tsx` 第 302 行：

```typescript
const res = await fetch(`${API_BASE}/policies/evaluate`, {
  method: "POST",
  ...
  body: JSON.stringify(simForm),
});
```

后端 `gov_handlers_abac.go` 第 361-369 行期望的路由格式为：

```
POST /gov/policies/{id}/evaluate
```

后端通过 `extractItemID(r, "/gov/policies")` 提取路径中资源 ID，然后用 `strings.HasSuffix(policyID, "/evaluate")` 判断是否为评估请求。

当客户端请求 `POST /gov/policies/evaluate` 时，后端会：
1. 提取 `policyID = "evaluate"`（将 `evaluate` 视为策略 ID）
2. `strings.HasSuffix("evaluate", "/evaluate")` 返回 `false`
3. `isEvaluate` 为 false
4. 返回 `405 METHOD_NOT_ALLOWED`：`"策略 POST 仅支持 /evaluate 子路径"`

**影响**：策略模拟评估功能**当前不可用**。前端将 URI 路径 `/gov/policies/evaluate` 视为一个独立端点，但后端将其解析为对 ID 为 `evaluate` 的策略执行 POST 操作。

**建议修复**：
- 方案 A（推荐）：后端新增独立路由 `/gov/policies/evaluate`，不依赖 `{id}/evaluate` 子路径模式。
- 方案 B：前端改为 `POST /gov/policies/{policyId}/evaluate`，并在请求体中传入 `policy_id`。

---

## 汇总

| 任务 | 子检查项数 | 通过 | 失败 | 状态 |
|------|-----------|------|------|------|
| 任务1：8模块页面完整性 | 8 | 8 | 0 | PASS |
| 任务2：ABAC动态菜单 | 3 | 3 | 0 | PASS |
| 任务3：Pipeline接入验证 | 3 | 3 | 0 | PASS |
| 任务4：前端API调用一致性 | 20 | 19 | 1 | **1 FAIL** |
| **合计** | **34** | **33** | **1** | |

**唯一未通过项**：`abac/page.tsx` 的策略模拟评估 POST 请求路径 `/gov/policies/evaluate` 与后端期望的 `/gov/policies/{id}/evaluate` 不匹配，导致策略模拟功能不可用。
