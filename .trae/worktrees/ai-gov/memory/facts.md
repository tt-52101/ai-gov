# 事实提取 · 项目知识基线

> 从全部历史会话中提取的事实性信息，作为后续会话的上下文注入基础。

## 项目身份
- 名称：企业级 AI 智能网关治理平台（Token 治理底座）
- 代号：ai-gov
- 技术栈：Go + PostgreSQL/TiDB/OceanBase + K8s
- 开源底座：TokenHub (astaxie) 主仓库 + AxonHub (looplj) 计价吸收

## 文档体系
- 原始需求本源：`docs/history/原始产品功能设计需求讨论.md`
- **正式主线基线：`docs/prd/AI-GOV-PRD-v3.2.0.md`（v3.2.0，~1250行，40表DDL，25守恒定理，ABAC引擎）**
- 过渡版本：`docs/prd/AI-GOV-PRD-v2.0.2.md`（v2.0.2，917行，价值回填版）
- 历史最终版：`docs/prd/AI-GOV-PRD-v2.0.1.md`（v2.0.1，852行）
- v3.x 前身：`docs/prd/AI-GOV-PRD-v3.0.0-融合架构完整方案.md`、`AI-GOV-PRD-v3.1.0.md`
- 丢失分析：`docs/memory/需求回溯与丢失分析-2026-07-31.md`
- 复盘文件：`docs/history/复盘/`

## PRD 版本链路
```
原始讨论(505行) → v1.0.0(558) → v1.0.1(688) → v1.0.2(487) → v2.0.0(672) → v2.0.1(852)
```
v3.x 系列（v3.0.0, v3.1.0）存在但独立于 v2.x 链路。

## 数据库 Schema
- `schema/ai-gov.sql` — 原始 DDL
- `schema/ai-gov-fusion-minimal.sql` — 融合最小 DDL
- `docs/analysis/DDL-GAP-ANALYSIS-v1.0.md` — DDL 差距分析
- `docs/analysis/FUSION-DDL-MAPPING-v1.0.md` — 融合映射

## 设计专题文档
- `docs/spec/AI-GOV.md` — 产品规格
- `docs/spec/design/资金守恒定理.md` — 资金守恒理论
- `docs/spec/design/资金守恒缺口修复方案.md` — 修复方案
- `docs/spec/design/资金守恒缺口审计.md` — 缺口审计
- `docs/spec/design/两阶段提交（2PC）与分布式事务补偿.md` — 事务方案
- `docs/spec/design/账本与辅助缓存：功能设计与可复用实现方案.md` — 账本缓存
- `docs/spec/design/routing/模型路由策略动态矩阵：融合设计与实现方案.md` — 路由策略

## 第三方源码
- `third-party/TokenHub/` — astaxie/TokenHub（含分析报告和融合方案）
- `third-party/axonhub/` — looplj/axonhub（含 .trae 规则）
- `third-party/融合架构可行性论证报告.md`
- `third-party/融合架构演进设计方案-llm平移与控制数据面分离.md`
- `third-party/融合架构演进设计方案-v2-llm全量平移与双轨保留.md`
- `third-party/融合DB模型设计-v1-三方融合GORM模型.md`
- `third-party/项目分析规约.md`

## 需求丢失清单（已编号）
详见 `lost-requirements.md`，共 17 项 L-01~L-17。
P0: L-01~L-06（对外售卖预留、多级预算、多模态计价、固定摊销、非功能验收、AUD-01 断链）
P1: L-07~L-11（架构原则、缓存折扣、模型预判器、对账流程、性能数值）
P2: L-12~L-17（安全报表、调用追踪UI、验收焦点列、API枚举、增强目录、价值叙事）

## 下一步基准
**当前正式主线基线：v3.2.0**
已完成：A（v2.0.2 价值回填）→ B（v3.2.0 主线跃迁）
L-01~L-17 处理状态：14项已在 v3.1.0 解决 + 2项在 v3.2.0 解决（L-14/L-17）+ 1项设计取舍（L-02 四级预算→双层预算）
后续：进入详细设计（DDL对齐、API设计、组件架构）
