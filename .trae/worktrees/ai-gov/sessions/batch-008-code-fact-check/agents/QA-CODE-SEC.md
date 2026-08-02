# 单兵轨迹 · QA-CODE-SEC

| 项 | 值 |
|----|-----|
| Agent ID | QA-CODE-SEC |
| 所属蜂群 | batch-008 代码事实核查 |
| 作战指令 | 基于事实代码核查安全治理域（abac/authz/modelgrant/ui_permission/audit/security） |
| 执行耗时 | 约 4 分钟 |
| 验收判定 | ❌ 不通过——1 项 P0 + 3 项 P1 |

## 情报收集（逐文件）

### abac 包
- engine.go：Evaluate L52-135（83行略超标）评估顺序 deny→allow→role→default deny ✅；resolveSubjectRoles L208-236 scope_party_id 过滤 ✅
- model.go：四轴常量 L22-31 ✅
- builtin.go：SeedBuiltinPolicies L138-265（127行超标）4条SOD策略+4角色绑定 ✅
- policy.go/role.go：CRUD 完整 ✅

### authz 包
- grant.go：Evaluate L94-116 DENY 优先 ✅
- middleware.go：L44-96 鉴权中间件 ⚠️ 未集成 scope_party_id 过滤

### modelgrant 包
- checker.go：CheckAccess L45-79 DENY 优先 ✅；ConsumeQuota L133-175 乐观锁 WHERE version=? ✅；loadGrantsForCascade L180-212 无 if typ==principal.Type 守卫 ✅
- model.go：ModelGrant 含 PartyID 字段 L68 ✅（batch-007 FIX-E 修复确认）

### ui_permission 包
- projector.go：ProjectMenus/ProjectRoutes/ProjectActions ✅
- projector_test.go：L73,117,137,337,345 ❌ int64/string 类型错误
- store_test.go：L19,30,80,86,108 ❌ int64/string 类型错误

### audit 包
- event.go：RecordEvent L40-70 仅 INSERT ✅；isConfigMutationAction L85-92 快照强校验 ✅
- anchor.go：AnchorChain L30-73 SHA-256 哈希链 ✅

### security 包
- egress.go：CheckEgress L73-99 INTERNAL_ONLY 阻断 ✅；HYBRID_ALLOWED L86-91 P2骨架直接放行 ❌
- hooks.go：NoopHook L95-108 空实现 ❌

## 战果产出

| 文件 | 行数 | 关键发现 |
|------|------|---------|
| abac/engine.go | 302 | Evaluate 83行略超标；scope_party_id 实现完整 |
| abac/builtin.go | 265 | SeedBuiltinPolicies 127行超标（P2） |
| ui_permission/projector_test.go | 345 | 5处 int64/string 类型错误（P0-3） |
| ui_permission/store_test.go | 108 | 5处 int64/string 类型错误（P0-3） |
| security/egress.go | 99 | HYBRID_ALLOWED 白名单未实现（P1-3） |
| security/hooks.go | 164 | NoopHook 空实现（P1-4） |

## 发现结论

### P0 阻塞（1 项）
- P0-3: ui_permission 包测试代码类型不匹配（10+ 处），go vet/test 失败

### P1 重要（3 项）
- P1-3: SEC-02 出网白名单未实现
- P1-4: SEC-03 内容安全未实现
- P1-5: SEC-04 异常流量拦截未实现

## 编译/测试结果
- go build ✅
- go vet ❌（ui_permission 类型错误）
- go test abac/authz/modelgrant/audit/security ✅ PASS
- go test ui_permission ❌ FAIL [build failed]
