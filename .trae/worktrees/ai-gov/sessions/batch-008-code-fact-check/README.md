# 任务批次 008：基于事实代码的 GA 差距核查作战

| 项 | 值 |
|----|-----|
| 批次编号 | batch-008 |
| 任务主题 | 直接读取代码文件核查实现完整性，禁止依赖任何形式报告 |
| 执行日期 | 2026-08-01 |
| 蜂群配置 | 5 Agent 全部并行（按 PRD §9 功能域分配） |
| 核查铁律 | 必须直接读取代码 + 引用行号 + 执行 go build/vet/test，禁止引用 docs/delivery/acceptance/ 报告 |
| 验收结论 | ❌ 不通过——发现 8 项 P0 阻塞 + 12 项 P1 重要，多份历史报告未暴露的真实代码缺陷 |

---

## 一、蜂群配置矩阵

| Agent | 核查域 | 目标代码 | PRD 引用 | 核查动作 | 结论 |
|-------|--------|----------|---------|---------|------|
| QA-CODE-FUND | 资金治理 | fund/pricing/idempotency/party | §9.2/§9.3/§13.1/§0.3.1 F-CON | 逐文件读取+grep占位+go test | ❌ 2 项 P0 |
| QA-CODE-SEC | 安全治理 | abac/authz/modelgrant/ui_permission/audit/security | §9.5/§9.6/§7.7/§0.3.2 D-CON | 逐文件读取+grep占位+go test | ❌ 1 项 P0 |
| QA-CODE-ROUTE-API | 路由+API | routing/pipeline/gov_handlers/http | §9.7/§9.9/§13.1/14步管线 | 14步串联+55端点逐个验证+grep"待实现" | ❌ 3 项 P0 |
| QA-CODE-UI | 前端UI | frontend/8模块+middleware+错误码 | §9.9 UI-01~14/UX-1/UX-2 | 逐模块读取+ABAC投影+编译验证 | ⚠️ 5 项 UI 条款未实现 |
| QA-CODE-DATA | 数据+部署 | schema/deploy/scripts | §10/§12/§11.7 | 40表DDL+迁移+部署配置 | ❌ 2 项 P0 |

---

## 二、P0 阻塞缺陷清单（8 项——GA 前必须修复）

| # | 缺陷 | 代码证据 | 影响范围 | 来源 Agent |
|---|------|----------|----------|-----------|
| P0-1 | **party 包 int64/TEXT 类型不匹配**——validateChannel 生产路径失效 | party/model.go:131-132 (string) vs party/store.go:104 (int64) vs party/service.go:220 (int64) | 所有资金划拨通道校验失败，UUID 无法解析为 int64 | QA-CODE-FUND |
| P0-2 | **fund 包 newUUID 伪 UUID**——时间拼接，并发碰撞风险 | fund/service.go:421-429 使用 time.Now() 而非 crypto/rand | allocation/freeze/ledger 主键并发碰撞 | QA-CODE-FUND |
| P0-3 | **ui_permission 包测试代码类型不匹配**——10+ 处 int64/string 错误 | projector_test.go:73,117,137,337,345；store_test.go:19,30,80,86,108 | go vet/test 失败，违反 §7.6 质量门禁 | QA-CODE-SEC |
| P0-4 | **Key 吊销假实现**——返回成功但无 DB 操作 | gov_handlers_fund.go:693-697 | 已吊销 Key 仍可调用，安全凭证失控 | QA-CODE-ROUTE-API |
| P0-5 | **FreezeFunds 为 stub**——未实际冻结资金 | store_integration.go:342-354 注释"待 fund.Service 集成" | Pipeline 路径资金预扣失效，违反守恒铁律 | QA-CODE-ROUTE-API |
| P0-6 | **16 处"待实现"占位端点**——Key详情/轮换、仪表盘、安全报表、调用追踪等 | gov_handlers.go:700,734,783,817；gov_handlers_fund.go:132,621,692,702,1257,1268；gov_handlers_abac.go:1225,1234,1242,1288,1295,1302 | API-01"治理API对等"未达成 | QA-CODE-ROUTE-API |
| P0-7 | **liquidations.liquidation_type 字段缺失** | schema/ai-gov-fusion-v3.2.sql:217-229（无该字段），PRD §10.1:915 要求 | 组织合并/拆分流程无法标识清算类型 | QA-CODE-DATA |
| P0-8 | **K8s Helm Chart 缺失** | 全局无 Chart.yaml，PRD §11.4:1062 要求"K8s Helm（生产）" | 生产环境 K8s 部署无法进行 | QA-CODE-DATA |

---

## 三、P1 重要缺陷清单（12 项）

| # | 缺陷 | 代码证据 | 来源 Agent |
|---|------|----------|-----------|
| P1-1 | 清算状态机 5 阶段 vs PRD context.md 4 阶段不一致 | fund/lifecycle.go:438-454 vs PRD context.md | QA-CODE-FUND |
| P1-2 | validateChannel 降级路径无生产防护（PartyService==nil 时仅字符串比较） | fund/service.go:354-361 | QA-CODE-FUND |
| P1-3 | SEC-02 出网白名单未实现（HYBRID_ALLOWED 直接放行） | security/egress.go:86-91 | QA-CODE-SEC |
| P1-4 | SEC-03 内容安全未实现（仅 NoopHook 空实现） | security/hooks.go:95-108 | QA-CODE-SEC |
| P1-5 | SEC-04 异常流量拦截告警未实现 | security 包无相关代码 | QA-CODE-SEC |
| P1-6 | logDecision 未持久化（注释称写数据库但仅 slog 输出） | routing/profile.go:404-418 | QA-CODE-ROUTE-API |
| P1-7 | MaxAttempts 未用于重试切换（仅存储钳位，ExecuteProfile 不引用） | routing/profile.go:151,198-311 | QA-CODE-ROUTE-API |
| P1-8 | amount 字段 float64（PRD 要求 NUMERIC 字符串禁止浮点） | gov_handlers_fund.go:32 | QA-CODE-ROUTE-API |
| P1-9 | 5 个 UI 条款未实现（UI-05 Key/UI-08 密钥仓库/UI-09 模型权限/UI-10 安全报表/UI-11 调用追踪） | frontend/app/(console)/gov/ 无对应模块 | QA-CODE-UI |
| P1-10 | 国产环境完全缺失（无 OceanBase/TiDB 驱动，仅文档层面提及） | store.go:30-31 仅 postgres+sqlite | QA-CODE-DATA |
| P1-11 | 无 Prometheus 指标导出代码 | 全项目无 prometheus client 代码 | QA-CODE-DATA |
| P1-12 | strategies 子包零测试覆盖（12 策略无 _test.go） | go test 输出 [no test files] | QA-CODE-ROUTE-API |

---

## 四、与历史报告的差异（关键发现）

本批次核查发现多份历史报告未暴露的真实代码缺陷：

| 历史报告结论 | 代码事实核查结论 | 差异性质 |
|-------------|----------------|---------|
| batch-004 "~55端点真实实现" | 16 处"待实现"占位 + Key 吊销假实现 | ❌ 报告高估 |
| batch-007 "R6-03 CheckBudgetCap 已修复" | CheckBudgetCap 修复了，但 FreezeFunds 仍是 stub | ❌ 报告不完整 |
| batch-001 "party 包测试通过" | party 包测试编译失败（int64/TEXT 类型不匹配） | ❌ 报告失实 |
| batch-007 "24漏洞全部关闭" | R6-02 stream 降级路径仍存在；R6-15 流式续期仅对非降级路径生效 | ⚠️ 部分修复 |
| PROGRESS-SUMMARY "无已知阻塞缺陷" | 8 项 P0 阻塞（含 2 项金融级致命：伪 UUID + 类型不匹配） | ❌ 严重低估 |

---

## 五、编译/测试事实

| 命令 | 范围 | 结果 |
|------|------|------|
| go build ./... | 全项目 | ✅ 通过 |
| go vet ./... | 全项目 | ❌ 失败（party/ui_permission 类型错误） |
| go test fund/ | fund 包 | ✅ PASS |
| go test pricing/ | pricing 包 | ✅ PASS |
| go test idempotency/ | idempotency 包 | ✅ PASS |
| go test party/ | party 包 | ❌ FAIL [build failed] |
| go test abac/authz/modelgrant/audit/security/ | 5 个安全包 | ✅ PASS |
| go test ui_permission/ | ui_permission 包 | ❌ FAIL [build failed] |
| go test routing/ | routing 包 | ✅ PASS（但 strategies 子包零测试） |
| npx next build | 前端 | ✅ 通过 |

---

## 六、单兵作战记录

| Agent | 记录文件 |
|-------|---------|
| QA-CODE-FUND | `agents/QA-CODE-FUND.md` |
| QA-CODE-SEC | `agents/QA-CODE-SEC.md` |
| QA-CODE-ROUTE-API | `agents/QA-CODE-ROUTE-API.md` |
| QA-CODE-UI | `agents/QA-CODE-UI.md` |
| QA-CODE-DATA | `agents/QA-CODE-DATA.md` |

---

## 七、合规声明

本批次严格遵循 AGENTS.md §7 五阶段生命周期：
- **事实驱动**：每条发现附代码行号证据，禁止引用报告文档
- **独立作战**：5 Agent 独立核查，互不通信
- **存证完整**：批次 README + 5 份单兵记录
- **铁律遵守**：每个 Agent 提示词第一行声明中文注释铁律 + 只读核查禁修改代码
