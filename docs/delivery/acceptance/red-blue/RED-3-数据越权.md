# RED-3：数据越权攻击面（IDOR + 水平越权）

## 概述

**评估日期**：2026-07-31
**评估范围**：治理 API 全部单品端点（`/v1/gov/*/{id}`）的归属校验与 ABAC scope_party_id 机制
**代码基线**：`ai-gov-fusion/backend/internal/server/`

---
## 1. 攻击面总览

| # | 端点 | 方法 | resourceType | lookupResourceParty 映射 | requireGovItemAuth 返回值检查 | 风险等级 |
|---|------|------|-------------|--------------------------|---------------------------|---------|
| 1 | `/v1/gov/accounts/{id}` | GET | account | fund_accounts.party_id | **不检查** (`_, _`) | **严重** |
| 2 | `/v1/gov/parties/{id}` | DELETE | party | parties.id (self) | 正确检查 (`_, ok`) | 低 |
| 3 | `/v1/gov/keys/{id}` | DELETE | key | api_keys.party_id | **不检查** (`_, _`) | **严重** |
| 4 | `/v1/gov/allocations/{id}` | GET | allocation | fund_allocations.party_id | **不检查** (`_, _`) | **严重** |
| 5 | `/v1/gov/model-grants/{id}` | DELETE | model_grant | model_grants.party_id | **不检查** (`_, _`) | **严重** |
| 6 | `/v1/gov/route-profiles/{id}` | DELETE | route_profile | route_profiles.party_id | **不检查** (`_, _`) | **严重** |
| 7 | 所有使用 `requireGovItemAuth` 的端点 | — | — | `party_edge`/`party_member` 缺失映射 | — | 高 |

---
## 2. 逐端点攻击路径分析

### 2.1 handleAccountItem -- GET /v1/gov/accounts/{id}

**文件**：`gov_handlers_fund.go`

**代码路径**：

```
行 145-171: handleAccountItem → 行 148-150 → handleGetAccount
行 176-194: handleGetAccount
```

**行 177 调用**：
```go
_, _ = h.requireGovItemAuth(w, r, "fund.balance.read", "account", accountID)
```

**问题**：返回值被 `_, _` 丢弃。`requireGovItemAuth` 鉴权失败时已在 ResponseWriter 中写入了 403 错误响应，但 `handleGetAccount` **不检查返回值，继续执行**：

```
行 179-193: 继续查询 DB，执行 okJSON(w, acct) 写入 200 + 敏感账户数据
```

**攻击效果**：攻击者携带任意有效 Token（但无 `fund.balance.read` 权限，或 scope_party_id 不匹配），请求后可收到 HTTP 403 状态码，但响应体中**拼接包含被拒绝资源的完整账户数据**（JSON 拼接：403 错误 JSON + 200 账户详情 JSON）。

**资源归属校验路径**：`lookupResourceParty(DB, ctx, "account", accountID)`（行 269）→ 查询 `fund_accounts.party_id`（行 313-314）。party_id 被传入 `abac.Engine.Evaluate` 作为 `Resource.PartyID`（行 274）。ABAC 引擎在 `resolveSubjectRoles`（engine.go 行 208-236）按 scope_party_id 过滤角色。**前提是 ABAC 引擎已启用**（行 267 `if h.deps.ABACEngine != nil`）。若 ABAC 引擎未配置（开发模式），则跳过鉴权直接放行。

### 2.2 handlePartyItem -- DELETE /v1/gov/parties/{id}

**文件**：`gov_handlers.go`

**代码路径**：

```
行 658-677: case http.MethodDelete
行 660-663:
  _, ok := h.requireGovItemAuth(w, r, "iam.party.write", "party", partyIDStr)
  if !ok { return }
```

**返回值检查**：**正确**。使用 `_, ok` 模式并在 ok 为 false 时 return。

**资源归属校验**：`lookupResourceParty` 对 "party" 类型查询 `parties.id`（行 312），即 party 自身的 ID 即为 party_id。ABAC 引擎通过 scope_party_id 过滤角色绑定。Party 的写操作需要 `iam.party.write` 权限，该权限仅在有匹配 scope_party_id 的角色绑定时生效。

**结论**：权限校验正确，无 IDOR 风险。但请注意，`extractItemID(r, "/gov/parties")` 的行 581 使用的 URL 前缀与路由注册的 `/v1/gov/parties/` 前缀**不匹配**（详见第 4 节）。

### 2.3 handleKeyItem -- DELETE /v1/gov/keys/{id}

**文件**：`gov_handlers_fund.go`

**代码路径**：

```
行 675-689: handleKeyItem
行 681-683:
  case http.MethodDelete:
    _, _ = h.requireGovItemAuth(w, r, "iam.key.delete", "key", keyID)
    okJSON(w, map[string]any{"deleted": true, "id": keyID})
```

**问题 1**：返回值被 `_, _` 丢弃。鉴权失败后仍执行 `okJSON` 写入 200。

**问题 2**：DELETE 操作当前为占位实现（行 683 直接返回 `okJSON`），未真正执行删除逻辑。但若后续补全删除逻辑（DB delete），泄露风险升级为实际数据破坏。

**资源归属校验路径**：`lookupResourceParty` 对 "key" 类型查询 `api_keys.party_id`（行 315-316）。party_id 传入 ABAC 做 scope 过滤。Key 的 party_id 在创建时通过 `handleCreateKey`（行 767）设置。

### 2.4 handleAllocationItem -- GET /v1/gov/allocations/{id}

**文件**：`gov_handlers_fund.go`

**代码路径**：

```
行 622-643: handleAllocationItem
行 624: _, _ = h.requireGovItemAuth(w, r, "fund.ledger.read", "allocation", allocID)
行 633-639: 查询 DB 并返回分配记录
```

**问题**：返回值被 `_, _` 丢弃。鉴权失败后仍执行 DB 查询并返回数据。

**资源归属校验**：`lookupResourceParty` 对 "allocation" 类型查询 `fund_allocations.party_id`（行 317-318）。**注意**：此校验仅检查分配记录自身的 party_id 列，**不验证**分配的 `src_account_id` 和 `dst_account_id` 是否属于同一 party。若数据层面存在不一致（分配记录的 party_id 指向 A，但其 src/dst account 属于 B），攻击者可通过读取本 party 的分配间接获取其他 party 的账户引用信息。

**附加攻击面**：若 FundService.Store 未对 allocation 按 party_id 过滤，存在跨 party 数据泄露。

### 2.5 handleModelGrantItem -- DELETE /v1/gov/model-grants/{id}

**文件**：`gov_handlers_fund.go`

**代码路径**：

```
行 1043-1075: handleModelGrantItem
行 1060-1072: case http.MethodDelete
行 1061: _, _ = h.requireGovItemAuth(w, r, "routing.model_grant.write", "model_grant", grantID)
行 1067: modelgrant.DeleteModelGrant(db, grantID) -- 实际执行删除
```

**问题**：返回值被 `_, _` 丢弃。**这是最危险的端点之一**——DELETE 操作有**实际副作用**（行 1067 调用 `DeleteModelGrant` 从 DB 删除记录）。鉴权失败（403 已写入）后，仍会执行物理删除。

**资源归属校验路径**：`lookupResourceParty` 对 "model_grant" 类型查询 `model_grants.party_id`（行 319-320）。若 party_id 查询因 DB 故障返回空字符串（行 338-346），scope 过滤完全失效。

### 2.6 handleRouteProfileItem -- DELETE /v1/gov/route-profiles/{id}

**文件**：`gov_handlers_fund.go`

**代码路径**：

```
行 1135-1200: handleRouteProfileItem
行 1185-1197: case http.MethodDelete
行 1186: _, _ = h.requireGovItemAuth(w, r, "routing.route_profile.write", "route_profile", profileIDStr)
行 1192: routing.DeleteProfile(db, profileID) -- 实际执行删除
```

**问题**：同样丢弃返回值，鉴权失败后仍执行 DELETE。此端点与 model_grant 一样有实际副作用。

**附加发现**：GET（行 1149）和 PUT（行 1162）同样丢弃返回值，存在数据泄露（GET）和数据篡改（PUT）风险。

### 2.7 lookupResourceParty 资源类型覆盖完整性

**文件**：`gov_handlers.go` 行 297-347

**当前已映射的资源类型**：

| resourceType | 表名 | party_id 列 | 行号 |
|-------------|------|------------|------|
| party | parties | id | 312 |
| account | fund_accounts | party_id | 313-314 |
| key | api_keys | party_id | 315-316 |
| allocation | fund_allocations | party_id | 317-318 |
| model_grant | model_grants | party_id | 319-320 |
| route_profile | route_profiles | party_id | 321-322 |
| model_price | — | —（返回 ""） | 323 |
| role | — | —（返回 ""） | 323 |
| policy | — | —（返回 ""） | 323 |
| subject_role_binding | — | —（返回 ""） | 323 |

**缺失映射的资源类型（落入 default 返回 ""）**：

| resourceType | 调用位置 | 风险 |
|-------------|---------|------|
| `party_edge` | gov_handlers.go:743 | 高 -- party_edge 有明确的 party 归属，应查询 party_edges 表并关联 party_id |
| `party_member` | gov_handlers.go:822 | 高 -- party_member 有明确的 party 归属，应查询 party_members 表的 party_id |
| `ui_menu` | gov_handlers_abac.go:792, 805, 823 | 低 -- UI 菜单为系统级资源 |
| `ui_route` | gov_handlers_abac.go:891, 904, 922 | 低 -- UI 路由为系统级资源 |
| `ui_action_binding` | gov_handlers_abac.go:990, 1003, 1021 | 低 -- UI 按钮绑定为系统级资源 |
| `audit_event` | gov_handlers_abac.go:1139 | 低 -- 审计事件为系统级资源 |
| `request_log` | gov_handlers_abac.go:1169 | 低 -- 请求日志为系统级资源 |
| `model_route` | gov_handlers_fund.go:1225, 1228 | 中 -- 待定（占位实现） |
| `grant` | gov_handlers_abac.go:716 | 低 -- 授权记录为系统级资源 |

**关键缺失**：`party_edge` 和 `party_member` 两种类型属于**有明确 party 归属**的资源，但 `lookupResourceParty` 未映射。这意味着：

1. `handlePartyEdgeItem`（gov_handlers.go 行 743）调用 `requireGovItemAuth` 时，`lookupResourceParty` 对 `"party_edge"` 返回 `""`。
2. 返回的 `""` 作为 `Resource.PartyID` 传入 ABAC（行 274-275）。
3. ABAC 的 `resolveSubjectRoles` 对空 PartyID 不做 scope 过滤（engine.go 行 225-228：`if scopePartyID != nil && *scopePartyID != ""`）。
4. **结果**：任何拥有全局 `iam.party.write` 角色的用户均可删除任意 party edge，无论其 scope_party_id 绑定为何。

---
## 3. 根源分析：requireGovItemAuth 返回值模式不一致

**统计**：

- **正确检查返回值**（`gctx, _ := ...` + `if gctx == nil { return }`）：6 处
- **不检查返回值**（`_, _ = ...`）：**27 处**

**正确模式**（仅以下端点）：

| 端点 | 文件 | 行号 |
|------|------|------|
| handlePartyItem GET | gov_handlers.go | 594-596 |
| handlePartyItem PATCH | gov_handlers.go | 617-619 |
| handlePartyItem DELETE | gov_handlers.go | 660-663 |
| handleAllocate | gov_handlers_fund.go | 277-278 |
| handleLiquidate | gov_handlers_fund.go | 375-378 |
| handleUpdateBudget | gov_handlers_fund.go | 462-465 |
| handleRoleItem PUT | gov_handlers_abac.go | 218-219 |
| handlePolicyItem PUT | gov_handlers_abac.go | 387-389 |

**错误模式**（返回值丢弃，共 27 处，以下为高影响端点）：

| 端点 | 文件 | 行号 | 操作 |
|------|------|------|------|
| handleGetAccount | gov_handlers_fund.go | 177 | GET 读取敏感数据 |
| handleGetAccountLedgers | gov_handlers_fund.go | 204 | GET 读取流水 |
| handleAllocationItem GET | gov_handlers_fund.go | 624 | GET 读取 |
| handleKeyItem GET | gov_handlers_fund.go | 679 | GET 读取 |
| handleKeyItem DELETE | gov_handlers_fund.go | 682 | DELETE 占位 |
| handleModelGrantItem GET | gov_handlers_fund.go | 1048 | GET 读取 |
| handleModelGrantItem DELETE | gov_handlers_fund.go | 1061 | DELETE 实际删除 |
| handleRouteProfileItem GET | gov_handlers_fund.go | 1149 | GET 读取 |
| handleRouteProfileItem PUT | gov_handlers_fund.go | 1162 | PUT 更新 |
| handleRouteProfileItem DELETE | gov_handlers_fund.go | 1186 | DELETE 实际删除 |

**机制解释**：

`requireGovItemAuth` 的实现（gov_handlers.go 行 243-282）中，当鉴权失败时：
1. 调用 `writeError(w, r, ...)` -- 向 ResponseWriter 写入 403 状态码和错误 JSON
2. 返回 `(nil, false)`

当调用方使用 `_, _` 丢弃返回值时：
- `WriteHeader(403)` 已执行，HTTP 状态码设为 403
- 但处理函数继续执行，最终调用 `okJSON` 调用 `writeJSON` → `WriteHeader(200)`（**被忽略**，因为 Header 已写入）
- `json.NewEncoder(w).Encode(data)` 仍会将数据 **追加写入** 响应体

结果：客户端收到 HTTP 403 + 拼接的 JSON（错误消息 + 敏感数据），虽然 HTTP 状态码正确，但**响应体中包含本应拒绝访问的数据**。

---
## 4. 路径前缀不匹配风险

**发现**：路由注册使用 `/v1/gov/*` 前缀，但 `extractItemID` 使用 `/gov/*` 前缀。

| 路由注册（gov_handlers.go） | extractItemID 前缀 | 行号 |
|--------------------------|-------------------|------|
| `/v1/gov/parties/` (行 115) | `/gov/parties` | 581 |
| `/v1/gov/party-edges/` (行 117) | `/gov/party-edges` | 732 |
| `/v1/gov/party-members/` (行 119) | `/gov/party-members` | 811 |
| `/v1/gov/accounts/` (行 123) | `/gov/accounts/` (extractAccountAction 行 110) | 145 |
| `/v1/gov/allocations/` (行 125) | `/gov/allocations` | 623 |
| `/v1/gov/keys/` (行 129) | `/gov/keys` | 676 |
| `/v1/gov/model-prices/` (行 133) | `/gov/model-prices` | 951 |
| `/v1/gov/model-grants/` (行 137) | `/gov/model-grants` | 1044 |
| `/v1/gov/route-profiles/` (行 141) | `/gov/route-profiles` | 1136 |
| `/v1/gov/model-routes/` (行 144) | `/gov/model-routes` | 1222 |
| `/v1/gov/roles/` (行 150) | `/gov/roles` | 189 |
| `/v1/gov/policies/` (行 152) | `/gov/policies` | 364 |
| `/v1/gov/subject-role-bindings/` (行 154) | `/gov/subject-role-bindings` | 603 |
| `/v1/gov/grants/` (行 156) | `/gov/grants` | 713 |

**影响**：若请求未经过反向代理剥离 `/v1` 前缀，`extractItemID` 将无法正确提取资源 ID。例如：

- URL: `/v1/gov/parties/123`
- `extractItemID(r, "/gov/parties")` → `path[13:]` = `"es/123"`（错误，应返回 `"123"`）
- 后续 `strconv.ParseInt("es/123", 10, 64)` 报错 → `"party_id 格式无效"`

同样，`extractAccountAction` 的 `strings.HasPrefix(path, "/gov/accounts/")` 对 `/v1/gov/accounts/acc_123` 返回 false → 返回 `("", "")` → accountID 为空。

**说明**：若存在反向代理（nginx）做 `StripPrefix("/v1")`，此问题不成立。但代码本身缺乏防御性，建议统一前缀或从路由注册提取常量。

---
## 5. ABAC 架构层分析

### 5.1 scope_party_id 过滤机制

ABAC `Engine.Evaluate`（engine.go 行 52-135）的 IDOR 防护依赖于**两层**：

**第 1 层 -- 角色过滤**（engine.go 行 60-67）：
```go
var scopePartyID *string
if resource.PartyID != "" {
    scopePartyID = &resource.PartyID
}
roleIDs, err := e.resolveSubjectRoles(ctx, subject, scopePartyID)
```

`resolveSubjectRoles`（行 208-236）仅保留 scope_party_id 为空（全局角色）或匹配资源 party_id 的角色绑定。

**第 2 层 -- 策略条件匹配**（engine.go 行 312-346）：
`matchPolicyConditions` **不检查 PartyID**。它只匹配 `axis`、`actions`、`resource_type` 三个维度。

### 5.2 保护缺口

1. **全局角色绕过**：若用户被授予全局角色（scope_party_id=NULL）且该角色拥有 `fund.balance.read` 权限，则该用户可读取**所有账户**，无 party 限制。这是一个**有意设计**（超级管理员行为），但需在文档中明确标注。

2. **lookupResourceParty 失败回退**：当 DB 查询失败或资源不存在时，返回 `""`（行 338-346）。空 party_id 导致 scope 过滤**完全失效**（engine.go 行 225：`if scopePartyID != nil && *scopePartyID != ""`）。若攻击者能触发 DB 查询失败（如发送超长 ID），可绕过 scope 过滤。

3. **ABAC 引擎未配置**：`requireGovItemAuth` 行 267：`if h.deps.ABACEngine != nil && action != ""` -- 若 ABACEngine 为 nil（开发模式），鉴权完全跳过，任何认证用户可访问任意资源。

### 5.3 requireGovAuth（列表端点）的 Resource.ID

`requireGovAuth`（gov_handlers.go 行 199-232）设置 `resource := abac.Resource{Type: "gov_api", ID: r.URL.Path}`（行 226）。这意味着 Resource.ID 是 URL 路径（如 `/v1/gov/parties`），而非实际资源 ID。这导致 ABAC 策略的 resource_type 条件对列表端点使用 "gov_api" 而非实际资源类型，**与单品端点不一致**。

---
## 6. 修复建议

### 6.1 紧急修复（严重）-- 修复所有 requireGovItemAuth 返回值检查

在所有使用 `_, _ = h.requireGovItemAuth(...)` 的调用点，改为：

```go
gctx, ok := h.requireGovItemAuth(w, r, "<action>", "<resourceType>", resourceID)
if !ok {
    return
}
```

或在仅需鉴权不需 gctx 的场景：

```go
if _, ok := h.requireGovItemAuth(w, r, "<action>", "<resourceType>", resourceID); !ok {
    return
}
```

**影响文件与行号**：

| 文件 | 需修复行号 |
|------|-----------|
| gov_handlers_fund.go | 177, 204, 624, 679, 682, 685, 958, 971, 1048, 1061, 1149, 1162, 1186, 1225, 1228 |
| gov_handlers_abac.go | 193, 276, 374, 429, 447, 606, 716, 792, 805, 823, 891, 904, 922, 990, 1003, 1021, 1139, 1169 |
| gov_handlers.go | 743, 822 |

### 6.2 高优先级 -- 补全 lookupResourceParty 映射

添加以下资源类型映射：

```go
case "party_edge":
    mapping = &partyQuery{table: "party_edges", idColumn: "id", col: "party_id"}
case "party_member":
    mapping = &partyQuery{table: "party_members", idColumn: "id", col: "party_id"}
```

**文件**：gov_handlers.go 行 309-329，在 `case "route_profile"` 之后添加。

### 6.3 中优先级 -- 统一 URL 前缀

抽取常量定义路由前缀，在路由注册和 `extractItemID` 调用之间保持一致：

```go
const govRoutePrefix = "/v1/gov"
```

或确认反向代理剥离 `/v1` 的行为已文档化并经过测试。

### 6.4 中优先级 -- 加强 lookupResourceParty 失败处理

当 DB 查询失败时（gov_handlers.go 行 338-345），不应静默返回 `""`。建议返回一个特殊的哨兵值或记录审计事件，避免攻击者通过触发 DB 错误绕过 scope 过滤。

### 6.5 增强建议 -- 显式归属校验

在 `requireGovItemAuth` 内（或 handler 层）增加显式的归属校验：除了依赖 ABAC 的 scope_party_id 角色过滤外，直接检查 `resource.party_id` 是否在主体的允许 scope 列表中。这将提供**纵深防御**，避免全局角色滥用。

---
## 7. 验证命令

```bash
# 验证：无权限用户尝试读取其他组织账户
curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer <other_party_user_token>" \
  http://localhost:8080/v1/gov/accounts/<target_party_account_id>

# 验证：无权限用户尝试删除其他组织 ModelGrant
curl -s -X DELETE -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer <other_party_user_token>" \
  http://localhost:8080/v1/gov/model-grants/<target_party_grant_id>

# 验证：party_edge scope_party_id 过滤是否生效
curl -s -X DELETE -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer <scoped_user_token>" \
  http://localhost:8080/v1/gov/party-edges/<other_party_edge_id>
```

**期望结果**：全部返回 403。
**当前预期结果**（修复前）：返回值检查缺失的端点可能返回 200 或拼接数据（状态码 403 + 泄露数据）。

---
## 8. 影响评级

| 维度 | 评级 | 说明 |
|------|------|------|
| 可利用性 | **高** | 仅需有效 API Key，无需高级技巧 |
| 影响范围 | **高** | 涉及所有治理域（Fund/Key/Routing/Party） |
| 数据敏感性 | **高** | 账户余额、流水、密钥、授权、路由档案均为敏感治理数据 |
| 持久性 | **中** | 部分为只读泄露，部分为实际写/删操作 |
| 总体 | **严重** | P0 漏洞，需立即修复 |

**修复工作量**：约 30 处代码修改，分布在 3 个文件。预计 1-2 小时完成修改 + 回归测试。
