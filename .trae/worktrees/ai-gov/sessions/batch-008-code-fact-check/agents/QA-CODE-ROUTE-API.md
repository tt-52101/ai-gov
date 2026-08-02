# 单兵轨迹 · QA-CODE-ROUTE-API

| 项 | 值 |
|----|-----|
| Agent ID | QA-CODE-ROUTE-API |
| 所属蜂群 | batch-008 代码事实核查 |
| 作战指令 | 基于事实代码核查路由调度域与控制台API域（routing/pipeline/gov_handlers/http） |
| 执行耗时 | 约 5 分钟 |
| 验收判定 | ❌ 不通过——3 项 P0 + 5 项 P1 |

## 情报收集（逐文件）

### routing 包
- strategy.go：Strategy 接口 L61-72 ✅；MaxDeltaCap=0.20 L176 ✅；RouteProfile.PartyID L153-154 ✅
- registry.go：12策略注册 ✅
- profile.go：ExecuteProfile L198-311（113行超标）；applyPriceCap δ过滤 ✅；logDecision L404-418 ❌ 注释称持久化但仅 slog
- decision.go：Decision 结构体 ✅
- strategies/：12 策略全部真实实现 ✅；但零测试覆盖 ❌

### pipeline.go
- 14步管线编排器 L185-514 ✅ 真实实现
- [4] ModelGrant L306-313 ⚠️ 仅当 modelName != "" 时执行
- [6] 价格过滤δ L330 ⚠️ 注释"由 Router 内部处理"
- [8] 冻结 L358-367 ✅；defer 补偿 L372-400 ✅（R6-14修复）
- [11] 流式续期 L404-430 ✅ goroutine（R6-15修复）
- Execute 函数 L237-490（253行严重超标）

### pipeline_handler.go
- stream 降级 L67-74 ⚠️ 仍降级到 fallback
- fallbackChatCompletions ModelGrant 注入 L143-158 ✅（R6-02修复）

### store_integration.go
- CheckBudgetCap L262-338 ✅ 真实实现（R6-03修复）
- CheckModelAccess L199-218 ✅ 无守卫（R6-10修复）
- FreezeFunds L342-354 ❌ stub 占位"待 fund.Service 集成"

### gov_handlers*.go
- requireGovAuth L200-233 ✅ 返回值检查（R6-01修复）
- lookupResourceParty L311 ✅ 返回 (string, error)（R6-13修复）
- DELETE /parties L670-689 ✅ 真实软删除
- allocate channel 必填 L309 ✅；liquidate party_id 必填 L403 ✅
- amount 字段 L32 ❌ float64（PRD 要求 NUMERIC 字符串）
- Key 吊销 L693-697 ❌ 假实现返回成功但无 DB 操作
- 16 处"待实现"占位 ❌

### http.go
- Pipeline 注入 /v1/chat/completions L171 ✅（GAP-002修复）
- Cookie 安全属性 L2061-2063 ✅ HttpOnly+Secure+SameSite（R6-09修复）

## 战果产出

| 文件 | 行数 | 关键发现 |
|------|------|---------|
| routing/profile.go | 418 | logDecision 未持久化（P1-6）；ExecuteProfile 113行超标 |
| pipeline.go | 514 | Execute 253行严重超标（P1-5）；14步真实实现 |
| store_integration.go | 399 | FreezeFunds stub（P0-5） |
| gov_handlers_fund.go | 1268 | Key吊销假实现（P0-4）；amount float64（P1-8）；6处"待实现" |
| gov_handlers_abac.go | 1303 | 6处"待实现"（P0-6） |
| gov_handlers.go | 817 | 4处"待实现"（P0-6） |

## 发现结论

### P0 阻塞（3 项）
- P0-4: Key 吊销假实现——返回成功但无 DB 操作
- P0-5: FreezeFunds 为 stub——未实际冻结资金
- P0-6: 16 处"待实现"占位端点

### P1 重要（5 项）
- P1-6: logDecision 未持久化
- P1-7: MaxAttempts 未用于重试切换
- P1-8: amount 字段 float64
- P1-12: strategies 子包零测试覆盖
- P1-5(隐含): pipeline.go Execute 253行超标

## 编译/测试结果
- go build ✅
- go vet routing ✅
- go test routing ✅ PASS（strategies 子包 [no test files]）
