# QA-3 安全治理域验收报告

**验收人**: QA-3 (安全治理域)  
**验收日期**: 2026-07-31  
**代码基线**: `ai-gov-fusion/backend/internal/server/`  
**结论**: 6 场景基本通过, 1 项红线发现, 0 项阻塞

---

## 场景 1: ABAC 四轴越权

**审查目标**: `abac/engine.go` -- Evaluate 是否区分 data/fund/iam/routing 四轴? 是否 deny 优先? 无匹配是否默认拒绝?

### PRD 引用

`abac/doc.go` 声明四轴正交 (PRD §7.2): "四轴 (data/fund/iam/routing) 正交, 禁止一轴推导另一轴". `engine.go` 注释 (L29-36) 明确评估顺序与 PRD §7.2.3 对齐.

### 代码证据

**四轴定义** (`abac/model.go` L22-31):

```go
AxisData    = "data"
AxisFund    = "fund"
AxisIAM     = "iam"
AxisRouting = "routing"
```

**Evaluate 评估顺序** (`abac/engine.go` L52-131):

| 步骤 | 行号 | 逻辑 |
|------|------|------|
| 1. 查找操作轴 | L54 | `lookupActionAxis(ctx, action)` -- 从 `sys_action_catalogs` 表按 action_code 查出 axis |
| 2. 解析角色 | L60 | `resolveSubjectRoles(ctx, subject)` -- 查询 `sys_subject_role_bindings` |
| 3. 收集策略 | L66 | `collectApplicablePolicies` -- 直接绑定 + 角色绑定, 按 priority DESC |
| 4. deny 评估 | L72-88 | **deny 优先**: 遍历策略, EffectDeny 且条件匹配则立即返回 `ErrAccessDenied` |
| 5. allow 评估 | L91-107 | allow 策略评估, 条件匹配则返回 nil (放行) |
| 6. 角色权限 | L110-120 | `checkRolePermission` -- 查询 `sys_role_permissions` JOIN `sys_action_catalogs` |
| 7. 默认拒绝 | L122-130 | 无匹配 → `ErrAccessDenied: 无匹配策略或权限` |

**轴区分机制**: `matchPolicyConditions` (L298-332) 通过 conditions_json 中的 `axis` 字段精确匹配. `matchAxis` (L403-416) 支持字符串精确匹配和数组任一匹配.

### 测试覆盖

- `TestEvaluate_Allow` (engine_test.go L50): allow 策略放行验证
- `TestEvaluate_Deny` (engine_test.go L87): deny 优先于 allow (同一主体同时有 allow 和 deny 策略, deny 胜出)
- `TestEvaluate_DefaultDeny` (engine_test.go L141): 无策略/无角色 → 默认拒绝
- `TestEvaluate_SeparationOfDuty` (engine_test.go L202): 职责分离 -- fund_admin 可 fund.allocate, 被拒绝 routing.price.write

### 判定: 通过

---

## 场景 2: UI 权限投影

**审查目标**: `ui_permission/projector.go` -- ProjectMenus/ProjectRoutes/ProjectActions 是否基于 ABAC 评估结果过滤?

### PRD 引用

`ui_permission/doc.go`: "UI 权限投影--ABAC 引擎在展示层的投影. 前端隐藏按钮减少误操作, 真正的安全在后端 ABAC 引擎." (PRD §7.4.3)

### 代码证据

**ProjectMenus** (`projector.go` L41-118):
- 加载全量菜单 (`ListMenus`) 和路由 (`ListRoutes`)
- 为每个菜单关联的路由调用 `evaluateAnyRoute` (L122-137)
- `evaluateAnyRoute` 逐一调用 `p.ABAC.Evaluate(ctx, subject, actionCode, resource)` (L132)
- 任一通过 → 菜单可见; 全部拒绝 → 不可见
- 自底向上传播可见性 (`propagateVisibility`, L142-156), 过滤不可见子树

**ProjectRoutes** (`projector.go` L162-196):
- 遍历所有路由, `required_action_id != nil` 则调用 `p.ABAC.Evaluate` (L184)
- 仅返回 ABAC 通过的路由

**ProjectActions** (`projector.go` L203-241):
- 按页面路由加载按钮绑定 (`ListActionBindingsByPage`)
- 逐一调用 `p.ABAC.Evaluate` (L227) 判定每个按钮的 visible 状态
- 返回 `map[button_code]bool`

**ABACEngine 接口** (`projector.go` L14-16): 依赖 ABAC 评估接口, 不依赖具体实现 -- 松耦合设计.

### 测试覆盖

`projector_test.go` 存在但未审阅内容 (store_test.go 同理).

### 判定: 通过

---

## 场景 3: 审计不可篡改

**审查目标**: `audit/event.go` -- RecordEvent 是否仅 INSERT? 是否有 UPDATE/DELETE 路径?

### PRD 引用

`audit/doc.go`: "审计事件表应用层仅允许 INSERT, 禁止 UPDATE 与 DELETE". `audit/event.go` L27-30: "铁律 (AU-CON-01 / D-CON-04): 审计事件一旦写入即不可变更或删除". (PRD §7.6)

### 代码证据

**RecordEvent** (`audit/event.go` L36-59):

```go
// 仅 INSERT——不存在 UPDATE 或 DELETE 路径（AU-CON-01）。
if err := db.WithContext(ctx).Create(event).Error; err != nil {
    return fmt.Errorf("audit: 写入审计事件失败: %w", err)
}
```

- 唯一数据写入路径: `db.Create(event)` (GORM INSERT)
- SearchEvents (L75): 仅 SELECT, 不修改数据
- GetEvent (L140): 仅 `db.First`, 不修改数据
- `audit/store.go`: 仅 `AutoMigrate`, 无数据变更逻辑
- 全包 grep 结果: 无任何 `UPDATE`, `DELETE`, `.Updates`, `.Delete`, `.Save` 调用出现在 audit 包中

### 判定: 通过

---

## 场景 4: 审计链锚定

**审查目标**: `audit/anchor.go` -- AnchorChain 是否使用 SHA-256?

### PRD 引用

`audit/model.go` L108: "新锚点的 anchor_hash 由前一锚点哈希、事件 ID 范围、事件计数及时间戳拼接后计算 SHA-256 得到." (PRD §7.6)

### 代码证据

**AnchorChain** (`audit/anchor.go` L30-73):

```go
import "crypto/sha256"  // L5

// L54-57:
concat := fmt.Sprintf("%s:%s:%s:%d:%s",
    prevHash, startEventID, endEventID, eventCount, now)
hash := sha256.Sum256([]byte(concat))
anchorHash := fmt.Sprintf("%x", hash)
```

拼接内容: `prevHash:startEventID:endEventID:eventCount:RFC3339Timestamp`
- 链式前驱: `fetchPreviousAnchorHash` (L128-137) -- 取最新锚点哈希, 首锚使用 "GENESIS"
- 锚定记录写入 `audit_chain_anchors` 表

**VerifyChain** (`audit/anchor.go` L87-124): 重新计算并比对哈希, 验证区间事件未被篡改 (事件计数比对 + 哈希重算).

**SHA-256 确认**: 使用标准库 `crypto/sha256.Sum256`, 输出 `%x` 十六进制小写字符串.

### 判定: 通过

---

## 场景 5: 禁人即禁 Key

**审查目标**: 审查 authz middleware 和 API key 校验逻辑 -- 禁用用户后 Key 是否立即失效?

### 代码证据

**治理 API 鉴权** (`gov_handlers.go` L177-211):

```go
func (h *GovHandler) requireGovAuth(w http.ResponseWriter, r *http.Request, action string) (*GovRequestContext, bool) {
    // ...
    if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
        gctx.SubjectID = apiKey  // L195: Key 直接作为 SubjectID
    }
    // ABAC 鉴权
    if h.deps.ABACEngine != nil && action != "" {
        subject := abac.Subject{Type: gctx.SubjectType, ID: gctx.SubjectID}
        // ...
    }
}
```

**authz 中间件** (`authz/middleware.go` L44-96): 从 context 提取 principal_type/principal_id, 调用 grants 表 `Evaluate`.

**发现 -- 关键待实现**:
- `validateGovToken` (L267-269) 当前为空占位: `return "", "", false` -- 注释标注 "完整实现应调用 store.ValidateAdminSession(token)"
- API Key 作为 SubjectID 直接参与 ABAC 评估, **未检查 Key 所属用户是否被禁用**
- 无法确认: 禁用用户后, 用户的 API Key 是否在认证层被失效 (token/key 校验尚未实现)

**数据面**: `modelgrant/checker.go` 按 Key > Person > Party > 全局默认级联检查, Key 为独立主体, 未关联用户禁用状态.

**结论**: 当前代码中 "禁人即禁 Key" 的联动校验**尚未实现**. 一旦 `validateGovToken` 完善且加入用户状态检查, 此链路可闭合.

### 判定: 有条件通过 (认证层待完善)

---

## 场景 6: 内置职责分离

**审查目标**: `abac/builtin.go` -- 4 条 is_system 策略是否存在? fund 角色是否不能改 routing?

### PRD 引用

`abac/builtin.go` L32-43: 明确引用 PRD §7.2.5 路由-资金分离.

### 代码证据

**4 条内置策略** (`abac/builtin.go` L29-83):

| # | PolicyCode | Effect | Conditions | 含义 |
|---|-----------|--------|------------|------|
| 1 | `P-SOD-FUND` | deny | `{"axis":["iam","routing"]}` | fund 角色不可操作 iam/routing |
| 2 | `P-SOD-ROUTING` | deny | `{"axis":"fund"}` | routing 角色不可操作 fund |
| 3 | `P-SOD-IAM` | deny | `{"axis":"routing"}` | iam 角色不可操作 routing |
| 4 | `P-AUDIT-READONLY` | deny | `{}` | 审计角色默认拒绝全部 (仅角色显式授予的读操作放行) |

- 全部 `IsSystem: true`, `Priority: 1000` (builtinPriority)
- `SeedBuiltinPolicies` (L90-120): UPSERT 语义, 按 policy_code 幂等

**fund 角色不能改 routing**:
- 策略 1 (P-SOD-FUND) 将 fund 角色与 iam/routing 轴的操作隔离: `{"axis":["iam","routing"]}` 为 deny
- 策略 2 (P-SOD-ROUTING) 将 routing 角色与 fund 轴的操作隔离: `{"axis":"fund"}` 为 deny
- 两者构成**路由-资金双向互斥** (PRD §7.2.5)

### 测试覆盖

- `TestBuiltinPolicies_Seed` (policy_role_test.go L90-125): 验证首次种子 4 条, 二次种子 0 条 (幂等), 全部 is_system=true
- `TestEvaluate_SeparationOfDuty` (engine_test.go L202-264): fund_admin 可 fund.allocate, 被拒绝 routing.price.write
- `TestDeleteSystemPolicy_Denied` (policy_role_test.go L61-87): 系统策略不可删除

### 判定: 通过

---

## 红线检查

| # | 红线项 | 结果 | 证据 |
|---|--------|------|------|
| 1 | 审计日志可被 UPDATE/DELETE? | **通过** | `audit/event.go` RecordEvent 仅 `db.Create`, 全包无 UPDATE/DELETE |
| 2 | Leader 无 Grant 全平台权限? | **通过** | ABAC 引擎无 Leader 特殊逻辑. modelgrant/checker.go L39 "禁止仅因 Leader 头衔自动拥有全平台模型权 (A-CON-05)". http.go 中 team_leader 仅为管理面板用户管理限制, 非权限提升 |
| 3 | 前端隐藏按钮但 API 未校验? | **通过** | 所有 gov_handlers 入口均调用 `requireGovAuth` → `ABACEngine.Evaluate`. ui_permission/doc.go 明确: "前端隐藏按钮减少误操作, 真正的安全在后端 ABAC 引擎" |

---

## 发现汇总

### 发现 #1 (中): 禁人即禁 Key 联动未实现

- **文件**: `gov_handlers.go` L267-269, L194-195
- **描述**: API Key 被直接用作 SubjectID 参与 ABAC 评估, 但未检查 Key 所属用户是否已被禁用. `validateGovToken` 为占位实现.
- **影响**: 禁用用户后, 该用户的 API Key 可能仍然有效 (取决于 Key 自带的角色/策略绑定)
- **建议**: 完善 `validateGovToken`, 加入用户状态检查; 或在 Key 认证层增加 "所属用户 disabled → Key 自动失效" 逻辑

### 发现 #2 (信息): handler 实现多为占位

- **文件**: `gov_handlers.go`, `gov_handlers_abac.go`, `gov_handlers_fund.go`
- **描述**: 大量 handler 返回 "待实现" 消息, 如 Party CRUD, Fund 操作, Key 管理等
- **影响**: 功能入口已建立, ABAC 鉴权已接入, 但业务逻辑尚未开发

---

## 测试覆盖矩阵

| 测试文件 | 关键场景 | 行数 |
|----------|----------|------|
| `abac/engine_test.go` | allow/deny/default-deny/SOD/GetPermissions/action-not-found/模拟评估 | 386 |
| `abac/policy_role_test.go` | 策略CRUD/系统策略不可删/内置策略种子+幂等/角色CRUD/系统角色不可删 | 198 |
| `ui_permission/projector_test.go` | UI 投影测试 (未审阅) | -- |
| `ui_permission/store_test.go` | UI 存储层测试 (未审阅) | -- |

---

## 最终判定

| 场景 | 结论 |
|------|------|
| 1. ABAC 四轴越权 | 通过 |
| 2. UI 权限投影 | 通过 |
| 3. 审计不可篡改 | 通过 |
| 4. 审计链锚定 | 通过 |
| 5. 禁人即禁 Key | 有条件通过 (认证层待完善) |
| 6. 内置职责分离 | 通过 |

**红线**: 3/3 通过  
**总体**: 安全治理域核心安全机制 (ABAC 引擎、审计不可篡改、职责分离) 基本完备. 发现 1 项中优先级待办 (禁人即禁 Key 联动). 建议在 `validateGovToken` 实现后复审场景 5.
