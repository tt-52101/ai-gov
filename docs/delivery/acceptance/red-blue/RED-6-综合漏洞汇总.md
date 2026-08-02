# RED-6 综合漏洞扫描汇总报告

| 项 | 内容 |
|----|------|
| 审计编号 | RED-6 |
| 审计范围 | 综合漏洞扫描——技术债务、硬编码密钥、前端路由白名单、RED-1~3 汇总 |
| 审计日期 | 2026-07-31 |
| 扫描代码范围 | `backend/internal/server/`（所有 .go） + `frontend/`（所有 .tsx/.ts） |
| 总发现数 | 22（RED-1~3 遗留 17 + RED-6 新增 5） |

---

## 第一部分：TODO/FIXME/HACK 技术债务扫描

**命令**: `grep -rn "TODO|FIXME|HACK|XXX" --include="*.go" --include="*.tsx" --include="*.ts"`（已排除 node_modules 和 .next）

### 发现 T.1 -- Dashboard 环比数据均为硬编码占位值

| 项目 | 内容 |
|------|------|
| 严重程度 | 低 |
| 文件 | `frontend/app/(console)/gov/dashboard/page.tsx` |
| 行号 | 173, 181, 189, 197 |
| 代码 | `trend={12.5 /* TODO: 替换为真实环比数据，需要后端提供上期基准值 */ }` 等 4 处 |

**详情**: 仪表盘 4 个统计卡片（总预算池、累计消耗、活跃主体、待审批）的 trend 值均为硬编码浮点数，标注为 TODO 待替换。不影响安全性，但属于已知技术债务。

**修复建议**: 后端需提供 `GET /gov/dashboard/stats` 返回 `current_value` 和 `previous_value` 字段，前端计算 `trend = (current - previous) / previous * 100`。

---

### 发现 T.2 -- CSP 安全策略标记为 TODO v1.2

| 项目 | 内容 |
|------|------|
| 严重程度 | 中 |
| 文件 | `web/guardian-gateway/src/proxy.ts` |
| 行号 | 19, 49 |

```typescript
// break 全站。CSP 留待 v1.2 配合 nonce 化后启用 (TODO v1.2)。
// CSP — TODO v1.2 (需配合 next/script nonce + Tailwind 安全配置, 否则 break 全站)
```

**详情**: Content-Security-Policy 头未启用。项目标注将在 v1.2 版本中配合 nonce 机制启用。当前全站无 CSP 保护，增加了 XSS 和代码注入的风险敞口。

**修复建议**: v1.2 前至少启用报告模式（`Content-Security-Policy-Report-Only`），先收集违规日志而不阻断功能。

---

### 发现 T.3 -- 前端通知渠道 placeholder URL

| 项目 | 内容 |
|------|------|
| 严重程度 | 低 |
| 文件 | `frontend/features/admin/domain/catalog.tsx` |
| 行号 | 276 |

**详情**: Discord 通知渠道的 webhook URL 填充为全零和全 X 占位符：
```
discord: "https://discord.com/api/webhooks/000000000000000000/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
```
其他渠道（飞书 L272、钉钉 L273、企业微信 L274）使用 `xxxx` 占位符 Slack (L275) 使用 `REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL`，均无害但不符合生产标准。这不是硬编码泄露（无有效凭证），但占位符可能误导运维人员。

---

## 第二部分：硬编码 Secret/Token 扫描

**命令**: `grep -rn "password|secret|token|api_key" --include="*.go" --include="*.tsx"`（已排除 struct tag 和环境变量 fallback 噪声）

### 发现 S.1 -- 默认密钥通过环境变量 fallback 暴露

| 项目 | 内容 |
|------|------|
| 严重程度 | 高 |
| 文件 | `backend/internal/server/config.go` |
| 行号 | 83, 84, 90 |

```go
AdminToken:             getenv("TOKENHUB_ADMIN_TOKEN", "dev_admin_token"),              // L83
BootstrapAdminPassword: getenv("TOKENHUB_BOOTSTRAP_ADMIN_PASSWORD", "admin123456"),      // L84
SecretKey:              getenv("TOKENHUB_SECRET_KEY", "dev_tokenhub_secret_key"),       // L90
```

**详情**: 三项核心密钥均在 `getenv` 调用中提供代码内嵌 fallback 默认值。虽然有 `ValidateForStartup()`（L119-153）在生产环境调用 `weakProductionSecretReason` 做长度/占位符检测，但仍存在以下风险：

1. 若 `TOKENHUB_ENV` 未正确设为生产环境值（如误设 `dev`），`ValidateForStartup` 直接放行（L136-137）
2. 若部署配置中遗漏环境变量设置，服务将静默使用代码内嵌的弱密钥启动
3. `BootstrapAdminPassword` 默认值 `"admin123456"` 仅为 12 字节阈值（L146），刚好满足生产最低长度要求

**Positives**: `www.authenticate()` 机制（L140-151）对 `"dev"`、`"development"`、`"local"`、`"test"` 以外的环境要求密钥长度 >= 32（token/secret）或 >= 12（password），并检查是否为已知占位符。这是防御层，但依赖于 `TOKENHUB_ENV` 正确设置。

**修复建议**: 
- 将 `BootstrapAdminPassword` 默认值也改为更长的占位符（如 `"change-me-tokenhub-admin-password"`），避免刚好满足最低阈值
- 考虑在启动阶段检测到使用默认值时输出醒目的 WARNING 而不是仅依赖 `ValidateForStartup`

---

### 发现 S.2 -- 测试认证凭证被注入到 live 命令环境

| 项目 | 内容 |
|------|------|
| 严重程度 | 低 |
| 文件 | `backend/internal/server/anthropic_messages_live_test.go` |
| 行号 | 348, 352 |

```go
"TOKENHUB_LIVE_ARK_API_KEY":          "upstream-secret-must-not-leak",    // L348
"TOKENHUB_UNRELATED_TEST_CREDENTIAL": "unrelated-secret-must-not-leak",  // L352
```

**详情**: 该测试文件（`TestLiveCommandEnvironmentExcludesCredentialsAndDefaultAliases`）通过 `t.Setenv` 注入伪造凭证到测试环境，然后验证 `liveCommandEnvironment()` 函数不会将这些环境变量泄漏到子命令中。这是**测试本身验证泄漏防护**的正面用例——凭证名中的 `must-not-leak` 即是对预期的显式标记。真正的测试对错误的防护是验证通过，而非代码含漏洞。

**判定**: 此为安全防护的测试用例，非漏洞。`t.Setenv` 在测试结束后自动恢复环境变量。

---

### 发现 S.3 -- 通知渠道 webhook/access_token 占位符可见于前端

| 项目 | 内容 |
|------|------|
| 严重程度 | 低 |
| 文件 | `frontend/features/admin/domain/catalog.tsx` |
| 行号 | 271-278 |

**详情**: 通知渠道 URL 映射对象中，飞书（L272）和钉钉（L273）包含形如 `xxxxxxxx` 的占位 token 参数，企业微信（L274）包含形如 `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` 的占位 key。Slack（L275）为 `REPLACE_WITH_YOUR_SLACK_WEBHOOK_URL`，均为明显占位符。无有效凭证泄露。

**判定**: 无风险。占位符仅为 UI 表单的示例值，不含有效凭证。

---

## 第三部分：前端 Console-Router 白名单逻辑审查

**审查文件**: `frontend/app/(console)/console-router.tsx`

### 发现 R.1 -- ABAC 权限网关为 FAIL-OPEN 模式

| 项目 | 内容 |
|------|------|
| 严重程度 | 高 |
| 文件 | `frontend/app/(console)/console-router.tsx` |
| 行号 | 56-58 |

```typescript
.catch(() => {
  // API 不可用时回退——允许访问（避免锁死）
  if (!cancelled) setAbacReady(true);
});
```

**详情**: 当 `/v1/gov/ui-permissions/snapshot` API 不可用时（超时、网络错误、HTTP 错误），catch 分支直接设置 `abacReady = true`，跳过权限检查，允许访问所有页面。这是一个典型的 FAIL-OPEN 设计，注释明确标注"避免锁死"。

攻击路径：
1. 攻击者通过 DDoS 使后端 API 暂时不可用，或触发 `/gov/ui-permissions/snapshot` 端点返回错误
2. 前端 catch 分支触发，`setAbacReady(true)` + `abacDenied` 保持 `false`
3. 用户无需 ABAC 校验即可访问旧系统的所有受限页面（如 /projects、/users、/api-keys）

**对比**: gov 路由 (`isGovRoute`) 在第 33-35 行选择直接标记 ready（因为 `/gov/*` 由后端 handler 做 ABAC 鉴权），这是合理的。但非 gov 路由的旧 AdminConsole SPA 完全依赖前端白名单，FAIL-OPEN 是高风险决策。

**修复建议**:
1. 将 catch 分支改为 FAIL-CLOSED：API 不可用时拒绝访问（设置 `abacDenied = true`），重定向到 `/gov/dashboard` 或显示错误提示
2. 或至少增加重试逻辑（指数退避 + 最大 3 次）后再决定放行还是阻断

---

### 发现 R.2 -- UI 权限快照端点后端为空实现

| 项目 | 内容 |
|------|------|
| 严重程度 | 高 |
| 文件 | `backend/internal/server/gov_handlers_abac.go` |
| 行号 | 220-223 |

```go
func (h *GovHandler) handleUIPermissionSnapshot(w http.ResponseWriter, r *http.Request) {
    _, _ = h.requireGovAuth(w, r, "")  // action = "" → 跳过 ABAC 评估！
    okJSON(w, map[string]string{"message": "UI 权限快照——待实现"})
}
```

**详情**: 此端点返回占位 JSON 而非实际权限数据。前端的 `/v1/gov/ui-permissions/snapshot` 请求将收到 `{"message": "UI 权限快照——待实现"}` 而非 `{"routes": [...]}`。结合 R.1 的 FAIL-OPEN 策略，这意味着：

1. 正常流程（API 返回 HTTP 200）：响应不包含 `routes` 字段 → `data.routes` 为 `undefined` → `allowed` 为 `[]` → `allowed.length > 0` 为 `false` → 跳过白名单检查 → `abacReady = true` + `abacDenied = false` → 允许访问**所有**页面
2. 错误流程（API 返回错误）：catch 分支直接放行

两条路径均导致白名单机制形同虚设。

**修复建议**:
1. 后端实现 `handleUIPermissionSnapshot`：调用 `GetPermissions` 返回当前用户的 UI 路由白名单
2. 使用正确的 action 值（建议 `iam.ui.read`）

---

### 发现 R.3 -- 路由前缀匹配存在绕过风险

| 项目 | 内容 |
|------|------|
| 严重程度 | 低 |
| 文件 | `frontend/app/(console)/console-router.tsx` |
| 行号 | 48-49 |

```typescript
const isAllowed = allowed.some(
  (r) => r.route_path === pathname || pathname.startsWith(r.route_path + "/")
);
```

**详情**: 当前使用前缀匹配（`startsWith`）做路由白名单判断。例如若 `r.route_path = "/api-keys"`，则 `/api-keys-legacy` 也会被误放行。这是 Next.js App Router 已知的边界情况，当前项目路由级别暂无 `/api-keys` 与 `/api-keys-legacy` 同时存在的情况，但属于架构层面的设计缺陷。

**修复建议**: 改用完整路径匹配或使用 Next.js App Router 的 `pathname.match()` 精确匹配。

---

## 第四部分：RED-1~3 历史发现汇总

> 注: RED-4 和 RED-5 报告未在 `docs/delivery/acceptance/red-blue/` 目录中发现（仅存在 RED-1、RED-2、RED-3、RED-3-复验）。若后续产生，请追加至本矩阵。

### 4.1 整体分级矩阵

| # | 来源 | 攻击面/发现 | 状态 | 严重程度 |
|---|------|-----------|------|---------|
| 1 | RED-1 1.1 | IDOR——全端点无资源归属校验 | 已关闭（RED-3 复验确认） | 高 |
| 2 | RED-1 2.1 | ABAC 绕过——/gov/ui-permissions/snapshot 空 action | **未修复** | 高 |
| 3 | RED-1 2.2 | ABAC 绕过——无全局中间件兜底 | 未修复 | 中 |
| 4 | RED-1 2.3 | ABAC 绕过——ABACEngine=nil 跳过鉴权 | 未修复 | 中 |
| 5 | RED-1 3.1 | 条件注入——conditions_json 无 schema 校验 | 未修复 | 低 |
| 6 | RED-1 4.1 | Key 伪造——治理 API 不校验 Key 哈希 | **未修复** | 高 |
| 7 | RED-1 4.2 | Key 伪造——裸 SHA-256 无盐值 | 未修复 | 中 |
| 8 | RED-1 4.3 | Key 伪造——时序攻击面 | 未修复 | 低 |
| 9 | RED-1 5.1 | 未注册动作——正确默认拒绝 | 已防御 | -- |
| 10 | RED-1 S.1 | handleAccountItem 写操作共用只读权限 | 已关闭（RED-3 复验确认） | 高 |
| 11 | RED-2 1 | 重复划拨——幂等键可选 | **未修复** | 高 |
| 12 | RED-2 2 | 并发竞态——Settle freeze 未锁定 | 未修复 | 中 |
| 13 | RED-2 5 | 绕过通道校验 (validateChannel 占位) | **未修复** | 严重 |
| 14 | RED-3 1 | INTERNAL_ONLY 出网管控失效 | 已关闭（RED-3 复验确认） | 严重 |
| 15 | RED-3 2 | 密钥明文泄漏 | 通过（无风险） | -- |
| 16 | RED-3 3 | 审计日志篡改 | 通过（已防御） | -- |
| 17 | RED-3 4 | SQL 注入 | 通过（已防御） | -- |
| 18 | RED-3 5 | 日志敏感信息 | 未修复 | 低 |
| 19 | RED-3 6 | 错误信息泄露——内部标识暴露 | 已关闭（RED-3 复验确认） | 高 |

### 4.2 RED-3 复验结论

| 缺陷编号 | 状态 | 关键修复 |
|---------|------|---------|
| FIX-B | 已关闭 | pipeline.go L263-287 注入 network_class + 调用 CheckEgress；compliance.go L29-46 消费 CtxKeyNetworkClass 剔除 external 候选 |
| FIX-D | 已关闭 | engine.go L225-229 scope_party_id 过滤；18 个单品端点接入 requireGovItemAuth；sanitizeError 脱敏 |

---

## 第五部分：修复优先级清单

### 严重 — 阻塞发布（release blocker）

| 优先级 | 编号 | 问题 | 文件 | 行号 | 修复方向 |
|--------|------|------|------|------|---------|
| 严重 | RED-2.5 | 通道校验占位——`validateChannel()` 不查询 party_edges | `backend/internal/server/fund/service.go` | 286-298 | 在 Allocate 事务中调用 `party.Service.CanAllocate()` 或内联边查询 |
| 高 | RED-1.2.1 | `/gov/ui-permissions/snapshot` 空 action 绕过 ABAC | `backend/internal/server/gov_handlers_abac.go` | 220-223 | 传入正确的 action 值（如 `iam.ui.read`） |
| 高 | RED-6.R.1 | 前端 ABAC 权限网关 FAIL-OPEN | `frontend/app/(console)/console-router.tsx` | 56-58 | catch 改为 FAIL-CLOSED 或增加重试 |
| 高 | RED-6.R.2 | UI 权限快照后端为空实现，前端无法获取白名单 | `backend/internal/server/gov_handlers_abac.go` | 220-223 | 实现 `GetPermissions` 投影逻辑，返回实际 routes 数组 |

### 高 — 必须修复（下一迭代）

| 优先级 | 编号 | 问题 | 文件 | 行号 | 修复方向 |
|--------|------|------|------|------|---------|
| 高 | RED-1.4.1 | 治理 API X-API-Key 不校验哈希 | `backend/internal/server/gov_handlers.go` | 194-195 | 对 X-API-Key 值做 SHA-256 哈希后查询 gov_api_keys 表 |
| 高 | RED-2.1 | IdempotencyKey 可选——允许重复划拨 | `backend/internal/server/fund/service.go` | 82 | 强制要求幂等键（或拒绝无幂等键的 Allocate 请求） |
| 高 | RED-6.S.1 | 默认密钥 fallback 值存在于代码中 | `backend/internal/server/config.go` | 83-90 | 将 fallback 值改为更明显的占位符；`ValidateForStartup` 增加更严格的默认值检测 |

### 中 — 建议修复（技术债务收敛）

| 优先级 | 编号 | 问题 | 文件 | 行号 | 修复方向 |
|--------|------|------|------|------|---------|
| 中 | RED-1.2.2 | gov 路由无全局鉴权中间件兜底 | `backend/internal/server/gov_handlers.go` | 167-172 | 在 `wrapGovHandler` 中注入默认鉴权检查 |
| 中 | RED-1.2.3 | ABACEngine=nil 时跳过鉴权 | `backend/internal/server/gov_handlers.go` | 202 | 生产环境强制要求 ABACEngine 非 nil，否则返回 503 |
| 中 | RED-1.4.2 | HashSecret 使用裸 SHA-256 无盐值 | `backend/internal/server/types.go` | 1122-1125 | 改为 HMAC-SHA256 配合每密钥独立随机盐值 |
| 中 | RED-2.2 | Settle freeze 读取无行锁 | `backend/internal/server/fund/sqlstore/pg.go` | 181 | GetFreeze 增加 FOR UPDATE 或 UpdateFreezeStatus 增加状态前置条件 |
| 中 | RED-6.T.2 | CSP 头未启用（TODO v1.2） | `web/guardian-gateway/src/proxy.ts` | 19, 49 | v1.2 前启用 CSP Report-Only 模式 |

### 低 — 优化项

| 优先级 | 编号 | 问题 | 文件 | 行号 | 修复方向 |
|--------|------|------|------|------|---------|
| 低 | RED-1.3.1 | conditions_json 无 schema 校验 | `backend/internal/server/abac/policy.go` | 36-38 | 增加 JSON Schema 校验（axis、actions、resource_type 三字段） |
| 低 | RED-1.4.3 | ValidateAPIKey 时序攻击面 | `backend/internal/server/store.go` | 1537 | 使用 `crypto/subtle.ConstantTimeCompare` |
| 低 | RED-3.5 | 日志含内部系统 ID | `backend/internal/server/fund/service.go` 等 | 多处 | 非开发环境对 account_id/freeze_id 做哈希截断 |
| 低 | RED-6.T.1 | Dashboard trend 硬编码占位 | `frontend/app/(console)/gov/dashboard/page.tsx` | 173,181,189,197 | 后端提供上期基准值 API |
| 低 | RED-6.T.3 | 通知渠道 placeholder URL | `frontend/features/admin/domain/catalog.tsx` | 271-278 | 替换为有意义的示例值或空白字符串 |
| 低 | RED-6.R.3 | 路由前缀匹配边界风险 | `frontend/app/(console)/console-router.tsx` | 48-49 | 改用精确匹配或 pathname.match() |

---

## 第六部分：防御有效性评估

### 已确认为防御有效的机制

| 防御机制 | 位置 | 说明 |
|---------|------|------|
| 未注册 action 默认拒绝 | `abac/engine.go:185-197, 52-57` | `ErrActionNotFound` 直接导致评估失败 |
| 负/零金额拒绝 | `fund/service.go:74-76` `fund/freeze.go:34-36` | `LessThanOrEqual` 全覆盖 |
| 自划拨拒绝 | `fund/service.go:77-79` `fund/lifecycle.go:233-235` | 双入口均校验 |
| 死锁预防（加锁排序） | `fund/service.go:112-115` | 字典序固定锁定顺序 |
| 乐观锁（version 检查） | `fund/sqlstore/pg.go:109-112` | 收支平衡更新检查 version |
| SELECT FOR UPDATE | `fund/sqlstore/pg.go:91-103` | 行锁防并发竞态 |
| 参数化查询（防 SQL 注入） | 全量 GORM 查询 | 零拼接 SQL |
| 审计日志不可篡改 | `audit/event.go:36-59` | 仅 INSERT，无 UPDATE/DELETE |
| 出网管控（INTERNAL_ONLY） | `pipeline.go:263-287` | 经 RED-3 复验确认已修复 |
| scope_party_id 过滤 | `abac/engine.go:225-229` | 经 RED-3 复验确认已修复 |
| 单品端点归属校验 | `gov_handlers.go:230-269` | 18 个端点均已接入 requireGovItemAuth |
| 错误脱敏 | `gov_handlers.go:346-358` | sanitizeError 对非 HTTPError 返回通用消息 |
| 审计敏感键过滤 | `http.go:8643-8651` | isSensitiveAuditKey 对 authorization/password/secret 等脱敏 |
| 生产环境密钥校验 | `config.go:119-153` | ValidateForStartup 检查默认值和最小长度 |

### 确认为防御缺失的关键缺口

| 缺口 | 来源 | 说明 |
|------|------|------|
| 资金通道校验完全绕过 | RED-2.5 | validateChannel 是占位实现 |
| 治理 API 认证缺失 | RED-1.4.1 | X-API-Key 不校验哈希 |
| UI 权限快照空实现 | RED-1.2.1 / RED-6.R.2 | 端点无 ABAC + 返回占位 JSON |
| 前端权限网关 FAIL-OPEN | RED-6.R.1 | API 不可用时放行所有页面 |
| 幂等键可选导致可重复划拨 | RED-2.1 | 无幂等键时不拒绝 |

---

## 附录：扫描文件清单

| 文件 | 扫描内容 | 绝对路径 |
|------|---------|---------|
| console-router.tsx | ABAC 白名单 / FAIL-OPEN | `D:\ai-work\grok\a-gov\ai-gov-fusion\frontend\app\(console)\console-router.tsx` |
| dashboard/page.tsx | TODO 技术债务 | `D:\ai-work\grok\a-gov\ai-gov-fusion\frontend\app\(console)\gov\dashboard\page.tsx` |
| catalog.tsx | 通知渠道 placeholder URL | `D:\ai-work\grok\a-gov\ai-gov-fusion\frontend\features\admin\domain\catalog.tsx` |
| config.go | 默认密钥 fallback | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\config.go` |
| anthropic_messages_live_test.go | 测试凭证泄漏防护验证 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\anthropic_messages_live_test.go` |
| gov_handlers_abac.go | UI 权限快照空实现 | `D:\ai-work\grok\a-gov\ai-gov-fusion\backend\internal\server\gov_handlers_abac.go` |
| proxy.ts | CSP TODO v1.2 | `D:\ai-work\grok\a-gov\ai-gov-fusion\web\guardian-gateway\src\proxy.ts` |
| RED-1-权限提升.md | 历史发现引用 | `D:\ai-work\grok\a-gov\docs\delivery\acceptance\red-blue\RED-1-权限提升.md` |
| RED-2-资金攻击.md | 历史发现引用 | `D:\ai-work\grok\a-gov\docs\delivery\acceptance\red-blue\RED-2-资金攻击.md` |
| RED-3-数据泄露.md | 历史发现引用 | `D:\ai-work\grok\a-gov\docs\delivery\acceptance\red-blue\RED-3-数据泄露.md` |
| RED-3-数据泄露-复验.md | RED-3 复验结论 | `D:\ai-work\grok\a-gov\docs\delivery\acceptance\red-blue\RED-3-数据泄露-复验.md` |

---

*报告结束。RED-1~3 历史发现中 6 项已在 RED-3 复验中关闭。当前仍有 4 项严重/高级别缺口（RED-2.5、RED-1.2.1、RED-6.R.1、RED-6.R.2）属于 Release Blocker，建议优先修复。*
