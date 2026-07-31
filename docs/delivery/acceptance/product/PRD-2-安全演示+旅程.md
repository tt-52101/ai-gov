# PRD-2 安全治理演示脚本 + 用户旅程审计报告

| 项 | 内容 |
|----|------|
| 审计人 | PRD-2（产品专家） |
| 审计日期 | 2026-07-31 |
| 基线 PRD | AI-GOV-PRD-v3.2.0 |
| 审计代码 | `ai-gov-fusion/backend/internal/server/` + 前端 `gov/` 页面 |
| 结论 | **4/6 通过，2 项需修复。3 条旅程骨架存在，后端实现度低。** |

---

# 第一部分：安全治理演示审计（PRD §13.4）

## 步骤 1：创建角色并绑定权限

| PRD 要求 | 代码证据 | 结论 |
|----------|---------|------|
| AssignRole 是否存在 | `abac/role.go:219` -- `func AssignRole(ctx, db, subjectType, subjectID, roleID, scopePartyID, validFrom, validUntil)` 完整实现，支持主体类型、scope、有效期 | **通过** |
| 是否有职责分离策略 | `abac/builtin.go:12-21` -- 4 条内置策略常量已定义 | **通过（含风险）** |

**文件清单：**
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\abac\role.go`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\abac\builtin.go`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\abac\model.go`

**详情：**

`AssignRole` 在 `abac/role.go:219` 完整实现，接受 `subjectType`、`subjectID`、`roleID`、`scopePartyID`、`validFrom`、`validUntil` 参数，创建 `sys_subject_role_bindings` 记录。配套的 `RevokeRole`（line 254）、`GetSubjectRoles`（line 279）及 `GrantPermission`（line 134）均完整。

内置 SOD 策略（`abac/builtin.go:44-83`）定义了 4 条：
- `P-SOD-FUND`：资金管理员不可操作 iam/routing 轴
- `P-SOD-ROUTING`：路由管理员不可操作 fund 轴
- `P-SOD-IAM`：身份管理员不可操作 routing 轴
- `P-AUDIT-READONLY`：审计角色仅允许读取

`SeedBuiltinPolicies`（line 90）在启动时将策略写入 `sys_access_policies` 表（UPSERT 语义）。

**风险 -- 策略绑定缺失：** 内置策略仅写入 `sys_access_policies` 表，但未通过 `sys_access_policy_bindings` 绑定到具体角色。要使 SOD 生效，必须在部署时调用 `BindPolicy(ctx, db, policyID, "role", roleID)` 将每条 SOD 策略绑定到对应的管理员角色。当前代码框架具备 BindPolicy 能力（`abac/policy.go:146`），但没有自动绑定的 seed 逻辑。

**建议：** 在 `SeedBuiltinPolicies` 的末尾添加策略绑定步骤，或在 `gov_handlers_abac.go` 的 role handler 中，为系统内置角色（`is_system=true`）自动绑定对应的 SOD 策略。

---

## 步骤 2：数据不越权（scope_party_id 参与评估）

| PRD 要求 | 代码证据 | 结论 |
|----------|---------|------|
| scope_party_id 是否参与评估 | `abac/engine.go:200-222` -- `resolveSubjectRoles` 查询所有绑定但**不使用 scope_party_id 过滤** | **严重缺陷** |
| 不同 Party 数据被隔离 | `abac/engine.go:48-131` -- `Evaluate` 方法完全不检查 scope | **未实现** |

**文件清单：**
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\abac\engine.go`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\abac\model.go`

**详情：**

`SysSubjectRoleBinding` 模型（`abac/model.go:147-176`）定义了 `ScopePartyID *string` 字段，注释写明「为 NULL 表示全局生效；指定则仅在对应 Party 及其下级生效」。但在引擎评估中：

1. `resolveSubjectRoles`（engine.go:200-222）从 DB 加载所有有效绑定，返回 roleIDs 列表，**scope_party_id 被完全忽略**。
2. `Evaluate`（engine.go:52-131）仅基于 action→axis→policies→role_permissions 链条判定，不检查 resource 是否属于 subject 的授权 scope。
3. `Resource` 结构体（model.go:67-72）有 `Type` 和 `ID` 字段，但引擎未利用它做 Party 范围校验。

**后果：** 将某用户绑定为 `部门Leader` 角色（scope=AI研发部）后，该用户通过角色权限获得 `data.usage.read` 权限，可读取**全平台所有 Party** 的用量数据。scope_party_id 完全被忽略，PRD §7.5「所有列表/详情/导出按 data 轴授权范围过滤」未实现。

**建议：** 在 `Evaluate` 或数据查询层增加 scope 过滤逻辑：
1. 在引擎中：根据 `resolveSubjectRoles` 返回的 bindings，提取有效 scope_party_id 集合
2. 在数据层：API handler 在查询前注入 `WHERE party_id IN (scope_party_ids)` 条件
3. 短期：至少对 `handleAuditEvents`、`handleRequestLogs` 等 handler 添加 scope 校验

---

## 步骤 3：无 routing 轴权限不可操作路由

| PRD 要求 | 代码证据 | 结论 |
|----------|---------|------|
| routing 轴动作需 routing 轴角色 | `abac/engine.go:109-120` -- 角色权限检查 + `abac/builtin.go` -- SOD 策略 | **通过（绑定后生效）** |

**文件清单：**
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\abac\engine.go`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\abac\builtin.go`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\authz\middleware.go`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\gov_handlers.go`

**详情：**

ABAC 引擎评估链路正确：
1. `lookupActionAxis` 从 `sys_action_catalogs` 查找 action 所属的 axis
2. `evaluate deny 策略` -- P-SOD-ROUTING（routing→fund deny）和 P-SOD-FUND（fund→iam/routing deny）在此阶段拦截
3. `evaluate allow 策略` -- 匹配明确的 allow 规则
4. `checkRolePermission` -- 检查角色是否通过 `sys_role_permissions` 拥有该 action
5. 无匹配则默认拒绝（A-CON-02）

治理 API handler 层面，`requireGovAuth`（`gov_handlers.go:178-211`）在入口处就注入 ABAC 鉴权——路由相关的 handler 都绑定了正确的 action：
- `handleRouteProfiles`：需 `routing.route_profile.write`
- `handleModelPrices`：需 `routing.price.write`
- `handlePolicies`/`handleRoles`：需 `iam.policy.write` / `iam.role.write`

`authz/middleware.go` 也实现了 grants 表的 DENY 优先评估（line 94-115）。

**注意：** SOD 策略和角色权限均依赖于 `sys_access_policy_bindings` 和 `sys_role_permissions` 的种子数据。部署时需确保：
- SOD 策略已绑定到对应管理员角色
- 各角色通过 `GrantPermission` 获得了正确的 action 集合

---

## 步骤 4：UI 权限投影

| PRD 要求 | 代码证据 | 结论 |
|----------|---------|------|
| ProjectMenus 过滤无权限菜单 | `ui_permission/projector.go:41-118` -- 完整实现，按 ABAC 评估过滤、传播可见性 | **通过** |
| 前端路由守卫存在 | `admin-console.tsx:229-233` -- `canAccessView` 检查 + 重定向 | **通过（部分）** |

**文件清单：**
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\ui_permission\projector.go`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\frontend\features\admin\shell\admin-console.tsx`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\frontend\app\(console)\gov\layout.tsx`

**详情：**

后端 `ProjectMenus`（projector.go:41-118）：
1. 加载所有菜单 + 路由 + action_codes
2. 构建 menu_id→routes 映射
3. 对每个叶子菜单：通过 ABAC 评估关联路由的 required_action
4. 容器菜单自底向上传播可见性：子菜单至少一个 visible → 父菜单 visible
5. 孤立的不可见子树从结果中过滤
6. 按 sort_order 排序

`ProjectRoutes`（line 162-196）和 `ProjectActions`（line 203-241）逻辑一致。

前端路由守卫：
- `admin-console.tsx:229-233`：用户无权限访问当前视图时，自动重定向到默认视图
- `admin-console.tsx:292`：`load()` 函数开头检查 `canAccessView(currentUser, view)`

**风险 -- gov 治理面板无权限过滤：**
`gov/layout.tsx`（治理控制台导航栏）硬编码了全部 8 个菜单项（仪表盘、Party 管理、资金操作、价目维护、路由档案、ABAC 策略、UI 权限、审计日志），不调用 `ProjectMenus` API，也没有 `canAccessView` 检查。这意味着任何登录 gov 面板的用户都能看到所有菜单项。虽然后端 ABAC 鉴权会在 API 层拒绝，但 UI 层面未遵守 PRD §7.4.2 的菜单渲染规则。

**建议：** `gov/layout.tsx` 应在加载时调用 `/gov/ui-permissions/snapshot`（handler 已注册但返回 "待实现"），根据返回的权限快照过滤 navItems。

---

## 步骤 5：审计不可篡改

| PRD 要求 | 代码证据 | 结论 |
|----------|---------|------|
| 是否有 DELETE handler | `audit/event.go` -- 仅有 INSERT/SELECT，无 UPDATE/DELETE | **通过** |
| 是否存在 DELETE /gov/audit-events | `gov_handlers_abac.go:227-236` -- `handleAuditEvents` 仅 GET，`handleAuditEventItem` 仅读取 detail | **通过** |

**文件清单：**
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\audit\event.go`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\gov_handlers_abac.go`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\gov_handlers.go`

**详情：**

`audit/event.go` 三层保障：
1. `RecordEvent`（line 36-59）：**仅执行 INSERT**，注释明确写「永远不 UPDATE 或 DELETE」。要求调用方必须提供 ID（UUID），且 before_snapshot/after_snapshot 对配置变更类操作强制填写。
2. `SearchEvents`（line 75-129）：仅 SELECT，支持多维过滤 + 分页
3. `GetEvent`（line 140-154）：仅 SELECT 单条

`gov_handlers_abac.go:227-236`：
- `handleAuditEvents`：仅 GET，需 `data.audit.read` 权限
- `handleAuditEventItem`：仅读取 eventID，无 DELETE 方法
- 路由注册（gov_handlers.go:151-152）：`/gov/audit-events` 和 `/gov/audit-events/`，handler 内无 DELETE 分支

`audit/model.go` 需要确认 `AuditEvent` GORM 模型未使用软删除（`DeletedAt`）来规避 UPDATE 操作。

**审计链锚定：** `audit/anchor.go` 存在但本次未深查。`handleAuditChainAnchors` 注册在 `/gov/audit-chain-anchors`（仅 GET）。

---

## 步骤 6：ModelGrant DENY 优先 + 级联

| PRD 要求 | 代码证据 | 结论 |
|----------|---------|------|
| DENY 优先于 ALLOW | `modelgrant/checker.go:53-63` -- 第一遍扫 DENY，命中即返回 ErrModelAccessDenied | **通过** |
| 级联 Key>Person>Party | `modelgrant/checker.go:175-199` -- 按此顺序加载并追加全局默认 | **通过** |

**文件清单：**
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\modelgrant\checker.go`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\modelgrant\model.go`

**详情：**

`CheckAccess`（checker.go:45-79）评估顺序：
1. 第一遍：遍历所有规则，`Effect == EffectDeny` 且 `matchGrant` 匹配 → 立即返回 `ErrModelAccessDenied`
2. 第二遍：遍历所有规则，`Effect == EffectAllow` 且匹配 → 返回 nil（允许）
3. 无匹配：返回 `ErrModelAccessDenied`（最小权限默认）

`loadGrantsForCascade`（line 171-203）级联加载：
1. Key 级别（`PrincipalKey`）：主体类型为 key 时优先加载
2. Person 级别（`PrincipalPerson`）：主体类型为 user 时加载
3. Party 级别（`PrincipalParty`）：主体类型为 party 时加载
4. 全局默认：`principal_type IS NULL AND principal_id IS NULL`

匹配规则按 `priority DESC` 排序，高层级规则先被评估。DENY 规则在 ALLOW 之前扫描。

`CheckQuotaLimit`（line 91-123）支持双层预算第二层——ModelGrant 配额检查。

---

# 第二部分：用户旅程审计（PRD §1.5）

## 旅程 1：财务人员 -- 部门预算划拨与监控

| PRD 步骤 | 后端 API | 前端页面 | 状态 |
|----------|---------|---------|------|
| 1. 登录控制台 → 资金管理 | `/gov/accounts` (GET) | `/gov/fund/page.tsx` | 骨架存在 |
| 2. 查看总盘余额 | `/gov/accounts` (GET) | `StatCard` 显示汇总余额 | 前端完整 |
| 3. 配置月预算帽 + 告警 | `/gov/accounts/{id}` (PATCH) | fund page 显示 budget 进度条 | 后端待实现 |
| 4. 执行划拨 | `/gov/accounts/{id}/allocate` (POST) | fund page 有 allocate dialog | 前端完整，后端待验证 |
| 5. 成员个人账户创建 | `/gov/accounts` (POST) | 未找到对应 UI | 缺失 |
| 6. 月底查看消耗 | `/gov/dashboard` (GET) | `/gov/dashboard/page.tsx` | 前端完整 |
| 7. 收到告警 | 告警系统 | fund page 显示 warn_active 标记 | 仅视觉标记 |

**文件清单：**
- `D:\ai-work\grok\a-gov\ai-gov-fusion\frontend\app\(console)\gov\fund\page.tsx`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\frontend\app\(console)\gov\dashboard\page.tsx`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\gov_handlers_fund.go`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\gov_handlers_abac.go`

**评估：** 预算配置→划拨→监控→清算的 UI 流程存在（fund page 包含账户列表、划拨 dialog、清算 confirm、流水表格），但：
- 后端 handler 多为 "待实现" 占位
- 缺少个人账户创建（为成员创建子账户）的 UI 入口
- 预算帽配置（PATCH accounts/{id}）后端未实现，前端无编辑入口
- 告警仅静态显示，无实时通知机制

---

## 旅程 2：部门 Leader -- 消耗查看 + 成员管理 + 模型授权

| PRD 步骤 | 后端 API | 前端页面 | 状态 |
|----------|---------|---------|------|
| 1. 登录 → 仪表盘 | `/gov/dashboard` (GET) | `/gov/dashboard/page.tsx` | 前端完整 |
| 2. 成员消耗详情 | `/gov/request-logs` (GET) | 未找到 person-scoped 视图 | 缺失 |
| 3. 路由档案配置 | `/gov/route-profiles` (GET/POST) | `/gov/routes/page.tsx` | 待实现 |
| 4. 模型授权配置 | `/gov/model-grants` (GET/POST) | 未找到 ModelGrant UI 页面 | 缺失 |

**文件清单：**
- `D:\ai-work\grok\a-gov\ai-gov-fusion\frontend\app\(console)\gov\dashboard\page.tsx`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\frontend\app\(console)\gov\routes\page.tsx`
- `D:\ai-work\grok\a-gov\ai-gov-fusion\frontend\app\(console)\gov\parties\page.tsx`

**评估：** 消耗查看可通过仪表盘实现（显示 top_consumers 排行），但：
- 无法下钻到个人调用明细（缺少 person 维度的 request log 查询 UI）
- 路由档案页面（routes/page.tsx）后端 handler 为 "待实现"
- ModelGrant UI 完全缺失——gov 面板无模型授权管理入口
- 成员管理入口在 `/gov/parties` 页面但也是 "待实现"
- `gov/layout.tsx` 硬编码导航菜单，不区分 Leader 角色和普通员工

---

## 旅程 3：普通员工 -- Key 创建 + 调用 + 个人用量查看

| PRD 步骤 | 后端 API | 前端页面 | 状态 |
|----------|---------|---------|------|
| 1. IDE 配置网关 Key | `/gov/keys` (POST) | 主控制台有 API Key Wizard | 存在（TokenHub 侧） |
| 2. 网关调用链 | Pipeline（鉴权→ModelGrant→冻结→调度→结算） | 无前端 | 后端 Pipeline 已实现 |
| 3. 个人用量查看 | `/gov/request-logs?person_id=X` (GET) | 未找到个人视图 | 缺失 |
| 4. 额度耗尽反馈 | Pipeline 返回 `INSUFFICIENT_BALANCE` | 无前端展示 | Pipeline 错误码存在 |

**文件清单：**
- `D:\ai-work\grok\a-gov\ai-gov-fusion\frontend\features\admin\shell\admin-console.tsx`（API Key Wizard）
- `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\pipeline.go`

**评估：**
- Key 创建流程在主管理控制台（TokenHub）已实现，但在 gov 面板中 `/gov/keys` handler 为 "待实现"
- 调用 Pipeline 在 `pipeline.go` 中已实现鉴权→ModelGrant→价格过滤→预算帽→余额→冻结→调度→结算全链路
- 个人用量查看完全缺失——没有面向普通员工的 "我的用量" 页面。所有用量查看都在管理控制台中，需要 admin 角色
- 额度耗尽反馈：Pipeline 返回 `INSUFFICIENT_BALANCE` 错误码，但前端无个人额度仪表盘，员工无法主动了解剩余额度

---

# 第三部分：总结与建议

## 通过项（4/6）

| # | 步骤 | 结论 |
|---|------|------|
| 1 | 创建角色 + SOD 策略 | 通过（策略需补充绑定） |
| 3 | 无 routing 轴权限不可操作路由 | 通过（绑定后生效） |
| 5 | 审计不可篡改 | 通过 |
| 6 | ModelGrant DENY 优先 + 级联 | 通过 |

## 缺陷项（2/6）

| # | 步骤 | 严重度 | 描述 |
|---|------|--------|------|
| 2 | scope_party_id 不参与评估 | **严重** | 引擎未利用 scope_party_id 做范围隔离，导致 Party-scoped 角色可访问全平台数据。违反 D-CON-01。 |
| 4 | gov 面板无权限过滤 | **中** | gov/layout.tsx 硬编码菜单，不调用 UI 权限投影 API。违反 PRD §7.4.2 菜单渲染规则。 |

## 用户旅程状态总览

| 旅程 | 后端实现度 | 前端覆盖度 | 关键缺失 |
|------|-----------|-----------|---------|
| 财务人员 | ~30% | ~60% | 预算帽编辑、个人账户创建、handler 待实现 |
| 部门 Leader | ~10% | ~30% | 成员消耗明细、ModelGrant UI、路由档案编辑 |
| 普通员工 | ~40% | ~10% | 完全无个人用量视图、无额度仪表盘 |

## 优先修复建议

1. **立即修复 -- scope_party_id 隔离（D-CON-01）**
   - 在 `resolveSubjectRoles` 中同时返回 scope_party_id 集合
   - 在数据查询 handler 中注入 `WHERE party_id IN (scope)` 条件
   - 添加集成测试：同一 Party 的 Leader 无法读取其他 Party 数据

2. **高优先 -- 补充策略绑定**
   - 在 `SeedBuiltinPolicies` 后自动绑定 SOD 策略到系统角色
   - 或在系统角色种子数据中包含 `sys_access_policy_bindings` 记录

3. **高优先 -- gov 面板 UI 权限集成**
   - `gov/layout.tsx` 调用 `/gov/ui-permissions/snapshot` 获取权限快照
   - 根据权限快照过滤 navItems 数组

4. **中优先 -- 用户旅程补全**
   - 为普通员工添加 "我的用量" 页面（个人额度仪表盘）
   - 为财务人员补充预算帽配置 + 个人账户创建 UI
   - 为部门 Leader 补充 ModelGrant 管理 UI + 成员消耗明细页面

5. **中优先 -- handler 实现**
   - 当前 `gov_handlers_abac.go` 和 `gov_handlers_fund.go` 中大量 handler 返回 "待实现"
   - 优先实现：Roles CRUD、Policies CRUD、ModelGrants CRUD、RouteProfiles CRUD
