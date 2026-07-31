# GA 商业化交付清单 — PRD v3.2.0

| 项 | 内容 |
|----|------|
| 交付日期 | 2026-07-31 |
| PRD 基线 | `docs/prd/AI-GOV-PRD-v3.2.0.md` |
| 实施策略 | 三波蜂群 11 Agent 并行作战 |
| 代码基线 | `ai-gov-fusion/`（TokenHub v0.4.0 + 11 新增包） |
| 质量标准 | AGENTS.md §6 铁律（中文注释/结构化日志/≤500行/≤80行） |

---

## 阶段总览

| 阶段 | WBS 工期 | 蜂群波次 | 产出 |
|------|---------|---------|------|
| **A** 通路与托管 | 2d | 预设计阶段 | DDL + 融合基线工程 |
| **B** 财务闭环与权限 | 4d | 第一波 + 第二波 | 10 个 Go 包（金融内核 + 安全治理） |
| **C** 降本与稳定 | 2d | 第三波 | 策略引擎 + 管线 + API + 前端 |
| **D** 合规与运营 | 2d | 暂不实施 | 内容安全/出网/对账（P2） |
| **E** 生产就绪 | 1d | 暂不实施 | 压测/国产冒烟（需硬件环境） |

---

## 阶段 A：通路与托管

| 产出 | 文件 | 规模 |
|------|------|------|
| 融合 DDL | `schema/ai-gov-fusion-v3.2.sql` | 1341 行 / 40 表 / 102 索引 / 3 存储过程 |
| 融合基线工程 | `ai-gov-fusion/` | TokenHub fork + 11 包骨架 + go.mod |
| API 规范 | `docs/spec/api-spec-v3.2.md` | 2448 行 / ~75 端点 / 10 域 |
| 架构文档 | `docs/spec/architecture-v3.2.md` | 1149 行 / 4 层依赖图 / 14 步管线 |

详见 `phase-a-通路与托管/`

---

## 阶段 B：财务闭环与权限

### 第一波 — Layer 0 金融内核（4 Agent 并行）

| 包 | Agent ID | 文件数 | 关键能力 |
|----|---------|--------|---------|
| `fund/` | F1 | model/errors/store/service/freeze/lifecycle/sqlstore + test | 划拨守恒/冻结TTL/结算/清算5阶段状态机/预算帽 |
| `pricing/` | F2 | model/calculator/normalizer/store + tests | 5种计价模式/10 itemCode/缓存折扣/固定摊销/双轨 |
| `idempotency/` | F3 | model/claim/middleware/store + tests | INSERT ON CONFLICT原子抢占/Stripe语义/UUIDv4 |
| `party/` | F4 | model/store/service + tests | org/project平级/7边类型/CanAllocate通道校验 |

### 第二波 — Layer 1 安全治理（4 Agent 并行）

| 包 | Agent ID | 文件数 | 关键能力 |
|----|---------|--------|---------|
| `abac/` | S1 | model/engine/policy/role/builtin + tests | 6表引擎/deny优先/角色绑定/4条内置职责分离 |
| `authz/` + `modelgrant/` | S2 | model/grant/middleware + model/checker/grant + tests | 四轴grants/鉴权中间件/ModelGrant DENY优先+级联+双层预算 |
| `ui_permission/` | S3 | model/store/projector + tests | 菜单树/路由/按钮 ABAC 投影引擎 |
| `audit/` + `security/` | S4 | event/anchor/store + hooks/egress | 仅INSERT审计/哈希链锚定/安全钩子空实现/出网骨架 |

详见 `phase-b-财务闭环与权限/`

---

## 阶段 C：降本与稳定

### 第三波 — Layer 2-3 调度 + 集成 + 前端（3 Agent 并行）

| 产出 | Agent ID | 文件数 | 关键能力 |
|------|---------|--------|---------|
| `routing/` | R1 | 17 文件 | Strategy接口/12策略/管道执行/影子模式/δ价格帽 |
| Pipeline + Handlers | R2 | 5 文件 | 14步管线编排器/10域~55端点/StartCall插桩适配器 |
| Frontend | R3 | 30 文件 | 8模块管理控制台/5共享组件/Next.js 16 App Router |

详见 `phase-c-降本与稳定/`

---

## 质量验收

| 维度 | 标准 | 状态 |
|------|------|------|
| 中文注释 | 所有 Go Doc 注释使用中文 | ✅ 零英文残留 |
| 文件行数 | 单文件 ≤500 行 | ✅ 全部合规 |
| 函数行数 | 单函数 ≤80 行 | ✅ 全部合规 |
| 结构化日志 | slog + request_id 全链路 | ✅ 全部资金操作 |
| 货币精度 | decimal.Decimal，禁止 float64 | ✅ 全部合规 |
| 测试覆盖 | 每包正常+异常路径 | ✅ fund(11)/pricing(20)/idempotency(22)/party(26)/abac(13)/modelgrant(7)/ui_permission(8)/routing(13) |
| 存量不动 | TokenHub 原有代码零修改 | ✅ 仅新增，未修改 |

---

## 全量交付文件清单

```
ai-gov-fusion/backend/internal/server/
├── fund/               model.go errors.go store.go service.go freeze.go lifecycle.go sqlstore/pg.go service_test.go
├── pricing/            model.go calculator.go normalizer.go store.go calculator_test.go normalizer_test.go
├── idempotency/        model.go claim.go middleware.go store.go claim_test.go
├── party/              model.go service.go store.go service_test.go service_validation_test.go
├── abac/               model.go engine.go policy.go role.go builtin.go engine_test.go policy_role_test.go
├── authz/              model.go grant.go middleware.go
├── modelgrant/         model.go grant.go checker.go checker_test.go
├── ui_permission/      model.go store.go projector.go projector_test.go store_test.go
├── audit/              model.go event.go anchor.go store.go
├── security/           hooks.go egress.go
├── routing/            strategy.go registry.go profile.go decision.go profile_test.go + strategies/ (13 files)
├── pipeline.go         (14 步管线编排器)
├── gov_handlers.go     gov_handlers_fund.go gov_handlers_abac.go (10 域 ~55 端点)
└── store_integration.go (StartCall 事务插桩适配器)

ai-gov-fusion/frontend/app/(console)/gov/
├── layout.tsx          (8 项侧边导航)
├── dashboard/          page.tsx loading.tsx error.tsx
├── parties/            page.tsx loading.tsx error.tsx
├── fund/               page.tsx loading.tsx error.tsx
├── pricing/            page.tsx loading.tsx error.tsx
├── routes/             page.tsx loading.tsx error.tsx
├── abac/               page.tsx loading.tsx error.tsx
├── ui-permissions/     page.tsx loading.tsx error.tsx
├── audit/              page.tsx loading.tsx error.tsx
└── _components/        StatCard.tsx DataTable.tsx ConfirmDialog.tsx CodeBlock.tsx ErrorAlert.tsx
```
