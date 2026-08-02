# FIX-A 单兵作战记录

| 属性 | 值 |
|------|-----|
| Agent ID | FIX-A |
| 修复域 | 鉴权返回值丢弃 → `if _, ok := ...; !ok { return }` |
| 修改文件 | `gov_handlers.go`、`gov_handlers_fund.go`、`gov_handlers_abac.go` |
| 修复数 | 65 处 |
| 对应漏洞 | R6-01（D-CON-01、R2-4/R2-5/R2-6、RED-3 新增） |

## 修复方法

全量扫描 `_, _ = h.requireGovAuth(` 和 `_, _ = h.requireGovItemAuth(` 模式 → 逐一替换为 `if _, ok := ...; !ok { return }`。

## 验证

- `grep "_, _ = h.requireGovAuth\|_, _ = h.requireGovItemAuth"` → 零匹配
- `go vet ./internal/server/...` → 无编译错误
