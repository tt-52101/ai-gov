# RED-2: Cookie + 白名单风险扫除报告

**日期**: 2026-07-31
**范围**: 红蓝对抗第2轮——Cookie 会话风险 + 前端白名单绕过 + 后端二次鉴权覆盖率
**铁律**: 中文报告、PRD引用 + 代码行号

---

## 攻击面 1: Cookie 会话风险

### 1.1 Cookie 认证架构概览

前端中间件 `middleware.ts` (行 51-53) 依赖两个 cookie 做路由守卫:

```typescript
// middleware.ts:51-53
const sessionCookie =
  request.cookies.get("gov_session")?.value ||
  request.cookies.get("tokenhub_session")?.value;
```

### 1.2 关键发现: Cookie 仅在前端设置，后端从未签发

**排查结果**: 在整个 Go 后端代码库中搜索 `Set-Cookie`、`http.SetCookie`、`SameSite`、`HttpOnly`、`Secure`，返回结果为 **零**。后端在所有认证流程中均未设置任何 cookie。

- 登录接口 `handleAdminLogin` (`http.go:2008-2035`): 在 JSON 响应体中返回 `token`，不使用 `Set-Cookie`
- OAuth 回调 `oauthRedirectWithSession` (`http.go:2863-2868`): 将 session token 通过 URL fragment (`oauth_token`) 传递，不使用 cookie

**结论**: `gov_session` 和 `tokenhub_session` 这两个 cookie 只能由前端 JavaScript (`document.cookie` 或类似 API) 设置。这带来以下风险:

| 风险项 | 严重度 | 说明 |
|--------|--------|------|
| **无法设置 HttpOnly** | 高危 | 客户端 JS 设置的 cookie 不可能具有 HttpOnly 属性，这意味着任何 XSS 漏洞都可以直接窃取该 cookie |
| **Secure/SameSite 缺失** | 中危 | 代码中未发现任何 Secure 或 SameSite 配置，cookie 可能通过非加密连接传输，且对 CSRF 完全无防护 |
| **前端路由守卫可绕过** | 中危 | `middleware.ts:55` 仅做 cookie 存在性检查 (`if (!sessionCookie)`)，不校验 token 有效性与过期时间 |

### 1.3 后端认证路径（不受 Cookie 影响）

值得肯定的是，后端 API 层 **不依赖 cookie 做认证**。所有 `/v1/gov/*` handler 通过以下方式认证:

- **`requireGovAuth`** (`gov_handlers.go:199-231`): 从 `Authorization: Bearer <token>` (行 210) 或 `X-API-Key` (行 215) 提取凭证
- **`validateGovToken`** (`gov_handlers.go:433-497`): 对 Bearer token 执行 SHA-256 哈希后在 `gov_api_keys` 表中查询，校验密钥状态 (行 457)、过期时间 (行 462)、所有者用户状态 (行 468-483)

**结论**: 即使 cookie 被窃取，攻击者也只能访问前端路由 UI，而无法通过后端 API 获取或修改数据——前提是后端所有端点的 `requireGovAuth` 均正确生效（详见攻击面 3 的严重发现）。

### 1.4 建议

1. 由后端在登录/OAuth 回调时通过 `Set-Cookie` 响应头设置 session cookie，并设置 `HttpOnly; Secure; SameSite=Strict`
2. 在 `middleware.ts` 中增加 token 校验（向 `/v1/gov/ui-permissions/snapshot` 探活），而非仅检查 cookie 存在性
3. 考虑迁移到纯 Bearer token + `localStorage`/`sessionStorage` 方案，彻底消除 cookie 攻击面

---

## 攻击面 2: 前端白名单绕过风险（用户点名）

### 2.1 前端 ABAC 白名单检查机制

`console-router.tsx` (行 39-62) 在前端通过调用 `/v1/gov/ui-permissions/snapshot` 获取当前用户可访问的路由列表，然后在前端进行路由可见性判断:

```typescript
// console-router.tsx:39-62
fetch("/v1/gov/ui-permissions/snapshot")
  .then((res) => {
    ...
    return res.json();
  })
  .then((data: { routes?: Array<{ route_path: string }> }) => {
    const allowed = data.routes || [];
    const isAllowed = allowed.some(
      (r) => r.route_path === pathname || pathname.startsWith(r.route_path + "/")
    );
    if (!isAllowed && allowed.length > 0) {
      setAbacDenied(true);  // 客户端重定向到 /gov/dashboard
    }
    ...
  })
  .catch(() => {
    // API 不可用时回退——允许访问（避免锁死）  ← 行 56-58: 关键风险点
    if (!cancelled) setAbacReady(true);
  });
```

### 2.2 风险分析

| 风险 | 严重度 | 详情 |
|------|--------|------|
| **前端白名单可绕过** | 中危 | 攻击者通过浏览器 DevTools 修改 JavaScript、拦截并篡改 fetch 响应，即可绕过路由白名单检查。这是所有客户端权限检查的固有缺陷。`console-router.tsx:47-53` |
| **Fail-Open 设计** | 高危 | `console-router.tsx:56-58`: 当 `/v1/gov/ui-permissions/snapshot` API 不可用时，前端**直接放行**所有路由访问——"API 不可用时回退——允许访问（避免锁死）"。攻击者可通过网络拦截/阻断该 API 请求，绕过所有前端路由限制 |
| **snapshot endpoint 权限过于宽松** | 中危 | `handleUIPermissionSnapshot` (`gov_handlers_abac.go:1040`) 调用 `requireGovAuth` 时**传入空的 action 字符串 `""`**，导致 ABAC 策略引擎被跳过（`gov_handlers.go:223` 的 `action != ""` 条件）。任何持有有效 Bearer token 的用户均可调用此端点，获取完整的菜单树和路由列表投影 |

### 2.3 真实防护层

真正的安全保障在后端: 每个 `/v1/gov/*` API 端点都经过 `requireGovAuth`/`requireGovItemAuth` 做 ABAC 鉴权。即使攻击者绕过了前端路由白名单，也无法在后端获取或操作数据——**前提是后端所有 handler 均正确校验了鉴权结果**（见攻击面 3）。

### 2.4 建议

1. 将前端路由白名单检查定位为 "UX 优化" 而非安全控制——代码注释中应明确标注 "非安全边界，仅用于菜单可见性过滤"
2. Fail-open 策略应改为 fail-safe: API 不可用时默认拒绝访问（当前为允许访问），或至少要求用户重新登录
3. `handleUIPermissionSnapshot` 应传入具体的 action (如 `"iam.ui.read"`) 以受 ABAC 策略保护

---

## 攻击面 3: 后端二次鉴权覆盖率 —— 严重发现

### 3.1 鉴权机制说明

`gov_handlers.go` 提供两个鉴权入口函数:

- **`requireGovAuth`** (行 199-231): 用于集合端点（列表/创建），从 Header 提取 Bearer token 或 X-API-Key，校验后调用 `ABACEngine.Evaluate` 评估权限。返回 `(*GovRequestContext, bool)`。
- **`requireGovItemAuth`** (行 243-282): 用于单品端点（GET/PATCH/DELETE 单个资源），额外查询资源所属 PartyID 以支持 scope_party_id 角色绑定和 IDOR 防护。

### 3.2 路由注册全量覆盖

`RegisterGovHandlers` (`gov_handlers.go:110-180`) 注册了 **33 个路由**，覆盖 11 个治理域（Party、Fund、Key、Pricing、ModelGrant、Routing、ABAC、UI Permission、Audit、Reconciliation、Dashboard）。所有路由均通过 `wrapGovHandler` 包装，但 `wrapGovHandler` 本身**不执行鉴权**（行 186-191），鉴权由各自的 handler 函数内部调用完成。

### 3.3 严重漏洞: 大量 Handler 忽略鉴权返回值，执行流越过鉴权失败继续执行

**这是本次红蓝对抗发现的最严重漏洞。**

#### 漏洞机制

`requireGovAuth` / `requireGovItemAuth` 在鉴权失败时:
1. 通过 `writeError` 向 HTTP response writer 写入 401/403 错误响应
2. 返回 `nil, false`

正确的调用模式应当检查返回值:
```go
gctx, _ := h.requireGovAuth(w, r, "fund.balance.write")
if gctx == nil {
    return  // 正确: 鉴权失败时立即终止
}
```

**但代码库中大量 handler 使用了错误的模式:**
```go
_, _ = h.requireGovAuth(w, r, "iam.role.write")  // 错误: 丢弃返回值
// 执行流继续，即使鉴权已失败!
```

由于 Go 的 `http.ResponseWriter` 在首次 `WriteHeader` 后，后续的 `WriteHeader` 调用会被忽略，客户端确实会收到 401/403 错误——**但 handler 内部的数据库操作（DELETE、CREATE、UPDATE）会继续执行！** 这意味着无认证的攻击者可以:
1. 发送无认证凭证的 DELETE/POST 请求
2. 收到 403 响应（以为操作被拒绝）
3. 但实际上数据库中的记录已被删除或修改

#### 3.3.1 受影响的 DELETE 端点（共 9 个）

| Handler | 文件 | 忽略返回值行号 | 越权执行行号 | 越权操作 |
|---------|------|--------------|------------|---------|
| `handlePartyMemberItem` DELETE | gov_handlers.go:822 | 822 | 830 | `RemoveMember` — 移除 Party 成员 |
| `handlePartyEdgeItem` DELETE | gov_handlers.go:743 | 743 | 751 | `DeleteEdge` — 删除关系边 |
| `handleRoleItem` DELETE | gov_handlers_abac.go:276 | 276 | 282 | `DeleteRole` — 删除角色 |
| `handlePolicyItem` DELETE | gov_handlers_abac.go:429 | 429 | 435 | `DeletePolicy` — 删除策略 |
| `handleSubjectRoleBindingItem` DELETE | gov_handlers_abac.go:606 | 606 | 613 | `RevokeRole` — 撤销角色绑定 |
| `handleGrantItem` DELETE | gov_handlers_abac.go:716 | 716 | 723 | `db.Delete(&GovGrant{})` — 删除授权记录 |
| `handleUIMenuItem` DELETE | gov_handlers_abac.go:823 | 823 | 829 | `DeleteMenu` — 删除 UI 菜单 |
| `handleUIRouteItem` DELETE | gov_handlers_abac.go:922 | 922 | 928 | `DeleteRoute` — 删除 UI 路由 |
| `handleUIActionBindingItem` DELETE | gov_handlers_abac.go:1021 | 1021 | 1027 | `DeleteActionBinding` — 删除 UI 按钮绑定 |

#### 3.3.2 受影响的 POST/CREATE 端点（共 3 个）

| Handler | 文件 | 忽略返回值行号 | 越权执行行号 | 越权操作 |
|---------|------|--------------|------------|---------|
| `handleUIMenus` POST | gov_handlers_abac.go:745 | 745 | 756 | `CreateMenu` — 创建 UI 菜单 |
| `handleUIRoutes` POST | gov_handlers_abac.go:844 | 844 | 855 | `CreateRoute` — 创建 UI 路由 |
| `handleUIActionBindings` POST | gov_handlers_abac.go:943 | 943 | 954 | `CreateActionBinding` — 创建 UI 按钮绑定 |

#### 3.3.3 受影响的只读端点（约 25 个）

此外，以下 GET 端点的 handler 同样忽略了鉴权返回值，虽然数据库不会被修改，但鉴权失败的请求仍继续执行数据库查询逻辑——这违反了最小权限原则，且在多线程/连接池场景下可能导致竞态问题:

| Handler | 文件 | 忽略返回值行号 |
|---------|------|--------------|
| `handlePartyEdges` GET | gov_handlers.go:719 | 719 |
| `handleActionCatalogs` GET | gov_handlers_abac.go:101 | 101 |
| `handleRoles` GET | gov_handlers_abac.go:170 | 170 |
| `handleRoleItem` GET | gov_handlers_abac.go:193 | 193 |
| `handlePolicies` GET | gov_handlers_abac.go:343 | 343 |
| `handlePolicyItem` GET | gov_handlers_abac.go:374 | 374 |
| `handlePolicyItem` POST/evaluate | gov_handlers_abac.go:447 | 447 |
| `handlePolicyEvaluate` POST | gov_handlers_abac.go:483 | 483 |
| `handleSubjectRoleBindings` GET | gov_handlers_abac.go:581 | 581 |
| `handleGrants` GET | gov_handlers_abac.go:682 | 682 |
| `handleUIMenus` GET | gov_handlers_abac.go:763 | 763 |
| `handleUIMenuItem` GET | gov_handlers_abac.go:792 | 792 |
| `handleUIMenuItem` PUT | gov_handlers_abac.go:805 | 805 |
| `handleUIRoutes` GET | gov_handlers_abac.go:862 | 862 |
| `handleUIRouteItem` GET | gov_handlers_abac.go:891 | 891 |
| `handleUIRouteItem` PUT | gov_handlers_abac.go:904 | 904 |
| `handleUIActionBindings` GET | gov_handlers_abac.go:961 | 961 |
| `handleUIActionBindingItem` GET | gov_handlers_abac.go:990 | 990 |
| `handleUIActionBindingItem` PUT | gov_handlers_abac.go:1003 | 1003 |
| `handleAuditEvents` GET | gov_handlers_abac.go:1081 | 1081 |
| `handleAuditEventItem` GET | gov_handlers_abac.go:1139 | 1139 |
| `handleRequestLogs` GET | gov_handlers_abac.go:1162 | 1162 |
| `handleRequestLogTrace` GET | gov_handlers_abac.go:1169 | 1169 |
| `handleAuditChainAnchors` GET | gov_handlers_abac.go:1175 | 1175 |
| `handleReconciliationRuns` POST/GET | gov_handlers_abac.go:1184/1190 | 1184/1190 |
| `handleReconciliationRunItem` GET | gov_handlers_abac.go:1202 | 1202 |
| `handleDashboard` GET | gov_handlers_abac.go:1213 | 1213 |
| `handleSecurityReports` GET | gov_handlers_abac.go:1218 | 1218 |
| `handleTrace` GET | gov_handlers_abac.go:1223 | 1223 |
| `handleAccounts` GET | gov_handlers_fund.go:127 | 127 |
| `handleAllocations` GET | gov_handlers_fund.go:610 | 610 |
| `handleModelPrices` GET | gov_handlers_fund.go:923 | 923 |

### 3.4 正确实现的端点（对照）

以下端点正确检查了鉴权返回值——当 `gctx == nil` 时立即 `return`:

| Handler | 文件 | 行号 |
|---------|------|------|
| `handleParties` POST/GET | gov_handlers.go | 521-523, 542-544 |
| `handlePartyItem` GET/PATCH/DELETE | gov_handlers.go | 594-596, 617-619, 660-662 |
| `handlePartyEdges` POST | gov_handlers.go | 692-694 |
| `handlePartyMembers` POST | gov_handlers.go | 771-773 |
| `handleRoles` POST | gov_handlers_abac.go | 123-125 |
| `handleRoleItem` PUT | gov_handlers_abac.go | 218-220 |
| `handlePolicies` POST | gov_handlers_abac.go | 297-299 |
| `handlePolicyItem` PUT | gov_handlers_abac.go | 387-389 |
| `handleSubjectRoleBindings` POST | gov_handlers_abac.go | 532-534 |
| `handleGrants` POST | gov_handlers_abac.go | 629-631 |
| `handleUIPermissionSnapshot` GET | gov_handlers_abac.go | 1040-1042 |
| `handleAllocate` POST | gov_handlers_fund.go | 277-279 |
| `handleLiquidate` POST | gov_handlers_fund.go | 375-378 |
| `handleUpdateBudget` PATCH | gov_handlers_fund.go | 462-464 |
| `handleCreateKey` POST | gov_handlers_fund.go | 703-705 |
| `handleListKeys` GET | gov_handlers_fund.go | 816-818 |
| `handleModelPrices` PUT | gov_handlers_fund.go | 873-875 |
| `handleModelGrants` POST | gov_handlers_fund.go | 994-996 |
| `handleRouteProfiles` POST | gov_handlers_fund.go | 1087-1089 |

### 3.5 统计数据

| 分类 | 数量 |
|------|------|
| 注册路由总数 | 33 |
| 正确校验鉴权返回值的 handler | 18 |
| **忽略鉴权返回值的 handler** | **~37** (含同一 handler 的多个 HTTP method) |
| **越权 DELETE（严重）** | **9** |
| **越权 CREATE（严重）** | **3** |
| 越权 READ（中等） | ~25 |

### 3.6 建议修复方案

**紧急修复（P0）**: 所有 DELETE/CREATE handler 必须改为检查鉴权返回值:

```go
// 修复前 (错误):
_, _ = h.requireGovItemAuth(w, r, "iam.role.write", "role", roleID)
if db == nil { ... }
if err := abac.DeleteRole(...) { ... }  // 越权执行

// 修复后 (正确):
_, ok := h.requireGovItemAuth(w, r, "iam.role.write", "role", roleID)
if !ok {
    return  // 鉴权失败，立即终止
}
if db == nil { ... }
if err := abac.DeleteRole(...) { ... }
```

**高优先级修复（P1）**: 所有 GET handler 同样应检查鉴权返回值，确保鉴权失败的请求不会继续执行任何业务逻辑。

**预防措施**: 考虑将 `requireGovAuth`/`requireGovItemAuth` 重构为 middleware 模式（类似 `authz.NewMiddleware`），在路由层统一拦截，消除每个 handler 手动调用鉴权的模式——这是当前大规模遗漏的根本原因。

---

## 总结

| 编号 | 发现 | 严重度 | 涉及文件 |
|------|------|--------|---------|
| R2-1 | Cookie 由前端 JS 设置，无法使用 HttpOnly，缺乏 Secure/SameSite | 高危 | middleware.ts:51-53 |
| R2-2 | 前端路由白名单 fail-open，API 不可用时放行所有路由 | 高危 | console-router.tsx:56-58 |
| R2-3 | `handleUIPermissionSnapshot` ABAC 鉴权被跳过（action=""） | 中危 | gov_handlers_abac.go:1040 |
| R2-4 | **9 个 DELETE handler 忽略鉴权返回值，可无认证越权删除数据** | **严重** | gov_handlers.go:822/743; gov_handlers_abac.go:276/429/606/716/823/922/1021 |
| R2-5 | **3 个 POST/CREATE handler 忽略鉴权返回值，可无认证越权创建资源** | **严重** | gov_handlers_abac.go:745/844/943 |
| R2-6 | ~25 个 GET handler 忽略鉴权返回值，鉴权失败后继续执行业务逻辑 | 中危 | 详见 3.3.3 节 |
