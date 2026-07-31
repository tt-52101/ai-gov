// Package reconciliation 对账接口契约——阶段 D 实现具体逻辑。
// 当前文件仅定义接口，不做任何实现，供 P0 阶段预留集成点。
package reconciliation

import (
	"context"
	"time"
)

// ReconciliationService 对账服务接口——阶段 D 实现具体逻辑。
//
// 职责（PRD §9.8 AUD-03）：
//   - 启动对账运行：设置账期区间，初始状态为 pending。
//   - 导入上游账单：从外部计费系统（如厂商账单 CSV/API）解析 BillingItem 列表。
//   - 差异比对与分类：将上游账单与内部账本逐项比对，输出差异分类报告。
//
// P0 阶段约定：
//   - 本接口仅作为"预留接口契约"存在——所有方法的具体实现延后至阶段 D。
//   - P0 阶段已在路由层注册占位端点（POST /gov/reconciliation-runs 和
//     GET /gov/reconciliation-runs/{id}），返回 501 暂未实现。
//   - 对账数据模型（ReconciliationRun 等）见 model.go。
type ReconciliationService interface {
	// StartRun 启动一次新的对账运行。
	//
	// 参数:
	//   - ctx: 请求上下文。
	//   - periodStart: 对账周期起始时间（含）。
	//   - periodEnd: 对账周期结束时间（含）。
	//
	// 返回值:
	//   - 新创建的对账运行记录（状态为 pending）。
	//   - 若参数无效或数据库写入失败，返回错误。
	//
	// 阶段 D 实现要点：校验 periodStart < periodEnd，确保同一账期不重复启动。
	StartRun(ctx context.Context, periodStart, periodEnd time.Time) (*ReconciliationRun, error)

	// ImportUpstreamBilling 导入上游账单数据到指定对账运行。
	//
	// 参数:
	//   - ctx: 请求上下文。
	//   - runID: 目标对账运行 ID。
	//   - billingData: 从上游计费系统解析的账单明细列表。
	//
	// 返回值:
	//   - 导入成功后返回 nil。
	//   - 若 runID 不存在或状态非 pending/importing，返回错误。
	//
	// 阶段 D 实现要点：
	//   - 将 billingData 写入临时表或 reconciliation_runs 的 JSONB 字段。
	//   - 更新 run 状态为 importing（首次导入）或保持在 importing。
	//   - 支持分批导入——同一 run 可多次调用 ImportUpstreamBilling。
	ImportUpstreamBilling(ctx context.Context, runID string, billingData []UpstreamBillingItem) error

	// CompareAndClassify 执行上游账单与内部账本的逐项比对，生成差异报告。
	//
	// 参数:
	//   - ctx: 请求上下文。
	//   - runID: 目标对账运行 ID。
	//
	// 返回值:
	//   - 更新后的对账运行记录——含 upstream_total、internal_total、difference
	//     及 variance_reasons 字段。
	//   - 若 runID 不存在或状态不允许比对（如仍为 pending），返回错误。
	//
	// 阶段 D 实现要点：
	//   - 按 account_id + event_time 做 join 比对上游与内部账本。
	//   - 差异分类：upstream_extra / internal_extra / amount_mismatch / timing_shift。
	//   - 比对完成后将 status 设为 matched（无差异）或 variance（有差异）。
	//   - 记录审计事件（调用 audit.RecordEvent）以保留对账快照。
	CompareAndClassify(ctx context.Context, runID string) (*ReconciliationRun, error)
}
