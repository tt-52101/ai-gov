# RED-5 单兵作战记录

| 属性 | 值 |
|------|-----|
| Agent ID | RED-5 |
| 审计域 | 路由策略访问越权（routing 轴 SOD + scope + delta + config） |
| 执行时间 | 2026-07-31 |
| 审计文件 | `routing/strategy.go`, `routing/profile.go`, `gov_handlers.go`, `gov_handlers_fund.go`, `abac/builtin.go`, `abac/seed.go` |
| 审计项 | 5 |
| 通过 | 2 |
| 缺陷 | 3（严重 1、高危 1、中危 1） |

---

## 审计方法

五大审计项：(AP-1) RouteProfile scope 鉴权；(AP-2) delta 硬上限 20% 校验；(AP-3) SOD 跨轴互斥绑定；(AP-4) route-strategies 端点鉴权；(AP-5) strategies_json Config 注入。

---

## 发现详情

### AP-1 (高危) 路由档案 scope 鉴权绕过

`gov_handlers.go` L321-322 `lookupResourceParty` 查询 `route_profiles.party_id`，但 `RouteProfile` 模型（`routing/strategy.go` L144-156）**不含 `party_id` 字段**。查询失败静默返回 `""`，`requireGovItemAuth` 中的 scope_party_id 角色过滤对路由档案完全失效。

威胁：绑定到特定 Party 的角色本应只能管理所属 Party 的路由，实际上可管理全部路由。

### AP-2 (通过) delta 硬上限 20%

`routing/profile.go` L44-47 和 L117-120 硬编码 `MaxDeltaCap = 0.20`。`strategy.go` L174 消费此常量。不依赖前端 slider，双防有效。

### AP-3 (严重) SOD 跨轴互斥策略已定义但从未绑定

`abac/builtin.go` L46-82 定义了 4 条内置 SOD 策略：
- P-SOD-FUND：阻止资金角色操作路由
- P-SOD-IAM：阻止身份角色操作路由

但 `SeedBuiltinPolicies`（L84-120）仅创建策略定义，**不执行策略绑定**。`BootstrapBaseData`（`seed.go` L146-170）也未创建 ABAC 系统角色或绑定。当前 4 条 SOD 策略形同虚设。

威胁：拥有 `fund.*` 权限的账户可同时操作 `routing.*`，违反职责分离。

### AP-4 (通过) route-strategies 端点鉴权

`gov_handlers_fund.go` L1208：`requireGovAuth(w, r, "routing.route_profile.read")` 正确执行 ABAC 鉴权。

### AP-5 (中危) strategies_json Config 注入

`CreateProfile`（`profile.go` L36-73）未校验 `Strategies` 数组内容。`StrategyBinding.Config`（`strategy.go` L134）为 `json.RawMessage`，可注入任意 JSON 持久化，无白名单校验。

---

## 关键代码位置

| 文件 | 行号 | 内容 |
|------|------|------|
| `routing/strategy.go` | 144-156 | RouteProfile 缺失 party_id |
| `gov_handlers.go` | 321-322 | 查询不存在的 party_id 列 |
| `abac/builtin.go` | 84-120 | SeedBuiltinPolicies 未绑定 SOD 策略 |
| `routing/profile.go` | 58-60, 36-73 | CreateProfile/UpdateProfile 未校验策略白名单 |
