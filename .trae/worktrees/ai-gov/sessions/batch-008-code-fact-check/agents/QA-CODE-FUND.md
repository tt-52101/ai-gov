# 单兵轨迹 · QA-CODE-FUND

| 项 | 值 |
|----|-----|
| Agent ID | QA-CODE-FUND |
| 所属蜂群 | batch-008 代码事实核查 |
| 作战指令 | 基于事实代码核查资金治理域（fund/pricing/idempotency/party）实现完整性 |
| 执行耗时 | 约 4 分钟 |
| 验收判定 | ❌ 不通过——2 项 P0 + 3 项 P1 |

## 情报收集（逐文件）

### fund 包
- fund/service.go：Allocate L79-155 ✅；allocateExecute L176-318（143行超标）；validateChannel L352-414 ⚠️；newUUID L421-429 ❌ 伪 UUID
- fund/freeze.go：Freeze L30-77 ✅；freezeCheckBudget L81-110 ✅；Settle L214-305（92行超标）
- fund/lifecycle.go：RenewFreeze L23-89 ✅；Liquidate 5阶段 L438-454 ⚠️ 与 PRD 4阶段不一致
- fund/model.go：Account 含 version 乐观锁 ✅；Ledger 只追加 ✅；defaultFreezeTTL=15min ✅
- fund/sqlstore/pg.go：SELECT FOR UPDATE ✅；乐观锁 WHERE version=? ✅

### pricing 包
- calculator.go：5种计价模式 ✅；cache_discount_ratio ✅
- normalizer.go：OpenAI/Anthropic 规范化 ✅；10 itemCode ✅
- model.go：双轨结构 ✅

### idempotency 包
- claim.go：INSERT 优先 + UNIQUE 约束 ✅；Release 方法 ✅
- middleware.go：POST/PUT/PATCH 检查 ✅

### party 包
- service.go：ValidateChannel L220-285 ❌ int64 签名 vs model TEXT 类型；newPartyID L356-365 ✅ crypto/rand
- model.go：7 种边类型 ✅；PartyEdge.SrcPartyID/DstPartyID 为 string ⚠️
- store.go：FindEdge 签名 int64 ❌ 类型不匹配

## 战果产出

| 文件 | 行数 | 关键发现 |
|------|------|---------|
| fund/service.go | 429 | newUUID 伪 UUID（P0-2）；allocateExecute 143行超标 |
| fund/freeze.go | 412 | Settle 92行超标；孤儿结算逻辑不完整 |
| fund/lifecycle.go | 454 | 5阶段 vs PRD 4阶段不一致 |
| party/service.go | 365 | ValidateChannel int64 签名类型不匹配（P0-1） |
| party/store.go | 287 | FindEdge/DeleteEdge/ListEdges 全部 int64 签名 |

## 发现结论

### P0 阻塞（2 项）
- D1: party int64/TEXT 类型不匹配——validateChannel 生产路径失效
- D2: fund newUUID 伪 UUID——并发碰撞风险

### P1 重要（3 项）
- D3: 清算状态机 5阶段 vs PRD 4阶段
- D4: validateChannel 降级路径无生产防护
- D5: Settle 孤儿结算逻辑不完整

## 编译/测试结果
- go build ✅
- go vet ❌（party 类型错误）
- go test fund/pricing/idempotency ✅ PASS
- go test party ❌ FAIL [build failed]
