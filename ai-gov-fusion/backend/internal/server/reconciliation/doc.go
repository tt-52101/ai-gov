// Package reconciliation 实现对账模块——上游账单与内部账本的数据一致性校验。
//
// 模块职责（PRD §9.8 AUD-03）：
//   - 定义对账运行数据模型（ReconciliationRun）与差异分类结构（VarianceReason）。
//   - 声明对账服务接口契约（ReconciliationService）供阶段 D 实现。
//   - 提供 GORM AutoMigrate + 基础 CRUD 操作。
//
// P0 阶段范围：
//   - 数据模型 + 接口契约 + HTTP 占位端点——不实现任何对账业务逻辑。
//   - 对账端点已注册于 /gov/reconciliation-runs 和 /gov/reconciliation-runs/{id}。
//   - go build 通过，接口可被后续阶段引用。
//
// 文件组织：
//   - model.go: 数据模型（ReconciliationRun / VarianceReason / UpstreamBillingItem）。
//   - contract.go: ReconciliationService 接口定义。
//   - store.go: GORM AutoMigrate + 基础 CRUD 操作。
package reconciliation
