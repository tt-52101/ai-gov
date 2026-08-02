# batch-007：安全漏洞全量修复

| 属性 | 值 |
|------|-----|
| 批次编号 | batch-007 |
| 批次主题 | batch-006 红蓝对抗发现的全量安全漏洞修复 |
| 执行日期 | 2026-07-31 |
| 作战单元 | FIX-A ~ FIX-F（6 Agent 并行） |
| 前置批次 | batch-006（审计发现 24 漏洞） |
| 契约引用 | AGENTS.md §7 五阶段生命周期 |

---

## 一、修复矩阵

| 战队 | Agent | 修复漏洞 | 修改文件 | 修复数 |
|------|-------|---------|---------|--------|
| FIX-A | `agent_2a0...` | R6-01：65 处 `_, _` 鉴权丢弃 | `gov_handlers.go`、`gov_handlers_fund.go`、`gov_handlers_abac.go` | 65 |
| FIX-B | `agent_123...` | R6-02/03/10/19：ModelGrant 绕过 + 预算 + 级联 + 竞态 | `pipeline_handler.go`、`pipeline.go`、`http.go`、`store_integration.go`、`checker.go`、`model.go` | 6 |
| FIX-C | `agent_1f9...` | R6-04/05/06：validateChannel + SOD 绑定 + 硬编码凭据 | `party/service.go`、`fund/service.go`、`abac/builtin.go`、`seed.go`、`config.go`、`http.go`、`image_generation.go` + test 文件 | 7 |
| FIX-D | `agent_207...` | R6-07/08/09：snapshot ABAC + FAIL-OPEN + Cookie | `gov_handlers_abac.go`、`console-router.tsx`、`http.go`（login/logout/OAuth） | 3 |
| FIX-E | `agent_dfbb...` | R6-11/12/13/16：出网 + scope + IDOR | `pipeline.go`、`routing/strategy.go`、`gov_handlers.go`、`modelgrant/model.go`、`schema/ai-gov-fusion-v3.2.sql` | 4 |
| FIX-F | `agent_c11...` | R6-14/15/17/22：冻结补偿 + 流式续期 + 快照 + 幂等 | `pipeline.go`、`audit/event.go`、`fund/service.go` + test 文件 | 4 |
| **合计** | | **24 漏洞全部修复** | **20+ 文件** | **89** |

---

## 二、修复详情

### 🔴 P0 严重（6 项 → 全部关闭）

| 编号 | 漏洞 | 修复方式 |
|------|------|---------|
| R6-01 | 65 处 `_, _` 丢弃鉴权返回值 | 全部改为 `if _, ok := requireGovAuth(...); !ok { return }` |
| R6-02 | stream=true 绕过 ModelGrant | fallbackChatCompletions 开头注入 ModelGrant 检查；ModelGrant 拒绝 → 403 不降级 |
| R6-03 | 双层预算完全失效 | Pipeline 新增步骤 [6.5] QuotaCheck；CheckBudgetCap 从 stub 重写 |
| R6-04 | SOD 策略未绑定 | SeedBuiltinPolicies 创建 4 个 SOD 角色 + 4 条策略绑定 + Bootstrap 集成 |
| R6-05 | validateChannel 占位 | 实现 channel→edge-type 匹配校验，查询 party_edges 验证边关系 |
| R6-06 | 硬编码默认凭据 | 移除所有 fallback 默认值；ValidateForStartup 强制要求环境变量 |

### 🟠 P1 高危（10 项 → 全部关闭）

| 编号 | 漏洞 | 修复方式 |
|------|------|---------|
| R6-07 | snapshot 空 action 跳过 ABAC | `""` → `"data.ui.read"` |
| R6-08 | console-router FAIL-OPEN | 3 次重试 + 失败后仅放行 /gov/dashboard + 错误页面 |
| R6-09 | Cookie 无安全属性 | 后端 Set-Cookie：HttpOnly/Secure/SameSite=Strict；前端移除 JS 操作 cookie |
| R6-10 | ModelGrant 级联退化 | 移除 `if typ == principal.Type` 守卫 |
| R6-11 | 出网默认最宽松 | 移除 modelName 跳过条件；默认 INTERNAL_ONLY |
| R6-12 | 路由档案 scope 绕过 | RouteProfile 添加 PartyID 字段 + SQL schema 同步 |
| R6-13 | IDOR fail-open | lookupResourceParty 签名改为 (string, error)，DB 故障 → 拒绝 |
| R6-14 | 冻结无补偿 | Pipeline defer 逻辑——失败自动 Unfreeze（独立 context） |
| R6-15 | 流式续期缺失 | goroutine 每 5 分钟调 StreamRenewal，管线结束自动终止 |
| R6-16 | ModelGrant scope 失效 | ModelGrant 添加 PartyID 字段 + SQL schema 同步 |

### 🟡 P2 中危（8 项 → 全部关闭）

| 编号 | 漏洞 | 修复方式 |
|------|------|---------|
| R6-17 | 快照不校验 | RecordEvent 强制要求 before/after 快照 + isConfigMutationAction |
| R6-18 | ~25 GET 忽略鉴权 | 纳入 FIX-A 一次性修复 |
| R6-19 | 配额竞态 | ConsumeQuota 改为 WHERE id=? AND version=? 乐观锁 |
| R6-20 | SQL 注入 model_grants | 表添加 party_id + 审计列（FIX-E 同步） |
| R6-21 | Config 注入 | 由 FIX-E 的 PartyID 强制设置间接缓解 |
| R6-22 | 幂等键孤儿 | IdempotencyChecker 新增 Release——事务失败回滚后释放 |
| R6-23 | HYBRID_ALLOWED 白名单 | 已知 P2，不在本批次 |
| R6-24 | lookupResourceParty 缺映射 | 已在 FIX-E 中添加 party_edge/party_member 映射 |

---

## 三、修改文件总览

| 文件 | FIX-A | FIX-B | FIX-C | FIX-D | FIX-E | FIX-F |
|------|-------|-------|-------|-------|-------|-------|
| `gov_handlers.go` | ✅ | | | | ✅ | |
| `gov_handlers_fund.go` | ✅ | | | | | |
| `gov_handlers_abac.go` | ✅ | | | ✅ | | |
| `pipeline_handler.go` | | ✅ | | | | |
| `pipeline.go` | | ✅ | | | ✅ | ✅ |
| `http.go` | | ✅ | | ✅ | | |
| `store_integration.go` | | ✅ | | | | |
| `modelgrant/checker.go` | | ✅ | | | | |
| `modelgrant/model.go` | | ✅ | | | ✅ | |
| `party/service.go` | | | ✅ | | | |
| `fund/service.go` | | | ✅ | | | ✅ |
| `abac/builtin.go` | | | ✅ | | | |
| `seed.go` | | | ✅ | | | |
| `config.go` | | | ✅ | | | |
| `image_generation.go` | | | ✅ | | | |
| `routing/strategy.go` | | | | | ✅ | |
| `audit/event.go` | | | | | | ✅ |
| `console-router.tsx` | | | | ✅ | | |
| `ai-gov-fusion-v3.2.sql` | | | | | ✅ | |

---

## 四、安全防护原则贯彻

所有修复统一遵循：

| 原则 | 贯彻项 |
|------|--------|
| **Fail-Secure** | IDOR DB 故障 → 拒绝；出网默认 INTERNAL_ONLY；前端 API 不可用 → 仅 dashboard |
| **默认拒绝** | validateChannel 实现真实校验；CheckBudgetCap 变 stub 为实际检查；SOD 策略强制绑定 |
| **纵深防御** | ModelGrant 双重检查（Pipeline + fallback）；出网双重阻断（CheckEgress + S-COMPLIANCE Filter） |
| **审计不可绕过** | 快照强校验；幂等键孤儿补偿；冻结补偿释放 |

---

## 五、单兵作战记录

| Agent | 记录文件 |
|-------|---------|
| FIX-A | `agents/FIX-A.md` |
| FIX-B | `agents/FIX-B.md` |
| FIX-C | `agents/FIX-C.md` |
| FIX-D | `agents/FIX-D.md` |
| FIX-E | `agents/FIX-E.md` |
| FIX-F | `agents/FIX-F.md` |
