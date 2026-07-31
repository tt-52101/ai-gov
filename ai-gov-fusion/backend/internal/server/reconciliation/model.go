// Package reconciliation 实现上游账单与内部账本的对账逻辑——阶段 D 完成具体实现。
// 当前阶段（P0）仅提供数据模型、接口契约和 HTTP 端点骨架。
package reconciliation

import (
	"time"
)

// ── 对账运行状态常量 ──────────────────────────────────────────────────────────

const (
	// StatusPending 对账运行已创建，等待上游账单导入。
	StatusPending = "pending"
	// StatusImporting 上游账单导入中。
	StatusImporting = "importing"
	// StatusComparing 差异比对中。
	StatusComparing = "comparing"
	// StatusMatched 对账通过——上游总额与内部总额一致，无差异。
	StatusMatched = "matched"
	// StatusVariance 存在差异——上游与内部数据不一致，需人工核查。
	StatusVariance = "variance"
	// StatusResolved 差异已人工确认、处理完毕。
	StatusResolved = "resolved"
	// StatusError 对账过程发生系统错误。
	StatusError = "error"
)

// ── 数据模型 ──────────────────────────────────────────────────────────────────

// ReconciliationRun 对账运行记录——表示一次完整的上游账单与内部账本比对流程。
//
// GORM 表: reconciliation_runs
//
// 生命周期（PRD §9.8 AUD-03）：
//   pending → importing → comparing → matched | variance → resolved
//
// 差异分类存储在 variance_reasons（JSONB）中，每项为 VarianceReason 序列化结果。
// upstream_total 和 internal_total 为对账周期内分别从上游账单和内部账本汇总的金额。
type ReconciliationRun struct {
	// RunID 对账运行主键，UUID 前缀 run_。
	RunID string `json:"run_id" gorm:"primaryKey"`

	// PeriodStart 对账周期起始时间（含）。
	PeriodStart time.Time `json:"period_start" gorm:"index;not null"`

	// PeriodEnd 对账周期结束时间（含）。
	PeriodEnd time.Time `json:"period_end" gorm:"index;not null"`

	// Status 对账运行当前状态，使用本包 Status* 常量。
	Status string `json:"status" gorm:"type:varchar(32);not null;default:pending;index"`

	// UpstreamTotal 上游账单汇总金额（单位：分或最小货币精度）。
	UpstreamTotal int64 `json:"upstream_total" gorm:"not null;default:0"`

	// InternalTotal 内部账本汇总金额（单位：分或最小货币精度）。
	InternalTotal int64 `json:"internal_total" gorm:"not null;default:0"`

	// Difference 上游总额与内部总额之间的差额（上游 - 内部）。
	// 正值表示上游多计费，负值表示上游少计费。
	Difference int64 `json:"difference" gorm:"not null;default:0"`

	// VarianceReasons 差异分类明细（JSONB 格式）。
	//
	// 每项为 VarianceReason 的 JSON 序列化结果，记录差异金额与原因分类。
	// 对账通过时此字段为空数组 "[]"。
	//
	// 差异分类（PRD §9.8）：
	//   - upstream_extra: 上游多计——仅上游存在、内部无记录的计费项
	//   - internal_extra: 内部多计——内部有记录、上游无对应计费项
	//   - amount_mismatch: 金额不一致——同一计费项双方记录金额不同
	//   - timing_shift: 时间偏移——账期边界附近的时间戳差异所致
	VarianceReasons string `json:"variance_reasons,omitempty" gorm:"type:jsonb"`

	// CreatedAt 对账运行创建时间戳。
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime;index"`
}

// TableName 覆盖 GORM 默认表名。
func (ReconciliationRun) TableName() string { return "reconciliation_runs" }

// ── 请求/结果结构体 ───────────────────────────────────────────────────────────

// VarianceReason 差异分类明细——描述一条具体的对账差异。
type VarianceReason struct {
	// Category 差异分类：upstream_extra / internal_extra / amount_mismatch / timing_shift。
	Category string `json:"category"`

	// Amount 差异金额（单位：分或最小货币精度）。
	Amount int64 `json:"amount"`

	// Description 差异说明——人类可读的描述信息。
	Description string `json:"description"`

	// ReferenceID 关联的资源标识（如账单行 ID、账本记录 ID），用于溯源。
	ReferenceID string `json:"reference_id,omitempty"`
}

// UpstreamBillingItem 上游账单项——从外部计费系统导入的单条计费记录。
//
// 阶段 D 实现具体的导入解析逻辑后，此结构体将被用于
// ReconciliationService.ImportUpstreamBilling 的输入参数。
type UpstreamBillingItem struct {
	// BillingID 外部计费系统赋予的唯一标识。
	BillingID string `json:"billing_id"`

	// AccountID 计费项关联的账户 ID。
	AccountID string `json:"account_id"`

	// Amount 计费金额（单位：分或最小货币精度）。
	Amount int64 `json:"amount"`

	// Currency 币种代码（如 "CNY"、"USD"）。
	Currency string `json:"currency"`

	// EventTime 计费事件发生时间。
	EventTime time.Time `json:"event_time"`

	// Description 计费项描述——从上游账单原文提取。
	Description string `json:"description,omitempty"`
}
