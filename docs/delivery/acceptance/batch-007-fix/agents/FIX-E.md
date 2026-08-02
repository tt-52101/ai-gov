# FIX-E 单兵作战记录

| 属性 | 值 |
|------|-----|
| Agent ID | FIX-E |
| 修复域 | 出网管控 + scope 鉴权 + IDOR fail-secure |
| 修改文件 | `pipeline.go`、`routing/strategy.go`、`gov_handlers.go`、`modelgrant/model.go`、`ai-gov-fusion-v3.2.sql` |
| 修复数 | 4 漏洞 + 2 Schema 列 |
| 对应漏洞 | R6-11、R6-12、R6-13、R6-16 |

## 修复要点

1. **出网默认**：移除 `modelName != ""` 跳过；默认 `INTERNAL_ONLY`（最严格）
2. **RouteProfile PartyID**：结构体 + SQL schema 添加 party_id 列
3. **IDOR fail-secure**：lookupResourceParty 签名 `(string, error)`，DB 故障 → 500 拒绝
4. **ModelGrant PartyID**：结构体 + SQL schema 添加 party_id 列 + 索引
