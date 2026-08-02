# RED-5 红蓝对抗：路由策略访问越权

**报告日期**：2026-07-31
**审计范围**：路由档案 CRUD（route_profiles）与策略组合管理
**铁律**：中文报告、代码行号

---

## 攻击路径总览

| 编号 | 攻击向量 | 严重度 | 状态 |
|------|----------|--------|------|
| AP-1 | CreateProfile / UpdateProfile / DeleteProfile ABAC 鉴权 | 中 (HIGH) | 发现缺陷 |
| AP-2 | δ 硬上限 20% 校验 | 无 | 通过 |
| AP-3 | routing.route_profile.write 跨轴越权 | 高 (CRITICAL) | 发现缺陷 |
| AP-4 | /v1/gov/route-strategies GET 未授权访问 | 无 | 通过 |
| AP-5 | strategies_json 策略组合注入 | 中 (MEDIUM) | 发现缺陷 |

---

## AP-1：CreateProfile / UpdateProfile / DeleteProfile ABAC 鉴权

### 1.1 代码路径

**包层函数**（`routing/profile.go`）：
- `CreateProfile` — 行 36-73，纯业务函数，无 ABAC 逻辑
- `UpdateProfile` — 行 103-150，纯业务函数，无 ABAC 逻辑
- `DeleteProfile` — 行 155-166，纯业务函数，无 ABAC 逻辑

**HTTP 层鉴权**（`gov_handlers_fund.go`）：
- `handleRouteProfiles` POST（创建）— 行 1087：`requireGovAuth(w, r, "routing.route_profile.write")`
- `handleRouteProfileItem` PUT（更新）— 行 1162：`requireGovItemAuth(w, r, "routing.route_profile.write", "route_profile", profileIDStr)`
- `handleRouteProfileItem` DELETE（删除）— 行 1186：`requireGovItemAuth(w, r, "routing.route_profile.write", "route_profile", profileIDStr)`

### 1.2 鉴权流程

`requireGovItemAuth`（`gov_handlers.go` 行 243-282）执行以下步骤：
1. 从 Header 提取 Bearer Token 或 X-API-Key，解析 SubjectID
2. 若未认证，返回 401 `AUTH_INVALID_KEY`
3. 调用 `lookupResourceParty` 查询资源所属 party_id
4. 构造 `abac.Resource{Type: "route_profile", ID: resourceID, PartyID: partyID}`
5. 调用 `ABACEngine.Evaluate(subject, "routing.route_profile.write", resource)`
6. 若引擎为 nil 或 action 为空，**跳过 ABAC**（行 267）

### 1.3 缺陷：RouteProfile 缺少 party_id 列

**位置**：`gov_handlers.go` 行 321-322 与 `routing/strategy.go` 行 144-156

`lookupResourceParty` 函数为 `route_profile` 资源类型查询 `route_profiles` 的 `party_id` 列：
```go
case "route_profile":
    mapping = &partyQuery{table: "route_profiles", idColumn: "id", col: "party_id"}
```

但 `RouteProfile` GORM 模型（`strategy.go` 行 144-156）**不包含 party_id 字段**：
```go
type RouteProfile struct {
    ID          int64            `json:"id" gorm:"primaryKey;autoIncrement"`
    Name        string           `json:"name" gorm:"uniqueIndex;not null"`
    Description string           `json:"description,omitempty" gorm:"type:text"`
    Strategies  []StrategyBinding `json:"strategies" gorm:"serializer:json;column:strategies_json;..."`
    DeltaCap    decimal.Decimal  `json:"delta_cap" gorm:"type:numeric(18,6);..."`
    MaxAttempts int              `json:"max_attempts" gorm:"default:3"`
    Shadow      bool             `json:"shadow" gorm:"default:false"`
    Status      string           `json:"status" gorm:"default:active"`
    CreatedAt   time.Time        `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt   time.Time        `json:"updated_at" gorm:"autoUpdateTime"`
}
```

**影响**：
- `lookupResourceParty` 的 SQL 查询尝试 SELECT `party_id` FROM `route_profiles` WHERE `id` = ?
- GORM AutoMigrate（`store.go` 行 481-521）**不包含** `routing.RouteProfile`，因此 `route_profiles` 表由 `routing.Migrate`（`strategy.go` 行 187-189）创建，确实不含 `party_id` 列
- 查询失败时 `lookupResourceParty` 静默返回空字符串（`gov_handlers.go` 行 338-346），仅记录 WARN 日志
- 导致 `Resource.PartyID` 始终为空，“按 scope_party_id 过滤”的角色绑定（ABAC `resolveSubjectRoles` 函数，`engine.go` 行 208-236）**对路由档案完全失效**

**结论**：**route_profile 资源的 ABAC scope 鉴权被静默绕过**。所有已登录用户只需获得 `routing.route_profile.write` 操作权限（通过角色或直接 grant），即可操作任意路由档案，不受 party 作用域限制。严重度：**HIGH**。

### 1.4 鉴权结论

| 操作 | HTTP 鉴权点 | 行号 | 鉴权方式 | 结论 |
|------|------------|------|----------|------|
| CreateProfile | requireGovAuth | gov_handlers_fund.go:1087 | ABAC 引擎 | 通过（引擎 eval） |
| UpdateProfile | requireGovItemAuth | gov_handlers_fund.go:1162 | ABAC 引擎 | **缺陷**：scope 绕过 |
| DeleteProfile | requireGovItemAuth | gov_handlers_fund.go:1186 | ABAC 引擎 | **缺陷**：scope 绕过 |

---

## AP-2：δ 配置硬上限 20%

### 2.1 前端校验

`frontend/app/(console)/gov/routes/page.tsx`：
- 行 379-381：HTML range input `max={0.2}` `step={0.01}` — 前端 slider 硬限制
- 行 187-189：JS 保存前二次校验 `if (editorForm.delta_cap > 0.2)` → 拒绝并提示

### 2.2 后端硬编码校验

`routing/strategy.go` 行 173-174：
```go
const MaxDeltaCap = 0.20
```

`routing/profile.go`：
- `CreateProfile` — 行 44-47：`if profile.DeltaCap.GreaterThan(maxDelta)` → 返回 `ErrDeltaCapExceeded`
- `UpdateProfile` — 行 117-120：相同的校验逻辑

### 2.3 结论

**后端存在硬编码校验，不依赖前端 slider。** 即使攻击者通过 curl 构造 `delta_cap: 0.99` 绕过前端，`CreateProfile` / `UpdateProfile` 包级函数会拒绝超过 20% 的值。前端 + 后端双重防御有效。**通过**。

---

## AP-3：routing.route_profile.write 跨轴越权

### 3.1 权限点定义

`authz/model.go` 行 74：
```go
ActionRouteProfileWrite = "route_profile.write"
```

该操作属于 `routing` 轴（行 31：`AxisRouting = "routing"`）。

### 3.2 内置 SOD 策略

`abac/builtin.go` 定义了四条内置策略：

| 策略编码 | 效果 | 条件 | 行号 | 说明 |
|----------|------|------|------|------|
| P-SOD-FUND | deny | `{"axis":["iam","routing"]}` | 46-54 | 资金管理员不可操作身份/路由 |
| P-SOD-ROUTING | deny | `{"axis":"fund"}` | 55-63 | 路由管理员不可操作资金 |
| P-SOD-IAM | deny | `{"axis":"routing"}` | 64-72 | 身份管理员不可操作路由 |
| P-AUDIT-READONLY | deny | `{}` | 73-82 | 审计角色拒绝所有写操作 |

### 3.3 SOD 策略生效路径

ABAC 引擎评估流程（`abac/engine.go` 行 52-135）：

1. `lookupActionAxis` — 查找 `routing.route_profile.write` 对应的轴 = `routing`
2. `resolveSubjectRoles` — 获取主体当前有效角色列表
3. `collectApplicablePolicies`（行 243-302）— 收集策略：
   - **直接绑定**：`sys_access_policy_bindings` 中 subject_type=user, subject_id=主体ID
   - **角色绑定**：`sys_access_policy_bindings` 中 subject_type=role, subject_id IN (主体角色ID列表)
4. 先评估 deny 策略（行 76-92）— 若 axis 条件匹配 `routing`，拒绝
5. 再评估 allow 策略
6. 最后评估角色权限（`sys_role_permissions` JOIN `sys_action_catalogs`）
7. 默认拒绝

### 3.4 缺陷：SOD 策略依赖绑定

**位置**：`abac/builtin.go` 行 84-120

`SeedBuiltinPolicies` 仅将策略定义写入 `sys_access_policies` 表，**不执行** `BindPolicy`。SOD 策略的生效依赖管理员手动将策略绑定到对应角色。

**攻击场景**：
- 若 P-SOD-FUND 未绑定到任何 fund 轴角色 → 持有 fund 角色 + routing 角色的用户可同时操作资金和路由
- 若 P-SOD-IAM 未绑定到任何 iam 轴角色 → 持有 iam 角色 + routing 角色的用户可同时操作身份和路由
- 引擎在第 6 步的 `checkRolePermission`（行 348-363）只检查角色是否拥有该操作，**不检查跨轴互斥**

**现状**：`BootstrapBaseData`（`seed.go` 行 146-170）和 `SeedDemoData`（`seed.go` 行 35-140）均**未**执行 SOD 策略绑定或创建 ABAC 角色定义。`seedDefaultRoleConfigs`（行 275-317）仅创建 admin_resource（kind="role-configs"），不创建 `sys_roles`。

**结论**：**SOD 内置策略目前仅以策略定义存在，未经绑定无法生效。** 一个主体可以同时被授予 `fund.balance.write` 和 `routing.route_profile.write` 的角色权限，ABAC 引擎不会跨轴互斥。严重度：**CRITICAL**。

### 3.5 handler 层二次确认

`gov_handlers_fund.go` 行 1087（POST）、1162（PUT）、1186（DELETE）全部使用 `requireGovAuth` / `requireGovItemAuth` 传入 `"routing.route_profile.write"`。传入空 action 会跳过 ABAC 鉴权（`gov_handlers.go` 行 267），但当前传入的是合法 action 字符串，不满足跳过条件。handler 层鉴权入口正确。

---

## AP-4：/v1/gov/route-strategies GET 未授权访问

### 4.1 路由注册

`gov_handlers.go` 行 142：
```go
mux.HandleFunc("/v1/gov/route-strategies", wrapGovHandler(h.handleRouteStrategies))
```

### 4.2 鉴权检查

`gov_handlers_fund.go` 行 1202-1212：
```go
func (h *GovHandler) handleRouteStrategies(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        writeError(w, r, NewHTTPError(405, ...))
        return
    }
    _, _ = h.requireGovAuth(w, r, "routing.route_profile.read")
    ...
}
```

`requireGovAuth` 要求：
1. 有效认证凭证（Bearer Token 或 X-API-Key）
2. ABAC 引擎评估 `routing.route_profile.read` 权限（若引擎启用）

### 4.3 结论

**通过**。该端点通过 `requireGovAuth` 执行完整的认证 + ABAC 鉴权链路。未授权访问将被 401 或 403 拦截。

---

## AP-5：策略组合注入攻击（strategies_json）

### 5.1 数据流

1. **前端**（`routes/page.tsx` 行 200-204）：构造 `strategies_json: [{code, enabled, priority, config:{}}]`，策略 code 来自硬编码的 `allStrategyCodes` 列表（行 56-60），仅 12 个合法值
2. **HTTP 入站**（`gov_handlers_fund.go` 行 1096）：`readJSON[routing.RouteProfile](w, r)` — **直接反序列化整个请求体**，包括 `strategies_json` 数组
3. **包层无校验**（`routing/profile.go` 行 36-73）：
   - `CreateProfile` 仅校验：nil、name 非空、deltaCap <= 20%、maxAttempts 范围
   - **不校验** Strategies 字段中的 code、priority、config
   - 行 58-60：仅把 nil 初始化为空切片
4. **持久化**：`db.Create(profile)` — 策略数组通过 GORM serializer:json 存入 `strategies_json` 列
5. **管道执行时校验**（`routing/profile.go` 行 317-341）：
   - `resolveStrategies` 按 priority 排序，跳过 disabled 的
   - 调用 `GetStrategy(code)` 查找已注册策略
   - 若未注册，记录 WARN 日志并跳过（行 332-337）

### 5.2 注入向量

| 字段 | 类型 | 攻击可能性 | 分析 |
|------|------|-----------|------|
| `code` | string | **低** | 未注册的策略在管道执行时被跳过；但可存储任意字符串到数据库 |
| `enabled` | bool | 无 | 布尔值无注入风险 |
| `priority` | int | **低** | 仅影响排序，但可以输入极大值制造管道顺序异常 |
| `config` | `json.RawMessage` | **中 (MEDIUM)** | 任意 JSON 内容，不经验证直接存储和传递给策略实现 |

### 5.3 缺陷：Config 字段无校验

**位置**：`routing/strategy.go` 行 134-135：
```go
// Config 策略级配置 JSON，由各策略自行解析。
Config json.RawMessage `json:"config,omitempty"`
```

攻击者可注入任意 JSON 到 Config 字段：
- 嵌套极深的 JSON（DoS by JSON parsing）
- 超大 Config（数据库膨胀）
- 恶意 JSON 键（若策略实现使用 `json.Unmarshal` 到 `map[string]any` 并逐键处理，可能引发意外行为）

当前 12 个策略注册在 `routing/strategies/register.go` 中，均通过 `register.go` 的 `RegisterAll` 函数注册。各策略在 Pipeline 执行时从 `StrategyBinding.Config` 解析配置。**问题在于**：Config 的校验完全委托给各策略实现，包层和 handler 层均不进行任何校验。

### 5.4 结论

**中危**。虽然策略代码被前端限制为 12 个合法值，但通过 curl 等工具可直接发送任意 `strategies_json`。未注册的策略代码在管道执行时被跳过，因此无法注入恶意策略逻辑，但：
- 可注入任意 JSON 到 Config 字段并持久化
- 若未来策略实现的 Config 解析存在漏洞（如反射、模板注入），此入口即在写入时未设防
- `priority` 无上限校验，可输入极大值扰乱管道排序

**建议**：在 `CreateProfile` / `UpdateProfile` 中增加策略白名单校验。

---

## 综合风险评级

| 风险项 | 严重度 | 可利用性 | 总评级 |
|--------|--------|----------|--------|
| route_profile 缺失 party_id，scope 鉴权绕过 | HIGH | 高 | **HIGH** |
| SOD 策略未绑定，跨轴互斥失效 | CRITICAL | 中（需管理员错误配置） | **HIGH** |
| strategies_json Config 注入 | MEDIUM | 低-中 | **MEDIUM** |

---

## 修复建议

### R-1：RouteProfile 增加 party_id 字段（或修正 lookupResourceParty）

**方案 A**：在 `RouteProfile` struct 增加 `PartyID` 字段并迁移表：
- `routing/strategy.go` 行 144-156 添加 `PartyID *string`
- `routing/profile.go` `CreateProfile` / `UpdateProfile` 从请求中提取
- 执行 GORM AutoMigrate

**方案 B**：修正 `lookupResourceParty`，将 `route_profile` 从映射中移除或改为返回空：
- `gov_handlers.go` 行 321-322 改为返回空或合并到系统级资源（行 323-325）

### R-2：SeedBuiltinPolicies 同步绑定 SOD 策略

在 `SeedBuiltinPolicies` 中自动将 SOD 策略绑定到对应的系统角色：
- `abac/builtin.go` 行 84-120，在创建策略后执行 `BindPolicy`
- 或创建专门的 bootstrap 函数 `SeedABACBindings`，在 `BootstrapBaseDataWithConfig`（`seed.go` 行 146）中调用

### R-3：CreateProfile / UpdateProfile 增加策略白名单校验

在 `routing/profile.go` `CreateProfile`（行 36）和 `UpdateProfile`（行 103）中，遍历 `profile.Strategies`：
- 校验每个 `binding.Code` 是否在 `GetRegistered()` 返回的列表中
- 校验 `binding.Priority` 范围（如 0-200）
- 校验 `binding.Config` JSON 有效性且大小限制（如 < 4KB）

### R-4：前端 delta_cap 校验加固

`routes/page.tsx` 行 379：`max={0.2}` 限制仅对合法 UI 操作生效。当前已有后端硬编码校验（profile.go 行 44-47），前端仅作友好提示。**无需额外加固**。

---

## 审计文件清单

| 文件 | 关键行号 | 用途 |
|------|----------|------|
| `routing/profile.go` | 36-73, 103-150, 155-166 | Create/Update/DeleteProfile 包层实现 |
| `routing/strategy.go` | 144-156, 173-174 | RouteProfile 模型、MaxDeltaCap 常量 |
| `routing/strategy.go` | 122-136 | StrategyBinding 结构体（含 Config 字段） |
| `routing/registry.go` | 36-54 | GetStrategy / GetRegistered 注册表 |
| `gov_handlers_fund.go` | 1079-1212 | handleRouteProfiles / handleRouteProfileItem / handleRouteStrategies |
| `gov_handlers.go` | 195-282 | requireGovAuth / requireGovItemAuth / lookupResourceParty |
| `gov_handlers.go` | 140-142 | 路由注册 |
| `abac/builtin.go` | 44-120 | 内置 SOD 策略定义与种子 |
| `abac/engine.go` | 52-135 | ABAC 评估流程（deny → allow → role → default-deny） |
| `abac/engine.go` | 243-302 | collectApplicablePolicies 策略收集逻辑 |
| `abac/engine.go` | 348-363 | checkRolePermission |
| `abac/model.go` | 181-208 | SysAccessPolicy 模型 |
| `authz/model.go` | 71-78 | routing 轴操作常量 |
| `seed.go` | 146-170, 275-317 | BootstrapBaseData / seedDefaultRoleConfigs |
| `store.go` | 481-521 | AutoMigrate 模型列表（不含 RouteProfile） |
| `routes/page.tsx` | 187-189, 378-381, 200-204 | 前端 delta_cap 校验与 strategies_json 构造 |
