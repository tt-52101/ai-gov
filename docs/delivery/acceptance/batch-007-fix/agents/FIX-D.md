# FIX-D 单兵作战记录

| 属性 | 值 |
|------|-----|
| Agent ID | FIX-D |
| 修复域 | 前端 FAIL-OPEN + Cookie + snapshot ABAC |
| 修改文件 | `gov_handlers_abac.go`、`console-router.tsx`、`http.go` |
| 修复数 | 3 漏洞 |
| 对应漏洞 | R6-07、R6-08、R6-09 |

## 修复要点

1. **snapshot ABAC**：空 action `""` → `"data.ui.read"`
2. **FAIL-OPEN → FAIL-CLOSED**：3 次重试 + 失败后仅放行 /gov/dashboard + 错误页 UI
3. **Cookie 安全**：后端 login/logout/OAuth 三处 Set-Cookie：HttpOnly/Secure/SameSite=Strict；前端零 JS cookie
