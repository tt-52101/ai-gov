# FIX-F 单兵作战记录

| 属性 | 值 |
|------|-----|
| Agent ID | FIX-F |
| 修复域 | 管线补偿 + 流式续期 + 快照 + 幂等 |
| 修改文件 | `pipeline.go`、`audit/event.go`、`fund/service.go` + test 文件 |
| 修复数 | 4 漏洞 |
| 对应漏洞 | R6-14、R6-15、R6-17、R6-22 |

## 修复要点

1. **冻结补偿**：Pipeline 新增 Unfreeze 字段 + defer 逻辑（独立 context.Background，10s 超时）
2. **流式续期**：goroutine 每 5 分钟调 StreamRenewal，管线返回时 renewCancel() 终止
3. **快照强校验**：RecordEvent 对配置变更操作强制 before/after 非空 + isConfigMutationAction 判断
4. **幂等键补偿**：IdempotencyChecker 新增 Release——事务失败回滚后释放已 Claim 的键
