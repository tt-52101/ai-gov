# 阶段 B 存证：财务闭环与权限

| 项 | 内容 |
|----|------|
| 对应 WBS | §11.7 阶段 B（4d） |
| 产出日期 | 2026-07-31 |
| 蜂群波次 | 第一波（Layer 0）+ 第二波（Layer 1） |
| Agent 数量 | 8 Agent（4+4 并行） |

---

## 第一波：Layer 0 金融内核（无依赖，4 Agent 并行）

### [F1] `fund/` — 资金治理核心

| 文件 | 行数 | 职责 |
|------|------|------|
| `model.go` | 423 | Account/Ledger/Freeze/Allocation/Liquidation + Decimal 类型 |
| `errors.go` | 167 | FundError + 8 哨兵错误 |
| `store.go` | 123 | Store 接口（20 方法）+ Tx 接口 |
| `service.go` | 337 | Allocate（守恒+死锁安全+通道校验+幂等） |
| `freeze.go` | 381 | Freeze（预算帽+TTL）/ Settle（退款+孤儿结算） |
| `lifecycle.go` | 438 | RenewFreeze / UnfreezeTimeout / Liquidate（5 阶段状态机） |
| `sqlstore/pg.go` | 393 | PgStore（GORM + SELECT FOR UPDATE + 乐观锁） |
| `service_test.go` | 876 | 11 测试（划拨守恒/幂等/冻结/结算/清算） |

### [F2] `pricing/` — 双轨计价引擎

| 文件 | 行数 | 职责 |
|------|------|------|
| `model.go` | 268 | 10 itemCode + 5 计价模式 + PriceJSON/PricingTier |
| `calculator.go` | 255 | CalculateCost/Sell/DualTrack + tieredPrice + EstimateSell |
| `normalizer.go` | 189 | NormalizeOpenAI / NormalizeAnthropic（缺字段标记不伪造） |
| `store.go` | 98 | UpsertPrice（INSERT ON CONFLICT） |
| 测试 | 559 | 20 测试全部通过 |

### [F3] `idempotency/` — 幂等键引擎

| 文件 | 行数 | 职责 |
|------|------|------|
| `model.go` | 132 | Record + 3 状态常量 + 响应编解码 |
| `claim.go` | 309 | Claim（INSERT-first 原子抢占）/ Complete / Fail |
| `middleware.go` | 173 | HTTP 中间件 + UUID v4 校验 |
| `store.go` | 113 | GORM CRUD + CleanExpired |
| `claim_test.go` | 682 | 22 测试（并发/回放/冲突/过期回收） |

### [F4] `party/` — Party 统一主体

| 文件 | 行数 | 职责 |
|------|------|------|
| `model.go` | 229 | Party/PartyEdge/PartyMember + 7 边类型常量 |
| `service.go` | 277 | CreateParty（项目可不挂父级）/ CreateEdge / CanAllocate |
| `store.go` | 202 | GORM CRUD + Migrate |
| 测试 | 627 | 26 测试全部通过 |

---

## 第二波：Layer 1 安全治理（依赖 Layer 0，4 Agent 并行）

### [S1] `abac/` — ABAC 策略引擎

| 文件 | 行数 | 职责 |
|------|------|------|
| `model.go` | 266 | 6 表 GORM 模型 + Subject/Resource |
| `engine.go` | 483 | Evaluate（deny→allow→角色→默认拒绝）/ GetPermissions |
| `policy.go` | 326 | 策略 CRUD + 绑定 + EvaluatePolicy 模拟 |
| `role.go` | 358 | 角色 CRUD + 权限授予/撤销 + 主体角色分配 |
| `builtin.go` | 120 | 4 条内置职责分离策略 + 幂等种子写入 |
| 测试 | 582 | 13 测试全部通过 |

### [S2] `authz/` + `modelgrant/` — 授权 + 模型治理

| 包 | 文件 | 行数 | 职责 |
|----|------|------|------|
| authz | `model.go` | 142 | Grant + 四轴常量 + 21 动作常量 |
| authz | `grant.go` | 127 | CRUD + Evaluate（DENY 优先） |
| authz | `middleware.go` | 121 | HTTP 鉴权中间件（URL→轴/动作映射） |
| modelgrant | `model.go` | 95 | Principal + ModelGrant + quota_limit |
| modelgrant | `grant.go` | 117 | CRUD |
| modelgrant | `checker.go` | 219 | CheckAccess（Key>Person>Party 级联）/ CheckQuotaLimit / ConsumeQuota |
| modelgrant | 测试 | 264 | 7 测试全部通过 |

### [S3] `ui_permission/` — UI 权限投影

| 文件 | 行数 | 职责 |
|------|------|------|
| `model.go` | 107 | SysUIMenu/SysUIRoute/SysUIActionBinding |
| `store.go` | 312 | CRUD + FK 校验 |
| `projector.go` | 302 | ProjectMenus（二遍自底向上）/ ProjectRoutes / ProjectActions |
| 测试 | 496 | 8 测试全部通过 |

### [S4] `audit/` + `security/` — 审计 + 安全

| 包 | 文件 | 行数 | 职责 |
|----|------|------|------|
| audit | `model.go` | 165 | AuditEvent + AuditChainAnchor + 9 行动常量 |
| audit | `event.go` | 154 | RecordEvent（仅 INSERT）/ SearchEvents / GetEvent |
| audit | `anchor.go` | 152 | AnchorChain（SHA-256）/ VerifyChain |
| audit | `store.go` | 21 | Migrate |
| security | `hooks.go` | 201 | Hook 接口 + NoopHook + Chain 职责链 |
| security | `egress.go` | 100 | CheckEgress（INTERNAL_ONLY 阻断 + HYBRID 放行） |

---

## 中文注释质量

| 包 | 英文残留 |
|----|---------|
| fund | 0 行 |
| pricing | 0 行 |
| idempotency | 0 行 |
| party | 0 行 |
| abac | 1 行 |
| authz | 1 行 |
| modelgrant | 5 行 |
| ui_permission | 1 行 |
| audit | 2 行 |
| security | 1 行 |
