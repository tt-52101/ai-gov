# RED-6 单兵作战记录

| 属性 | 值 |
|------|-----|
| Agent ID | RED-6 |
| 审计域 | 综合漏洞扫描（TODO/FIXME/HACK/硬编码 secret + 前端白名单 + ABAC 完整性） |
| 执行时间 | 2026-07-31 |
| 审计文件 | 全部 `*.go` + `*.ts`/`*.tsx` |
| 扫描项 | 全量 |
| Release Blocker | 4 |

---

## 审计方法

三阶段扫描：(1) 全量 Go 文件静态扫描——TODO/FIXME/HACK 注释 + 硬编码 secret/key + 占位函数；(2) 前端 ABAC 路由白名单审计——console-router.tsx 逻辑审查；(3) ABAC 端点完整性——验证所有 gov handler 的 action 参数非空。

---

## 发现详情

### BLOCKER-1 (严重) `validateChannel()` 占位——可绕过 party 边关系校验

`fund/service.go` L286-298：`validateChannel` 函数体为占位实现，不执行任何实际校验。任何 channel 参数均可通过，绕过 party 边关系划拨约束。

### BLOCKER-2 (高危) `/v1/gov/ui-permissions/snapshot` 传入空 action 跳过 ABAC

`gov_handlers_abac.go` L220-223：`h.requireGovAuth(w, r, "")` 传入空字符串 action。`requireGovAuth`（gov_handlers.go L223）中 `action != ""` 条件为 false，ABAC 策略评估被完全跳过，仅校验 Token 有效性。

### BLOCKER-3 (高危) `console-router.tsx` FAIL-OPEN

`console-router.tsx` L56-58：ABAC 权限 API 不可用时静默放行所有页面访问。正确行为应为 fail-closed——API 不可用时拒绝访问或仅允许 `/gov/dashboard`。

### BLOCKER-4 (高危) `config.go` AdminToken/SecretKey 代码内嵌默认值

`config.go` L83-90：当环境变量未设置时，回退到硬编码的默认值。生产环境中遗漏环境变量配置将导致使用可预测的默认凭据。

---

## 关键代码位置

| 文件 | 行号 | 内容 |
|------|------|------|
| `fund/service.go` | 286-298 | validateChannel 占位 |
| `gov_handlers_abac.go` | 220-223 | snapshot 传入空 action |
| `console-router.tsx` | 56-58 | FAIL-OPEN |
| `config.go` | 83-90 | 硬编码默认凭据 |
