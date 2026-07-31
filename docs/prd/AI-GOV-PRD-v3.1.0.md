# 企业级 AI 智能网关治理平台（Token 治理底座）
## 产品需求与融合架构完整方案

| 项 | 内容 |
|----|------|
| 文档版本 | **3.1.0（全领域守恒定理 + ABAC 安全治理 + 完整闭环）** |
| 文档状态 | 可立项、可详细设计、可开发、可测试、可验收、可商用交付 |
| 日期 | 2026-07-31 |
| 读者 | 产品、研发、测试、UE、财务、安全、架构、交付、运维、合规 |
| 基线代码 | TokenHub v0.4.0（主仓库底座，Apache 2.0） |
| 吸收代码 | AxonHub itemCode 计价引擎 + 渠道探针 + 多维负载评分（llm/ LGPL-3.0，动态链接或 sidecar） |
| 参考设计 | ai-gov.sql 安全治理面（ABAC 引擎、UI 权限、审计链、复式账本思想） |
| 基线 DDL | `schema/ai-gov-fusion-minimal.sql`（29 表 + 本版 ABAC/UI 治理扩展） |

---

# 第 0 章 文档说明、设计哲学与守恒定理

## 0.1 文档性质

本文是独立完整的**产品需求规格 + 融合架构详细设计 + DDL 数据模型 + 安全治理规范**四合一文本。阅读与执行本文无需依赖任何其它文稿。所有核心需求、守恒定理、物理世界模型、安全治理架构、功能矩阵、数据纪律、验收标准与分期交付均在正文中以可执行粒度写明。

## 0.2 产品设计哲学与核心洞察

**Token 是 AI 时代的电力。但 Token ≠ 电力——同等数量 Token 在不同模型下的价格、业务价值完全不同，简单累加式机械统计毫无意义，必须具备业务感知能力。**

本产品不运营模型供给、不对外倒卖 Token，只在企业内部提供：

1. 统一模型调用入口
2. 精确计量与双轨计价
3. 主体经费账本、划拨与清算
4. 安全控制与不可抵赖审计

**核心矛盾：厂商总账单 ≠ 企业内部分户电表。** 模型供应商仅能统计单一账号用量，无法关联企业内部组织架构。企业普遍多云、多模型混合使用，各家厂商 Token 计价规则、统计口径不统一，多张异构账单难以统一核算。

**差异化定位：普通 API 中转工具解决「请求能不能送达模型」；本产品解决「AI 能源如何计量、分配、管控、追溯」。这不是一个 API 通道，这是一台 AI 财务精算机。**

**目标闭环：**

```text
模型接入 → 安全拦截 → 智能调度 → 实时扣费 → 资金回笼与清算 → 对账审计
```

**行业趋势判断：** 随着 Agent 大规模落地、全员 AI 助手普及，Token 将成为企业持续性核心生产成本。企业关注点将从「模型单价」转向「资源分配、投入产出、Token 使用效率」。管理公理：无计量则无管理；无归因则无法优化；缺少自有 Token 网关，企业无法真正掌控 AI 资产。长期趋势：每家规模化企业都会部署属于自己的 Token 智能电表。

## 0.3 全领域守恒定理

以下定理是本产品的**不可配置取消的宪法级约束**，覆盖财务、数据、权限、调度、审计五个治理域。任何功能设计、代码实现、测试用例、运维操作均不得违反。

### 0.3.1 财务守恒定理

| 编号 | 定理 | 违反示例 | 检测手段 |
|------|------|---------|---------|
| F-CON-01 | **余额守恒**：任何余额变更必须有对应流水记录；禁止无流水改账 | 直接 UPDATE accounts.available_balance 绕过 ledgers | `trg_append_only` 触发器 + 定期余额核对 |
| F-CON-02 | **划拨守恒**：划拨操作一方减少、另一方增加，金额相等，同一事务 | 划出 100，划入 99（差额消失） | 事务内双边校验：`abs(src_delta + dst_delta) ≤ ε` |
| F-CON-03 | **冻结守恒**：冻结金额 = 可用余额减少量；解冻金额 ≤ 冻结金额；结算金额 ≤ 冻结金额 | 冻结 100 但可用余额只减 50；结算 120 超过冻结 100 | `available_balance + frozen_balance = 常数`（在无其它操作的时间窗内） |
| F-CON-04 | **禁止负余额**：默认成功路径不允许账户余额为负 | 先调用后欠费；透支未记录 | CHECK `available_balance >= 0`（应用层校验前快照） |
| F-CON-05 | **幂等写**：所有资金写操作（划拨、清算、补偿）必须支持幂等键，重复请求不产生二次记账 | 网络重试导致同一划拨执行两次 | `idempotency_records` UNIQUE 抢占 |
| F-CON-06 | **管理员也不例外**：平台管理员也必须经账本服务与流水变更余额，不得提供绕过守恒的后门接口 | 超级管理员直接 SQL 改余额 | 无直接 DB 写入权限；仅通过 fund API |

### 0.3.2 数据安全守恒定理

| 编号 | 定理 | 违反示例 | 检测手段 |
|------|------|---------|---------|
| D-CON-01 | **数据不越权**：所有列表/详情/导出接口必须应用数据范围过滤器；禁止仅凭知道 UUID 读取未授权 Party 数据 | `/api/usage?party_id=<别人的ID>` 返回数据 | 中间件强制 Scope 注入；API 测试用例覆盖 IDOR |
| D-CON-02 | **数据不出境**：标记为 INTERNAL_ONLY 的主体、人或 Key，其请求不得产生任何外网上游流量 | INTERNAL_ONLY 用户的请求被路由到海外 OpenAI | 路由层 S-COMPLIANCE 硬策略 + DNS 层阻断 |
| D-CON-03 | **密钥不透传**：上游 API Key 仅保存在网关侧加密存储；调用方只持有企业下发的网关 Key；完整明文不落日志、不二次回显 | 上游 Key 出现在日志中；前端回显完整 Key | 日志脱敏中间件；API 响应仅返回 `key_prefix` + `key_suffix` |
| D-CON-04 | **审计不可篡改**：所有管理员配置变更必须保存 before/after 快照；审计日志不可编辑、不可删除；保留不少于 180 天 | 管理员修改 δ 后删除审计记录 | `audit_events` 表应用层禁止 UPDATE/DELETE；定期哈希锚定 |

### 0.3.3 权限守恒定理

| 编号 | 定理 | 违反示例 | 检测手段 |
|------|------|---------|---------|
| A-CON-01 | **四轴正交**：data、fund、iam、routing 四轴权限独立判定；禁止一轴推导另一轴特权 | 有 routing 权限的人自动获得划拨权 | 鉴权中间件逐轴独立校验；无隐式继承 |
| A-CON-02 | **最小权限默认**：未显式授予即拒绝；新用户默认仅拥有本人用量查看 + 持有 Key 的调用能力 | 新用户默认能看全平台报表 | 默认角色仅含 `usage.read:self` + `key.call:self` |
| A-CON-03 | **职责分离**：仅有 routing 轴权限者不能划拨；仅有 fund 轴权限者不能改全局路由与上游密钥；审计角色默认只读 | 路由管理员顺便划拨经费 | ABAC 策略中显式互斥约束 |
| A-CON-04 | **模型权限独立**：模型可用范围与资金扣费主体必须解耦；允许调用某模型不等于有钱调用；有钱不等于能调用所有模型 | ModelGrant 只检查一次，绕过 DENY 规则 | ModelGrant deny 优先于 allow；每次调用前强制校验 |
| A-CON-05 | **Leader 不万能**：禁止仅因 Leader 头衔自动拥有全平台数据/资金/模型权限；Leader 权限必须显式 Grant | 部门负责人默认能看所有人日志、划拨所有经费 | Leader 模板是显式 Grant 操作，可审计、可撤销 |

### 0.3.4 调度守恒定理

| 编号 | 定理 | 违反示例 | 检测手段 |
|------|------|---------|---------|
| S-CON-01 | **账户锁定**：全流程扣费账户 = 鉴权时 Key 绑定的账户；调度策略不得修改 account_id | 路由策略将请求转发到另一个账户扣费 | account_id 在鉴权时注入 context，调度层只读 |
| S-CON-02 | **价格锚定**：候选落地内部价不得超过 P_request × (1+δ)；δ 默认 0，硬上限 20%；无合格候选时拒绝调用，不得先调用后欠费 | δ=30% 被保存；无候选时仍调用最便宜的模型 | δ 配置保存时 CHECK ≤20%；候选集为空直接返回 NO_ROUTE_WITHIN_PRICE_CAP |
| S-CON-03 | **δ 变更审计**：δ 的任何修改必须记为关键配置变更审计（操作者、档案 ID、前后值、时间），不可仅写普通操作日志 | 某人将 δ 从 0 改为 20% 无审计记录 | audit_events 表 before/after 快照强制写入 |

### 0.3.5 审计守恒定理

| 编号 | 定理 | 违反示例 | 检测手段 |
|------|------|---------|---------|
| AU-CON-01 | **全链路可追溯**：每次调用必须记录 request_id、人、Key、账户、主体、模型、渠道、用量分项、双轨金额、路由结果、安全结果、耗时 | 调用日志只有 request_id 和 status_code | request_logs 表字段完整性校验 |
| AU-CON-02 | **资金操作全审计**：所有资金写操作（划拨、清算、补偿、冻结/解冻/结算）必须关联 audit_event_id | 一笔划拨成功但没有审计记录 | 资金 API 层强制写入 audit_events + 关联回 ledger |
| AU-CON-03 | **配置变更全快照**：价目、路由、授权、资金规则、主体关键信息、δ、预算帽的任何修改保存变更前与变更后快照 | 修改了模型价格但审计日志只写"已修改" | audit_events 强制 before_snapshot + after_snapshot NOT NULL |

## 0.4 能力优先级

| 优先级 | 能力域 | 说明 |
|--------|--------|------|
| **P0** | 统一接入与密钥托管 | 消灭野连接 |
| **P0** | **财务治理与计量闭环** | 双轨 cost/sell、Party 账本、划拨清算、预算帽分码——**第一护城河** |
| **P0** | **安全治理体系**（ABAC 引擎 + 四轴授权 + ModelGrant + UI 权限 + 审计链） | 数据不越权、权限不越权——**国企/金融合规底线** |
| **P0** | 调度与资金强制边界 | 价格帽 δ、账户锁定、禁止先调后欠费 |
| **P0** | 治理 API 与资金写幂等 | 与控制台对等，Idempotency-Key |
| **P1** | 可插拔路由策略矩阵与降本高可用 | 12 种策略可组合 |
| **P1** | 基础运营报表与仪表盘 | |
| **P2** | 内容安全与出网强化、配置变更快照强化、上游自动对账 | |
| **架构 P0** | 数据面安全扩展点 | 阶段 B 预留钩子 |

---

# 第 1 章 产品综述

## 1.1 要解决的问题

1. 员工与团队私自采购上游 API Key，调用黑箱，无法统一禁用与审计。
2. 财务只能拿到厂商总账单，无法按组织、项目、人员、应用做内部结算与毛利分析。
3. 成本失控：无预算预扣、无实时阻断、无双轨；预算帽与「账户真没钱」若同一错误码则运营无法精细抓手。
4. 合规风险：敏感数据可能经未审批路径出境；管理员改配置缺少不可篡改留痕；权限模型若存在隐式推导则无法满足等保 2.0 三级审计要求。
5. 可用性与成本：仅写死四层降级不足以支撑商业灵活组合；调度不受内部价格约束会击穿账本。

## 1.2 价值主张

| 主张 | 说明 |
|------|------|
| 统一治理 | 全部模型调用经单一企业网关，消灭野连接 |
| 财务闭环 | 主动预算、预扣与实扣、主体间划拨与清算、双轨计量、预算帽分码、对账 |
| 安全合规 | 密钥托管、出网策略、内容安全、ABAC 引擎 + 四轴正交授权 + UI 权限治理 + 全链路不可篡改审计 |
| 降本增效 | 在准确计量与价格约束前提下，按可插拔策略矩阵做成本/健康/权重/亲和/智能分类等调度 |

## 1.3 产品边界

**本产品负责：**

- 安全治理体系：ABAC 策略引擎、四轴正交授权（data/fund/iam/routing）、ModelGrant 模型访问控制、UI 菜单/路由/操作权限、管理员操作不可篡改审计
- 身份、成员、主体（Party：组织与项目）、统一关系边
- 上游模型与私有化资源接入、上游密钥托管
- 调用接入（OpenAI / Anthropic Messages / Gemini / Embeddings / Images 六协议兼容）、安全拦截扩展点、可插拔路由调度
- Token 计量、双轨计价（cost/sell）、多模态计价（文本/图片/音频/视频）、固定摊销模式、缓存折扣
- 账本、划拨、预扣结算、预算帽（Account 级 + ModelGrant 模型级）、清算、组织变更
- 调用审计、配置变更审计、对账与成本报表
- 管理控制台与对等的控制面治理 API
- 运维可观测

**本产品不负责：**

- Agent / 工作流编排、Skill 市场、面向终端用户的业务对话产品 UI
- 作为云厂商对外售卖 Token 或模型本身
- 同一请求拆分到多个账本分摊扣费
- 默认透支信用（负余额作为正常成功路径）
- 由业务客户端负责冻结续期
- 以公链/联盟链作为热路径余额系统记录

**预留扩展点**（架构级设计，非功能实现）：
- 对外 Token 售卖：预留 `tenants` 表（多租户隔离）、外部定价层级 JSON 结构、用量同步 gRPC 接口签名
- 多币种、复杂审批流、HR/ERP 深度双向同步

**与上层契约：** OpenAI 兼容 HTTP API（六协议兼容）；业务标签（user/task/agent/skill 等）仅用于归因，不替代鉴权与扣费账户。

## 1.4 目标用户

| 角色 | 主要诉求 |
|------|----------|
| 平台管理员 / 架构师 | 部署、资源池、路由档案、系统配置、上游密钥、模型目录、ABAC 策略管理 |
| 安全 / 合规官 | 出网策略、ABAC 权限审计、UI 权限审查、拦截监控、审计检索、保留策略合规 |
| 财务 / 成本控制 | 账本、划拨、内部加价、预算帽、报表、对账、清算 |
| 组织或项目负责人（Leader） | 本主体经费、成员与 Key、消耗可见、模型范围（在 Grant 授权内） |
| 实体人（员工/外包） | 持有 Key、合法调用、查看本人用量 |

## 1.5 典型用户旅程

### 旅程 1：财务人员——部门预算划拨与监控

```text
1. 财务管理员登录控制台 → 进入"资金管理"模块（需 fund 轴 allocate + budget.write 权限）
2. 查看公司总盘余额：¥500,000
3. 选择"AI 研发部"（Party type=org）→ 配置月预算帽 ¥80,000，告警比例 80%
4. 执行划拨：从公司总盘划入 ¥80,000 → 系统校验 fund 轴授权 + parent 通道 + 幂等键 → 事务内双边记账
5. 为研发部成员张三创建个人 Account，通过 allocates 边划入 ¥2,000 月度个人经费
6. 月底查看：仪表盘显示研发部消耗 ¥72,000（预算使用率 90%），张三消耗 ¥1,800
7. 收到告警：研发部已达告警比例 80%，但未达上限，调用继续
```

### 旅程 2：部门 Leader——团队消耗管理与模型授权

```text
1. AI 研发部 Leader 登录控制台 → 仪表盘显示本部门消耗趋势、成员用量排行
2. 发现成员李四的 GPT-4 消耗异常高 → 查看李四的调用明细：大量简单翻译任务使用了 GPT-4
3. 在"路由档案"中为研发部配置 ha-cost 策略（PRI + HEALTH + WEIGHT + COST），将简单任务自动路由到低成本模型
4. 在"模型权限"中为研发部授予 GPT-4、Claude-3.5、DeepSeek-V3 的 ModelGrant（ALLOW），但禁止使用 GPT-4.5-preview（DENY）
5. 下周复查：GPT-4 消耗下降 40%，整体成本下降 25%
```

### 旅程 3：普通员工——日常调用与个人额度管理

```text
1. 员工张三在 IDE 中配置网关 Key（sk-xxxx...）→ 发起代码生成请求
2. 网关：Key 鉴权 → 人未禁用 → ModelGrant 校验（研发部有 GPT-4 权限）→ 价格过滤 → 预算帽检查（部门预算剩余 10%）→ 个人账户余额检查（¥1,800 剩余 ¥200）→ 冻结 ¥15 → 调度 → 上游调用 → 结算 ¥12
3. 张三在控制台查看本人用量：本月已消耗 ¥1,800 / ¥2,000 个人额度
4. 额度耗尽后，下一次调用返回 INSUFFICIENT_BALANCE，张三联系 Leader 申请追加经费
```

---

# 第 2 章 物理世界主体模型（Party 多态）

## 2.1 设计出发点

真实企业 AI 经费与编制形态是多态的。系统不得假设唯一结构「公司 → 部门 → 项目 → 人」，也不得强制「项目必须挂在某个部门下」。企业是图，不是树。

## 2.2 典型场景

| 场景 | 组织 | 项目 | 关系与资金 |
|------|------|------|------------|
| 部门日常额度 | 有常设部门 | 可不建项目 | 部门主体持有账本，人的 Key 扣部门账本 |
| 部门主责专项 | 有 | 有 | 可用 owns 关联；项目可自有账本 |
| 公司级战略项目 | 多个事业部 | 有，可与部门平级 | 多 sponsors 出资到同一项目账本；独立 Leader |
| 跨主体协作 | 内外部组织 | 有 | 人可只加入项目、不加入内部部门 |
| 矩阵制 | 产品线、区域等 | 有 | 多协作边；一次调用只扣一个账本 |
| 结项还钱 | — | 结束 | 剩余资金回流；Key 失效或改绑 |
| 个人实验经费 | 部门下发 | — | 部门通过 allocates 边拨入个人 Account；个人 Key 扣个人 Account |
| 组织合并 | A+B→C | — | A/B 清算余额划入 C；历史流水关联 A/B |
| 组织拆分 | C→A+B | — | C 按比例划出到 A/B |

## 2.3 统一主体 Party

- 组织（org）与项目（project）同一层语义，均可：持有账本、拥有成员、指定 Leader、作为授权资源、被授予 ModelGrant。
- 项目不必然从属于某个组织。
- 控制台可分「组织」「项目」导航，底层模型与权限引擎必须统一。

## 2.4 关系边

| 关系类型 | 含义 | 与资金的关系 |
|----------|------|----------------|
| parent | 组织汇报树上下级 | 默认允许上级 → 下级划拨 |
| sponsors | 出资 | 默认允许出资方 → 被出资方划拨 |
| owns | 主责 / 主办 | 不自动产生划拨权 |
| participates | 协作 | 不自动产生划拨权 |
| allocates | Party → Person 注入个人经费 | 允许；仅用于 Person Account |
| merged_into | 组织合并（A 并入 C） | A 清算余额划入 C；走清算类流程 |
| split_from | 组织拆分（C 拆出 B） | C 按比例划出到 B；走清算类流程 |

无关系时：独立项目或独立组织费用池均可成立。

## 2.5 人、Key、账本、Leader

| 对象 | 物理对应 | 规则 |
|------|----------|------|
| Person | 实体人 | 通过成员关系进入一个或多个 Party |
| Person Account | 可选个人小金库 | 由所属 Party 通过 allocates 边注入；仅支持预算帽 + Key 消费扣费；不支持对外划拨 |
| API Key | 消费凭证 | 必须绑定且仅绑定一个 account_id；调用与扣费的唯一入口；必须归属实体人 |
| Account | 小金库 | 归属某个 Party（或个人）；可用余额与冻结金额；含预算帽热字段 |
| Leader | 负责人 | 对本 Party 默认责任角色；权限以显式 Grant 为准（A-CON-05：禁止头衔万能） |

**资金流转强制规则（F-CON-01~06 约束）：**

1. 划拨：一方减少、另一方增加，金额相等，同一事务。
2. 预扣：可用余额转入冻结，必须带 freeze_id 与过期时间；预扣前可经预算帽 + ModelGrant 模型级预算检查。
3. 结算：按实际内部价从冻结结清，多退少补。
4. 清算/组织变更：按状态机阻断新调用、排空冻结后回流资金并收口 Key。
5. 任何余额变更必须有流水；禁止无流水改账；默认不允许成功路径产生负余额。
6. 关键资金写操作必须支持幂等。

**消费强制规则：** 内部结算金额（sell）只从 Key 绑定账户扣除，归因到 Key 所属实体人；业务标签只做报表归因，不形成第二套资金。

## 2.6 可迭代与扩展原则

- **稳定核心：** 计量、双轨、账本守恒、Key 扣费、ABAC 四轴授权、ModelGrant、统一入口、价格上限与账户锁定、冻结生命周期、清算/组织变更状态机、治理 API 幂等、预算帽分码、审计不可篡改。
- **扩展面：** 新关系类型、新 itemCode、新路由策略、价目分时、安全钩子、对账连接器、划拨白名单。
- **存储纪律：** 热字段原子；复杂进 JSON；禁止无限加表加列。
- **发布演进：** 可先只计量双轨再实扣；策略可 shadow 后再生效。

---

# 第 3 章 开源底座选型与融合架构

## 3.1 三方能力矩阵

| 维度 | TokenHub v0.4.0（底座） | AxonHub（吸收方） | 融合架构 |
|------|--------------------------|-------------------|----------|
| 语言/框架 | Go 1.26 + GORM + Next.js 16 | Go 1.26 + Ent + Uber fx + React 19 | Go + GORM + Next.js（TokenHub 主线） |
| 数据库 | SQLite / PostgreSQL 双模 | SQLite / PG / MySQL / TiDB | SQLite（MVP）/ PostgreSQL（生产） |
| 入站协议 | OpenAI / Anthropic Messages / Responses / Codex / Embeddings / Images | 9 类（+ Gemini / AI SDK / Jina / Doubao / Copilot / Claude Code） | 6 协议（TokenHub 主线）+ 按需扩展 |
| 路由策略 | 优先级+权重+策略+亲和+健康 6 策略 Failover | 4 模式 × 7 策略 + 三态熔断 + 探针 | 策略矩阵 12 种（含 S-CLASSIFY）+ 档案组合 |
| 计价 | 单轨 input/output_price_per_1m | itemCode 级 4 模式 + JSON 价目 + costItems | **双轨** cost/sell + 5 种计价模式（含 amortization_fixed）+ 缓存折扣 |
| 资金闭环 | ❌ 事后统计 cost_usd | ❌ 仅 total_cost 字段 | **新建** fund 包 |
| 授权模型 | RBAC（admin/member） | 多角色 + Ent Privacy | **新建** ABAC 引擎 + grants 四轴 + model_grants + UI 权限 |
| 部署 | Docker Compose + systemd | K8s Helm + Docker Compose | Docker Compose（MVP）+ K8s Helm（生产） |

## 3.2 融合策略

```
TokenHub（主仓库底座，Apache 2.0）
    │
    ├── 直接复用：Provider 适配器、6 协议兼容、API Key 管理、限流配额桶、
    │             Prometheus 指标、SQLite/PG 双模、systemd 加固、多实例协调
    │
    ├── 扩展改造：routing → 策略矩阵引擎（抽象 Strategy 接口，新增 S-CLASSIFY）
    │             authz → ABAC 引擎（sys_action_catalogs + sys_roles + sys_access_policies）
    │             party → parties + party_edges（统一主体，含 7 种关系边）
    │             models → 双轨 sell 价格热字段 + 多模态 itemCode
    │             ui → sys_ui_menus + sys_ui_routes + sys_ui_action_bindings
    │
    ├── 从零构建：fund（accounts/ledgers/freezes/allocations/liquidations）
    │             pricing（model_prices 双轨 JSON + 5 种计价模式 + 缓存折扣）
    │             idempotency（Idempotency-Key 原子抢占）
    │             modelgrant（ModelGrant ALLOW/DENY + quota_limit）
    │
    └── AxonHub 吸收：itemCode 计算口径（cost_calc.go）→ pricing 包
                      4 模式 × 7 策略 + 三态熔断 → routing 包
                      渠道探针 channel_probes → 运维面
                      上游配额状态 provider_quota_status → 运维面
                      llm/ 子模块：首选通过 Go plugin 动态链接；
                      若不可行，则 AxonHub 作为独立 sidecar 通过 HTTP/gRPC
                      暴露计价+评分 API，TokenHub 通过适配器调用
```

## 3.3 路由：可插拔策略矩阵（12 种）

| 策略代码 | 名称 | 作用 | 来源 | 可禁用 |
|----------|------|------|------|--------|
| S-COMPLIANCE | 合规网络 | INTERNAL_ONLY 等硬过滤 | 新建 | **硬策略，不可对受限主体关闭** |
| S-PRI | 优先级分组 | 主备硬分组与顺序 | TokenHub | 可 |
| S-HEALTH | 健康与熔断 | 三态熔断（Closed/HalfOpen/Open）+ 探针 | TokenHub + AxonHub | 可 |
| S-WEIGHT | 权重与负载 | 按权重与历史负载分配 | TokenHub | 可 |
| S-AFFINITY | 会话亲和 | 同会话优先同一渠道 | TokenHub | 可 |
| S-COST | 成本感知 | 合格集内偏向低价 | TokenHub + AxonHub lowestCost | 可 |
| S-LATENCY | 延迟感知 | TTFT 或端到端延迟 | AxonHub | 可 |
| S-ERROR | 错误率感知 | 近期成功率惩罚 | AxonHub | 可 |
| S-RATE | 限流感知 | 降低触达 RPM/TPM/429 概率 | AxonHub | 可 |
| S-TAG | 业务标签 | 按请求 Tag 定向或加权 | TokenHub | 可 |
| S-CLASSIFY | 智能任务分类 | 轻量级模型预判任务复杂度，简单任务自动路由低成本模型 | 新建 | 可（阶段 C 可选实现，需额外推理开销） |
| S-CACHE | 缓存兜底 | 最后手段降级渠道 | TokenHub | 可 |

**档案预设：**
- `simple-failover`：PRI + HEALTH
- `ha-cost`：PRI + HEALTH + WEIGHT + COST + AFFINITY
- `compliance-strict`：S-COMPLIANCE 强制 + PRI + HEALTH
- `intelligent-save`：S-CLASSIFY + S-COST + S-HEALTH（阶段 C）

---

# 第 4 章 双轨计价

## 4.1 费用项 itemCode 基线

| itemCode | 含义 | 来源 |
|----------|------|------|
| prompt_tokens | 输入文本 Token | AxonHub cost_calc.go |
| completion_tokens | 输出文本 Token | AxonHub cost_calc.go |
| prompt_cached_tokens | 缓存读 Token | AxonHub cost_calc.go |
| prompt_write_cached_tokens | 缓存写 Token | AxonHub cost_calc.go |
| completion_reasoning_tokens | 推理输出 Token | AxonHub cost_calc.go |
| prompt_audio_tokens | 音频输入 Token | AxonHub usage_log |
| completion_audio_tokens | 音频输出 Token | AxonHub usage_log |
| image_count | 图片张数 | AxonHub Image Generation |
| image_resolution_tier | 图片分辨率档位（standard/hd/4k） | AxonHub Image Generation |
| video_duration_seconds | 视频时长（秒） | AxonHub Video Generation |

## 4.2 计价模式

| 模式 | 说明 | 适用场景 |
|------|------|---------|
| flat_fee | 按次固定费用 | API 调用次数计费 |
| usage_per_unit | 按单位用量（通常每 1M Token 或每张图片） | Token 计费、图片计费 |
| usage_tiered | 阶梯价格（分段累计） | 大客户折扣 |
| usage_volume | 总量落档后整单同一单价 | 包量套餐 |
| amortization_fixed | **按月/年固定摊销额，不按 Token 计量** | 私有化部署模型（vLLM 推理集群）内部成本分摊 |

## 4.3 双轨 + 缓存折扣

| 轨道 | 含义 | 折扣规则 |
|------|------|---------|
| cost | 上游成本 | 上游实际价格 |
| sell | 内部结算价 | sell = cost × (1 + markup)；对缓存类 itemCode（prompt_cached_tokens / prompt_write_cached_tokens）可配置 `cache_discount_ratio`（如 0.5 = 50% 折扣），sell = cost × (1 + markup) × cache_discount_ratio |

## 4.4 价目 JSON 结构

```json
{
  "items": [
    {
      "itemCode": "prompt_tokens",
      "cost": {"mode": "usage_per_unit", "rate": 0.002},
      "sell": {"mode": "usage_per_unit", "rate": 0.003}
    },
    {
      "itemCode": "prompt_cached_tokens",
      "cost": {"mode": "usage_per_unit", "rate": 0.0005},
      "sell": {"mode": "usage_per_unit", "rate": 0.00075,
               "cache_discount_ratio": 0.5}
    },
    {
      "itemCode": "image_count",
      "cost": {"mode": "flat_fee", "rate": 0.02},
      "sell": {"mode": "flat_fee", "rate": 0.03}
    },
    {
      "itemCode": "amortization_fixed",
      "cost": {"mode": "amortization_fixed", "monthly_rate": 5000.00},
      "sell": {"mode": "amortization_fixed", "monthly_rate": 5000.00}
    }
  ],
  "schedule": {"timezone": "Asia/Shanghai", "overrides": []}
}
```

## 4.5 用量规范化

适配层：上游 usage → 内部 itemCode；缺失记 0 + `usage_incomplete` + 标记进对账差异；禁止伪造明细。

---

# 第 5 章 预算帽配置数据模型

## 5.1 双层预算体系

| 层级 | 挂载点 | 字段 | 语义 |
|------|--------|------|------|
| **Account 级** | accounts 表 | budget_limit_amount 等 | 控制"这个账户还有多少钱可花" |
| **ModelGrant 级** | model_grants 表 | quota_limit | 控制"这个主体在这个模型上最多花多少" |

两层取**交集**：任意一层触发阻断即拒绝。

## 5.2 Account 级预算帽字段

| 字段 | 类型 | 含义 |
|------|------|------|
| budget_limit_amount | DECIMAL NULL | 预算上限；NULL=未启用 |
| budget_warn_ratio | DECIMAL NULL | 告警比例（如 0.80）；只告警不阻断 |
| budget_period | VARCHAR(24) DEFAULT 'none' | none / calendar_month / calendar_day / custom |
| budget_period_start | TIMESTAMPTZ | custom 起点 |
| budget_period_end | TIMESTAMPTZ | custom 终点 |
| budget_consumed_amount | DECIMAL NOT NULL DEFAULT 0 | 本周期已确认 sell 累计 |
| budget_version | BIGINT NOT NULL DEFAULT 0 | 配置乐观锁 |

## 5.3 ModelGrant 级预算字段

| 字段 | 类型 | 含义 |
|------|------|------|
| quota_limit | DECIMAL NULL | 该授权下累计消费上限；NULL=不限制 |

ModelGrant.quota_limit 与 Account.budget_limit_amount 取交集——两者取最严格的限制生效。

## 5.4 预扣判定顺序（修正后）

```text
1. Key 鉴权 → 人状态 → Account 绑定
2. 安全钩子（若启用）
3. ModelGrant 模型过滤（DENY 剔除 → ALLOW 候选集）
4. 构建候选渠道 → 价格合格集过滤（P_candidate ≤ P_request × (1+δ)，δ≤20%）
5. 若 ModelGrant.quota_limit 非空 → 模型级预算检查（失败 → MODEL_BUDGET_EXCEEDED）
6. 若 Account.budget_limit_amount 非空 → Account 级预算帽检查（失败 → BUDGET_CAP_EXCEEDED）
7. 冻结金额 = 合格候选集上预估 sell 最大值；可用余额 ≥ 冻结额（失败 → INSUFFICIENT_BALANCE）
8. 写入 freeze
9. 策略矩阵选路 → 托管上游密钥调用
10. 流式场景网关续期冻结
11. 用量规范化 → 双轨结算 → 审计
12. 返回兼容响应 + 分码错误
```

---

# 第 6 章 错误码定义全表

## 6.1 资金与额度

| code | HTTP | 含义 |
|------|------|------|
| BUDGET_CAP_EXCEEDED | 402 | Account 级预算上限命中（余额可能>0） |
| MODEL_BUDGET_EXCEEDED | 402 | ModelGrant 级预算上限命中 |
| INSUFFICIENT_BALANCE | 402 | 可用余额不足以完成本次冻结 |
| ACCOUNT_FROZEN_OR_CLOSED | 403 | 账户停用或清算中 |
| FREEZE_EXPIRED | 409 | 结算时冻结已失效 |
| IDEMPOTENCY_CONFLICT | 409 | 同键处理中或同键异参 |
| IDEMPOTENCY_REPLAY | 200 | 同键重放成功结果 |

## 6.2 鉴权与身份

| code | HTTP | 含义 |
|------|------|------|
| AUTH_INVALID_KEY | 401 | Key 无效或吊销 |
| AUTH_USER_DISABLED | 403 | 归属人已禁用 |
| AUTH_KEY_NO_ACCOUNT | 403 | Key 未绑定账户 |
| AUTHZ_DENIED | 403 | 控制面 ABAC 授权不通过 |
| MODEL_ACCESS_DENIED | 403 | ModelGrant 不允许该模型 |

## 6.3 路由与价格

| code | HTTP | 含义 |
|------|------|------|
| NO_ROUTE_WITHIN_PRICE_CAP | 422 | 无价格合格候选 |
| NO_ROUTE_AVAILABLE | 503 | 无可用健康路由 |
| ROUTE_COMPLIANCE_BLOCKED | 403 | 合规策略剔除全部候选 |

## 6.4 安全

| code | HTTP | 含义 |
|------|------|------|
| COMPLIANCE_NETWORK_BLOCKED | 403 | 网络策略阻断（如 INTERNAL_ONLY 请求外网模型） |
| CONTENT_BLOCKED | 403 | 内容安全阻断 |
| RATE_LIMITED | 429 | 网关或策略限流 |

## 6.5 上游与系统

| code | HTTP | 含义 |
|------|------|------|
| UPSTREAM_ERROR | 502 | 上游返回错误 |
| UPSTREAM_TIMEOUT | 504 | 上游超时 |
| INTERNAL_ERROR | 500 | 网关内部错误 |

---

# 第 7 章 安全治理体系

> **本章是本产品的合规核心。** 国企、金融、能源等行业的等保 2.0 三级审计要求决定了安全治理不是可选功能，而是交付底线。

## 7.1 安全治理全景架构

```text
                    ┌──────────────────────────────────────┐
                    │         ABAC 策略决策引擎              │
                    │  sys_access_policies + sys_roles      │
                    │  + sys_subject_role_bindings          │
                    │  + sys_action_catalogs                │
                    └──────────┬───────────────────────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        │                      │                      │
        ▼                      ▼                      ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────────┐
│  四轴正交授权  │    │  ModelGrant   │    │   UI 权限治理      │
│  data / fund  │    │  ALLOW/DENY   │    │  menus / routes   │
│  iam / routing│    │  + quota_limit│    │  / action_bindings│
└───────┬───────┘    └───────┬───────┘    └────────┬──────────┘
        │                    │                      │
        ▼                    ▼                      ▼
┌──────────────────────────────────────────────────────────┐
│                    审计不可篡改链                           │
│  audit_events (before/after 快照 + 不可 UPDATE/DELETE)     │
│  + 定期哈希锚定 (audit_chain_anchors)                      │
└──────────────────────────────────────────────────────────┘
```

## 7.2 ABAC 策略引擎

### 7.2.1 设计原则

ABAC（Attribute-Based Access Control）是本产品的**统一授权引擎**。所有控制面 API、数据面敏感操作、UI 可见性均由 ABAC 引擎判定。RBAC 是 ABAC 的子集——角色（Role）是主体属性之一，不作为独立授权体系。

**与简化的 `grants` 表的区别：** 直接 grant 模式（"用户 X 对资源 Y 有权限 Z"）在策略数量增长时产生组合爆炸。ABAC 通过策略（Policy）定义规则，通过属性（用户角色、部门、时间、IP）动态评估，支持策略复用、优先级排序、显式拒绝。

### 7.2.2 核心实体

| 表 | 职责 | 示例数据 |
|----|------|---------|
| `sys_action_catalogs` | 定义系统中所有可执行动作 | `fund.allocate`、`routing.price.write`、`data.usage.read`、`iam.key.create` |
| `sys_roles` | 定义角色模板 | `财务管理员`、`部门Leader`、`安全审计员`、`普通员工` |
| `sys_role_permissions` | 角色 → 动作的映射 | `财务管理员` → `fund.allocate, fund.ledger.read, fund.budget.write` |
| `sys_subject_role_bindings` | 主体 → 角色的绑定（含 scope） | `用户张三` → `部门Leader`（scope: party=AI研发部） |
| `sys_access_policies` | ABAC 策略规则 | "禁止 routing 角色执行 fund 动作"、"工作时间允许、非工作时间需审批" |
| `sys_access_policy_bindings` | 策略 → 主体的绑定 | `职责分离策略` → 全局生效 |

### 7.2.3 策略评估流程

```text
1. 提取主体属性：user_id → roles + party_memberships + ip + time
2. 提取资源属性：resource_type + resource_id + owner_party_id
3. 提取动作属性：action + axis
4. 加载适用策略：sys_access_policies WHERE 策略绑定到该主体或全局
5. 策略评估：
   a. 显式 DENY 规则 → 立即拒绝（DENY 优先）
   b. 显式 ALLOW 规则 → 匹配则通过
   c. 角色权限匹配 → 通过
   d. 无匹配 → 默认拒绝（A-CON-02：最小权限默认）
6. 记录审计：所有授权决策记录到 audit_events（含 evaluate_result + matched_policy_ids）
```

### 7.2.4 四轴正交授权

| 轴 | 动作示例 | 禁止推导 |
|----|---------|---------|
| data | `usage.read`、`report.read`、`member.read` | 不能推导 fund 或 routing 权限 |
| fund | `balance.read`、`ledger.read`、`allocate`、`liquidate`、`budget.write` | 不能推导 data（其它 Party 日志）或 routing |
| iam | `key.create/revoke/rotate`（绑户 ∈ 允许集）、`user.disable`、`member.add/remove` | 不能绑无权账户；不能改价目 |
| routing | `price.write`、`route_profile.write`（含 δ）、`channel.write`、`upstream_secret.write`、`model_catalog.write`、`model_grant.write` | 不能改 account_id；不能划拨 |

### 7.2.5 职责分离强制策略

| 策略 | 规则 | 审计要求 |
|------|------|---------|
| 路由-资金分离 | 拥有 routing 轴任一写权限的主体，自动被 DENY fund 轴所有写权限 | 策略 ID：`SEP-ROUTE-FUND`，不可禁用 |
| 审计只读 | 拥有审计角色的主体，仅授予 data 轴读权限，DENY 所有写权限 | 策略 ID：`AUDITOR-READONLY`，不可禁用 |
| Leader 不万能 | Leader 角色不自动授予任何权限；必须通过显式 sys_subject_role_bindings 绑定 | A-CON-05 |
| 自审批禁止 | 划拨操作的发起人与审批人不得为同一主体 | 策略 ID：`NO-SELF-APPROVAL` |

## 7.3 模型访问治理（ModelGrant）

| 字段 | 含义 |
|------|------|
| principal_type | party / person / key / role |
| principal_id | 对应 ID |
| model_id 或 model_tag | 单个模型或标签组 |
| effect | allow / deny（**deny 优先于 allow**） |
| priority | 冲突解析辅助 |
| quota_limit | DECIMAL NULL；该授权下累计消费上限（与 Account 预算帽取交集） |

级联顺序：Key > Person > Party > 全局默认。禁止仅因 Leader 头衔自动拥有全平台模型调用权。

## 7.4 UI 权限治理

> **定理：用户只能看到和操作其被授权的功能。** UI 权限是 ABAC 引擎在展示层的投影，不是独立的权限体系。

### 7.4.1 核心实体

| 表 | 职责 | 示例数据 |
|----|------|---------|
| `sys_ui_menus` | 定义菜单树结构 | `{id:"menu-fund", parent_id:"root", label:"资金管理", icon:"wallet"}` |
| `sys_ui_routes` | 定义前端路由到菜单的映射 | `{route:"/console/fund/allocate", menu_id:"menu-fund-allocate"}` |
| `sys_ui_action_bindings` | 定义 UI 操作按钮 → ABAC 动作的映射 | `{button:"划拨执行", required_action:"fund.allocate", resource_type:"account"}` |

### 7.4.2 渲染规则

```text
1. 用户登录 → ABAC 引擎评估用户所有有效权限 → 生成权限快照
2. 菜单渲染：仅展示 sys_ui_menus 中用户至少拥有一个子操作权限的菜单项
3. 路由守卫：访问 /console/fund/allocate → 校验 route 对应菜单是否可见 + required_action 是否在权限快照中
4. 按钮渲染：每个操作按钮通过 sys_ui_action_bindings 关联 required_action → 无权限则隐藏或置灰
5. 权限变更：用户角色变更后，权限快照失效 → 下次请求时重新评估
```

### 7.4.3 UI 权限 vs ABAC 权限的关系

```
UI 权限（前端展示层）     ABAC 权限（后端执行层）
      │                        │
      │  隐藏按钮 ≠ 安全        │  拒绝请求 = 安全
      │                        │
      ▼                        ▼
  sys_ui_action_bindings    sys_access_policies
  required_action  ──────►  action + resource
```

**安全原则：** UI 隐藏是不让用户看到、减少误操作，但真正的安全在后端 ABAC 引擎。即使前端绕过了 UI 限制直接调用 API，ABAC 引擎仍会拒绝。前端和后端共享同一动作定义（sys_action_catalogs），确保不会出现"前端按钮对应一个不存在的后端权限"。

## 7.5 数据不越权

- 所有列表/详情/导出按 data 轴授权范围过滤（D-CON-01）
- 禁止仅凭知道 UUID 读取未授权 Party 数据（防 IDOR）
- 余额流水归 fund、用量日志归 data，可只授其一
- 数据面所有查询自动注入 `WHERE party_id IN (授权的Party集合)` 条件

## 7.6 审计不可篡改

- `audit_events` 表应用层禁止 UPDATE/DELETE（AU-CON-02）
- 所有管理员配置变更强制保存 before_snapshot + after_snapshot（AU-CON-03）
- δ、预算帽、ModelGrant、路由策略的关键变更必须记录操作者、前后值、时间
- 定期生成哈希锚定记录（audit_chain_anchors），防批量篡改
- 保留不少于 180 天；冷热分离存储

## 7.7 验收红线清单

1. 无流水改余额（F-CON-01）
2. 划拨无通道（非 parent/sponsors/allocates 方向 / 非白名单）
3. Key 无 account 调用
4. 调度改扣费账户（S-CON-01）
5. 先调后欠费（价格帽外候选仍调用，S-CON-02）
6. Leader 无 Grant 即全平台数据/资金/模型权限（A-CON-05）
7. iam 建 Key 绑到无权账户
8. 预算帽与余额不足返回同一错误码
9. ModelGrant deny 后仍可调用该模型（A-CON-04）
10. 前端隐藏了按钮但 API 未校验权限
11. UI 显示了一个按钮但对应的 sys_action_catalogs 中不存在该动作
12. 审计日志可被 UPDATE 或 DELETE
13. INTERNAL_ONLY 用户请求产生了外网上游流量（D-CON-02）

---

# 第 8 章 关键边界规则与流程

## 8.1 预扣、价格约束与结算

1. **请求锚定内部价 P_request：** 逻辑模型 + 价目 + 预估用量 → 内部 sell 预估金额。
2. **候选价格约束：** P_candidate ≤ P_request × (1+δ)；**默认 δ=0**；**硬上限 20%**；改 δ 必须关键配置审计（S-CON-03）。
3. **ModelGrant 模型过滤：** DENY 规则剔除 → ALLOW 候选集。
4. **ModelGrant 模型级预算检查：** 若 quota_limit 非空且 consumed + 本次预估 > quota_limit → MODEL_BUDGET_EXCEEDED。
5. **预算帽检查**（若启用）→ BUDGET_CAP_EXCEEDED。
6. **冻结金额** = 价格合格候选集上预估 sell 的最大值。
7. **调度**仅在合格集执行；**account_id 在鉴权时锁定，调度不得修改**（S-CON-01）。
8. **结算**按实际用量与落地价目算 cost/sell；多退少补。

## 8.2 划拨路径规则

| 路径 | 默认是否允许 | 方向 |
|------|--------------|------|
| parent | 是 | 仅上级 → 下级 |
| sponsors | 是 | 仅出资方 → 被出资方 |
| allocates | 是 | Party → Person Account |
| owns / participates | 否 | 不自动开通 |
| 无关系双方 | 否 | 除非白名单 + fund 授权 |

## 8.3 冻结生命周期与流式续期

- 默认 TTL 15 分钟（可配 1–60 分钟）。
- 流式：网关自动续期同一 freeze_id，不增加冻结金额；累计上限可配（如 2 小时）。
- 客户端不负责续期。

## 8.4 清算状态机

```text
active
  → liquidating_block_new    // 拒绝新调用与新冻结
  → liquidating_drain        // 等待冻结清零
  → liquidating_transfer     // 余额转入目标账户
  → liquidated               // Key 收口，主体只读
```

## 8.5 组织变更流程

```
组织合并：active → merging_block_new → merging_drain → merging_transfer → merged
组织拆分：active → splitting → split_completed（按比例划出）
```

与清算共享冻结排空机制，复用幂等和流水基础设施。

## 8.6 治理 API 幂等

| 项 | 规则 |
|----|------|
| 适用范围 | 划拨、清算、组织变更、资金补偿等写操作 |
| 机制 | `Idempotency-Key`（UUID v4，≤255） |
| DDL | `idempotency_records` UNIQUE(scope, actor_id, idempotency_key) |
| 抢占 | INSERT ON CONFLICT 原子抢占 |
| 行为 | 同键同指纹重放首次结果；异指纹拒绝（409 IDEMPOTENCY_CONFLICT） |

## 8.7 账本技术

热账本 = PostgreSQL/SQLite + 只追加 `ledgers`。不采用区块链。流水禁止 UPDATE/DELETE。可选定期哈希摘要归档 WORM 增强。

---

# 第 9 章 功能需求编号全表

## 9.1 统一接入与模型资源池

| 编号 | 名称 | 描述 | 优先级 |
|------|------|------|--------|
| RES-01 | 多上游接入 | 公有+国内 API、OpenAI 兼容私有化（vLLM/Ollama/KServe） | P0 |
| RES-02 | 上游密钥仓库 | 网关托管；加密；明文不落日志（D-CON-03） | P0 |
| RES-03 | 密钥操作权限 | 仅 routing 轴 upstream_secret.write | P0 |
| RES-04 | 健康与状态 | 可用/降级/不可用 | P0 |
| RES-05 | 资源标签 | 成本、区域、是否内网 | P1 |
| RES-06 | 兼容 API | OpenAI + Anthropic + Gemini + Embeddings + Images 六协议 | P0 |
| RES-07 | 请求标识与标签 | request_id；业务标签仅归因 | P0 |
| RES-08 | 用量规范化 | 映射 itemCode（含多模态）；缺失记 0 + usage_incomplete | P0 |

## 9.2 双轨计价

| 编号 | 名称 | 描述 | 优先级 |
|------|------|------|--------|
| PRI-01 | 价目配置 | 渠道×模型双轨 JSON；变更审计（AU-CON-03） | P0 |
| PRI-02 | 用量解析 | 规范化分项（含多模态 image/audio/video）；缓存与输入不双计 | P0 |
| PRI-03 | 费用计算 | cost/sell 总额与 costItems；含缓存折扣（cache_discount_ratio） | P0 |
| PRI-04 | 落账字段 | 双轨、分项、账户、人、Key、Party、标签、freeze_id | P0 |
| PRI-05 | 固定摊销 | 私有化模型按月/年固定摊销（amortization_fixed），不按 Token | P1 |

## 9.3 主体、账本与资金

| 编号 | 名称 | 描述 | 优先级 |
|------|------|------|--------|
| FUN-01 | 主体管理 | org/project 同一层；Leader；项目可不挂靠 | P0 |
| FUN-02 | 关系管理 | 7 种边（parent/sponsors/owns/participates/allocates/merged_into/split_from） | P0 |
| FUN-03 | 账本 | balance+frozen；并发安全（F-CON-01/03/04） | P0 |
| FUN-04 | 划拨通道 | 见 8.2 | P0 |
| FUN-05 | 划拨执行 | 授权+通道+守恒+流水+幂等（F-CON-02/05） | P0 |
| FUN-06 | 预扣与结算 | 见 8.1 | P0 |
| FUN-07 | 告警比例与预算上限 | 双层（Account + ModelGrant）；告警不阻断；分码 | P0 |
| FUN-08 | 清算 | 见 8.4；幂等 | P0 |
| FUN-09 | 组织变更 | 见 8.5；幂等 | P0 |
| FUN-10 | 流水 | 只追加；含幂等键等关联字段（F-CON-01） | P0 |
| FUN-11 | 冻结超时与续期 | 见 8.3 | P0 |

## 9.4 Key 与成员

| 编号 | 名称 | 描述 | 优先级 |
|------|------|------|--------|
| KEY-01 | Key 生命周期 | 创建/轮换/吊销；存哈希 | P0 |
| KEY-02 | 绑定账户 | 必须且唯一 account_id（含 Person Account） | P0 |
| KEY-03 | 归属人 | 必须关联实体人 | P0 |
| KEY-04 | 绑户约束 | 目标账户 ∈ iam 允许集 | P0 |
| KEY-05 | 成员管理 | 加入/移出 Party | P0 |
| KEY-06 | 禁用联动 | 禁人后 Key 立即失效 | P0 |

## 9.5 安全治理体系

| 编号 | 名称 | 描述 | 优先级 |
|------|------|------|--------|
| SEC-GOV-01 | ABAC 策略引擎 | sys_action_catalogs + sys_roles + sys_role_permissions + sys_subject_role_bindings + sys_access_policies + sys_access_policy_bindings；DENY 优先；最小权限默认（A-CON-02） | P0 |
| SEC-GOV-02 | 四轴正交授权 | data/fund/iam/routing 独立判定（A-CON-01） | P0 |
| SEC-GOV-03 | 职责分离 | 路由-资金互斥；审计只读；Leader 不万能（A-CON-03/05） | P0 |
| SEC-GOV-04 | ModelGrant | ALLOW/DENY + quota_limit 模型级预算；deny 优先（A-CON-04） | P0 |
| SEC-GOV-05 | UI 权限治理 | sys_ui_menus + sys_ui_routes + sys_ui_action_bindings；菜单可见性 + 路由守卫 + 按钮显隐 | P0 |
| SEC-GOV-06 | 数据不越权 | 全接口 Scope 过滤；防 IDOR（D-CON-01） | P0 |
| SEC-GOV-07 | 审计不可篡改 | before/after 快照；不可 UPDATE/DELETE；≥180 天（AU-CON-01/02/03） | P0 |
| SEC-GOV-08 | 密钥安全 | 加密存储；无明文日志与回显（D-CON-03） | P0 |

## 9.6 调用安全

| 编号 | 名称 | 优先级 |
|------|------|--------|
| SEC-01 | 网络策略（INTERNAL_ONLY 零外网流量，D-CON-02） | P2（可升 P0） |
| SEC-02 | 出网范围（白名单） | P2 |
| SEC-03 | 内容安全（阻断/脱敏/强制内网） | P2 |
| SEC-04 | 异常流量（拦截告警） | P2 |
| SEC-05 | 扩展点（主路径钩子；阶段 B 就绪） | 架构 P0 |

## 9.7 路由与调度

| 编号 | 名称 | 优先级 |
|------|------|--------|
| RTE-01 | 策略引擎（注册/启停/混合组合） | P1 |
| RTE-02 | 策略矩阵（12 种，含 S-CLASSIFY） | P1 |
| RTE-03 | 高可用（重试切换；三态熔断；流式限制危险切换） | P1 |
| RTE-04 | 账户正交（不得改 account_id，S-CON-01） | P0 |
| RTE-05 | 价格约束（δ 默认 0、硬上限 20%、关键审计，S-CON-02/03） | P0 |
| RTE-06 | 决策可观测（候选与选择关联 request_id） | P1 |

## 9.8 审计与对账

| 编号 | 名称 | 优先级 | 说明 |
|------|------|--------|------|
| AUD-01 | 调用审计（全字段：request_id、人、Key、账户、Party、模型、渠道、用量分项、双轨金额、路由结果、安全结果、耗时） | P0 | AU-CON-01 |
| AUD-02 | 配置变更审计（before/after 快照；δ 与预算帽关键；不可篡改） | P0 | AU-CON-02/03 |
| AUD-03 | 对账（上游 vs cost；差异分类；P0 阶段预留接口契约：reconciliation_runs 最小字段 run_id/period_start/period_end/provider/status + 上游账单拉取标准接口签名） | P2（P0 预留契约） | — |
| AUD-04 | 报表（多维汇总；沿 parent/sponsors 边向上聚合；data 轴范围约束生效） | P0/P2 | — |

## 9.9 控制台与治理 API

| 编号 | 名称 | 描述 | 优先级 |
|------|------|------|--------|
| UI-01 | 角色化导航 | 按 ABAC 评估结果显示菜单（SEC-GOV-05） | P0 |
| UI-02 | 主体与关系 | Party/Leader/边/成员；组织变更操作 | P0 |
| UI-03 | 资金操作 | 划拨/流水/清算/预算帽；危险操作二次确认 | P0 |
| UI-04 | 价目维护 | 双轨编辑 + 缓存折扣 + 固定摊销配置 | P0 |
| UI-05 | Key 与成员 | 申请绑户吊销 | P0 |
| UI-06 | 路由档案 | 12 种策略与 δ 配置（受 20% 硬上限约束） | P1 |
| UI-07 | 仪表盘与报表 | 消耗/余额/预算/拦截 + 沿关系边向上聚合 | P1–P2 |
| UI-08 | 密钥仓库 | 无明文二次回显（D-CON-03） | P0 |
| UI-09 | 模型权限 | 目录与 ModelGrant（含 quota_limit） | P0 |
| UI-10 | 安全事件报表 | 外网调用统计/内容拦截排行/违规调度记录/ABAC 拒绝统计/审计异常告警；支持导出 | P2 |
| UI-11 | 全链路调用追踪 | 按 request_id / task_id / user_id 查询；可视化流转链路（决策路由、安全结果、Token 计量、耗时）；关联 route_attempt_logs | P1 |
| UI-12 | ABAC 策略管理 | sys_access_policies 编辑/测试/模拟评估；sys_roles 管理；sys_subject_role_bindings 管理 | P0 |
| UI-13 | UI 权限管理 | sys_ui_menus 编辑；sys_ui_action_bindings 配置 | P0 |
| UI-14 | 审计日志查询 | 按操作者/操作类型/时间/资源检索；before/after 快照对比视图；不可删除确认 | P0 |
| API-01 | 治理 API | 与控制台能力完全对等（主体/授权/划拨/清算/Key/价目/路由/密钥/ABAC 策略/UI 权限）；同一 ABAC 鉴权模型；资金写强制幂等键 | P0 |

---

# 第 10 章 融合 DDL 数据模型

## 10.1 核心表全景

### 第1组：用户与身份（2 表）

| 表 | 来源 | 关键字段 |
|----|------|---------|
| `users` | TokenHub + AxonHub 融合 | username, email, password_hash, role, status, oidc_issuer, oidc_subject |
| `admin_sessions` | TokenHub | token, user_id, expires_at |

### 第2组：Party 统一主体（3 表）

| 表 | 来源 | 关键字段 |
|----|------|---------|
| `parties` | 新建（统一 org/project） | type(org/project), name, parent_party_id, leader_user_id, status |
| `party_edges` | 新建（7 种关系边） | src_party_id, dst_party_id, edge_type(parent/sponsors/owns/participates/allocates/merged_into/split_from), allows_fund |
| `party_members` | TokenHub + AxonHub 融合 | party_id, user_id, role(leader/member/observer) |

### 第3组：资金治理（5 表）

| 表 | 来源 | 关键字段 |
|----|------|---------|
| `accounts` | 新建 | party_id, available_balance, frozen_balance, status, budget_limit_amount, budget_warn_ratio, budget_period, budget_consumed_amount, liquidation_stage, version |
| `ledgers` | 新建（只追加） | account_id, direction(debit/credit/freeze/unfreeze/settle), amount, balance_after, cost_amount, sell_amount, freeze_id, request_id, idempotency_key |
| `freezes` | 新建（含续期） | account_id, request_id, amount, estimated_sell, status, expires_at, max_lifetime_at, renewal_count, settle_amount, settle_cost |
| `allocations` | 新建 | src_account_id, dst_account_id, amount, channel, edge_id, idempotency_key, status |
| `liquidations` | 新建 | party_id, account_id, target_account_id, status(blocking/draining/refunding/closing/closed), liquidation_type(project_close/org_merge/org_split) |

### 第4组：API Key（1 表）

| 表 | 来源 | 关键字段 |
|----|------|---------|
| `api_keys` | TokenHub 扩展 + AxonHub | key_hash, key_prefix, owner_user_id, account_id, party_id, status, limit_daily_tokens, limit_monthly_cost_usd |

### 第5组：模型目录（4 表）

| 表 | 来源 | 关键字段 |
|----|------|---------|
| `providers` | TokenHub + AxonHub | name, type, base_url, credentials(加密), status, healthy |
| `provider_resources` | TokenHub | provider_id, resource_group, api_key(加密), rate_limit_rpm, failure_count, cooldown_until |
| `models` | TokenHub 扩展 + 双轨 | name, category, input_price_per_1m(cost), sell_input_price_per_1m, item_codes(JSON), data_classification, network_class |
| `provider_models` | TokenHub | provider_id, upstream_model, canonical_name |

### 第6组：定价与路由（3 表）

| 表 | 来源 | 关键字段 |
|----|------|---------|
| `model_prices` | 新建（AxonHub 吸收+双轨） | model_id, channel_id, reference_id, price_json(items[{itemCode, cost:{mode,rate}, sell:{mode,rate,cache_discount_ratio}}], schedule) |
| `model_routes` | TokenHub 扩展 | model_name, provider_resource_id, priority, weight, route_profile_id, price_cap_delta |
| `route_profiles` | 新建（策略矩阵） | name, strategies_json(12种), delta_cap, max_attempts |

### 第7组：安全治理（9 表）

| 表 | 来源 | 关键字段 |
|----|------|---------|
| `sys_action_catalogs` | 参考 ai-gov.sql | action_code, action_name, axis(data/fund/iam/routing), resource_type |
| `sys_roles` | 参考 ai-gov.sql | role_code, role_name, description, is_system(系统角色不可删除) |
| `sys_role_permissions` | 参考 ai-gov.sql | role_id, action_id |
| `sys_subject_role_bindings` | 参考 ai-gov.sql | subject_type(user/party), subject_id, role_id, scope_party_id(NULL=全局), valid_from, valid_until |
| `sys_access_policies` | 参考 ai-gov.sql | policy_code, policy_name, effect(allow/deny), conditions_json, priority, is_system |
| `sys_access_policy_bindings` | 参考 ai-gov.sql | policy_id, subject_type, subject_id |
| `sys_ui_menus` | 参考 ai-gov.sql | menu_code, parent_id, label, icon, sort_order |
| `sys_ui_routes` | 参考 ai-gov.sql | route_path, menu_id, required_action_id |
| `sys_ui_action_bindings` | 参考 ai-gov.sql | button_code, button_label, page_route, required_action_id |

### 第8组：授权治理（2 表）

| 表 | 来源 | 关键字段 |
|----|------|---------|
| `grants` | 新建（轻量直接授权，ABAC 补充） | principal_type, principal_id, axis, action, resource_type, resource_id, effect |
| `model_grants` | 新建 | principal_type, principal_id, model_id, model_tag, effect(allow/deny), priority, quota_limit |

> **grants vs ABAC：** `grants` 表用于直接授权（"张三对 AI 研发部账本有 balance.read 权限"），ABAC 引擎用于策略化授权（"所有部门 Leader 对本部门账本有 balance.read 权限"）。两者共存，ABAC 评估优先级高于 grants。

### 第9组：请求与用量（5 表）

| 表 | 来源 | 关键字段 |
|----|------|---------|
| `request_logs` | TokenHub 扩展 + AxonHub 双轨 | request_id, api_key_id, account_id, user_id, party_id, model_name, cost_usd, sell_usd, cost_items(JSON), usage_incomplete, error_code |
| `request_payload_logs` | TokenHub | request_id, request_body(脱敏), response_body(脱敏) |
| `route_attempt_logs` | TokenHub | request_id, attempt_index, route_id, provider_resource_id, invoked, latency_ms |
| `usage_records` | TokenHub 扩展 + AxonHub itemCode | request_id, input_tokens, prompt_cached_tokens, prompt_audio_tokens, image_count, video_duration_seconds, cost_usd, sell_usd, cost_items(JSON) |
| `quota_buckets` | TokenHub | key_id, scope(daily/monthly), bucket, requests, prompt_tokens, cost_usd |

### 第10组：可观测（2 表）

| 表 | 来源 | 关键字段 |
|----|------|---------|
| `channel_probes` | AxonHub | channel_id, total_request_count, success_request_count, health_status, consecutive_failures |
| `provider_quota_status` | AxonHub | channel_id, status, quota_data(JSON), next_reset_at |

### 第11组：基础设施（3 表）

| 表 | 来源 | 关键字段 |
|----|------|---------|
| `audit_events` | TokenHub 扩展 | actor_user_id, action, resource_type, resource_id, status, before_snapshot(JSON), after_snapshot(JSON), ip, user_agent |
| `audit_chain_anchors` | 参考 ai-gov.sql | anchor_hash, start_event_id, end_event_id, event_count, created_at |
| `idempotency_records` | 新建 | scope, actor_id, idempotency_key, request_hash, status, response_json, expires_at |

## 10.2 总计：40 表（vs 原 69 表 -42%）

新增 11 表均为安全治理必需（ABAC 引擎 6 表 + UI 权限 3 表 + 审计链锚定 1 表 + 幂等 1 表）。剪裁 40 表为过度工程化（CQRS 投影、复式账本 14 entry_type、事件总线、紧急信用、对账 5 表后置、复杂身份 9 表合并）。

## 10.3 存储纪律

- 价目、策略组合、分时进 JSON
- 热字段原子；流水只追加（应用层禁止 UPDATE/DELETE）
- 审计不可篡改（应用层禁止 UPDATE/DELETE audit_events）
- 禁止为每个扩展点无限加表加列

---

# 第 11 章 系统架构与开发指导

## 11.1 逻辑架构

```text
应用/Agent → 兼容 API（6 协议）
  → Key 鉴权 → 安全钩子
  → ModelGrant 模型过滤(DENY 剔除)
  → 锚定内部价 → 价格合格集过滤(δ)
  → ModelGrant 模型级预算(quota_limit) → Account 级预算帽
  → 冻结
  → 策略矩阵选路(12 种) → 上游调用
  → 流式续期 → 用量规范化 → 双轨结算
  → 审计

控制面：ABAC 引擎 统一鉴权
  ├── 身份与角色：sys_roles + sys_subject_role_bindings
  ├── 策略评估：sys_access_policies + sys_access_policy_bindings
  ├── UI 投影：sys_ui_menus + sys_ui_routes + sys_ui_action_bindings
  ├── 资金：Party / Account / Ledger / Freeze / Allocation / Liquidation
  ├── 模型：Model / Provider / ModelPrice / ModelGrant
  └── 审计：audit_events + audit_chain_anchors

治理 API：与控制台相同 ABAC 鉴权；资金写强制幂等
存储：SQLite（MVP）/ PostgreSQL（生产）
```

## 11.2 包划分

| 包 | 性质 | 职责 |
|----|------|------|
| `fund` | **新建** | accounts/ledgers/freezes/allocations/liquidations 资金闭环 |
| `pricing` | **新建** | model_prices 双轨 JSON + 5 种计价模式 + 缓存折扣 + 固定摊销 |
| `idempotency` | **新建** | Idempotency-Key 原子抢占 |
| `party` | **扩展** | parties + party_edges + party_members（7 种边，扩展 TokenHub projects） |
| `abac` | **新建** | ABAC 策略引擎（sys_action_catalogs + sys_roles + sys_access_policies + 策略评估） |
| `authz` | **扩展** | grants 四轴直接授权（ABAC 补充）+ 鉴权中间件 |
| `ui_permission` | **新建** | sys_ui_menus + sys_ui_routes + sys_ui_action_bindings + 权限快照生成 |
| `routing` | **扩展** | Strategy 接口抽象 + route_profiles（12 种策略） |
| `modelgrant` | **新建** | ModelGrant ALLOW/DENY + quota_limit 模型级预算 |
| `security` | **扩展** | 安全钩子执行链路（SEC-05）+ 内容安全 + 出网管控 |
| `audit` | **扩展** | audit_events + audit_chain_anchors（不可篡改） |

## 11.3 存量代码重构策略

TokenHub 当前架构存在两个巨石文件：`store.go`（195KB）和 `http.go`（295KB）。重构原则：

1. **新建包从零写**（fund/pricing/idempotency/abac/ui_permission/modelgrant）：不修改 TokenHub 现有代码，通过接口注入
2. **扩展包增量提取**（party/authz/routing/audit）：每次提取一个子功能 + 独立测试门禁
3. **存量不动**：TokenHub 原有 `store.go` 和 `http.go` 中未被提取的代码保持原样，提取后原代码改为调用新包
4. **门禁**：每次提取前后跑 TokenHub 现有测试套件 + 新增包的单元测试

## 11.4 技术栈

| 层 | 选择 |
|----|------|
| 后端语言 | Go 1.26 |
| ORM | GORM |
| 数据库 | SQLite（MVP）/ PostgreSQL 16（生产） |
| 缓存 | Redis 可选 |
| 前端 | Next.js 16 + TypeScript |
| 部署 | Docker Compose + systemd（MVP）/ K8s Helm（生产） |
| 国产适配 | 国产 CPU/OS 阶段 A 冒烟 |

## 11.5 策略引擎接口

```go
type Strategy interface {
    Filter(ctx context.Context, candidates []RouteCandidate) []RouteCandidate
    Score(ctx context.Context, candidates []RouteCandidate) []RouteCandidate
}

type RouteProfile struct {
    Strategies []StrategyConfig
    DeltaCap   float64
}
```

Pipeline: `candidates → S-COMPLIANCE → ModelGrant → price cap → S-CLASSIFY → remaining strategies → pick`

## 11.6 多方验收门禁

| 顺序 | 门禁 | 责任方 | 出口 |
|------|------|--------|------|
| 1 | Dev Complete | 研发 | 单测通过；迁移可反复执行；OpenAPI 一致 |
| 2 | QA | 测试 | 第 13 章全部 P0 用例通过 |
| 3 | UED | 设计 | 危险确认、错误文案、角色导航走查通过 |
| 4 | 产品 UAT | 产品 | 财务演示脚本 + ABAC 权限场景 + UI 权限场景 + 用户旅程签字 |
| 5 | 安全 | 安全 | 越权测试、IDOR 测试、密钥抽检、审计篡改测试、ABAC 策略评估测试通过 |
| 6 | 发布 | 架构/运维 | 回滚演练、监控、备份、NOTICE 就绪 |

## 11.7 分期 WBS

| 阶段 | 内容 | 工期 |
|------|------|------|
| A | Fork TokenHub、执行 DDL（40 表含 ABAC+UI 治理）、用量规范化、国产冒烟 | 2d |
| B | Party 账本/划拨/预算帽/冻结续期/清算/组织变更/双轨 model_prices/价格帽/ABAC 引擎/grants/ModelGrant/UI 权限/审计链/治理 API 幂等/安全钩子空实现 | 4d |
| C | 策略矩阵全量（12 种含 S-CLASSIFY）/决策日志/全链路调用追踪 UI/仪表盘+聚合 | 2d |
| D | 内容安全出网/变更快照强化/安全事件报表/对账接口契约落地 | 2d |
| E | 压测 HA/GA/文档与许可 | 1d |

**总工期：约 11 工作日。**

---

# 第 12 章 非功能需求

| 类别 | 要求 |
|------|------|
| 可用性 | 目标 99.9%；多实例故障切换 |
| 性能 | 单节点目标 5000 QPS；附加延迟 <50ms |
| 安全 | TLS 1.3；密钥 AES-256 加密存储；ABAC 最小权限；全链路审计不可篡改；UI 权限投影 |
| 部署 | 私有化；Docker Compose / K8s；离线/内网；国产环境阶段 A 验证 |
| 审计保留 | ≥180 天；冷热分离；哈希链锚定防篡改 |
| 可扩展 | 适配器、itemCode、策略、边类型、安全钩子、ABAC 策略 |
| 可观测 | 冻结任务、幂等冲突、预算帽命中、ABAC 拒绝统计、路由指标（Prometheus + Grafana） |

---

# 第 13 章 验收标准

## 13.1 功能验收

| 场景 | 通过条件 |
|------|----------|
| 统一接入 | ≥5 类公有 + 1 类私有化兼容 |
| 双轨与 item（含多模态） | cost/sell 及文本+图片+音频+视频分项正确 |
| 固定摊销 | 私有化模型按月摊销，不按 Token 计费 |
| 缓存折扣 | prompt_cached_tokens 的 sell = cost × (1+markup) × cache_discount_ratio |
| usage 不完整 | 有标记、不伪造 |
| 独立项目 / 组织池 / 出资划拨 | 守恒与通道正确 |
| 个人经费 | Party allocates → Person Account → 个人 Key 扣费 |
| 组织合并/拆分 | 按 8.5 流程余额正确转移 |
| 价格约束与 δ | 默认 0、硬上限 20%、变更有关键审计 |
| 双层预算帽 | Account 级 90% 帽 → BUDGET_CAP_EXCEEDED；ModelGrant 级 quota_limit → MODEL_BUDGET_EXCEEDED；余额不够 → INSUFFICIENT_BALANCE |
| 告警比例 | 80% 只告警不阻断 |
| 冻结超时 / 流式续期 | 符合 8.3 |
| 清算状态机 | 符合 8.4 |
| 幂等 | 重复写不双记 |
| ModelGrant deny 优先 | deny 后不可调该模型 |
| ABAC 四轴越权 | data 角色不能看未授权 Party 日志；fund 角色不能改路由；Leader 无 Grant 不可全平台操作 |
| ABAC 策略评估 | DENY 策略立即拒绝；无匹配策略默认拒绝；角色权限正常匹配 |
| UI 权限投影 | 无 fund.allocate 权限的用户看不到划拨菜单和按钮；直接访问 /console/fund/allocate 路由被守卫拦截 |
| 审计不可篡改 | audit_events 表 UPDATE/DELETE 被应用层拒绝；before/after 快照完整 |
| 调度不改账户 | 任意路由下 account 不变 |
| 策略矩阵启停组合 | 12 种策略可单独启停组合 |
| 禁人即禁 Key | 立即 |
| 治理 API | ABAC 对等鉴权与幂等 |
| INTERNAL_ONLY（启用时） | 无外网流量（D-CON-02） |
| 密钥安全 | 上游 Key 明文不落日志、不二次回显（D-CON-03） |
| 全链路调用追踪 | 按 request_id 查询可见完整流转链路 |

## 13.2 非功能验收

1. 在约定硬件下完成压力测试：单节点 ≥5000 QPS，附加延迟 <50ms
2. 安全扫描与密钥存储抽检通过
3. 第三方安全渗透测试通过：IDOR、权限提升、ABAC 绕过
4. 审计保留策略验证通过：≥180 天 + 不可篡改
5. 国产环境冒烟通过
6. 冻结超时任务与幂等窗口行为验证通过

## 13.3 财务演示脚本（必须可重复）

配置预算与加价 → 划拨 → 创建人 Key 绑项目账本 → 调用 → 核对 sell/cost/流水 → 演示 Account 级预算帽分码 → 演示 ModelGrant 级预算帽分码 → 演示余额不足分码 → 演示个人经费注入与消费 → 清算回流 → 组织合并余额转移 → 重复提交划拨仅入账一次。

## 13.4 安全治理演示脚本（必须可重复）

1. ABAC 策略评估：创建"AI研发部 Leader"角色 → 绑定 fund.allocate + data.usage.read（scope=AI研发部）→ 以 Leader 身份尝试读取"市场部"日志 → 拒绝（数据不越权）→ 尝试修改路由策略 → 拒绝（无 routing 轴权限）
2. UI 权限投影：以普通员工登录 → 控制台不显示"资金管理"菜单 → 直接访问 /console/fund/allocate → 路由守卫拒绝
3. 审计不可篡改：管理员修改 δ → audit_events 记录 before/after 快照 → 尝试通过 API 删除该审计记录 → 拒绝
4. ModelGrant deny 测试：为研发部配置 GPT-4 ALLOW + GPT-4.5-preview DENY → 尝试调用 GPT-4.5-preview → MODEL_ACCESS_DENIED

---

# 第 14 章 不在范围与预留扩展

**当前不在范围：**

- Agent 编排 IDE、提示词资产中心、C 端聊天产品
- 同一请求多账本分摊扣费
- 默认透支信用
- 客户端负责冻结续期
- 无权限审计的上游密钥写入
- 区块链热账本
- 应急储备池（多部门借用）

**预留扩展（架构级设计，P0 阶段预留接口与表结构）：**

| 扩展项 | 预留内容 |
|--------|---------|
| 对外 Token 售卖 | `tenants` 表（多租户隔离键）；外部定价层级 JSON 结构（retail/wholesale/enterprise）；用量同步 gRPC 接口签名 |
| 对账完整流程 | `reconciliation_runs` 最小字段（run_id, period_start, period_end, provider, status）；上游账单拉取标准接口签名 |
| 多币种 | `accounts.currency` 字段预留 |
| 复杂审批流 | `approval_requests` + `approval_decisions` 表预留 |
| HR/ERP 深度同步 | Party/Person 外部 ID 映射字段预留 |

---

# 第 15 章 术语表

| 术语 | 定义 |
|------|------|
| Token | 计费用量及缓存、推理、图片、音频、视频等衍生类型 |
| 网关 Key | 企业发给调用方的凭证，≠ 上游厂商密钥 |
| Party | 组织或项目等可持账本与成员的主体（org/project 同层多态） |
| Party Edge | 关系边（parent/sponsors/owns/participates/allocates/merged_into/split_from） |
| 双轨 | cost（上游成本）与 sell（内部结算价）并行计量 |
| 缓存折扣 | 对缓存命中的 Token 的 sell 应用 cache_discount_ratio（如 0.5） |
| 固定摊销 | 私有化模型按月/年固定成本摊销（amortization_fixed），不按 Token |
| δ | 候选相对锚定价允许上浮比例；默认 0；硬上限 20% |
| 双层预算 | Account 级（这个账户还有多少钱）+ ModelGrant 级（这个主体在这个模型上最多花多少），取交集 |
| ABAC | 基于属性的访问控制（Attribute-Based Access Control），本产品的统一授权引擎 |
| 四轴正交 | data/fund/iam/routing 权限独立判定，禁止一轴推导另一轴 |
| ModelGrant | 模型访问 ALLOW/DENY 规则；deny 优先；含 quota_limit 模型级预算 |
| UI 权限投影 | ABAC 权限在展示层的映射——菜单可见性、路由守卫、按钮显隐 |
| 审计不可篡改 | audit_events 应用层禁止 UPDATE/DELETE；定期哈希链锚定 |
| 策略矩阵 | 12 种可启停、可混合的路由策略（含 S-CLASSIFY 智能分类） |
| 治理 API | 与管理台对等的控制面 API，同一 ABAC 鉴权模型 |
| 幂等键 | 防写操作重试重复记账 |
| itemCode | 与上游账单对齐的费用项编码（含文本/图片/音频/视频） |
| TokenHub 底座 | 主仓库二次开发基线（Apache 2.0） |
| AxonHub 吸收 | 计价明细与多维评分思想来源（llm/ LGPL-3.0，动态链接或 sidecar） |

---

# 第 16 章 总结

本方案以 **全领域守恒定理**为宪法约束，在 TokenHub（Apache 2.0）主仓库底座上，吸收 AxonHub 的 itemCode 计价引擎与多维评分思想，参考 ai-gov.sql 的安全治理架构（ABAC 引擎 + UI 权限 + 审计链），构建了一个完整的企业级 AI 智能网关治理平台。

**核心特质：**

| 维度 | 特质 |
|------|------|
| **财务** | 双轨 cost/sell + 5 种计价模式 + 缓存折扣 + 固定摊销 + 双层预算帽 + 守恒账本 + 清算 + 组织变更 + 个人经费 |
| **安全** | ABAC 策略引擎 + 四轴正交 + ModelGrant + UI 权限投影 + 审计不可篡改 + 数据不越权 + 密钥不透传 + INTERNAL_ONLY 强制隔离 |
| **调度** | 12 种可插拔策略（含 S-CLASSIFY 智能分类）+ δ 价格帽 ≤20% + 账户锁定 + 三态熔断 |
| **工程** | TokenHub 底座（Apache 2.0）+ AxonHub 动态链接/sidecar + 40 表轻量 DDL + 存量代码渐进式重构策略 |
| **交付** | 11 工作日 WBS + 6 门禁 + 11 项交付物 + 4 套演示脚本（财务 + 安全治理 + ABAC + ModelGrant） |

**这不是一个 API 通道。这是一台 AI 财务精算机，一座国企级安全治理堡垒。每一条守恒定理都是不可配置的宪法，每一张 ABAC 策略表都是合规审计的护城河，每一个 UI 按钮的可见性都由权限引擎精确控制。**

---

**文档结束（定版 3.1.0 完整闭环）。**
