# 单兵轨迹 · QA-CODE-DATA

| 项 | 值 |
|----|-----|
| Agent ID | QA-CODE-DATA |
| 所属蜂群 | batch-008 代码事实核查 |
| 作战指令 | 基于事实代码核查数据层DDL与部署配置（schema/deploy/scripts） |
| 执行耗时 | 约 4 分钟 |
| 验收判定 | ❌ 不通过——2 项 P0 + 3 项 P1 |

## 情报收集（逐文件）

### DDL Schema
- schema/ai-gov-fusion-v3.2.sql：
  - 40 表 ✅（与 PRD §10.1 100% 符合）
  - 103 索引 ✅
  - 3 存储过程 ✅（evaluate_access / evaluate_access_via_roles / anchor_audit_chain）
  - model_grants.party_id ✅（batch-007 FIX-E 修复确认）
  - route_profiles.party_id ✅（batch-007 FIX-E 修复确认）
  - liquidations.liquidation_type ❌ 缺失（P0-7）
  - 无 trg_append_only 触发器 ⚠️

### 迁移脚本
- scripts/migrate-db-v3.2.mjs：27表/75索引/83 addColumn/17 INSERT，幂等可重运行 ✅
- scripts/e2e-db-seed-test.mjs：E2E 种子脚本 ✅

### 部署配置
- deploy/docker-compose.yml ✅
- deploy/docker-compose.postgres.yml ✅
- deploy/docker-compose.e2e.yml ✅
- deploy/.env.example ⚠️ 缺少 GOV_API_KEY_PREFIX/AUDIT_RETENTION_DAYS/BUDGET_WARN_RATIO_DEFAULT
- deploy/install.sh ⚠️ 无 psql -f schema 步骤（GAP-013）
- deploy/nginx.multi-instance.conf ✅
- deploy/native/install.sh ✅
- deploy/native/tokenhub.service ✅
- deploy/local/run-local.sh ✅
- K8s Helm Chart ❌ 全局无 Chart.yaml（P0-8）

### Dockerfile
- backend/Dockerfile 多阶段构建 ✅
- backend/Dockerfile.native 静态产物 ✅

### GORM AutoMigrate
- server/store.go:481-508 集中入口（17 模型）✅
- 11 个模块分散迁移（25 模型）✅
- 注释明确"40 张 v3.2 表由 SQL 脚本在部署时创建" ✅

### 国产环境
- store.go:30-31 仅导入 postgres+sqlite 驱动 ❌
- 无 OceanBase/TiDB/MySQL 驱动 ❌
- 无麒麟/统信UOS/鲲鹏/飞腾适配代码 ❌
- 所有"国产"提及仅在文档层面 ❌（P1-10）

### 可观测性
- 全项目无 prometheus client 代码 ❌（P1-11）

## 战果产出

| 文件 | 行数 | 关键发现 |
|------|------|---------|
| schema/ai-gov-fusion-v3.2.sql | 1341 | 40表完整；liquidation_type 缺失（P0-7） |
| scripts/migrate-db-v3.2.mjs | - | 27表/75索引/83addColumn，幂等 ✅ |
| deploy/ 目录 | - | Docker/systemd/nginx 完整；K8s Helm 缺失（P0-8） |
| backend/store.go | - | 仅 postgres+sqlite 驱动（P1-10） |

## 发现结论

### P0 阻塞（2 项）
- P0-7: liquidations.liquidation_type 字段缺失
- P0-8: K8s Helm Chart 缺失

### P1 重要（3 项）
- P1-10: 国产环境完全缺失（无 OceanBase/TiDB 驱动）
- P1-11(隐含): 无 Prometheus 指标导出代码
- HIGH-02: PRD 状态机命名矛盾（§8.4 vs §10.1）

### P2 一般（3 项）
- MID-01: 无 trg_append_only 触发器兜底
- MID-02: AuditEvent 双重迁移
- MID-03: 离线打包

## 阶段 A/E 验收焦点
- 阶段 A：DDL ✅、用量规范化 ✅、国产冒烟 ❌（未达成）
- 阶段 E：压测 HA ❌、K8s Helm ❌、文档许可 ❌（未实施）
