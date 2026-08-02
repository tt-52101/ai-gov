# 作战计划 plan-009：商业GA歼灭战 | 版本：v1.0 | 日期：2026-08-02 | 状态：已发布

> **基于事实**：batch-008 代码事实核查发现 8 项 P0 阻塞 + 12 项 P1 重要缺陷
> **依据**：蜂群作战通用标准方案（三铁律 + 四闸 + 五原则）
> **核心原则**：集中优势兵力、各个歼灭、断指而非伤指、实事求是

---

## 作战目的

**48 小时内**将 AI-GOV 从「8P0+12P1 缺陷状态」推进至**商业 GA 可交付状态**，通过 6 大蜂群全量 QA E2E 验收。

---

## 作战单元拆解

| 单元 | 目标 | 对应缺陷 | 依赖 | 验收标准 |
|------|------|---------|------|---------|
| U1 | 资金核心修复 | P0-1/P0-2/P0-5/P0-7 | 无 | 类型统一 + UUID 真随机 + FreezeFunds 真冻结 + DDL 字段补全 |
| U2 | 安全+路由修复 | P0-3/P0-4/P0-6 | 无 | 测试编译通过 + Key 真吊销 + 16 处占位真实现 |
| U3 | 部署基础设施 | P0-8 | 无 | K8s Helm Chart 可部署 |
| U4 | P1 重要缺陷 | P1-1~P1-12 | U1+U2 | 全部 P1 闭环 |
| U5 | 全面 QA E2E 验收 | 全部 | U1~U4 | 6 大蜂群验收全部通过 |

---

## 三波次作战计划

### 波次 1：P0 阻塞歼灭战（奇胜式·突击蜂群）

| 时间闸 | 规模闸 | 门禁闸 |
|--------|--------|--------|
| ≤4h | 4 Agent | go build+vet+test 全通过 |

| Agent | 作战单元 | 目标文件 | 修复内容 |
|-------|---------|---------|---------|
| FIX-FUND | U1 | `party/model.go`, `party/store.go`, `party/service.go`, `fund/service.go`, `store_integration.go`, `schema/ai-gov-fusion-v3.2.sql` | P0-1 类型统一（int64→string）、P0-2 UUID 改用 crypto/rand、P0-5 FreezeFunds 集成 fund.Service、P0-7 liquidation_type 字段 |
| FIX-SEC | U2 | `ui_permission/projector_test.go`, `ui_permission/store_test.go`, `gov_handlers_fund.go` | P0-3 测试代码类型修正、P0-4 Key 吊销实现 |
| FIX-ROUTE | U2 | `gov_handlers.go`, `gov_handlers_fund.go`, `gov_handlers_abac.go` 等 16 处占位 | P0-6 16 处占位端点真实现 |
| FIX-DEPLOY | U3 | 新建 K8s 配置 | P0-8 K8s Helm Chart 创建 |

### 波次 2：P1 重要缺陷修复（正合式）

| 时间闸 | 规模闸 |
|--------|--------|
| ≤6h | 3-4 Agent |

修复 P1-1~P1-12，依赖波次 1 完成。

### 波次 3：全面 QA E2E 验收（正合式·6 大蜂群并行）

| 时间闸 | 规模闸 |
|--------|--------|
| ≤8h | 16 Agent（6 蜂群） |

按 AGENTS.md 六大蜂群矩阵全面验收。

---

## DoD 清单（每个作战单元）

- [ ] 代码修改编译通过（go build ./...）
- [ ] 静态分析零告警（go vet ./...）
- [ ] 测试全部通过（go test ./...）
- [ ] 单兵轨迹存证（commit hash + timespan）
- [ ] 门禁报告输出（gate-report.json）
- [ ] 缺陷注册表更新（registry.jsonl）
- [ ] RTM 追溯无空缺

---

## 回滚策略

任一波次门禁 overall=false → 缺陷入 registry → 启动修复子批次 → 重跑门禁。

---

## 附录：缺陷注册表

| ID | 标题 | 严重度 | 状态 | 发现批次 |
|----|------|--------|------|---------|
| DEF-001 | party 包 int64/TEXT 类型不匹配 | P0 | open | batch-008 |
| DEF-002 | fund 包 newUUID 伪 UUID | P0 | open | batch-008 |
| DEF-003 | ui_permission 测试代码类型不匹配 | P0 | open | batch-008 |
| DEF-004 | Key 吊销假实现 | P0 | open | batch-008 |
| DEF-005 | FreezeFunds 为 stub | P0 | open | batch-008 |
| DEF-006 | 16 处"待实现"占位端点 | P0 | open | batch-008 |
| DEF-007 | liquidations.liquidation_type 字段缺失 | P0 | open | batch-008 |
| DEF-008 | K8s Helm Chart 缺失 | P0 | open | batch-008 |
| DEF-009~020 | P1-1~P1-12 重要缺陷 | P1 | open | batch-008 |