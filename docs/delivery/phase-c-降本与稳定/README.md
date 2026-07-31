# 阶段 C 存证：降本与稳定

| 项 | 内容 |
|----|------|
| 对应 WBS | §11.7 阶段 C（2d） |
| 产出日期 | 2026-07-31 |
| 蜂群波次 | 第三波（Layer 2-3） |
| Agent 数量 | 3 Agent 并行 |

---

## [R1] `routing/` — 12 策略可插拔路由引擎

**文件：** 17 个（4 核心 + 13 策略实现） · 全部测试通过

### 核心文件

| 文件 | 行数 | 职责 |
|------|------|------|
| `strategy.go` | 189 | Strategy 接口 + Candidate + RouteProfile + Migrate |
| `registry.go` | 62 | 全局策略注册表（Register/GetStrategy/GetRegistered） |
| `profile.go` | 419 | 档案 CRUD + ExecuteProfile 管道执行（δ>0.20 拒绝保存） |
| `decision.go` | 48 | Decision 决策日志 |
| `profile_test.go` | 412 | 13 测试（故障转移/δ帽/合规硬策略/影子模式） |

### 12 策略实现（`strategies/` 子目录）

| 文件 | 策略 | 核心规则 |
|------|------|---------|
| `compliance.go` | S-COMPLIANCE | INTERNAL_ONLY→剔除 external；硬策略不可关闭 |
| `health.go` | S-HEALTH | down→剔除；degraded→扣 5 分 |
| `priority.go` | S-PRI | 高优先级组未耗尽→剔除低优先级 |
| `weight.go` | S-WEIGHT | 权重归一化 0~10 打分 |
| `cost.go` | S-COST | EstSell 最低→满分 10，其余按比例 |
| `latency.go` | S-LATENCY | EWMA 延迟最低→满分 10 |
| `error.go` | S-ERROR | 错误率 100%→剔除；score=(1-rate)×10 |
| `rate.go` | S-RATE | 近期 429→降权 8 分；压力系数扣 0~5 |
| `affinity.go` | S-AFFINITY | 会话命中→+8 分 |
| `tag.go` | S-TAG | 业务标签不匹配→剔除；匹配→+7 分 |
| `cache.go` | S-CACHE | 缓存候选扣 10 分→最后手段 |
| `classify.go` | S-CLASSIFY | simple→+10；complex→-5（不鼓励降级） |

**管道顺序：** S-COMPLIANCE (Filter) → δ 价格帽 → S-CLASSIFY (Score) → 其余策略 (Filter+Score) → 按 Score 降序选取最优

---

## [R2] Gateway Pipeline + API Handlers

**文件：** 5 个 · go build + go vet 零告警

### `pipeline.go`（419 行）
14 步数据面管线编排器。`Pipeline` 结构体通过依赖注入组合所有步骤：
```
鉴权 → 安全钩子 → ModelGrant → 定价 → 预算帽 → 冻结 → 路由 → 上游调用 → 规范化 → 结算 → 审计
```
每步独立记录审计事件（步骤编号/名称/状态/耗时），PipelineResult 携带全部中间输出。

### `gov_handlers.go`（375 行）+ `gov_handlers_fund.go`（208 行）+ `gov_handlers_abac.go`（269 行）
治理 API HTTP handlers，涵盖 ~55 端点：
- 统一路由注册函数 `RegisterGovHandlers(mux, deps)`
- `GovDependencies` 聚合全部 11 个包的服务注入
- ABAC 鉴权中间件（`requireGovAuth`）
- net/http 标准库 + http.ServeMux + HandleFunc（与 TokenHub 一致）

### `store_integration.go`（328 行）
StartCall 事务插桩适配器：
- `NoopIntegrator` — 默认空操作（渐进替换）
- `DefaultIntegrator` — 生产实现（含 security.Hook / ModelGrant / 定价 / fund.Store）
- TokenHub 原有 store.go 和 http.go **零修改**

---

## [R3] Frontend — 8 模块管理控制台

**文件：** 30 个 · Next.js 16 App Router · TypeScript · lucide-react

### 共享组件（`_components/`，5 个）

| 组件 | 行数 |
|------|------|
| StatCard | 62 |
| DataTable（排序/分页/搜索/骨架屏） | 222 |
| ConfirmDialog（ESC/遮罩关闭） | 104 |
| CodeBlock（JSON+复制+折叠） | 90 |
| ErrorAlert（重试+关闭） | 70 |

### 8 模块页面

| 模块 | 路由 | 行数 | 功能要点 |
|------|------|------|---------|
| Dashboard | `/gov/dashboard` | 284 | 时段选择器 + 4 StatCard + 消耗柱状图 + 拦截统计 |
| Parties | `/gov/parties` | 668 | Party 列表+创建对话框+边管理面板+成员CRUD |
| Fund | `/gov/fund` | 502 | 账户卡片+划拨对话框+流水历史+清算确认 |
| Pricing | `/gov/pricing` | 547 | 价目列表+5模式编辑器+双轨预览+JSON视图 |
| Routes | `/gov/routes` | 536 | 档案列表+12策略开关/滑块+δ滑杆(≤20%) |
| ABAC | `/gov/abac` | 811 | 4标签：角色/策略/绑定/模拟评估器（含匹配链） |
| UI Permissions | `/gov/ui-permissions` | 592 | 3标签：菜单树/路由权限/按钮绑定 |
| Audit | `/gov/audit` | 356 | 筛选列表+详情快照并排对比+不可变标记 |

**布局：** 左侧 8 项导航 + 活跃路由高亮 + 版本徽章(v3.2.0)
**合规：** 每个页面独立 loading.tsx（骨架屏）+ error.tsx（重试UI）
