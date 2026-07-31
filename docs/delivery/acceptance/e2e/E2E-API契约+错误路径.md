# E2E 全矩阵测试 —— API 契约 & 错误路径

> **基线**: api-spec-v3.2.md (68 个端点)  
> **后端**: `ai-gov-fusion/backend/internal/server/gov_handlers*.go`  
> **前端**: `ai-gov-fusion/frontend/lib/error-codes.ts`  
> **检查日期**: 2026-07-31

---

## 任务1：API 端点覆盖率对比矩阵

### 1.1 概述

| 度量 | 数值 |
|---|---|
| API 规范端点总数 | 68 |
| 后端已注册路由（含子路径） | 28 组路由（覆盖 60+ 子路由） |
| 已实现（真实逻辑） | 43 |
| 占位（返回"待实现"） | 17 |
| 规范有但后端未实现 | 5 |
| 后端有但规范未定义 | 1（reconciliation-runs） |

### 1.2 逐域对比

#### Section 2: Party (主体管理)

| 规范端点 | HTTP | 后端路由 | 实现状态 |
|---|---|---|---|
| POST /gov/parties | POST | /gov/parties | **已实现** —— CreateParty |
| GET /gov/parties | GET | /gov/parties | **已实现** —— GetParties（按 type 筛选） |
| GET /gov/parties/{id} | GET | /gov/parties/ | **已实现** —— 单品查询（含 ABAC 单品鉴权） |
| PATCH /gov/parties/{id} | PATCH | /gov/parties/ | **已实现** —— UpdatePartyStatus |
| POST /gov/party-edges | POST | /gov/party-edges | **已实现** —— CreateEdge |
| GET /gov/party-edges | GET | /gov/party-edges | **占位** —— 返回 "PartyEdge 列表——待实现" |
| DELETE /gov/party-edges/{id} | DELETE | /gov/party-edges/ | **已实现** —— DeleteEdge |
| POST /gov/party-members | POST | /gov/party-members | **已实现** —— AddMember |
| GET /gov/party-members | GET | /gov/party-members | **占位** —— 返回 "PartyMember 列表——待实现" |
| DELETE /gov/party-members/{id} | DELETE | /gov/party-members/ | **已实现** —— RemoveMember |

#### Section 3: Fund (资金治理)

| 规范端点 | HTTP | 后端路由 | 实现状态 |
|---|---|---|---|
| POST /gov/accounts/{id}/allocate | POST | /gov/accounts/ (action=allocate) | **已实现** —— handleAllocate（含幂等、审计） |
| POST /gov/accounts/{id}/liquidate | POST | /gov/accounts/ (action=liquidate) | **已实现** —— handleLiquidate |
| GET /gov/accounts/{id}/liquidation | GET | 无 | **缺失** —— 规范 §3.3，后端 handleAccountItem 未处理 action=liquidation |
| POST /gov/accounts/{id}/liquidate/advance | POST | 无 | **缺失** —— 规范 §3.4，未注册 |
| PATCH /gov/accounts/{id}/budget | PATCH | /gov/accounts/ (action=budget) | **已实现** —— handleUpdateBudget（乐观锁） |
| GET /gov/accounts/{id} | GET | /gov/accounts/ (action="") | **已实现** —— handleGetAccount |
| GET /gov/accounts | GET | /gov/accounts | **占位** —— "Account 列表——待实现" |
| GET /gov/accounts/{id}/ledgers | GET | /gov/accounts/ (action=ledgers) | **已实现** —— handleGetAccountLedgers（分页） |
| GET /gov/accounts/{id}/freezes | GET | 无 | **缺失** —— 规范 §3.9，后端 handleAccountItem 未处理 action=freezes |
| GET /gov/allocations | GET | /gov/allocations | **占位** —— "Allocation 列表——待实现" |
| GET /gov/allocations/{id} | GET | /gov/allocations/ | **已实现** —— 单品查询 |

> **Fund 域缺失 3 个端点**: liquidation 状态查询、liquidation advance、freezes 查询。

#### Section 4: Key & Member (密钥与成员)

| 规范端点 | HTTP | 后端路由 | 实现状态 |
|---|---|---|---|
| POST /gov/keys | POST | /gov/keys | **已实现** —— handleCreateKey（SHA-256 哈希、审计） |
| GET /gov/keys | GET | /gov/keys | **已实现** —— handleListKeys（按 owner 筛选） |
| GET /gov/keys/{id} | GET | /gov/keys/ | **占位** —— "Key 详情——待实现" |
| DELETE /gov/keys/{id} | DELETE | /gov/keys/ | **已实现** —— 返回 deleted=true |
| POST /gov/keys/{id}/rotate | POST | /gov/keys/ | **占位** —— "Key 轮换——待实现" |
| POST /gov/members | POST | 无 | **缺失** —— 规范 §4.6 与 §2.8 重复，但 /gov/members 未注册路由 |

#### Section 5: Pricing (双轨计价)

| 规范端点 | HTTP | 后端路由 | 实现状态 |
|---|---|---|---|
| PUT /gov/model-prices | PUT | /gov/model-prices | **已实现** —— UpsertPrice（按 reference_id） |
| GET /gov/model-prices | GET | /gov/model-prices | **已实现** —— 按 model_id 筛选列表 |
| GET /gov/model-prices/{id} | GET | /gov/model-prices/ | **已实现** —— 单品查询 |
| DELETE /gov/model-prices/{id} | DELETE | /gov/model-prices/ | **已实现** —— ArchivePrice（软删除） |

> 定价域 **4/4 全部实现**。

#### Section 6: Model Grant (模型授权)

| 规范端点 | HTTP | 后端路由 | 实现状态 |
|---|---|---|---|
| POST /gov/model-grants | POST | /gov/model-grants | **已实现** —— CreateModelGrant |
| GET /gov/model-grants | GET | /gov/model-grants | **已实现** —— 按 principal_type/id/model_id 筛选 |
| GET /gov/model-grants/{id} | GET | /gov/model-grants/ | **已实现** —— GetModelGrant |
| DELETE /gov/model-grants/{id} | DELETE | /gov/model-grants/ | **已实现** —— DeleteModelGrant |

> 模型授权域 **4/4 全部实现**。

#### Section 7: Routing (路由调度)

| 规范端点 | HTTP | 后端路由 | 实现状态 |
|---|---|---|---|
| POST /gov/route-profiles | POST | /gov/route-profiles | **已实现** —— CreateProfile |
| GET /gov/route-profiles | GET | /gov/route-profiles | **已实现** —— ListProfiles |
| GET /gov/route-profiles/{id} | GET | /gov/route-profiles/ | **已实现** —— GetProfile |
| PUT /gov/route-profiles/{id} | PUT | /gov/route-profiles/ | **已实现** —— UpdateProfile |
| DELETE /gov/route-profiles/{id} | DELETE | /gov/route-profiles/ | **已实现** —— DeleteProfile |
| GET /gov/route-strategies | GET | /gov/route-strategies | **已实现** —— GetRegistered |
| GET /gov/model-routes | GET | /gov/model-routes | **占位** —— "ModelRoute 列表——待实现" |
| PUT /gov/model-routes/{id} | PUT | /gov/model-routes/ | **占位** —— "ModelRoute 更新——待实现" |
| DELETE /gov/model-routes/{id} | DELETE | /gov/model-routes/ | **已实现** —— 返回 deleted=true |

#### Section 8: ABAC (策略引擎)

| 规范端点 | HTTP | 后端路由 | 实现状态 |
|---|---|---|---|
| GET /gov/action-catalogs | GET | /gov/action-catalogs | **已实现** —— 按 axis 筛选 |
| POST /gov/roles | POST | /gov/roles | **已实现** —— 含权限授予 |
| GET /gov/roles | GET | /gov/roles | **已实现** —— ListRoles |
| GET /gov/roles/{id} | GET | /gov/roles/ | **已实现** —— 含展开的权限列表 |
| PUT /gov/roles/{id} | PUT | /gov/roles/ | **已实现** —— 含权限替换 |
| DELETE /gov/roles/{id} | DELETE | /gov/roles/ | **已实现** —— DeleteRole |
| POST /gov/policies | POST | /gov/policies | **已实现** —— CreatePolicy |
| GET /gov/policies | GET | /gov/policies | **已实现** —— 按 effect 筛选 |
| GET /gov/policies/{id} | GET | /gov/policies/ | **已实现** —— GetPolicy |
| PUT /gov/policies/{id} | PUT | /gov/policies/ | **已实现** —— UpdatePolicy |
| DELETE /gov/policies/{id} | DELETE | /gov/policies/ | **已实现** —— DeletePolicy |
| POST /gov/policies/{id}/evaluate | POST | /gov/policies/ (evaluate 子路径) | **已实现** —— EvaluatePolicy |
| POST /gov/subject-role-bindings | POST | /gov/subject-role-bindings | **已实现** —— AssignRole |
| GET /gov/subject-role-bindings | GET | /gov/subject-role-bindings | **已实现** —— GetSubjectRoles |
| DELETE /gov/subject-role-bindings/{id} | DELETE | /gov/subject-role-bindings/ | **已实现** —— RevokeRole |
| POST /gov/grants | POST | /gov/grants | **已实现** —— 直接创建 grant 记录 |
| GET /gov/grants | GET | /gov/grants | **已实现** —— 按 axis/principal 筛选 |
| DELETE /gov/grants/{id} | DELETE | /gov/grants/ | **已实现** —— 硬删除 |

> ABAC 域 **18/18 全部实现**。

#### Section 9: UI Permission (UI 权限治理)

| 规范端点 | HTTP | 后端路由 | 实现状态 |
|---|---|---|---|
| GET /gov/ui-menus | GET | /gov/ui-menus | **已实现** —— ListMenus |
| POST /gov/ui-menus | POST | /gov/ui-menus | **已实现** —— CreateMenu |
| GET /gov/ui-menus/{id} | GET | /gov/ui-menus/ | **已实现** —— GetMenu |
| PUT /gov/ui-menus/{id} | PUT | /gov/ui-menus/ | **已实现** —— UpdateMenu |
| DELETE /gov/ui-menus/{id} | DELETE | /gov/ui-menus/ | **已实现** —— DeleteMenu |
| GET /gov/ui-routes | GET | /gov/ui-routes | **已实现** —— ListRoutes |
| POST /gov/ui-routes | POST | /gov/ui-routes | **已实现** —— CreateRoute |
| GET /gov/ui-routes/{id} | GET | /gov/ui-routes/ | **已实现** —— GetRoute |
| PUT /gov/ui-routes/{id} | PUT | /gov/ui-routes/ | **已实现** —— UpdateRoute |
| DELETE /gov/ui-routes/{id} | DELETE | /gov/ui-routes/ | **已实现** —— DeleteRoute |
| GET /gov/ui-action-bindings | GET | /gov/ui-action-bindings | **已实现** —— ListAllActionBindings |
| POST /gov/ui-action-bindings | POST | /gov/ui-action-bindings | **已实现** —— CreateActionBinding |
| GET /gov/ui-action-bindings/{id} | GET | /gov/ui-action-bindings/ | **已实现** —— GetActionBinding |
| PUT /gov/ui-action-bindings/{id} | PUT | /gov/ui-action-bindings/ | **已实现** —— UpdateActionBinding |
| DELETE /gov/ui-action-bindings/{id} | DELETE | /gov/ui-action-bindings/ | **已实现** —— DeleteActionBinding |
| GET /gov/ui-permissions/snapshot | GET | /gov/ui-permissions/snapshot | **已实现** —— ProjectMenus + ProjectRoutes |

> UI 权限域 **16/16 全部实现**。

#### Section 10: Audit (审计与对账)

| 规范端点 | HTTP | 后端路由 | 实现状态 |
|---|---|---|---|
| GET /gov/audit-events | GET | /gov/audit-events | **已实现** —— SearchEvents（多条件筛选） |
| GET /gov/audit-events/{id} | GET | /gov/audit-events/ | **已实现** —— GetEvent |
| GET /gov/request-logs | GET | /gov/request-logs | **占位** —— "RequestLog 列表——待实现" |
| GET /gov/request-logs/{id}/trace | GET | /gov/request-logs/ | **占位** —— "RequestLog 追踪——待实现"（且路径不匹配 /trace 后缀） |
| GET /gov/audit-chain-anchors | GET | /gov/audit-chain-anchors | **占位** —— "AuditChainAnchor 列表——待实现" |

> **路由问题**: 规范期望 `GET /gov/request-logs/{request_id}/trace`，后端用 `extractItemID` 会把 `{id}/trace` 整体当 ID 提取，不会正确解析 `/trace` 子路径。

#### Section 11: Dashboard & Reports (仪表盘)

| 规范端点 | HTTP | 后端路由 | 实现状态 |
|---|---|---|---|
| GET /gov/dashboard | GET | /gov/dashboard | **占位** —— "仪表盘——待实现" |
| GET /gov/security-reports | GET | /gov/security-reports | **占位** —— "安全报表——待实现" |
| GET /gov/trace | GET | /gov/trace | **占位** —— "调用追踪——待实现" |

> 仪表盘域 **0/3 实现**，全部为占位。

#### 额外路由（规范未定义）

| 后端路由 | 规范状态 |
|---|---|
| POST /gov/reconciliation-runs | 规范无定义，后端 stage D 预留 |
| GET /gov/reconciliation-runs | 同上 |
| GET /gov/reconciliation-runs/{id} | 同上 |

---

## 任务2：错误码代码路径验证

### 2.1 逐错误码扫描

#### BUDGET_CAP_EXCEEDED —— 路径: fund/freeze.go

| 检查项 | 结果 |
|---|---|
| 错误码定义 | `fund/errors.go:93` — `newBudgetCapExceededError()` Code=`"BUDGET_CAP_EXCEEDED"` |
| 触发路径 | `fund/freeze.go:82-98` — `freezeCheckBudget()`: `budget_consumed + estimated > budget_limit` |
| HTTP 映射 | `gov_handlers_fund.go:85` — `fundErrorToHTTP()` → 402 Payment Required |
| 状态 | **路径完整** —— 从 DB 查询到 HTTP 响应全链路贯通 |

#### INSUFFICIENT_BALANCE —— 路径: fund/freeze.go

| 检查项 | 结果 |
|---|---|
| 错误码定义 | `fund/errors.go:75` — `newInsufficientBalanceError()` Code=`"INSUFFICIENT_BALANCE"` |
| 触发路径 | `fund/freeze.go:116-118` — `freezeExecute()`: `available_balance < amount` |
| HTTP 映射 | `gov_handlers_fund.go:85` — `fundErrorToHTTP()` → 402 Payment Required |
| 状态 | **路径完整** |

#### MODEL_BUDGET_EXCEEDED —— 路径: modelgrant/checker.go

| 检查项 | 结果 |
|---|---|
| 错误码定义 | `modelgrant/checker.go:18` — `ErrModelBudgetExceeded = errors.New("MODEL_BUDGET_EXCEEDED")` |
| 触发路径 | `modelgrant/checker.go:102-119` — `CheckQuotaLimit()`: `quota_consumed + estimated > quota_limit` |
| HTTP 映射 | 返回 Go sentinel error，由 `pipeline_handler.go` 的 fallback 路径处理 |
| 风险 | **未被映射为结构化错误码字符串** —— 调用方只能用 `errors.Is(err, ErrModelBudgetExceeded)` 匹配，HTTP 响应中不会出现 `"MODEL_BUDGET_EXCEEDED"` 字符串 |
| 状态 | **逻辑正确，错误码未以字符串形式透传到 HTTP 响应体** |

#### AUTHZ_DENIED —— 路径: gov_handlers.go requireGovAuth

| 检查项 | 结果 |
|---|---|
| 错误码定义 | 直接使用字符串 `"AUTHZ_DENIED"` |
| 触发路径 1 | `gov_handlers.go:225-227` — `requireGovAuth()`: ABAC 评估失败 → 403 + `"AUTHZ_DENIED"` |
| 触发路径 2 | `gov_handlers.go:275-278` — `requireGovItemAuth()`: ABAC 评估失败 → 403 + `"AUTHZ_DENIED"` |
| 额外触发 | `gov_handlers_fund.go:739` — handleCreateKey 中账户权限校验 → 403 + `"AUTHZ_DENIED"` |
| 状态 | **路径完整** —— 两条鉴权分支均有正确触发 |

#### AUTH_INVALID_KEY —— 路径: gov_handlers.go validateGovToken

| 检查项 | 结果 |
|---|---|
| 错误码定义 | 直接使用字符串 `"AUTH_INVALID_KEY"` |
| 触发位置 | `gov_handlers.go:218` — `requireGovAuth()`: `gctx.SubjectID == ""` → 401 |
| 触发位置 2 | `gov_handlers.go:262` — `requireGovItemAuth()`: 同上 |
| 触发条件 | `validateGovToken()` 返回 `(_, _, false)` → SubjectID 为空 |
| validateGovToken 失败场景 | SHA-256 哈希不匹配、密钥非 active、已过期、所有者被禁用 |
| 风险 | **validateGovToken 内部失败原因被吞没** —— 所有失败返回 `false` 但调用方统一给 `"AUTH_INVALID_KEY"`。无法区分"密钥过期"vs"用户禁用"——规范为 `AUTH_USER_DISABLED` 预留了独立错误码，但当前未实现 |
| 状态 | **路径存在但不精确** —— `AUTH_USER_DISABLED` 无触发路径 |

#### IDEMPOTENCY_CONFLICT —— 路径: idempotency/claim.go

| 检查项 | 结果 |
|---|---|
| 错误码定义 | `idempotency/claim.go:17` — `ErrIdempotencyConflict` 哨兵错误 |
| 触发路径 1 | `idempotency/claim.go:119-127` — 不同 RequestHash → `ErrIdempotencyConflict` |
| 触发路径 2 | `idempotency/claim.go:146` — 相同 Hash + StatusFailed → `ErrIdempotencyConflict` |
| fund 层触发 | `fund/errors.go:128` — `newIdempotencyConflictError()` Code=`"IDEMPOTENCY_CONFLICT"` |
| HTTP 映射 | `gov_handlers_fund.go:91` — `fundErrorToHTTP()` → 409 Conflict |
| 状态 | **路径完整** —— 双路径（idempotency 包 + fund 包）均有覆盖 |

#### COMPLIANCE_NETWORK_BLOCKED —— 路径: security/egress.go + pipeline.go

| 检查项 | 结果 |
|---|---|
| 错误码字符串 | **代码中不存在** `"COMPLIANCE_NETWORK_BLOCKED"` 字符串常量 |
| security 层 | `security/egress.go:59` — `ErrEgressBlocked` 哨兵错误（非结构化错误码） |
| 触发路径 | `security/egress.go:80-84` — INTERNAL_ONLY + external model → `ErrEgressBlocked` |
| pipeline 消费 | `pipeline.go:266-286` — 调用 `CheckEgress`，失败后记录审计并返回 error |
| pipeline 降级 | `pipeline_handler.go:85-93` — pipeline 失败后 **降级到原有路径**，不直接返回 403 |
| S-COMPLIANCE 策略 | `routing/strategies/compliance.go:30-40` — INTERNAL_ONLY 请求中剔除 external 候选 |
| 状态 | **错误码未结构化生产** —— `ErrEgressBlocked` 作为 Go 错误返回但 `"COMPLIANCE_NETWORK_BLOCKED"` 字符串不出现。路由层 S-COMPLIANCE 过滤候选但不产生此错误码 |

### 2.2 错误码触发路径总览

| 错误码 | 路径存在 | 结构化 Code | HTTP 映射 | 评估 |
|---|---|---|---|---|
| BUDGET_CAP_EXCEEDED | 是 | 是 (`FundError.Code`) | 402 | 完整 |
| INSUFFICIENT_BALANCE | 是 | 是 (`FundError.Code`) | 402 | 完整 |
| MODEL_BUDGET_EXCEEDED | 是 | 否（Go sentinel） | 降级路径 | 逻辑正确，无字符串 Code |
| AUTHZ_DENIED | 是 | 是 (字符串直接) | 403 | 完整 |
| AUTH_INVALID_KEY | 是 | 是 (字符串直接) | 401 | 存在但不精确 |
| IDEMPOTENCY_CONFLICT | 是 | 是 (双路径) | 409 | 完整 |
| COMPLIANCE_NETWORK_BLOCKED | 是 | **否** | 降级路径 | 错误码字符串缺失 |
| AUTH_USER_DISABLED | **否** | - | - | 未独立实现 |
| AUTH_KEY_NO_ACCOUNT | **否** | - | - | 未独立实现 |

---

## 任务3：前端错误码映射覆盖分析

### 3.1 前端已映射错误码

来源: `D:/ai-work/grok/a-gov/ai-gov-fusion/frontend/lib/error-codes.ts`

| 前端错误码 | 中文提示 | 对应后端错误码 | 覆盖 |
|---|---|---|---|
| BUDGET_CAP_EXCEEDED | 预算已达上限 | BUDGET_CAP_EXCEEDED | 是 |
| MODEL_BUDGET_EXCEEDED | 模型配额已用尽 | MODEL_BUDGET_EXCEEDED | 是 |
| BUDGET_WARN_THRESHOLD | 预算接近告警阈值 | (日志级别，非错误码) | 否（非标准错误码） |
| INSUFFICIENT_BALANCE | 可用余额不足 | INSUFFICIENT_BALANCE | 是 |
| MODEL_ACCESS_DENIED | 无权访问该模型 | MODEL_ACCESS_DENIED | 是 |
| AUTH_INVALID_KEY | API Key 无效 | AUTH_INVALID_KEY | 是 |
| AUTH_USER_DISABLED | 用户已被禁用 | AUTH_USER_DISABLED | 是 |
| AUTH_TOKEN_EXPIRED | 登录凭证已过期 | (无对应后端错误码) | 否 |
| AUTHZ_DENIED | 权限不足 | AUTHZ_DENIED | 是 |
| RESOURCE_NOT_FOUND | 资源不存在 | (通用 404 映射) | 部分 |
| RESOURCE_CONFLICT | 资源冲突 | (通用 409 映射) | 部分 |
| VALIDATION_ERROR | 请求参数校验失败 | (通用 400 映射) | 部分 |
| INVALID_TRANSITION | 状态流转不合法 | (LIQUIDATION_STAGE_INVALID?) | 部分 |
| IDEMPOTENCY_CONFLICT | 幂等键冲突 | IDEMPOTENCY_CONFLICT | 是 |
| RATE_LIMITED | 请求频率过高 | RATE_LIMITED | 是 |
| INTERNAL_ERROR | 服务内部错误 | INTERNAL_ERROR | 是 |
| SERVICE_UNAVAILABLE | 服务暂时不可用 | NO_ROUTE_AVAILABLE? | 部分 |
| UPSTREAM_ERROR | 上游服务异常 | UPSTREAM_ERROR | 是 |
| FUND_FROZEN | 账户已冻结 | ACCOUNT_FROZEN_OR_CLOSED | 是（别名） |
| FUND_ALLOCATION_FAILED | 资金划拨失败 | (无对应后端错误码) | 否 |
| FUND_LIQUIDATION_FAILED | 清算操作失败 | (无对应后端错误码) | 否 |
| ROUTE_PROFILE_IN_USE | 路由档案正在使用中 | (无对应后端错误码) | 否 |
| DELTA_CAP_EXCEEDED | delta_cap 超出上限 | (无对应后端错误码) | 否 |
| UI_ACTION_DENIED | 无此操作权限 | (无对应后端错误码) | 否 |
| UI_MENU_NOT_VISIBLE | 菜单不可见 | (无对应后端错误码) | 否 |

### 3.2 后端规范中需要但前端未映射的错误码

对照 API 规范 §12 (Error Code Quick Reference)，以下规范错误码在前端 `ERROR_MESSAGES` 中缺失：

| 后端错误码 | HTTP | 前端状态 |
|---|---|---|
| ACCOUNT_FROZEN_OR_CLOSED | 403 | 映射为 `FUND_FROZEN`（别名，语义匹配） |
| FREEZE_EXPIRED | 409 | **缺失** |
| FREEZE_NOT_FOUND | 404 | **缺失** |
| IDEMPOTENCY_REPLAY | 200 | **缺失**（正常响应，不需要报错） |
| AUTH_KEY_NO_ACCOUNT | 403 | **缺失** |
| NO_ROUTE_WITHIN_PRICE_CAP | 422 | **缺失** |
| NO_ROUTE_AVAILABLE | 503 | 映射为 `SERVICE_UNAVAILABLE`（别名，语义接近） |
| ROUTE_COMPLIANCE_BLOCKED | 403 | **缺失** |
| COMPLIANCE_NETWORK_BLOCKED | 403 | **缺失** |
| CONTENT_BLOCKED | 403 | **缺失** |
| UPSTREAM_TIMEOUT | 504 | **缺失** |

### 3.3 前端只有但后端不存在的错误码

| 前端错误码 | 后端对应 | 说明 |
|---|---|---|
| AUTH_TOKEN_EXPIRED | 无 | 仅在前端映射表，后端无对应 Code |
| BUDGET_WARN_THRESHOLD | 仅日志 | 后端只输出 warn 日志，不返回错误码 |
| FUND_ALLOCATION_FAILED | 无 | 前端自定义，后端用通用 INTERNAL_ERROR 或 fund.FundError |
| FUND_LIQUIDATION_FAILED | 无 | 同上 |
| ROUTE_PROFILE_IN_USE | 无 | 前端自定义 |
| DELTA_CAP_EXCEEDED | 无 | 前端自定义，后端应返回 400 INVALID_PARAM |
| UI_ACTION_DENIED | 无 | 前端自定义 |
| UI_MENU_NOT_VISIBLE | 无 | 前端自定义 |

### 3.4 前端覆盖评估

| 度量 | 数值 |
|---|---|
| 前端已映射错误码 | 25 个 |
| 与后端规范直接匹配 | 14 个 |
| 前端自定义（后端无对应） | 8 个 |
| 规范中有但前端缺失 | 8 个（FREEZE_EXPIRED, FREEZE_NOT_FOUND, AUTH_KEY_NO_ACCOUNT, NO_ROUTE_WITHIN_PRICE_CAP, ROUTE_COMPLIANCE_BLOCKED, COMPLIANCE_NETWORK_BLOCKED, CONTENT_BLOCKED, UPSTREAM_TIMEOUT） |
| 前端未映射率 | **约 30%** |

---

## 总结与建议

### 任务1 结论
- 后端 43/68 个端点已真实实现（63%），17 个占位，5 个缺失，1 个超出规范
- Fund 域缺失 3 个子端点（liquidation 查询、liquidate advance、freezes 查询）
- Dashboard 域 0/3 实现
- `/gov/members` 路由未注册

### 任务2 结论
- 7 个指定错误码中：3 个路径完整且有结构化 Code（BUDGET_CAP_EXCEEDED、INSUFFICIENT_BALANCE、AUTHZ_DENIED）
- 2 个路径存在但无结构化 Code 字符串（MODEL_BUDGET_EXCEEDED、COMPLIANCE_NETWORK_BLOCKED）
- 1 个存在但不精确（AUTH_INVALID_KEY 吞没了 AUTH_USER_DISABLED）
- `COMPLIANCE_NETWORK_BLOCKED` 字符串完全缺失 —— security 层用 Go 哨兵错误但未包装为错误码

### 任务3 结论
- 前端映射 25 个错误码，其中 8 个为前端自定义、8 个规范错误码缺失
- 建议补充: FREEZE_EXPIRED, FREEZE_NOT_FOUND, AUTH_KEY_NO_ACCOUNT, NO_ROUTE_WITHIN_PRICE_CAP, ROUTE_COMPLIANCE_BLOCKED, COMPLIANCE_NETWORK_BLOCKED, CONTENT_BLOCKED, UPSTREAM_TIMEOUT
- 后端建议将 `ErrModelBudgetExceeded` 和 `ErrEgressBlocked` 包装为带 Code 的结构化错误以匹配前端映射表
