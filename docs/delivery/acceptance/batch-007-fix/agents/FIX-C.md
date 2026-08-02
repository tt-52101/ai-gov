# FIX-C 单兵作战记录

| 属性 | 值 |
|------|-----|
| Agent ID | FIX-C |
| 修复域 | validateChannel + SOD 绑定 + 硬编码凭据 |
| 修改文件 | `party/service.go`、`fund/service.go`、`abac/builtin.go`、`seed.go`、`config.go`、`http.go`、`image_generation.go` |
| 修复数 | 7 文件 + 2 test |
| 对应漏洞 | R6-05（BLOCKER-1）、R6-04（AP-3）、R6-06（BLOCKER-4） |

## 修复要点

1. **validateChannel**：实现 channel→edge-type 映射校验 + party_edges 表查询验证 AllowsFund
2. **SOD 绑定**：SeedBuiltinPolicies 创建 4 个 SOD 角色 + 4 条策略绑定 + Bootstrap 集成
3. **硬编码凭据**：移除所有 fallback 默认值；ValidateForStartup 强制检查（dev 必填，非 dev 额外弱凭据检查）
