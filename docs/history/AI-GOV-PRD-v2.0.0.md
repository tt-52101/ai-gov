# 企业级 AI 智能网关治理平台（Token 治理底座）  
## 完整需求与设计定版方案（禁止细节丢失）

| 项 | 内容 |
|----|------|
| 文档版本 | 定版 2.0（全量合并历史既定结论，禁止再压缩） |
| 文档状态 | 完整可评审、可详细设计、可开发、可测试、可验收、可商用交付 |
| 日期 | 2026-07-31 |
| 读者 | 产品、研发、测试、UE、财务、安全、架构、交付、运维 |

**编制原则：** 本文为历史全部既定事实与详细设计的合并定稿。凡曾达成共识的边界、矩阵、错误码、数据模型、流程、验收与工程约束，均在正文保留可执行粒度，**不得以“摘要”“见上”“从略”替代。**

---

# 第 0 章 文档说明与设计哲学

## 0.1 文档性质

本文档是独立、完整、详细的产品需求规格与关键详细设计合并文本。阅读与执行无需依赖其它会话记录。所有核心需求、物理世界模型、开源融合策略、功能矩阵、资金与调度边界、预算帽数据模型、错误码全表、模型权限与正交授权、冻结/清算/幂等、数据纪律、非功能、验收、分期与二次开发指导均在正文写明。

## 0.2 设计哲学

Token 是 AI 时代的电力。本产品不运营模型供给、不对外倒卖 Token，只在企业内部提供：

- 统一模型调用入口  
- 精确计量与双轨计价  
- 主体经费账本、划拨与清算  
- 安全控制与不可抵赖审计  

**目标闭环：**

> 模型接入 → 安全拦截 → 智能调度 → 实时扣费 → 资金回笼与清算 → 对账审计。

## 0.3 核心公理（不可配置取消）

1. **无计量则无管理。**  
2. **无归因则无法优化。**  
3. **无企业自有调用入口则无法真正掌控 AI 资产与成本。**  
4. **无守恒账本则无财务治理护城河。**  
5. **调度不得改变扣费账户；内部结算价格受请求锚定与上浮上限约束；禁止先调用后欠费。**  
6. **数据范围、资金、身份密钥、调度与模型配置四轴正交授权；禁止一轴推导另一轴特权；模型可用范围与资金扣费主体必须解耦。**  
7. **平台管理员也必须经账本服务与流水变更余额，不得提供绕过守恒的后门接口。**

## 0.4 能力优先级（核心竞争力顺序，历史明确校正）

| 优先级 | 能力域 | 说明 |
|--------|--------|------|
| **P0** | 统一接入与密钥托管 | 消灭野连接 |
| **P0** | **财务治理与计量闭环** | 双轨、账本、划拨清算、Key 扣费、预算帽、守恒流水——**第一护城河** |
| **P0** | 正交授权、数据不越权、模型访问治理 | 四轴 + ModelGrant |
| **P0** | 调度与资金强制边界 | 价格帽、账户锁定、禁止先调后欠费 |
| **P0** | 治理 API 与资金写幂等 | 与控制台对等 |
| **P1** | **可插拔路由策略矩阵与降本高可用** | 单策略启停、混合组合，非固化四层 |
| **P1** | 基础运营报表与仪表盘 | |
| **P2** | 内容安全与出网强化、配置变更快照、上游自动对账 | 具体引擎可后置 |
| 架构级 P0 | 数据面安全扩展点 | 阶段 B 预留钩子，避免后期全链路改造 |

合同强制合规时，项目可将 SEC 能力提升为项目级 P0。

---

# 第 1 章 产品综述

## 1.1 要解决的问题

1. 员工与团队私自采购上游 API Key，调用黑箱，无法统一禁用与审计。  
2. 财务只能拿到厂商总账单，无法按组织、项目、人员、应用做内部结算与利润分析。  
3. 成本失控：无预算预扣、无实时阻断、无双轨可见性；预算帽与「真没钱」若混用同一错误码则运营抓手不足。  
4. 合规风险：敏感数据可能经未审批路径出境；配置变更缺少不可篡改留痕。  
5. 可用性与成本：仅写死「主→备→降级→缓存」四层不足以支撑商业灵活组合；调度不受内部价格约束会击穿账本。

## 1.2 价值主张

| 主张 | 说明 |
|------|------|
| 统一治理 | 全部调用经单一企业网关 |
| 财务闭环 | 预算、预扣实扣、划拨清算、双轨、预算帽分码、对账 |
| 安全合规 | 密钥托管、出网、内容安全、全链路与配置审计 |
| 降本增效 | 计量可信后，策略矩阵做成本/健康/权重/亲和等调度 |

## 1.3 产品边界

**负责：** 身份与正交授权、模型访问治理、主体与账本、密钥托管、兼容 API、安全扩展点、可插拔路由、双轨计价、预扣结算、预算帽、审计对账、管理台与治理 API、可观测。

**不负责：** Agent 编排、Skill 市场、C 端对话产品、对外售卖 Token、同一请求多账本分摊扣费、默认透支、客户端负责冻结续期、以公链/联盟链作为热路径余额系统记录。

**契约：** OpenAI 兼容 HTTP API（及必要其它协议兼容）；业务标签仅归因，不替代鉴权与扣费账户。

## 1.4 目标用户

平台管理员/架构师、财务/成本、安全合规、组织或项目 Leader、实体人（持 Key）。

---

# 第 2 章 物理世界主体模型（Party 多态）

## 2.1 设计出发点

不得假设唯一「公司→部门→项目→人」且项目必须挂部门。

## 2.2 典型场景（必须全部可落地）

| 场景 | 组织 | 项目 | 关系与资金 |
|------|------|------|------------|
| 部门日常额度 | 有 | 可不建 | 部门账本 + 人 Key |
| 部门主责专项 | 有 | 有 | owns 可选；项目可自有账本 |
| 公司级战略项目 | 多事业部 | 可与部门平级 | 多 sponsors；独立 Leader |
| 跨主体协作 | 内外部 | 有 | 人可只挂项目 |
| 矩阵制 | 多线 | 有 | 多边；一次调用只扣一个账本 |
| 结项还钱 | — | 结束 | 回流 + Key 收口 |

## 2.3 统一主体 Party

- **org 与 project 同一层语义**，均可：账本、成员、Leader、授权资源。  
- **项目不必然挂靠组织。**  
- UI 可分「组织/项目」入口，底层权限与账本引擎统一。

## 2.4 关系边

| 类型 | 含义 | 与资金 |
|------|------|--------|
| parent | 组织树 | 默认仅上级→下级划拨 |
| sponsors | 出资 | 默认仅出资方→被出资方 |
| owns | 主责 | **不**自动开通划拨 |
| participates | 协作 | **不**自动开通划拨 |

## 2.5 人、Key、账本、Leader

| 对象 | 规则 |
|------|------|
| Person | 成员进入一个或多个 Party |
| API Key | 必须绑定唯一 account_id；调用与扣费唯一入口；必须归属人 |
| Account | 归属 Party；balance + frozen |
| Leader | 默认责任角色；权限必须显式 Grant，禁止头衔万能 |

**资金流转强制规则：**

1. 划拨等额、同事务、双边。  
2. 预扣入冻结，必有 freeze_id 与过期时间；预扣前可经预算帽。  
3. 结算按实际 sell 多退少补。  
4. 清算走状态机。  
5. 无流水不改账；默认成功路径不出现负余额。  
6. 资金写操作幂等。

**消费强制规则：** sell 只扣 Key 绑定账户，归因到人；Tag 只做报表，不是第二套钱。

## 2.6 扩展原则

稳定核心语义；扩展走插件/JSON/边类型/钩子；热字段原子；禁止表与列无限膨胀。

---

# 第 3 章 开源底座选型与能力融合

## 3.1 功能矩阵对比

| 维度 | TokenHub | AxonHub |
|------|----------|---------|
| 定位 | 企业私有网关：入口、项目 Key、路由、配额、归因、RBAC、审计 | 多协议、强 Trace、细粒度成本、自适应负载 |
| 组织/项目 | 项目空间、成员、项目 Key | 偏通道与调用侧 |
| 路由 | 优先级、权重、回退、亲和、健康 | 多维评分、故障转移 |
| 计费 | 基础用量 | **itemCode、三模式、JSON 价目、costItems、缓存分项** |
| 私有化 | SQLite/PG 多实例明确 | 可私有化 |
| 本 PRD 缺口 | 缺双轨、守恒账本、划拨清算、四轴、价格帽、预算帽 | 缺企业内 Party 资金闭环 |

## 3.2 选型结论

- **主仓库底座：TokenHub**（最小化二次开发）。  
- **吸收 AxonHub：** 费用项与计算口径、价目 JSON、costItems、多维评分思想纳入策略实现。  
- 强制 **cost/sell 双轨**。

## 3.3 路由：可插拔策略矩阵（替代固化四层）

「主→备→降级→缓存」仅为档案预设，不是唯一实现。

| 策略代码 | 名称 | 作用 | 可禁用 |
|----------|------|------|--------|
| S-PRI | 优先级分组 | 主备硬分组 | 可 |
| S-HEALTH | 健康与熔断 | 失败率、冷却、半开 | 可 |
| S-WEIGHT | 权重与负载 | 权重与历史负载 | 可 |
| S-AFFINITY | 会话亲和 | 同会话同渠道 | 可 |
| S-COST | 成本感知 | 合格集内偏低价 | 可 |
| S-LATENCY | 延迟感知 | TTFT/端到端 | 可 |
| S-ERROR | 错误率感知 | 近期成功率 | 可 |
| S-RATE | 限流感知 | RPM/TPM/429 | 可 |
| S-TAG | 业务标签 | Tag 定向 | 可 |
| S-COMPLIANCE | 合规网络 | INTERNAL_ONLY 等 | **硬策略，不可对受限主体关闭** |
| S-CACHE | 缓存兜底 | 最后手段 | 可 |

**档案示例：**

- `simple-failover`：类四层（PRI+HEALTH+有限 WEIGHT）  
- `ha-cost`：PRI+HEALTH+WEIGHT+COST+AFFINITY  
- `compliance-strict`：S-COMPLIANCE 强制 + PRI+HEALTH  

引擎要求：策略可注册、可按档案启用/禁用、可加权评分或优先级链组合。跑策略前必须价格合格集过滤；**不得改 account_id**。

---

# 第 4 章 双轨计价（对齐 AxonHub 代码事实）

## 4.1 费用项 itemCode 基线

| itemCode | 含义 |
|----------|------|
| prompt_tokens | 输入（与缓存拆分） |
| completion_tokens | 输出 |
| prompt_cached_tokens | 缓存读 |
| prompt_write_cached_tokens | 缓存写 |
| completion_reasoning_tokens | 推理输出（若有） |
| 缓存 TTL 变体等 | 可扩展 |

## 4.2 计价模式

| 模式 | 说明 |
|------|------|
| flat_fee | 按次固定 |
| usage_per_unit | 通常每 1M Token |
| usage_tiered | 分段阶梯 |
| usage_volume（可选） | 总量落档后整单同价 |

## 4.3 双轨与价目结构

- **cost：** 上游成本（对账、毛利）  
- **sell：** 内部结算价（扣账本）  

价目：渠道×模型，JSON 示例结构：

```json
{
  "items": [
    {
      "itemCode": "prompt_tokens",
      "cost": { "mode": "usage_per_unit", "usagePerUnit": "..." },
      "sell": { "mode": "usage_per_unit", "usagePerUnit": "..." }
    }
  ],
  "schedule": {
    "timezone": "Asia/Shanghai",
    "overrides": []
  }
}
```

输入 Token 计算口径对齐 AxonHub：`PromptTokens - CachedTokens - WriteCachedTokens` 等，缓存单独计价。  
**禁止**「模型能力多维度档位」替代费用项。

## 4.4 用量规范化

适配层映射上游 usage → 内部 itemCode；缺失记 0 + `usage_incomplete`；禁止伪造明细。

---

# 第 5 章 预算帽配置数据模型（详细设计）

## 5.1 目标

支持低于余额 100% 的软预算上限；与「余额不足」分错误码。

## 5.2 挂载点

预算帽挂在 **Account**（一 Party 一账时即对该主体经费生效）。本定版不做 Key 级分帽。

## 5.3 字段设计（推荐 accounts 热字段，控表数量）

| 字段 | 类型 | 含义 |
|------|------|------|
| budget_limit_amount | DECIMAL NULL | 预算上限；NULL=未启用帽 |
| budget_warn_ratio | DECIMAL NULL | 告警比例如 0.80；只告警不阻断 |
| budget_period | VARCHAR NULL | none / calendar_month / calendar_day / custom |
| budget_period_start | TIMESTAMPTZ NULL | custom 起点 |
| budget_period_end | TIMESTAMPTZ NULL | custom 终点 |
| budget_consumed_amount | DECIMAL NOT NULL DEFAULT 0 | 本周期已确认消耗的 sell 累计 |
| budget_version | BIGINT/INT | 配置乐观锁 |

## 5.4 业务规则

1. `budget_limit_amount IS NULL`：不做帽校验。  
2. 上限可小于当前 balance（软帽）。  
3. 告警：`budget_consumed_amount / budget_limit_amount >= budget_warn_ratio` → 通知，**不拒绝调用**。  
4. 阻断：`budget_consumed_amount + 本次预估 sell > budget_limit_amount` → `BUDGET_CAP_EXCEEDED`。  
5. 周期任务重置 consumed（写审计事件）。  
6. 结算成功的 sell 计入 consumed；未消费解冻不计入。  
7. 改预算帽：fund 轴 `budget.write`；变更前后审计。

## 5.5 预扣判定顺序

```text
构建候选 → 价格合格集过滤
  → 若启用预算帽：帽校验（失败 BUDGET_CAP_EXCEEDED）
  → 可用余额 ≥ 冻结额（失败 INSUFFICIENT_BALANCE）
  → 写入 freeze
```

---

# 第 6 章 错误码定义全表

机器码全项目统一；HTTP 与 body.code 固定绑定。

## 6.1 资金与额度

| code | HTTP 建议 | 含义 | 达上游 |
|------|-----------|------|--------|
| BUDGET_CAP_EXCEEDED | 402 | 命中预算上限（余额可能>0） | 否 |
| INSUFFICIENT_BALANCE | 402 | 可用余额不足以冻结 | 否 |
| ACCOUNT_FROZEN_OR_CLOSED | 403 | 账户停用或清算中 | 否 |
| FREEZE_EXPIRED | 409 | 结算时冻结失效（多内部） | — |
| IDEMPOTENCY_CONFLICT | 409 | 处理中或同键异参 | 否 |
| IDEMPOTENCY_REPLAY | 200 | 同键重放 | — |

## 6.2 鉴权与身份

| code | HTTP 建议 | 含义 |
|------|-----------|------|
| AUTH_INVALID_KEY | 401 | Key 无效/吊销 |
| AUTH_USER_DISABLED | 403 | 人已禁用 |
| AUTH_KEY_NO_ACCOUNT | 403 | Key 未绑账户 |
| AUTHZ_DENIED | 403 | 控制面四轴拒绝 |
| MODEL_ACCESS_DENIED | 403 | ModelGrant 不允许该模型 |

## 6.3 路由与价格

| code | HTTP 建议 | 含义 | 达上游 |
|------|-----------|------|--------|
| NO_ROUTE_WITHIN_PRICE_CAP | 422/503 | 无价格合格候选 | 否 |
| NO_ROUTE_AVAILABLE | 503 | 无可用健康路由 | 否 |
| ROUTE_COMPLIANCE_BLOCKED | 403 | 合规剔除全部候选 | 否 |

## 6.4 安全

| code | HTTP 建议 | 含义 |
|------|-----------|------|
| COMPLIANCE_NETWORK_BLOCKED | 403 | 网络策略 |
| CONTENT_BLOCKED | 403 | 内容安全 |
| RATE_LIMITED | 429 | 限流 |

## 6.5 上游与系统

| code | HTTP 建议 | 含义 |
|------|-----------|------|
| UPSTREAM_ERROR | 502 | 上游错误 |
| UPSTREAM_TIMEOUT | 504 | 上游超时 |
| INTERNAL_ERROR | 500 | 网关内部错误 |

## 6.6 不作为调用错误码

告警比例命中：仅事件/Webhook，不返回调用错误码。

---

# 第 7 章 模型权限治理与正交授权（专章）

## 7.1 四轴分离

| 轴 | 回答 | 禁止推导 |
|----|------|----------|
| data | 能看哪些日志/报表/成员 | 不能推导划拨/改路由 |
| fund | 余额流水、划拨清算、预算帽 | 不能推导未授权 Party 全量日志 |
| iam | 人、Key、成员、禁人 | 不能绑无权账户；不能改价目 |
| routing | 价目、档案、策略、渠道、上游密钥、模型目录与 ModelGrant | 不能改 account_id；不能划拨 |

调用链：**模型是否允许 → 锁定账户 → 预算帽 → 冻结 → 调度（不改账户）**。

## 7.2 模型访问治理

| 层次 | 含义 |
|------|------|
| 目录层 | 逻辑模型对谁可见 |
| 调用层 | 谁允许发起对某模型的请求 |
| 合规层 | INTERNAL_ONLY 等硬限制 |
| 资金层 | 允许调用仍可能预算帽/余额失败 |

**逻辑模型 Model** → **渠道绑定** → **ModelGrant**。

### ModelGrant 字段

| 字段 | 含义 |
|------|------|
| principal_type | party / person / key / role |
| principal_id | ID |
| model_id 或 model_tag | 模型或标签组 |
| effect | allow / deny |
| priority | 冲突解析 |

**默认与冲突（定版必须固定并验收）：**

- deny 优先于 allow。  
- 级联顺序建议：Key > Person > Party > 全局默认（实现固定并文档化）。  
- 商用推荐显式 allow 列表；全局默认策略在详细设计选定「默认拒绝」或「默认发布目录可调」之一并测通。  
- **禁止**仅因 Leader 头衔自动拥有全平台模型权。

### 与 Party

- 项目可只授权国产模型集。  
- 默认模型范围随 Key 绑定账户所属 Party；跨 Party 须额外鉴权。

## 7.3 Grant 最小动作矩阵

**fund：** balance.read、ledger.read、allocate、liquidate、budget.write  

**data：** usage.read、report.read、member.read  

**iam：** key.create/revoke/rotate（绑户∈允许集）、user.disable、member.add/remove  

**routing：** price.write、route_profile.write（含 δ）、channel.write、upstream_secret.write、model_catalog.write、model_grant.write  

## 7.4 数据不越权

列表/详情/导出全部 Scope 过滤；防 IDOR；fund 与 data 分轴可只授其一。

## 7.5 调度正交

路由不得写 account_id；S-COST 仅在模型授权且价格合格集内排序；routing 管理员无 fund 不能划拨。

## 7.6 验收红线清单

1. 无流水改余额  
2. 划拨无通道  
3. Key 无 account 调用  
4. 调度改扣费账户  
5. 先调后欠费  
6. Leader 无 Grant 即全平台权限  
7. iam 建 Key 绑无权账户  
8. 预算帽与余额不足同一错误码  
9. ModelGrant deny 后仍可调该模型  

---

# 第 8 章 关键边界规则与流程

## 8.1 预扣、价格约束与结算（完整）

1. **\(P_{request}\)**：请求逻辑模型 + 价目 + 预估用量 → 内部 sell 预估。  
2. **候选约束：** \(P_{candidate} \le P_{request}(1+\delta)\)；**默认 δ=0**；**硬上限 20%**；改 δ 必须关键配置审计；不合格剔除；空集 → `NO_ROUTE_WITHIN_PRICE_CAP`，无上游调用。  
3. **模型访问：** ModelGrant 失败 → `MODEL_ACCESS_DENIED`。  
4. **预算帽：** 见第 5 章 → `BUDGET_CAP_EXCEEDED`。  
5. **冻结：** 合格集最大预估 sell；余额不足 → `INSUFFICIENT_BALANCE`；freeze_id+expires_at。  
6. **调度：** 仅合格集；**account_id 锁定**。  
7. **结算：** 实际 cost/sell；多退少补；冻结失效走补偿流水+告警。

## 8.2 划拨路径

| 路径 | 默认 | 方向 |
|------|------|------|
| parent | 是 | 仅上级→下级 |
| sponsors | 是 | 仅出资方→被出资方 |
| owns/participates | 否 | — |
| 无关系 | 否 | 白名单+fund 除外 |

清算回流：显式目标账户 + fund + 审计。

## 8.3 冻结与流式续期

- 默认 TTL 15 分钟（可配 1–60 分钟级窗口参数）。  
- **流式：网关自动续期同一 freeze_id，不增加金额**；累计上限可配（如 2 小时）。  
- 客户端不负责续期。  
- 超时后台解冻 + 流水。  
- 持久化为准。

## 8.4 清算状态机

```text
active
  → liquidating_block_new    // 拒新调用与新冻结
  → liquidating_drain        // 排空冻结
  → liquidating_transfer     // 余额划转
  → liquidated               // Key 收口，主体只读
```

有未到期冻结不得划走占用额；超时告警停留 drain。

## 8.5 调用主路径

1. Key 鉴权、人状态、account 绑定  
2. 安全钩子（若启用）  
3. ModelGrant  
4. \(P_{request}\)、候选、价格过滤  
5. 预算帽 → 冻结  
6. 策略矩阵选路 → 托管密钥上游调用  
7. 流式续期  
8. 用量规范化 → 双轨结算 → 审计  
9. 兼容响应 + 分码错误  

## 8.6 划拨路径

fund 授权 → 通道 → 幂等键 → 事务双边记账 → 审计。

## 8.7 治理 API 幂等

- 适用范围：划拨、清算、资金补偿等写操作。  
- 头：Idempotency-Key（建议 UUID v4，≤255）。  
- 存储：`(scope, actor_id, idempotency_key)` 唯一；request_hash；status；响应语义；expires（建议≥24h）。  
- 原子抢占：INSERT ON CONFLICT；禁止先 SELECT 再 INSERT。  
- 同键同指纹重放；异指纹拒绝。  
- 实现参考 Stripe 语义。

## 8.8 账本技术

热账本 = PostgreSQL/TiDB/OceanBase 等 ACID + 只追加 ledger。  
**不采用**公链/联盟链作热余额账本。  
可选：流水防改 + 哈希归档 WORM。

---

# 第 9 章 功能需求编号全表

### 9.1 接入 RES-01～08

多上游、密钥仓库、密钥权限 routing、健康、标签、兼容 API、request_id、用量规范化。

### 9.2 计价 PRI-01～04

价目、解析、计算、落账。

### 9.3 资金 FUN-01～10

主体、关系、账本、划拨通道、划拨执行、预扣结算、**告警比例与预算上限（分码）**、清算、流水、冻结超时与续期。

### 9.4 Key KEY-01～06

生命周期、绑账户、归属人、绑户约束、成员、禁人联动。

### 9.5 授权 AUTH-01～05

四轴、最小默认、Leader 模板、数据范围强制、职责分离。

### 9.6 模型 MODEL（专章落实）

目录、绑定、ModelGrant、调用前校验。

### 9.7 路由 RTE-01～06

引擎、矩阵、高可用、账户正交 P0、价格约束 P0、决策可观测。

### 9.8 安全 SEC-01～05

网络、出网、内容、异常流量、扩展点 P0。

### 9.9 审计 AUD-01～04

调用审计、配置变更（含 δ）、对账、报表。

### 9.10 UI / API-01

角色化导航、主体、资金、价目、Key、路由档案、仪表盘、密钥仓库、治理 API 幂等。

---

# 第 10 章 数据纪律（反膨胀）

**逻辑实体/表优先集合：**  
parties、party_edges、accounts（含预算帽字段）、ledgers、freezes、model_prices、model_grants、idempotency_records、grants、api_keys（+account_id）、用量日志（+cost/sell/cost_items/account_id/freeze_id）、审计日志。

复杂规则 JSON。禁止能力维宽表、策略一表一张、无流水改余额。

---

# 第 11 章 系统架构与二次开发指导

## 11.1 逻辑架构

TokenHub 进程：Admin/Governance API + Data Plane `/v1/*`；管道 Auth → SecurityHooks → ModelGrant → PriceCap → BudgetCap → Freeze → Router(Strategy Matrix) → Adapter → Settle。

新增包建议：`fund`、`pricing`、`idempotency`、`party`、`authz`、`routing`、`modelgrant`、`security`。

## 11.2 技术栈

Go、PostgreSQL 等、Redis 可选、K8s、国产阶段 A 冒烟。

## 11.3 幂等实现要点

表 idempotency_records；唯一约束抢占；存 status 与响应；与账本同事务或严格两阶段。

## 11.4 策略引擎要点

```text
interface Strategy { Filter/Score(ctx, candidates) }
Profile: 启用的策略列表 + 配置
Pipeline: candidates → COMPLIANCE → ModelGrant → price cap → strategies → pick
```

## 11.5 WBS 分期

| 阶段 | 内容 |
|------|------|
| A | Fork TokenHub、PG、用量规范化、国产冒烟 |
| B | Party/账本/划拨/预算帽/冻结续期/清算/双轨/价格帽/四轴/ModelGrant 骨架/治理 API 幂等/安全钩子空实现 |
| C | 策略矩阵全量、决策日志、仪表盘 |
| D | 内容安全出网、变更快照、对账 |
| E | 压测 HA、GA、LICENSE/NOTICE、运维手册 |

## 11.6 多方验收门禁

Dev Complete → QA → UED → 产品 UAT → 安全 → 发布（架构/运维）。

## 11.7 商用交付物

安装包、OpenAPI、迁移回滚、UAT 脚本、监控项、许可 NOTICE、运维手册。

---

# 第 12 章 非功能需求

| 类别 | 要求 |
|------|------|
| 可用性 | 目标 99.9%，多实例切换 |
| 性能 | 按硬件量化验收附加延迟与吞吐 |
| 安全 | TLS、密钥加密、最小权限、可审计 |
| 部署 | 私有化、K8s、离线、国产早期验证 |
| 审计保留 | ≥180 天，冷热分离 |
| 可扩展 | 适配器、itemCode、策略、边类型、钩子 |
| 可观测 | 冻结任务、幂等冲突、预算帽命中、健康指标 |

---

# 第 13 章 验收标准（完整）

| 场景 | 通过条件 |
|------|----------|
| 统一接入 | ≥5 公有 +1 私有化兼容 |
| 双轨与 item | cost/sell 分项正确 |
| usage 不完整 | 有标记不伪造 |
| 独立项目/组织池/出资划拨 | 守恒与通道正确 |
| 价格约束与 δ | 默认 0、硬上限 20%、变更关键审计 |
| 预算帽 vs 余额不足 | 90% 帽 → BUDGET_CAP_EXCEEDED；余额不够 → INSUFFICIENT_BALANCE |
| 告警比例 | 80% 只告警不阻断 |
| 冻结超时/流式续期 | 符合 8.3 |
| 清算状态机 | 符合 8.4 |
| 幂等 | 重复写不双记 |
| ModelGrant | deny 后不可调 |
| 四轴越权 | Leader/路由/资金职责分离成立 |
| 调度不改账户 | 任意路由 account 不变 |
| 策略矩阵启停组合 | 可测 |
| 禁人即禁 Key | 立即 |
| 治理 API | 对等鉴权 |
| INTERNAL_ONLY | 无外网流量 |

**财务演示脚本：** 预算与加价 → 划拨 → 人 Key 调用 → 双轨流水一致 → 预算帽分码 → 余额不足分码 → 清算 → 幂等划拨。

---

# 第 14 章 不在范围

Agent IDE、多账本分摊单次请求、默认透支、客户端冻结续期、无权限密钥写入、区块链热账本。多币种、复杂审批、HR/ERP 深度双向等单独立项。

---

# 第 15 章 术语表

Token、网关 Key、Party、双轨、请求锚定内部价、δ、预算帽、BUDGET_CAP_EXCEEDED、INSUFFICIENT_BALANCE、MODEL_ACCESS_DENIED、预扣/结算、冻结续期、清算、正交授权、ModelGrant、策略矩阵、治理 API、幂等键、itemCode、TokenHub 底座、AxonHub 计价吸收。

---

# 第 16 章 总结

本定版 2.0 将历史会话中全部既定结论合并为可执行全文，包括但不限于：

- 财务优先与物理世界 Party 多态、Leader、Key 扣费、守恒  
- TokenHub 底座 + AxonHub 计价与评分吸收  
- 可插拔路由策略矩阵全文  
- 价格帽、δ≤20%、关键审计  
- 预算帽数据模型与判定顺序  
- 错误码全表（预算帽与余额不足分码）  
- 模型权限 ModelGrant + 四轴正交 + 红线清单  
- 冻结 TTL 与流式网关续期、清算状态机、幂等、非 DLT 热账本  
- 薄 DDL、分期 WBS、多方门禁、验收与演示脚本  

**后续任何修订只能增量追加并标注变更说明，禁止再次整体压缩导致细节丢失。**

---

**文档结束（定版 2.0 全量）。**