# E2E-3 API 契约验证报告 —— 全矩阵逐端点比对

> **基线:** `docs/spec/api-spec-v3.2.md` (75 端点规范)  
> **后端:** `ai-gov-fusion/backend/internal/server/gov_handlers.go` + `gov_handlers_fund.go` + `gov_handlers_abac.go`  
> **前端:** `ai-gov-fusion/frontend/app/(console)/gov/*/page.tsx` (8 个页面)  
> **错误码文件:** `frontend/lib/error-codes.ts`  
> **验证日期:** 2026-07-31

---

## 1. 总览

| 指标 | 数值 |
|---|---|
| 规范定义端点总数 | 75 |
| 后端已注册路由 | 48 个 `HandleFunc` 覆盖 ~75 端点（集合+单品复用） |
| 真实实现（非占位） | 57 端点 |
| 占位/待实现（返回固定消息） | 12 端点 |
| 规范有但后端缺失 | 4 端点 |
| 后端有但规范未定义 | 2 端点 (reconciliation-runs) |
| 前端调用端点 | 33 个独立 fetch 调用 |
| 前端-后端 URL 不一致 | 3 处 |
| 前端-后端 请求体字段不一致 | 4 处 |

---

## 2. 端点对齐矩阵

### 图例

| 符号 | 含义 |
|---|---|
| PASS | 路由+方法+实现均已对齐 |
| STUB | 路由已注册但返回 "待实现" 消息 |
| MISSING | 规范定义但后端未注册 |
| EXTRA | 后端已注册但规范未定义 |
| MISMATCH | 方法或路径不一致 |

---

### 2.1 Party (主体管理) —— 10 端点

| # | 规范端点 | 方法 | 后端路由 | 后端方法 | ABAC 规范 | ABAC 后端 | 实现状态 | 判定 |
|---|---|---|---|---|---|---|---|---|
| 2.1 | POST /gov/parties | POST | /gov/parties | POST | iam.party.create | iam.party.create | REAL | PASS |
| 2.2 | GET /gov/parties | GET | /gov/parties | GET | data.party.read | data.party.read | REAL | PASS |
| 2.3 | GET /gov/parties/{party_id} | GET | /gov/parties/{id} | GET | data.party.read | data.party.read | REAL | PASS |
| 2.4 | PATCH /gov/parties/{party_id} | PATCH | /gov/parties/{id} | PATCH | iam.party.write | iam.party.write | 部分实现 | **MISMATCH** |
| 2.5 | POST /gov/party-edges | POST | /gov/party-edges | POST | iam.party.write | iam.party.write | REAL | PASS |
| 2.6 | DELETE /gov/party-edges/{edge_id} | DELETE | /gov/party-edges/{id} | DELETE | iam.party.write | iam.party.write | REAL | PASS |
| 2.7 | GET /gov/party-edges | GET | /gov/party-edges | GET | data.party.read | **iam.party.write** | STUB | **MISMATCH** |
| 2.8 | POST /gov/party-members | POST | /gov/party-members | POST | iam.member.create | **iam.member.write** | REAL | **MISMATCH** |
| 2.9 | DELETE /gov/party-members/{member_id} | DELETE | /gov/party-members/{id} | DELETE | iam.member.delete | iam.member.delete | REAL | PASS |
| 2.10 | GET /gov/party-members | GET | /gov/party-members | GET | data.member.read | data.member.read | STUB | PASS |

**Party 域详细问题:**

1. **PATCH /gov/parties/{id} —— 功能不完整:** 规范要求支持更新 name、description、leader_user_id、cost_center、status、metadata 全部字段。后端仅解析 `{"status": "..."}` 一个字段，其他字段被忽略。`handlePartyItem` 第 627-631 行仅定义了 `partyStatusUpdate` 结构体。

2. **GET /gov/party-edges —— ABAC 动作不一致:** 后端使用 `iam.party.write`（写权限），规范要求 `data.party.read`（读权限）。GET 请求不应需要写权限，属于授权策略错误。

3. **POST /gov/party-members —— ABAC 动作不一致:** 后端使用 `iam.member.write`，规范要求 `iam.member.create`。虽语义相近，但 ABAC 目录中 `iam.member.create` 是独立 action code（PRD 13.3），不一致意味着基于 `iam.member.write` 的角色绑定无法触发此端点。

4. **ID 类型不一致:** 规范使用 UUID 字符串，后端 `handlePartyItem`（第 585 行）将 `party_id` 解析为 `int64`，`handlePartyEdgeItem`（第 715 行）同样。这与数据库 schema 的 UUID/TEXT 主键设计冲突，实际运行时 ParseInt 会失败。

---

### 2.2 Fund (资金治理) —— 11 端点

| # | 规范端点 | 方法 | 后端路由 | 后端方法 | ABAC 规范 | ABAC 后端 | 实现状态 | 判定 |
|---|---|---|---|---|---|---|---|---|
| 3.1 | POST /gov/accounts/{id}/allocate | POST | /gov/accounts/{id}/allocate | POST | fund.allocate | **fund.balance.write** | REAL | **MISMATCH** |
| 3.2 | POST /gov/accounts/{id}/liquidate | POST | /gov/accounts/{id}/liquidate | POST | fund.liquidate | **fund.balance.write** | REAL | **MISMATCH** |
| 3.3 | GET /gov/accounts/{id}/liquidation | GET | -- | -- | fund.ledger.read | -- | -- | **MISSING** |
| 3.4 | POST /gov/accounts/{id}/liquidate/advance | POST | -- | -- | fund.liquidate | -- | -- | **MISSING** |
| 3.5 | PATCH /gov/accounts/{id}/budget | PATCH | /gov/accounts/{id}/budget | PATCH | fund.budget.write | **fund.balance.write** | REAL | **MISMATCH** |
| 3.6 | GET /gov/accounts/{id} | GET | /gov/accounts/{id} | GET | fund.balance.read | fund.balance.read | REAL | PASS |
| 3.7 | GET /gov/accounts | GET | /gov/accounts | GET | fund.balance.read | fund.balance.read | STUB | PASS |
| 3.8 | GET /gov/accounts/{id}/ledgers | GET | /gov/accounts/{id}/ledgers | GET | fund.ledger.read | fund.ledger.read | REAL | PASS |
| 3.9 | GET /gov/accounts/{id}/freezes | GET | -- | -- | fund.ledger.read | -- | -- | **MISSING** |
| 3.10 | GET /gov/allocations | GET | /gov/allocations | GET | fund.ledger.read | fund.ledger.read | STUB | PASS |
| 3.11 | GET /gov/allocations/{id} | GET | /gov/allocations/{id} | GET | fund.ledger.read | fund.ledger.read | REAL | PASS |

**Fund 域详细问题:**

1. **3 个端点完全缺失:**
   - `GET /gov/accounts/{id}/liquidation` —— 清算状态查询（规范 3.3），`handleAccountItem` 中仅处理 `allocate`、`liquidate`、`ledgers`、`budget` 四个 action，未注册 `liquidation` 子路径。
   - `POST /gov/accounts/{id}/liquidate/advance` —— 清算状态推进（规范 3.4），完全未注册。
   - `GET /gov/accounts/{id}/freezes` —— 冻结列表（规范 3.9），完全未注册。

2. **ABAC 动作大面积不一致（3 个端点）:** allocate 使用 `fund.balance.write` 而非 `fund.allocate`，liquidate 使用 `fund.balance.write` 而非 `fund.liquidate`，budget 使用 `fund.balance.write` 而非 `fund.budget.write`。这不符合四轴正交授权设计（PRD 7.2.4），`fund.allocate`、`fund.liquidate`、`fund.budget.write` 在 `sys_action_catalogs` 中是三个独立 action（PRD 13.2）。

3. **allocate 请求体字段差异:** 规范中 `channel` 为可选字段（规范 3.1 行 473 标记为 no），后端第 302-305 行将其作为必填字段校验。前端 fund page.tsx 完全不发送 `channel` 字段，运行时会被 400 拒绝。

4. **liquidate 请求体字段差异:** 后端 `GovLiquidateRequest` 包含 `party_id` 必填字段（gov_handlers_fund.go 第 46 行），但规范 3.2 的请求体中不含此字段。前端 fund page.tsx 也不发送 `party_id`。

---

### 2.3 Key (密钥管理) —— 5 端点

| # | 规范端点 | 方法 | 后端路由 | 后端方法 | 实现状态 | 判定 |
|---|---|---|---|---|---|---|
| 4.1 | POST /gov/keys | POST | /gov/keys | POST | REAL | PASS |
| 4.2 | GET /gov/keys | GET | /gov/keys | GET | REAL | PASS |
| 4.3 | GET /gov/keys/{key_id} | GET | /gov/keys/{id} | GET | STUB | PASS |
| 4.4 | DELETE /gov/keys/{key_id} | DELETE | /gov/keys/{id} | DELETE | REAL | PASS |
| 4.5 | POST /gov/keys/{key_id}/rotate | POST | /gov/keys/{id} | POST | STUB | PASS |

备注: Key 列表和创建为真实实现，详情查询和轮换返回占位消息。

---

### 2.4 Pricing (双轨计价) —— 4 端点

| # | 规范端点 | 方法 | 后端路由 | 后端方法 | 实现状态 | 判定 |
|---|---|---|---|---|---|---|
| 5.1 | PUT /gov/model-prices | PUT | /gov/model-prices | PUT | REAL | PASS |
| 5.2 | GET /gov/model-prices | GET | /gov/model-prices | GET | REAL | PASS |
| 5.3 | GET /gov/model-prices/{price_id} | GET | /gov/model-prices/{id} | GET | REAL | PASS |
| 5.4 | DELETE /gov/model-prices/{price_id} | DELETE | /gov/model-prices/{id} | DELETE | REAL | PASS |

Pricing 域全部对齐，4/4 真实实现。DELETE 返回 `{"archived": true, "id": "..."}` 符合规范的软删除语义。

---

### 2.5 Model Grant (模型授权) —— 4 端点

| # | 规范端点 | 方法 | 后端路由 | 后端方法 | 实现状态 | 判定 |
|---|---|---|---|---|---|---|
| 6.1 | POST /gov/model-grants | POST | /gov/model-grants | POST | REAL | PASS |
| 6.2 | GET /gov/model-grants | GET | /gov/model-grants | GET | REAL | PASS |
| 6.3 | GET /gov/model-grants/{grant_id} | GET | /gov/model-grants/{id} | GET | REAL | PASS |
| 6.4 | DELETE /gov/model-grants/{grant_id} | DELETE | /gov/model-grants/{id} | DELETE | REAL | PASS |

Model Grant 域全部对齐，4/4 真实实现。

---

### 2.6 Routing (路由调度) —— 9 端点

| # | 规范端点 | 方法 | 后端路由 | 后端方法 | 实现状态 | 判定 |
|---|---|---|---|---|---|---|
| 7.1 | POST /gov/route-profiles | POST | /gov/route-profiles | POST | REAL | PASS |
| 7.2 | GET /gov/route-profiles | GET | /gov/route-profiles | GET | REAL | PASS |
| 7.3 | GET /gov/route-profiles/{profile_id} | GET | /gov/route-profiles/{id} | GET | REAL | PASS |
| 7.4 | PUT /gov/route-profiles/{profile_id} | PUT | /gov/route-profiles/{id} | PUT | REAL | PASS |
| 7.5 | DELETE /gov/route-profiles/{profile_id} | DELETE | /gov/route-profiles/{id} | DELETE | REAL | PASS |
| 7.6 | GET /gov/route-strategies | GET | /gov/route-strategies | GET | REAL | PASS |
| 7.7a | GET /gov/model-routes | GET | /gov/model-routes | GET | STUB | PASS |
| 7.7b | PUT /gov/model-routes/{route_id} | PUT | /gov/model-routes/{id} | PUT | STUB | PASS |
| 7.7c | DELETE /gov/model-routes/{route_id} | DELETE | /gov/model-routes/{id} | DELETE | STUB | PASS |

Route Profile 5/5 真实实现。Model Routes 3/3 均为占位（返回 "待实现" 消息）。

---

### 2.7 ABAC (策略引擎) —— 18 端点

| # | 规范端点 | 方法 | 后端路由 | 后端方法 | 实现状态 | 判定 |
|---|---|---|---|---|---|---|
| 8.1 | GET /gov/action-catalogs | GET | /gov/action-catalogs | GET | REAL | PASS |
| 8.2 | POST /gov/roles | POST | /gov/roles | POST | REAL | PASS |
| 8.3 | GET /gov/roles | GET | /gov/roles | GET | REAL | PASS |
| 8.4 | GET /gov/roles/{role_id} | GET | /gov/roles/{id} | GET | REAL | PASS |
| 8.5 | PUT /gov/roles/{role_id} | PUT | /gov/roles/{id} | PUT | REAL | PASS |
| 8.6 | DELETE /gov/roles/{role_id} | DELETE | /gov/roles/{id} | DELETE | REAL | PASS |
| 8.7 | POST /gov/policies | POST | /gov/policies | POST | REAL | PASS |
| 8.8 | GET /gov/policies | GET | /gov/policies | GET | REAL | PASS |
| 8.9 | GET /gov/policies/{policy_id} | GET | /gov/policies/{id} | GET | REAL | PASS |
| 8.10 | PUT /gov/policies/{policy_id} | PUT | /gov/policies/{id} | PUT | REAL | PASS |
| 8.11 | DELETE /gov/policies/{policy_id} | DELETE | /gov/policies/{id} | DELETE | REAL | PASS |
| 8.12 | POST /gov/policies/{policy_id}/evaluate | POST | /gov/policies/{id}/evaluate | POST | REAL | PASS |
| 8.13 | POST /gov/subject-role-bindings | POST | /gov/subject-role-bindings | POST | REAL | PASS |
| 8.14 | GET /gov/subject-role-bindings | GET | /gov/subject-role-bindings | GET | REAL | PASS |
| 8.15 | DELETE /gov/subject-role-bindings/{binding_id} | DELETE | /gov/subject-role-bindings/{id} | DELETE | REAL | PASS |
| 8.16 | POST /gov/grants | POST | /gov/grants | POST | REAL | PASS |
| 8.17a | GET /gov/grants | GET | /gov/grants | GET | REAL | PASS |
| 8.17b | DELETE /gov/grants/{grant_id} | DELETE | /gov/grants/{id} | DELETE | REAL | PASS |

ABAC 域全部对齐，18/18 真实实现。后端 policies/evaluate 正确处理了 `/gov/policies/{id}/evaluate` 子路径路由。

---

### 2.8 UI Permission (UI权限治理) —— 16 端点

全部 CRUD 端点已注册且为真实实现:
- `/gov/ui-menus` — GET/POST, `/gov/ui-menus/{id}` — GET/PUT/DELETE (5 端点, REAL)
- `/gov/ui-routes` — GET/POST, `/gov/ui-routes/{id}` — GET/PUT/DELETE (5 端点, REAL)
- `/gov/ui-action-bindings` — GET/POST, `/gov/ui-action-bindings/{id}` — GET/PUT/DELETE (5 端点, REAL)
- `GET /gov/ui-permissions/snapshot` (1 端点, REAL)

UI Permission 域全部对齐，16/16 真实实现。

---

### 2.9 Audit (审计与对账) —— 7 端点 (+2 extra)

| # | 规范端点 | 方法 | 后端路由 | 后端方法 | 实现状态 | 判定 |
|---|---|---|---|---|---|---|
| 10.1 | GET /gov/audit-events | GET | /gov/audit-events | GET | REAL | PASS |
| 10.2 | GET /gov/audit-events/{event_id} | GET | /gov/audit-events/{id} | GET | REAL | PASS |
| 10.3 | GET /gov/request-logs/{request_id}/trace | GET | /gov/request-logs/{id} | GET | STUB | **MISMATCH** |
| 10.4 | GET /gov/request-logs | GET | /gov/request-logs | GET | STUB | PASS |
| 10.5 | GET /gov/audit-chain-anchors | GET | /gov/audit-chain-anchors | GET | STUB | PASS |
| -- | POST /gov/reconciliation-runs | POST | /gov/reconciliation-runs | POST | STUB | **EXTRA** |
| -- | GET /gov/reconciliation-runs/{id} | GET | /gov/reconciliation-runs/{id} | GET | STUB | **EXTRA** |

**Audit 域问题:**

1. **request-logs trace 路径不一致:** 规范定义 `GET /gov/request-logs/{request_id}/trace`，后端 `handleRequestLogTrace` 注册在 `/gov/request-logs/` 前缀下，实际匹配的是 `/gov/request-logs/{id}` 而非 `/gov/request-logs/{id}/trace`。缺少 `/trace` 子路径。

2. **reconciliation-runs (2 端点):** 后端额外注册但规范 v3.2.0 未定义，注释标注为"阶段 D 实现"。这是前瞻性注册。

---

### 2.10 Dashboard & Reports (仪表盘) —— 3 端点

全部已注册但返回占位消息: `GET /gov/dashboard`、`GET /gov/security-reports`、`GET /gov/trace`。

---

## 3. 前端-后端不一致清单

### 3.1 致命级别 —— 运行时必现错误

| # | 前端位置 | 前端调用 | 后端路由 | 问题 |
|---|---|---|---|---|
| 1 | `abac/page.tsx:302` | `POST /gov/policies/evaluate` | `POST /gov/policies/{id}/evaluate` | 前端漏了 `{policy_id}` 路径段。规范 8.12 要求 `/gov/policies/{policy_id}/evaluate`，后端正确实现了带 `{id}/evaluate` 的路由。前端请求 `/gov/policies/evaluate` 会匹配到 `/gov/policies/` 路由的 `handlePolicyItem`，进入 `isEvaluate` 分支但因 policyID 为空而 400。 |
| 2 | `parties/page.tsx:276` | `DELETE /gov/parties/{id}` | GET, PATCH only | 后端 `handlePartyItem` 只处理 GET 和 PATCH。前端点击删除按钮发送 DELETE 请求，后端返回 405。 |

### 3.2 请求体字段不一致 (运行时错误)

| # | 前端位置 | 字段 | 前端发送 | 后端期望 | 后果 |
|---|---|---|---|---|---|
| 3 | `fund/page.tsx:134-138` | `amount` | `number` (parseFloat) | `string` | `GovAllocateRequest.Amount` 类型为 `string`。前端发送 `{"amount": 80000}` (JSON number)，后端 JSON 解码到 `string` 字段会失败，返回 INVALID_JSON。 |
| 4 | `fund/page.tsx:134-138` | `channel` | 未发送 | **必填** | 后端第 302 行校验 `channel` 为空时返回 400。前端 allocate 表单没有 channel 选择器。 |
| 5 | `fund/page.tsx:168-171` | `party_id` | 未发送 | **必填** | 后端 `GovLiquidateRequest.PartyID` 为必填。前端 liquidate 表单缺少 `party_id` 字段。 |
| 6 | `parties/page.tsx:242-248` | `is_primary` | 未发送 | 可选 (默认 false) | 不影响功能，低优先级。 |

---

## 4. 错误码覆盖矩阵

### 4.1 后端实际错误码 vs 前端 `error-codes.ts` 映射

| 后端错误码 | HTTP | 出现位置 | 前端映射 | 状态 |
|---|---|---|---|---|
| AUTH_INVALID_KEY | 401 | requireGovAuth | AUTH_INVALID_KEY | 精确匹配 |
| AUTHZ_DENIED | 403 | requireGovAuth | AUTHZ_DENIED | 精确匹配 |
| INSUFFICIENT_BALANCE | 402 | fundErrorToHTTP | INSUFFICIENT_BALANCE | 精确匹配 |
| BUDGET_CAP_EXCEEDED | 402 | fundErrorToHTTP | BUDGET_CAP_EXCEEDED | 精确匹配 |
| IDEMPOTENCY_CONFLICT | 409 | fundErrorToHTTP | IDEMPOTENCY_CONFLICT | 精确匹配 |
| INTERNAL_ERROR | 500 | fundErrorToHTTP fallback | INTERNAL_ERROR | 精确匹配 |
| NOT_IMPLEMENTED | 501 | 各 handler | -- | **缺失** |
| METHOD_NOT_ALLOWED | 405 | 各 handler | -- | **缺失** |
| INVALID_PARAM | 400 | 各校验点 | VALIDATION_ERROR (不同 code) | **不精确** |
| INVALID_JSON | 400 | readJSON | VALIDATION_ERROR (不同 code) | **不精确** |
| INVALID_ID | 400 | routeprofile/ui | VALIDATION_ERROR (不同 code) | **不精确** |
| ACCOUNT_NOT_FOUND | 404 | handleGetAccount | RESOURCE_NOT_FOUND (不同 code) | **不精确** |
| PARTY_NOT_FOUND | 404 | handlePartyItem | RESOURCE_NOT_FOUND (不同 code) | **不精确** |
| ALLOCATION_NOT_FOUND | 404 | handleAllocationItem | RESOURCE_NOT_FOUND (不同 code) | **不精确** |
| NOT_FOUND | 404 | 各 handler | RESOURCE_NOT_FOUND (不同 code) | **不精确** |
| DB_UNAVAILABLE | 500 | 各 handler | INTERNAL_ERROR (不同 code) | **不精确** |
| CREATE_FAILED | 400/409 | 各 handler | -- | **缺失** |
| UPDATE_FAILED | 400 | 各 handler | -- | **缺失** |
| DELETE_FAILED | 404 | 各 handler | -- | **缺失** |
| CREATE_EDGE_FAILED | 400 | handlePartyEdges | -- | **缺失** |
| ADD_MEMBER_FAILED | 400 | handlePartyMembers | -- | **缺失** |
| REMOVE_MEMBER_FAILED | 404 | handlePartyMemberItem | -- | **缺失** |
| PARTY_LIST_FAILED | 500 | handleParties | -- | **缺失** |
| PARTY_QUERY_FAILED | 500 | handlePartyItem | -- | **缺失** |
| LEDGER_QUERY_FAILED | 500 | handleGetAccountLedgers | -- | **缺失** |
| ACCOUNT_QUERY_FAILED | 500 | handleGetAccount | -- | **缺失** |
| BUDGET_UPDATE_FAILED | 500 | handleUpdateBudget | -- | **缺失** |
| VERSION_CONFLICT | 409 | handleUpdateBudget | -- | **缺失** |
| KEY_CREATE_FAILED | 409 | handleCreateKey | -- | **缺失** |
| KEY_LIST_FAILED | 500 | handleListKeys | -- | **缺失** |
| UPSERT_FAILED | 409 | handleModelPrices | -- | **缺失** |
| LIST_FAILED | 500 | 各 List handler | -- | **缺失** |
| ARCHIVE_FAILED | 404 | handleModelPriceItem | -- | **缺失** |
| BIND_FAILED | 409 | handleSubjectRoleBindings | -- | **缺失** |
| REVOKE_FAILED | 404 | handleSubjectRoleBindingItem | -- | **缺失** |
| EVAL_FAILED | 500 | handlePolicyItem evaluate | -- | **缺失** |
| GRANT_PERM_FAILED | 500 | handleRoles | -- | **缺失** |
| PROJECT_FAILED | 500 | handleUIPermissionSnapshot | -- | **缺失** |
| SEARCH_FAILED | 500 | handleAuditEvents | -- | **缺失** |
| GET_FAILED | 500 | handleAuditEventItem | -- | **缺失** |

### 4.2 fundErrorToHTTP 映射的错误码

| 后端错误码 | HTTP | 前端映射 | 状态 |
|---|---|---|---|
| ACCOUNT_FROZEN_OR_CLOSED | 403 | FUND_FROZEN (code 不同!) | **不匹配** |
| ALLOCATION_CHANNEL_DENIED | 403 | -- | **缺失** |
| FREEZE_NOT_FOUND | 404 | -- | **缺失** |
| FREEZE_EXPIRED | 409 | -- | **缺失** |
| SELF_TRANSFER | 400 | -- | **缺失** |
| AMOUNT_MUST_BE_POSITIVE | 400 | -- | **缺失** |
| IDEMPOTENCY_KEY_REQUIRED | 400 | -- | **缺失** |
| LIQUIDATION_STAGE_INVALID | 422 | -- | **缺失** |

### 4.3 规范错误码但两端均未使用

| 规范错误码 | 后端 | 前端 |
|---|---|---|
| MODEL_BUDGET_EXCEEDED | 映射存在但无触发路径 | 已映射 |
| AUTH_KEY_NO_ACCOUNT | 未使用 | **缺失** |
| NO_ROUTE_WITHIN_PRICE_CAP | 未使用 | **缺失** |
| NO_ROUTE_AVAILABLE | 未使用 | **缺失** |
| ROUTE_COMPLIANCE_BLOCKED | 未使用 | **缺失** |
| COMPLIANCE_NETWORK_BLOCKED | 未使用 | **缺失** |
| CONTENT_BLOCKED | 未使用 | **缺失** |
| UPSTREAM_TIMEOUT | 未使用 | **缺失** |
| RATE_LIMITED | 未使用 | 已映射 |
| IDEMPOTENCY_REPLAY | 未使用 | **缺失** |

### 4.4 前端独有但规范不存在的错误码

BUDGET_WARN_THRESHOLD、AUTH_TOKEN_EXPIRED、RESOURCE_NOT_FOUND、RESOURCE_CONFLICT、VALIDATION_ERROR、INVALID_TRANSITION、FUND_ALLOCATION_FAILED、FUND_LIQUIDATION_FAILED、ROUTE_PROFILE_IN_USE、DELTA_CAP_EXCEEDED、UI_ACTION_DENIED、UI_MENU_NOT_VISIBLE —— 共 13 个。

---

## 5. 统计汇总

### 5.1 实现状态分布

| 类别 | 数量 | 占比 |
|---|---|---|
| 真实实现 | 57 | 76.0% |
| 占位/待实现 | 12 | 16.0% |
| 规范有但后端缺失 | 4 | 5.3% |
| 后端额外注册 | 2 | 2.7% |

### 5.2 问题严重性分布

| 严重性 | 数量 | 关键项 |
|---|---|---|
| 致命 (运行时必现) | 5 | policies/evaluate URL、DELETE parties、amount 类型、channel/party_id 缺失 |
| 高 (功能偏离) | 7 | PATCH party 不完整、3 个 ABAC 动作错误、3 端缺失、trace 路径不一致 |
| 中 (体验/安全) | 8 | ID 类型 int64 vs UUID、2 个 ABAC 动作错误、allocate channel 必填 vs 可选、liquidate party_id 额外 |
| 低 (待完善) | 12 | 12 占位端点、27 后端错误码无精确映射、13 前端独有错误码 |

### 5.3 错误码对齐统计

| 指标 | 值 |
|---|---|
| 后端实际返回的错误码种类 | 40 |
| 前端映射表条目数 | 27 |
| 精确匹配 (code 完全一致) | 6 |
| 泛化/不精确映射 | 7 |
| 后端返回但前端完全缺失 | 27 |
| 前端有但后端不返回 | 13 |

---

## 6. 修复建议 (按优先级)

### P0 —— 立即修复 (阻塞联调)

| # | 位置 | 措施 |
|---|---|---|
| 1 | `frontend abac/page.tsx:302` | 将 `POST /gov/policies/evaluate` 改为 `POST /gov/policies/{policy_id}/evaluate`，需先让用户选择 policy |
| 2 | `frontend fund/page.tsx:134-138` | allocate 请求体添加 `channel` 字段（下拉选择），`amount` 以字符串发送 |
| 3 | `frontend fund/page.tsx:168-171` | liquidate 请求体添加 `party_id` 字段 |
| 4 | `backend gov_handlers.go:627` | `handlePartyItem` 添加 `case http.MethodDelete` 分支，或前端改为 PATCH status=archived |
| 5 | `backend gov_handlers.go:302` | `handleAllocate` 的 `channel` 改为可选字段，默认值由 service 层推断 |

### P1 —— 高优先级

| # | 位置 | 措施 |
|---|---|---|
| 6 | `backend gov_handlers_fund.go` | `handleAllocate` ABAC 改为 `fund.allocate`，`handleLiquidate` 改为 `fund.liquidate`，`handleUpdateBudget` 改为 `fund.budget.write` |
| 7 | `backend` | 补充 3 个缺失端点: `GET /gov/accounts/{id}/liquidation`、`POST /gov/accounts/{id}/liquidate/advance`、`GET /gov/accounts/{id}/freezes` |
| 8 | `backend` | 修正 request-logs trace 路径: `/gov/request-logs/{id}/trace` |

### P2 —— 中优先级

| # | 位置 | 措施 |
|---|---|---|
| 9 | `frontend error-codes.ts` | 补充 27 个后端实际返回的错误码映射，移除 13 个不存在的错误码，将 `FUND_FROZEN` 改为 `ACCOUNT_FROZEN_OR_CLOSED` |
| 10 | `backend gov_handlers.go` | `handlePartyEdges` GET 使用 `data.party.read`，`handlePartyMembers` POST 使用 `iam.member.create` |
| 11 | `backend` | `GovLiquidateRequest` 移除 `party_id` 必填，从 account 记录反查 |
| 12 | `backend` | `handlePartyItem` PATCH 扩展为支持 name/description/leader/cost_center/metadata |

### P3 —— 低优先级

| # | 位置 | 措施 |
|---|---|---|
| 13 | `backend` | 若 DB ID 为 UUID/TEXT，修正 `handlePartyItem`、`handlePartyEdgeItem`、`handlePartyMemberItem` 的 `strconv.ParseInt` 为字符串比较 |
| 14 | 全部 12 占位端点 | 按规范实现具体业务逻辑 |

---

*报告生成: 2026-07-31 | 基于 api-spec-v3.2.md + gov_handlers.go (全 3 文件) + frontend 8 页面 + error-codes.ts*
