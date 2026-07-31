# RED-1 权限提升攻击面审计报告

| 项 | 内容 |
|----|------|
| 审计编号 | RED-1 |
| 审计范围 | 权限提升攻击面（IDOR / ABAC 绕过 / 条件注入 / Key 伪造 / 未注册动作） |
| 审计日期 | 2026-07-31 |
| 审计人员 | RED-1 (红蓝对抗安全专家) |
| PRD 基线 | `docs/prd/AI-GOV-PRD-v3.1.0.md` (0.3.3 权限守恒定理) |
| 代码基线 | `ai-gov-fusion/backend/` |
| 总发现数 | 9 (高风险 4 / 中风险 3 / 低风险 1 / 无风险 1) |

---

## 攻击面 #1: IDOR（水平越权）

### 发现 1.1 -- 所有单品资源端点均缺乏资源归属校验

**PRD 引用**: D-CON-01 "数据不越权：所有列表/详情/导出接口必须应用数据范围过滤器；禁止仅凭知道 UUID 读取未授权 Party 数据"

**代码位置**:

| 端点 | 文件 | 行号 | 提取 ID | 归属校验 |
|------|------|------|---------|---------|
| `GET/PATCH /gov/parties/{id}` | `gov_handlers.go` | 315-329 | `extractItemID(r, "/gov/parties")` | 无 |
| `DELETE /gov/party-edges/{id}` | `gov_handlers.go` | 343-352 | `extractItemID(r, "/gov/party-edges")` | 无 |
| `DELETE /gov/party-members/{id}` | `gov_handlers.go` | 366-375 | `extractItemID(r, "/gov/party-members")` | 无 |
| `GET/DELETE/POST /gov/keys/{id}` | `gov_handlers_fund.go` | 74-90 | `extractItemID(r, "/gov/keys")` | 无 |
| `GET/POST/PATCH /gov/accounts/{id}` | `gov_handlers_fund.go` | 21-35 | `extractItemID(r, "/gov/accounts")` | 无 |
| `GET/DELETE /gov/model-grants/{id}` | `gov_handlers_fund.go` | 137-150 | `extractItemID(r, "/gov/model-grants")` | 无 |
| `GET/PUT/DELETE /gov/roles/{id}` | `gov_handlers_abac.go` | 29-45 | `extractItemID(r, "/gov/roles")` | 无 |
| `GET/PUT/DELETE/POST /gov/policies/{id}` | `gov_handlers_abac.go` | 60-78 | `extractItemID(r, "/gov/policies")` | 无 |
| `DELETE /gov/subject-role-bindings/{id}` | `gov_handlers_abac.go` | 93-102 | `extractItemID(r, "/gov/subject-role-bindings")` | 无 |
| `DELETE /gov/grants/{id}` | `gov_handlers_abac.go` | 117-126 | `extractItemID(r, "/gov/grants")` | 无 |
| `GET/PUT/DELETE /gov/ui-menus/{id}` | `gov_handlers_abac.go` | 143-158 | `extractItemID(r, "/gov/ui-menus")` | 无 |
| `GET/PUT/DELETE /gov/ui-routes/{id}` | `gov_handlers_abac.go` | 173-188 | `extractItemID(r, "/gov/ui-routes")` | 无 |
| `GET/PUT/DELETE /gov/ui-action-bindings/{id}` | `gov_handlers_abac.go` | 203-218 | `extractItemID(r, "/gov/ui-action-bindings")` | 无 |
| `GET/PUT/DELETE /gov/route-profiles/{id}` | `gov_handlers_fund.go` | 167-183 | `extractItemID(r, "/gov/route-profiles")` | 无 |
| `PUT/DELETE /gov/model-routes/{id}` | `gov_handlers_fund.go` | 195-208 | `extractItemID(r, "/gov/model-routes")` | 无 |

**攻击路径描述**:
1. 攻击者通过合法途径获得任意有效的治理 API 凭证（如最低权限 API Key）
2. 枚举或猜测其他主体的资源 UUID（如 party ID、key ID、account ID）
3. 直接构造请求 `GET /gov/parties/<victim-id>` —— 当前代码仅校验了 ABAC 动作权限（如 `data.party.read`），但**不校验该 resource_id 是否属于当前主体**
4. 攻击者可以读取、修改、删除不属于自己的资源

**根本原因**: `gov_handlers.go` 第 178-211 行 `requireGovAuth` 中的 ABAC 评估使用 `r.URL.Path` 作为 resource ID（第 204 行），但 ABAC 的 `matchPolicyConditions` (`engine.go` 第 298-331 行) 并不校验 resource.ID 字段——只匹配 axis、actions、resource_type。这使得 ABAC 鉴权退化为纯动作权限检查，不包含资源级隔离。

**代码证据** (`gov_handlers.go` 第 202-208 行):
```go
if h.deps.ABACEngine != nil && action != "" {
    subject := abac.Subject{Type: gctx.SubjectType, ID: gctx.SubjectID}
    resource := abac.Resource{Type: "gov_api", ID: r.URL.Path}
    if err := h.deps.ABACEngine.Evaluate(r.Context(), subject, action, resource); err != nil {
```

注意 `Resource{Type: "gov_api", ID: r.URL.Path}` —— 资源类型硬编码为 `"gov_api"`，资源 ID 是整个路径。ABAC 条件的 `resource_type` 字段匹配的是 `"gov_api"`，而不是实际的资源类型（如 `"party"`, `"account"` 等）。

**当前防御状态**: **缺口** -- 无任何 resource-level IDOR 防护。ABAC 仅在动作维度（action）做权限判定，不在具体资源实例维度。

**建议修复**:
1. 在 `requireGovAuth` 之后，对单品操作增加 `SubjectID` 与 `resource.owner_id` 的归属校验
2. 或者扩展 ABAC 的 `Resource` 结构传递实际资源类型和 ID，并在条件评估中加入 `resource_id` 匹配

---

## 攻击面 #2: ABAC 绕过

### 发现 2.1 -- /gov/ui-permissions/snapshot 端点绕过 ABAC

**PRD 引用**: A-CON-02 "最小权限默认：未显式授予即拒绝"

**代码位置**: `gov_handlers_abac.go` 第 220-223 行

```go
func (h *GovHandler) handleUIPermissionSnapshot(w http.ResponseWriter, r *http.Request) {
    _, _ = h.requireGovAuth(w, r, "")  // action = "" → 跳过 ABAC 评估！
    okJSON(w, map[string]string{"message": "UI 权限快照——待实现"})
}
```

**攻击路径描述**:
1. `requireGovAuth` 在 `gov_handlers.go` 第 202 行的条件 `if h.deps.ABACEngine != nil && action != ""` —— 当 action 为空字符串时，ABAC 被跳过
2. `handleUIPermissionSnapshot` 传入空字符串 `""` 作为 action 参数
3. 任何通过认证（即拥有有效 X-API-Key）的主体都可以访问此端点，无需任何权限
4. UI 权限快照通常会暴露全局菜单/路由/按钮权限映射，属于敏感信息

**当前防御状态**: **缺口** -- 端点无 ABAC 保护，原因疑似 action 参数硬编码为空字符串（可能是占位遗留）。

**建议修复**: 为此端点指定合适的 action（如 `iam.ui.read`），与 `handleUIMenus` 等保持一致。

---

### 发现 2.2 -- gov 路由无全局鉴权中间件，依赖各 handler 自行调用 requireGovAuth

**PRD 引用**: A-CON-01 "四轴正交：data、fund、iam、routing 四轴权限独立判定"

**代码位置**:
- `gov_handlers.go` 第 94-161 行: `RegisterGovHandlers` 使用 `wrapGovHandler` 包装
- `gov_handlers.go` 第 167-172 行: `wrapGovHandler` 仅设置 Content-Type，**不执行任何鉴权**

```go
func wrapGovHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        fn(w, r)
    }
}
```

对比 `authz/middleware.go` 第 44-96 行 —— 存在一个正式的鉴权中间件 `NewMiddleware`，但**未应用于任何 /gov/* 路由**。

**攻击路径描述**:
1. 鉴权逻辑分散在每个 handler 内部，通过 `requireGovAuth` 调用
2. 如果新增 handler 时遗漏 `requireGovAuth` 调用，该端点将完全无保护
3. 当前代码中，由于所有 handler 使用了固定的 `handle*` 模式，尚无明显遗漏。但架构层面缺乏纵深防御——没有中间件兜底

**当前防御状态**: **潜在风险** -- 当前代码未出现遗漏，但架构设计缺乏中间件兜底保障。建议在 `wrapGovHandler` 或路由注册层增加默认鉴权检查。

---

### 发现 2.3 -- ABACEngine 为 nil 时完全跳过鉴权（开发模式无保护）

**PRD 引用**: A-CON-02 "最小权限默认：未显式授予即拒绝"

**代码位置**: `gov_handlers.go` 第 202 行

```go
if h.deps.ABACEngine != nil && action != "" {
```

**攻击路径描述**:
1. 如果部署时忘记配置 `ABACEngine`（如 `GovDependencies.ABACEngine` 为 nil），所有治理 API 端点将完全跳过 ABAC 鉴权
2. 只需要通过认证（拥有 X-API-Key）即可访问所有 /gov/* 端点

**当前防御状态**: **潜在风险** -- 开发环境的容错设计可能被误带到生产环境。建议在生产环境强制要求 ABACEngine 非 nil，否则所有 gov handler 返回 503。

---

## 攻击面 #3: ABAC 引擎条件注入

### 发现 3.1 -- CreatePolicy 对 conditions_json 无 schema 校验

**PRD 引用**: A-CON-03 "职责分离：仅有 routing 轴权限者不能划拨"

**代码位置**:
- `abac/policy.go` 第 23-58 行: `CreatePolicy` 函数
- `abac/engine.go` 第 298-331 行: `matchPolicyConditions` 函数

**分析**:

`CreatePolicy` 接受的字段校验仅限于:
- `PolicyCode` 非空（第 27 行）
- `PolicyName` 非空（第 30 行）
- `Effect` 为 allow 或 deny（第 33 行）
- `ConditionsJSON` 为空时默认设为 `"{}"`（第 36-38 行）

`ConditionsJSON` 的内容**完全没有结构校验**——任何有效 JSON 均可入库。

`matchPolicyConditions`（第 298-331 行）在评估时使用类型断言进行字段提取，仅识别 `axis`、`actions`、`resource_type` 三个字段。此设计本身是类型安全的。

**攻击路径描述**:
1. 攻击者通过合法途径创建或修改策略（需要 `iam.policy.write` 权限）
2. 在 `conditions_json` 中注入畸形 JSON（如超长字符串、深度嵌套对象、非标准字段）
3. 潜在影响：
   - 存储层：超大数据可能导致 DB 性能问题（DOS）
   - 评估层：`json.Unmarshal` 在畸形 JSON 上返回 error → `matchPolicyConditions` 返回 `false` → 条件不匹配 → 策略不生效
   - 如果攻击者能修改合法策略的 conditions_json 为畸形值，该策略将静默失效

**当前防御状态**: **潜在风险** -- 逻辑层是类型安全的（类型断言防御了注入攻击），但缺少输入校验导致的策略静默失效是真实风险。建议在 `CreatePolicy`/`UpdatePolicy` 中增加 conditions_json schema 校验。

---

## 攻击面 #4: API Key 伪造

### 发现 4.1 -- 治理 API 的 X-API-Key 认证完全未做哈希校验

**PRD 引用**: D-CON-03 "密钥不透传：上游 API Key 仅保存在网关侧加密存储"

**代码位置**: `gov_handlers.go` 第 194-195 行

```go
} else if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
    gctx.SubjectID = apiKey
}
```

**对比数据面认证** (`http.go` 第 1706-1718 行 + `store.go` 第 1533-1563 行):
- 数据面使用 `store.ValidateAPIKey(rawSecret, clientIP)` 进行完整的认证流程
- `ValidateAPIKey` 将原始密钥哈希后通过 `key_hash` 列查找（`store.go` 第 1538 行）
- 还校验 Key 状态、IP 白名单、过期时间、关联 Project 状态

**攻击路径描述**:
1. 治理 API 的 `X-API-Key` 认证路径直接将请求头值赋给 `gctx.SubjectID`
2. 没有哈希比对——任何字符串都能通过认证
3. 如果 `SubjectID` 随后被用于 ABAC 评估或日志记录，攻击者可以伪造任意身份
4. **但实际影响受限于**: `validateGovToken` (Bearer Token 路径) 永远是 `return "", "", false`，这意味着当前所有治理 API 认证实际上完全失效——只有 `X-API-Key` 头能通过认证（且不需要真实密钥）

**当前防御状态**: **缺口** -- 治理 API 认证完全未实现。Bearer Token 路径是占位实现（永远返回 false），X-API-Key 路径不校验哈希。

---

### 发现 4.2 -- HashSecret 使用裸 SHA-256，无可防范彩虹表攻击的盐值

**PRD 引用**: D-CON-03

**代码位置**: `types.go` 第 1122-1125 行

```go
func HashSecret(secret string) string {
    sum := sha256.Sum256([]byte(secret))
    return hex.EncodeToString(sum[:])
}
```

**攻击路径描述**:
1. `HashSecret` 使用裸 SHA-256（无盐值/HMAC），相同密钥永远产生相同哈希
2. 攻击者获得数据库读取权限后，可以:
   - 使用预计算的 SHA-256 彩虹表反查弱密钥
   - 发现两个用户使用了相同密钥（哈希值相同）

**当前防御状态**: **潜在风险** -- 数据面的 `ValidateAPIKey` 使用了此函数，如果数据库泄露且用户密钥强度不足，存在批量破解风险。建议使用 `bcrypt/scrypt/argon2` 或至少使用 HMAC-SHA256 配合每密钥独立盐值。

---

### 发现 4.3 -- ValidateAPIKey 存在时序攻击面

**PRD 引用**: 常规安全最佳实践

**代码位置**: `store.go` 第 1533-1563 行

```go
func (s *GormStore) ValidateAPIKey(rawSecret string, clientIP string) (Project, APIKey, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    var key APIKey
    if err := s.db.First(&key, "key_hash = ?", HashSecret(rawSecret)).Error; err != nil {
        return Project{}, APIKey{}, ErrInvalidAPIKey
    }
```

**攻击路径描述**:
1. GORM 的 `First` 查询执行标准 SQL `SELECT ... WHERE key_hash = ?` 匹配
2. 数据库进行字符串比较时，短前缀不匹配会快速返回，正确前缀匹配需要更多字节比较
3. 攻击者可以通过精确测量响应时间，逐字节推测出有效的 `key_hash`
4. 获得 key_hash 后，可通过彩虹表反查原始密钥（见 4.2）

**当前防御状态**: **潜在风险** -- 时序攻击需要本地网络环境且大量样本，实际利用难度较高，但在安全审计中应记录。建议在应用层添加常量时间比较（`crypto/subtle.ConstantTimeCompare`）。

---

## 攻击面 #5: 未注册动作

### 发现 5.1 -- 未注册 action 正确执行默认拒绝

**PRD 引用**: A-CON-02 "最小权限默认：未显式授予即拒绝"

**代码位置**:
- `abac/engine.go` 第 185-197 行: `lookupActionAxis`
- `abac/engine.go` 第 52-57 行: `Evaluate` 调用链

```go
func (e *Engine) lookupActionAxis(ctx context.Context, action string) (string, error) {
    var catalog SysActionCatalog
    err := e.DB.WithContext(ctx).
        Where("action_code = ?", action).
        First(&catalog).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return "", fmt.Errorf("%w: %s", ErrActionNotFound, action)
        }
        return "", fmt.Errorf("查询操作目录失败: %w", err)
    }
    return catalog.Axis, nil
}
```

在 `Evaluate` 中（第 52-57 行）:
```go
func (e *Engine) Evaluate(ctx context.Context, subject Subject, action string, resource Resource) error {
    // 步骤 1：查找操作所属的治理轴。
    actionAxis, err := e.lookupActionAxis(ctx, action)
    if err != nil {
        return err  // ← 直接返回错误，等同于拒绝
    }
```

**测试覆盖**: `engine_test.go` 第 326-340 行 `TestEvaluate_ActionNotFound` 验证了未注册 action 返回 `ErrActionNotFound`。

**攻击路径描述**: 如果攻击者构造一个不在 `sys_action_catalogs` 中的 action 值，`lookupActionAxis` 返回 `ErrActionNotFound`，`Evaluate` 直接返回该错误，导致访问被拒绝。该端点符合最小权限原则。

**当前防御状态**: **已防御** -- 未注册动作被默认拒绝，符合 A-CON-02。

---

## 附: 补充发现

### 发现 S.1 -- handleAccountItem 所有方法共用 fund.balance.read 权限

**PRD 引用**: A-CON-03 "职责分离"

**代码位置**: `gov_handlers_fund.go` 第 21-35 行

```go
func (h *GovHandler) handleAccountItem(w http.ResponseWriter, r *http.Request) {
    accountID := extractItemID(r, "/gov/accounts")
    _ = accountID
    _, _ = h.requireGovAuth(w, r, "fund.balance.read")  // ← 所有方法共用同一个 action
    switch r.Method {
    case http.MethodGet:
        okJSON(w, ...)
    case http.MethodPost:   // "Account 操作——待实现" → 应有写权限
        okJSON(w, ...)
    case http.MethodPatch:  // "Account 预算帽——待实现" → 应有 budget.write 权限
        okJSON(w, ...)
```

POST（账户操作）和 PATCH（预算帽设置）是写操作，应分别要求 `fund.account.write` 和 `fund.budget.write` 权限，而非与 GET 共用 `fund.balance.read`。

**当前防御状态**: **缺口** -- 操作方法共用只读权限，攻击者只要有余额查看权限即可执行写操作（待实现功能落地后）。

---

## 总结

| # | 攻击手法 | 状态 | 严重程度 |
|---|---------|------|---------|
| 1.1 | IDOR -- 全端点无资源归属校验 | **缺口** | **高** |
| 2.1 | ABAC 绕过 -- /gov/ui-permissions/snapshot 空 action | **缺口** | **高** |
| 2.2 | ABAC 绕过 -- 无全局中间件兜底 | 潜在风险 | 中 |
| 2.3 | ABAC 绕过 -- ABACEngine=nil 跳过鉴权 | 潜在风险 | 中 |
| 3.1 | 条件注入 -- conditions_json 无 schema 校验 | 潜在风险 | 低 |
| 4.1 | Key 伪造 -- 治理 API 不校验 Key 哈希 | **缺口** | **高** |
| 4.2 | Key 伪造 -- 裸 SHA-256 无盐值 | 潜在风险 | 中 |
| 4.3 | Key 伪造 -- 时序攻击面 | 潜在风险 | 低 |
| 5.1 | 未注册动作 -- 正确默认拒绝 | **已防御** | -- |
| S.1 | handleAccountItem 写操作共用只读权限 | **缺口** | **高** |
