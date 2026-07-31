# 项目上下文 · 完整知识库

## 产品定位
**企业级 AI 智能网关治理平台（Token 治理底座）**

核心隐喻：Token = AI 时代的电力。产品是企业内部的"智能电表 + 配电箱 + 财务结算中心"。
不做发电厂（不卖模型），只做企业内部 AI 能源控制面。

## 目标闭环
```
模型接入 → 安全拦截 → 智能调度 → 实时扣费 → 资金回笼与清算 → 对账审计
```

## 领域术语表
| 术语 | 定义 |
|------|------|
| Token | 计费用量及缓存、推理等衍生类型 |
| 网关 Key | 企业发给调用方的凭证，≠ 上游厂商密钥 |
| Party | 组织或项目等可持账本与成员的主体（org = project 同层语义） |
| 双轨 | cost（上游成本价）与 sell（内部结算价）并行计量 |
| 请求锚定内部价 | 约束调度候选的内部 sell 基准 |
| δ | 候选相对锚定价允许上浮比例；默认 0；硬上限 20% |
| 预算帽 | 可低于余额 100% 的配置上限；命中 → BUDGET_CAP_EXCEEDED |
| INSUFFICIENT_BALANCE | 可用余额不足以冻结（与预算帽分码） |
| MODEL_ACCESS_DENIED | ModelGrant 拒绝模型访问 |
| 预扣/结算 | 调用前冻结，调用后按实际内部价结清，多退少补 |
| 冻结续期 | 流式由网关延长同一 freeze_id 过期时间，不增加金额 |
| 清算 | 阻断新调用 → 排空冻结 → 余额回流 → Key 收口 |
| 正交授权 | data / fund / iam / routing 四轴分轴授权 |
| ModelGrant | 模型访问 allow/deny 规则；deny 优先 |
| 策略矩阵 | 11 个可启停、可混合的路由策略（PRI/HEALTH/WEIGHT/AFFINITY/COST/LATENCY/ERROR/RATE/TAG/COMPLIANCE/CACHE） |
| 治理 API | 与管理台对等的控制面 API；资金写操作强制幂等 |
| 幂等键 | Idempotency-Key（UUID v4），防写操作重试重复记账 |
| itemCode | 与上游账单对齐的费用项编码（prompt_tokens 等） |
| TokenHub | 主仓库底座（astaxie/TokenHub），Go 后端 + 管理台，企业私有 AI 网关 |
| AxonHub | 计价与多维评分思想来源（looplj/axonhub），itemCode 级明细 + JSON 价目 |

## 物理世界模型
- Party = org 与 project 同一层语义
- 关系边：parent（上级→下级划拨）、sponsors（出资方→被出资方）、owns（不开通划拨）、participates（不开通划拨）
- Key 必须绑定唯一 account_id；归属于实体人；扣费唯一入口
- 资金流转：划拨等额同事务 → 预扣入冻结 → 结算多退少补 → 清算走状态机 → 任何余额变更必须有流水
- 消费强制：sell 只扣 Key 绑定账户，归因到人；Tag 只做报表

## 技术选型
- 后端：Go（基于 TokenHub 二次开发）
- 热账本：PostgreSQL / TiDB / OceanBase（ACID SQL，非区块链）
- 缓存/短状态：Redis（可选）
- 部署：Docker / K8s；离线/内网；国产 CPU/OS 阶段 A 验证
- 审计保留：≥180 天

## 价格约束
- P_request: 请求锚定内部 sell 预估
- P_candidate ≤ P_request × (1+δ)；默认 δ=0；硬上限 20%
- δ 修改 → 关键配置变更审计
- 不合格候选 → 剔除；无合格候选 → NO_ROUTE_WITHIN_PRICE_CAP

## 预扣判定顺序
```
ModelGrant → 构建候选 → 价格合格集过滤 → 预算帽（若启用）→ 可用余额 ≥ 冻结额 → 写入 freeze → 策略矩阵选路 → 上游 → 结算
```

## 清算状态机
```
active → liquidating_block_new → liquidating_drain → liquidating_transfer → liquidated
```

## 开源底座
- 主仓库：TokenHub（astaxie/TokenHub）
- 吸收 AxonHub：itemCode 级计价 + JSON 价目 + 三模式 + 多维评分思想
- 强制 cost/sell 双轨（AxonHub 是单轨成本）

## 已知丢失需求清单（L-01 ~ L-17）
详见 `.trae/worktrees/ai-gov/memory/lost-requirements.md`
