# batch-006：红蓝对抗全系统安全审计

| 属性 | 值 |
|------|-----|
| 批次编号 | batch-006 |
| 批次主题 | 红蓝对抗——全系统安全漏洞扫除 |
| 执行日期 | 2026-07-31 |
| 作战单元 | RED-1 ~ RED-6（6 Agent 并行） |
| 审计范围 | Go 后端 + Next.js 前端 + SQLite Schema |
| PRD 基线 | `docs/prd/AI-GOV-PRD-v3.1.0.md` |
| 代码基线 | `ai-gov-fusion/backend/internal/server/` |
| 契约引用 | AGENTS.md §7 五阶段生命周期 |

---

## 一、作战矩阵

| Agent | 审计域 | 审计项数 | 发现漏洞 | 严重 | 高危 | 中危 |
|-------|--------|---------|---------|------|------|------|
| RED-1 | 守恒定理全量审计 | 24 定理 | 5 绕过 | — | 3 | 2 |
| RED-2 | Cookie + 白名单 + 鉴权完整性 | 6 攻击面 | 6 风险 | 2 | 2 | 2 |
| RED-3 | 数据泄露复验 (batch-002) | 2 缺陷 | 0（全部已关闭） | — | — | — |
| RED-4 | 模型访问越权 | 6 攻击路径 | 6 漏洞 | 2 | 1 | 3 |
| RED-5 | 路由策略访问越权 | 5 审计项 | 3 缺陷 | 1 | 1 | 1 |
| RED-6 | 综合漏洞扫描 + 前端审计 | 全量扫描 | 4 Release Blocker | 1 | 3 | — |
| **合计** | | | **24 独立漏洞** | **6** | **10** | **8** |

---

## 二、去重合并的独立漏洞清单

### 🔴 严重 — Release Blocker（6 项）

| 编号 | 来源 | 漏洞名称 | 受影响代码 |
|------|------|---------|-----------|
| **R6-01** | RED-1/D-CON-01, RED-2/R2-4/R2-5 | **12 个端点丢弃鉴权返回值——DELETE/POST 可无认证执行** | `gov_handlers.go` L743, L822, L718, L797; `gov_handlers_abac.go` L276, L429, L606, L716, L745, L823, L844, L922, L943, L1021 |
| **R6-02** | RED-4/V-4.1 | **`/v1/chat/completions` stream=true 完全绕过 ModelGrant** | `pipeline_handler.go` L58-69 (PipelineEnabled/stream 降级); `pipeline_handler.go` L84-93 (Pipeline 失败降级); `http.go` L705-810 (旧路径无 ModelGrant) |
| **R6-03** | RED-4/V-4.2 | **双层预算体系完全失效** | `store_integration.go` L252-265 (CheckBudgetCap stub); `checker.go` L91-122 (CheckQuotaLimit 未集成到 Pipeline); `checker.go` L133-166 (ConsumeQuota 无乐观锁) |
| **R6-04** | RED-5/AP-3 | **SOD 跨轴互斥策略已定义但从未绑定** | `abac/builtin.go` L84-120 (SeedBuiltinPolicies 只创建策略不绑定) |
| **R6-05** | RED-6/#1 | **`validateChannel()` 占位——可绕过 party 边关系校验** | `fund/service.go` L286-298 |
| **R6-06** | RED-6/#4 | **AdminToken/SecretKey 代码内嵌默认值** | `config.go` L83-90 |

### 🟠 高危（10 项）

| 编号 | 来源 | 漏洞名称 | 受影响代码 |
|------|------|---------|-----------|
| **R6-07** | RED-6/#2, RED-2/R2-3 | **`/v1/gov/ui-permissions/snapshot` 传入空 action 跳过 ABAC** | `gov_handlers_abac.go` L1040 |
| **R6-08** | RED-6/#3, RED-2/R2-2 | **`console-router.tsx` FAIL-OPEN——API 失败时静默放行所有页面** | `console-router.tsx` L56-58 |
| **R6-09** | RED-2/R2-1 | **Cookie 无 HttpOnly/Secure/SameSite——XSS 可窃取 session** | 全项目：后端从未设置 Set-Cookie，前端 JS 直接操作 cookie |
| **R6-10** | RED-4/V-4.3 | **ModelGrant 级联逻辑退化——`if typ == principal.Type` 只加载单层级** | `modelgrant/checker.go` L171-203; `store_integration.go` L196-215 (固定 Party 级) |
| **R6-11** | RED-1/D-CON-02 | **出网管控：modelName 为空跳过检查，默认策略为最宽松** | `pipeline.go` L267 (`modelName != ""` 条件跳过); L483 (默认 HYBRID_ALLOWED) |
| **R6-12** | RED-5/AP-1 | **路由档案 scope 鉴权绕过——`route_profiles` 表无 `party_id` 列** | `gov_handlers.go` L321-322; `routing/strategy.go` L144-156 |
| **R6-13** | RED-1/D-CON-07 | **IDOR：DB 故障时 scope 校验静默降级为全局放行 (fail-open)** | `gov_handlers.go` L338-344 |
| **R6-14** | RED-1/S-CON-02 | **管线冻结后失败无补偿回滚——资金锁定 15 分钟** | `pipeline.go` L327-336 |
| **R6-15** | RED-1/S-CON-03 | **流式续期全量委托 HTTP handler——管线不执行续期** | `pipeline.go` L362 |
| **R6-16** | RED-4/V-4.4 | **ModelGrant ABAC scope_party_id 失效——表缺 party_id 列** | `gov_handlers.go` L319-320; `schema/ai-gov-fusion-v3.2.sql` L529-541 |

### 🟡 中危（8 项）

| 编号 | 来源 | 漏洞名称 | 受影响代码 |
|------|------|---------|-----------|
| **R6-17** | RED-1/AU-CON-02 | **快照完整性仅注释声明——RecordEvent 不校验 before/after** | `audit/event.go` L36-58 |
| **R6-18** | RED-2/R2-6 | **~25 个 GET handler 忽略鉴权返回值** | `gov_handlers.go`、`gov_handlers_fund.go`、`gov_handlers_abac.go` 中 `_, _ = h.requireGovAuth(...)` 模式 |
| **R6-19** | RED-4/V-4.5 | **配额并发竞态——ConsumeQuota 声称乐观锁但未实现 version 检查** | `modelgrant/checker.go` L133-166 |
| **R6-20** | RED-4/V-4.6 | **直接 SQL 插入 `model_grants` 绕过 API 层 ABAC** | `model_grants` 表无 RLS、无触发器、无审计列 |
| **R6-21** | RED-5/AP-5 | **strategies_json Config 注入——json.RawMessage 无校验持久化** | `routing/strategy.go` L134 |
| **R6-22** | RED-1/F-CON-05 | **幂等键在事务外 Claim——事务失败后幂等键被"烧毁"** | `fund/service.go` L81 |
| **R6-23** | RED-3 | **HYBRID_ALLOWED 白名单未实现（阶段 D 骨架）** | `security/egress.go` L85-91 |
| **R6-24** | RED-3 | **lookupResourceParty 对 8 种 resourceType 无 party_id 映射** | `gov_handlers.go` L314-315 |

---

## 三、已确认关闭的已修复缺陷（RED-3 复验通过）

| 缺陷 | 状态 | 防御层 |
|------|------|--------|
| **FIX-B** INTERNAL_ONLY 出网管控 | ✅ 关闭 | 双重防护：CheckEgress 阻断 + S-COMPLIANCE Filter 剔除 |
| **FIX-D** scope_party_id / IDOR / 错误脱敏 | ✅ 关闭 | 18 个单品端点接入 requireGovItemAuth；sanitizeError 脱敏 |

---

## 四、修复优先级建议

### P0（阻塞 GA — 必须在本批次修复）

1. **R6-01** 12 端点鉴权绕过 → 所有 `_, _ = h.requireGovAuth(...)` 改为正确的返回值检查 + return
2. **R6-02** stream 绕过 ModelGrant → fallbackChatCompletions 添加 ModelGrant 检查，移除 stream 降级逻辑
3. **R6-03** 双层预算失效 → Pipeline 集成 CheckQuotaLimit；实现 CheckBudgetCap
4. **R6-06** 代码内嵌默认值 → 移除 fallback，启动时必须读取环境变量

### P1（GA 前必须修复）

5. **R6-04** SOD 策略未绑定 → SeedBuiltinPolicies 中绑定策略到角色
6. **R6-07** snapshot ABAC 绕过 → 传入正确 action
7. **R6-08** console-router FAIL-OPEN → 改为 fail-closed
8. **R6-09** Cookie 不安全 → 后端设置 HttpOnly/Secure/SameSite cookie
9. **R6-10** 级联退化 → 移除 if typ == principal.Type 守卫
10. **R6-11** 出网管控默认策略 → 默认 INTERNAL_ONLY 而非 HYBRID_ALLOWED

### P2（GA 后 sprint-2）

11-24. 其余中危项

---

## 五、单兵作战记录

| Agent | 记录文件 |
|-------|---------|
| RED-1 | `agents/RED-1.md` |
| RED-2 | `agents/RED-2.md` |
| RED-3 | `agents/RED-3.md` |
| RED-4 | `agents/RED-4.md` |
| RED-5 | `agents/RED-5.md` |
| RED-6 | `agents/RED-6.md` |

详细报告：`docs/delivery/acceptance/red-blue/RED-{1..6}-*.md`

---

## 六、合规声明

本批次严格遵循 AGENTS.md §7 五阶段生命周期：策划 → 执行 → 验证 → 存证 → 复盘。

- **事实驱动**：每条发现附 PRD 条款引用 + 代码行号证据
- **独立作战**：6 Agent 独立审计，交叉去重
- **存证完整**：batch README + 6 份单兵记录 + 6 份详细报告
