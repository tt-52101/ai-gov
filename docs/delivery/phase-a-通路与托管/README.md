# 阶段 A 存证：通路与托管

| 项 | 内容 |
|----|------|
| 对应 WBS | §11.7 阶段 A（2d） |
| 产出日期 | 2026-07-31 |
| 负责 Agent | Task A (DDL) + Task D (融合基线) + Task B (API) + Task C (架构) |

---

## 产出 1：融合 DDL — `schema/ai-gov-fusion-v3.2.sql`

**来源：** 以 `ai-gov-fusion-minimal.sql`（29 表）为基底，按 PRD v3.2.0 §10 补充 11 张新表。

**指标：** 1341 行 · 40 表 · 102 索引 · 3 存储过程 · PostgreSQL 16 方言

### 新增表（11 张）

| # | 表名 | 所属组 | 说明 |
|---|------|--------|------|
| 30 | `sys_action_catalogs` | ABAC | 四轴动作目录 |
| 31 | `sys_roles` | ABAC | 角色定义（含 is_system 保护） |
| 32 | `sys_role_permissions` | ABAC | 角色→动作 N:M |
| 33 | `sys_subject_role_bindings` | ABAC | 主体→角色（含 scope/有效期） |
| 34 | `sys_access_policies` | ABAC | 策略定义（conditions_json） |
| 35 | `sys_access_policy_bindings` | ABAC | 策略→主体绑定 |
| 36 | `sys_ui_menus` | UI | 自引用菜单树 |
| 37 | `sys_ui_routes` | UI | 路由→菜单→动作 |
| 38 | `sys_ui_action_bindings` | UI | 按钮→动作绑定 |
| 39 | `sys_config` | 基础设施 | 系统 KV 配置 |
| 40 | `audit_chain_anchors` | 审计 | SHA-256 哈希链锚定 |

### 存储过程（3 个）
- `evaluate_access()` — ABAC 策略 + grants 综合评估
- `evaluate_access_via_roles()` — 角色权限评估
- `anchor_audit_chain()` — SHA-256 审计链锚定

---

## 产出 2：融合基线工程 — `ai-gov-fusion/`

**来源：** TokenHub v0.4.0（`third-party/TokenHub/`）完整复制 + 11 个新包骨架。

**指标：** TokenHub 原有 60+ 源文件零修改 · 11 个 doc.go 包声明 · Go 1.26 + GORM

### 包结构
```
backend/internal/server/
├── (TokenHub 原有代码，存量不动)
├── fund/doc.go           ← NEW
├── pricing/doc.go        ← NEW
├── idempotency/doc.go    ← NEW
├── party/doc.go          ← EXTEND
├── abac/doc.go           ← NEW
├── authz/doc.go          ← EXTEND
├── modelgrant/doc.go     ← NEW
├── ui_permission/doc.go  ← NEW
├── audit/doc.go          ← EXTEND
├── security/doc.go       ← EXTEND
└── routing/doc.go        ← EXTEND
```

---

## 产出 3：API 规范 — `docs/spec/api-spec-v3.2.md`

**指标：** 2448 行 · 10 域 · ~75 端点

| 域 | 端点数 |
|----|--------|
| Party 主体管理 | 10 |
| Fund 资金治理（含幂等） | 11 |
| Key & Member | 6 |
| Pricing 双轨计价 | 4 |
| Model Grant | 4 |
| Routing 路由调度 | 7 |
| ABAC 策略引擎 | 17 |
| UI Permission | 10 |
| Audit 审计 | 5 |
| Dashboard | 3 |

---

## 产出 4：架构文档 — `docs/spec/architecture-v3.2.md`

**指标：** 1149 行 · 7 章

| 章节 | 内容 |
|------|------|
| §1 总览 | 三平面架构图 |
| §2 包架构 | 4 层依赖图 + 11 包 NEW/EXTEND 分类 |
| §3 数据面管线 | 14 步完整链路 + StartCall 插桩点 |
| §4 控制面 | ABAC + UI 权限投影 + 审计不可篡改 |
| §5 关键接口 | Strategy/FundService/ABACEngine/UIProjector/ModelGrantChecker |
| §6 数据库映射 | 40 表按域分组 + 包归属 |
| §7 部署拓扑 | MVP SQLite vs 生产 PostgreSQL |
