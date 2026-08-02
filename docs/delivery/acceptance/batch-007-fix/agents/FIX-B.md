# FIX-B 单兵作战记录

| 属性 | 值 |
|------|-----|
| Agent ID | FIX-B |
| 修复域 | ModelGrant + 双层预算 + 级联 + 配额竞态 |
| 修改文件 | `pipeline_handler.go`、`pipeline.go`、`http.go`、`store_integration.go`、`checker.go`、`model.go` |
| 修复数 | 6 文件 |
| 对应漏洞 | R6-02（V-4.1）、R6-03（V-4.2）、R6-10（V-4.3）、R6-19（V-4.5） |

## 修复要点

1. **stream 绕过**：fallbackChatCompletions 开头注入 ModelGrant.CheckAccess
2. **Pipeline 失败降级**：ModelGrant 拒绝 → 403 不降级
3. **模型级配额**：Pipeline 新增步骤 [6.5] QuotaCheck
4. **账户级预算帽**：CheckBudgetCap stub → 完整预算周期+超限检查
5. **级联修复**：移除 `if typ == principal.Type` 单层级守卫
6. **乐观锁**：ConsumeQuota WHERE id=? AND version=?
