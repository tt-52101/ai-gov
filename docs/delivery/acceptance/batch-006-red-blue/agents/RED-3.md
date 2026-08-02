# RED-3 单兵作战记录

| 属性 | 值 |
|------|-----|
| Agent ID | RED-3 |
| 审计域 | 数据越权攻击面（IDOR + 水平越权）+ batch-002 复验 + `_, _` 丢弃深度扫描 |
| 执行时间 | 2026-07-31 |
| 审计文件 | `gov_handlers.go`, `gov_handlers_fund.go`, `gov_handlers_abac.go`, `abac/engine.go`, `pipeline.go`, `security/egress.go`, `routing/strategies/compliance.go` |
| 发现 | 3 新增 + 2 已关闭 |

---

## Phase 1：batch-002 缺陷复验

### FIX-B：INTERNAL_ONLY 出网管控 → ✅ 已关闭

双重防护验证通过：
1. `pipeline.go` L264-287：`resolveNetworkClass` → CtxKeyNetworkClass → CheckEgress 阻断
2. `routing/strategies/compliance.go` L29-46：S-COMPLIANCE Filter 剔除 external 候选

### FIX-D：scope_party_id / IDOR / 错误脱敏 → ✅ 已关闭

- `abac/engine.go` L225-229：`resolveSubjectRoles` scope_party_id 过滤正确
- 18 个单品端点全部接入 `requireGovItemAuth`
- `sanitizeError`（gov_handlers.go L346-358）脱敏正确

---

## Phase 2：深层 `_, _` 丢弃扫描

在全量扫描中发现 **27 处** `requireGovItemAuth` 使用 `_, _` 模式丢弃返回值（范围远超 RED-2 报告的 12 处）：

| 端点 | 文件 | 行号 | HTTP 方法 | 副作用 |
|------|------|------|-----------|--------|
| handleGetAccount | gov_handlers_fund.go | 177 | GET | 读取敏感账户数据 |
| handleKeyItem DELETE | gov_handlers_fund.go | 682 | DELETE | 占位，补全后为实际删除 |
| handleAllocationItem GET | gov_handlers_fund.go | 624 | GET | 读取划拨记录 |
| handleModelGrantItem DELETE | gov_handlers_fund.go | 1061 | DELETE | **有实际 delete** |
| handleRouteProfileItem GET | gov_handlers_fund.go | 1149 | GET | 读取路由档案 |
| handleRouteProfileItem PUT | gov_handlers_fund.go | 1162 | PUT | **有实际 update** |
| handleRouteProfileItem DELETE | gov_handlers_fund.go | 1186 | DELETE | **有实际 delete** |

正确模式参考：`handlePartyItem`（gov_handlers.go L594/617/660）、`handleAllocate`（gov_handlers_fund.go L277）、`handleLiquidate`（gov_handlers_fund.go L375）均正确检查返回值。

---

## Phase 3：lookupResourceParty 覆盖度

`gov_handlers.go` L309-329 switch 语句缺失 `"party_edge"` 和 `"party_member"` 映射。这两种资源有明确 party 归属，但落入 default 返回 `""`，导致 ABAC scope_party_id 过滤失效。

---

## Phase 4：路由前缀不匹配

路由注册使用 `/v1/gov/parties/` 而 `extractItemID`（gov_handlers.go L423-443）使用 `/gov/parties` 前缀。若无反向代理剥离 `/v1` 前缀，所有单品端点 URL 解析均会失败。

---

## 关键代码位置

| 文件 | 行号 | 内容 |
|------|------|------|
| `gov_handlers_fund.go` | 177, 624, 682, 1061, 1149, 1162, 1186 | `_, _` 丢弃返回值（7 处关键） |
| `gov_handlers_abac.go` | 多处 | `_, _` 丢弃返回值（~20 处） |
| `gov_handlers.go` | 309-329 | lookupResourceParty 缺 party_edge/party_member |
| `gov_handlers.go` | 423-443 | extractItemID 前缀 `/gov/parties` vs 注册 `/v1/gov/parties/` |
