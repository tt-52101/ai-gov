# RED-2 单兵作战记录

| 属性 | 值 |
|------|-----|
| Agent ID | RED-2 |
| 审计域 | Cookie 安全 + 前端白名单 + 鉴权返回值完整性 |
| 执行时间 | 2026-07-31 |
| 审计文件 | `middleware.ts`, `console-router.tsx`, `gov_handlers.go`, `gov_handlers_fund.go`, `gov_handlers_abac.go` |
| 攻击面 | 6 |
| 严重 | 2 |
| 高危 | 2 |
| 中危 | 2 |

---

## 审计方法

三大攻击面独立审计：(1) Cookie 安全配置——检查 Set-Cookie 头、HttpOnly/Secure/SameSite 属性；(2) 前端路由白名单——检查 console-router.tsx 的 fail-open 行为；(3) 鉴权返回值完整性——全量扫描 `requireGovAuth`/`requireGovItemAuth` 调用，验证是否检查返回值并 return。

---

## 发现详情

### R2-1 (高危) Cookie 无 HttpOnly/Secure/SameSite

后端从未通过 `Set-Cookie` 设置 cookie。`gov_session`/`tokenhub_session` 完全由前端 JavaScript 操作，不可能具有 HttpOnly 属性。XSS 攻击可直接窃取 session。

### R2-2 (高危) 前端白名单 fail-open

`console-router.tsx` L56-58：权限 API 不可用时直接放行所有路由访问，违反最小权限原则。应 fail-closed（API 不可用时拒绝访问或仅允许 dashboard）。

### R2-3 (中危) UI 权限快照跳过 ABAC

`gov_handlers_abac.go` L1040：`handleUIPermissionsSnapshot` 传入空 action 字符串 `""`，`requireGovAuth` 中 `action != ""` 条件为 false，跳过 ABAC 策略评估，仅校验 Token 有效性。

### R2-4 (严重) 9 个 DELETE 端点可无认证越权删除

使用 `_, _ = h.requireGovItemAuth(...)` 模式丢弃鉴权返回值的端点：
- 删除角色：`gov_handlers_abac.go` L276
- 删除策略：L429
- 撤销角色绑定：L606
- 删除授权记录：L716
- 删除 UI 菜单：L823
- 删除 UI 路由：L922
- 删除 UI 按钮绑定：L1021
- 移除 Party 成员：`gov_handlers.go` L822
- 删除关系边：L743

鉴权函数向 ResponseWriter 写入 403，但 handler 不 return，后续 DELETE 操作仍执行。攻击者收到 403 响应但数据已被删除。

### R2-5 (严重) 3 个 POST/CREATE 端点同样越权

- UI 菜单创建：`gov_handlers_abac.go` L745
- UI 路由创建：L844
- UI 按钮绑定创建：L943

### R2-6 (中危) ~25 个 GET handler 忽略鉴权返回值

虽无数据修改风险，但作为信息泄露面存在。违反最小权限原则。

---

## 根本原因

错误模式 `_, _ = h.requireGovAuth(...)` 在整个代码库中被大规模复制粘贴。正确模式应为：

```go
gctx, ok := h.requireGovAuth(w, r, "action.code")
if !ok {
    return
}
```

建议将鉴权逻辑从 handler 内手动调用重构为 middleware 拦截。
