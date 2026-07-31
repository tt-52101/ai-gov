package routing

import "time"

// Decision 路由决策日志——记录一次完整路由管道的执行结果。
//
// 每次 ExecuteProfile 调用均产生一个 Decision 实例，记录输入候选、
// 输出候选、最终选中的渠道以及完整的策略执行链。决策日志用于审计追溯
// 和路由效果分析。
//
// GORM 表: route_decisions
type Decision struct {
	// ID 决策日志主键。
	ID int64 `json:"id" gorm:"primaryKey;autoIncrement"`

	// ProfileName 使用的路由档案名称。
	ProfileName string `json:"profile_name" gorm:"not null;index"`

	// CandidatesIn 进入管道的候选数量。
	CandidatesIn int `json:"candidates_in" gorm:"not null"`

	// CandidatesOut 管道输出后的候选数量。
	CandidatesOut int `json:"candidates_out" gorm:"not null"`

	// Selected 最终选中的渠道 ID（0 表示无可用候选）。
	Selected int64 `json:"selected" gorm:"not null"`

	// StrategyChain 策略执行链，按执行顺序记录策略 ID。
	StrategyChain []string `json:"strategy_chain" gorm:"serializer:json"`

	// InputSnapshot 输入候选的 ChannelID 快照，用于审计追溯。
	InputSnapshot []int64 `json:"input_snapshot" gorm:"serializer:json"`

	// Shadow 是否以影子模式执行（仅记录不路由）。
	Shadow bool `json:"shadow" gorm:"default:false"`

	// Error 路由过程中发生的错误消息（如有）。
	Error string `json:"error,omitempty" gorm:"type:text"`

	// Timestamp 决策时间戳。
	Timestamp time.Time `json:"timestamp" gorm:"not null;index"`

	// CreatedAt 数据库写入时间。
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (Decision) TableName() string { return "route_decisions" }
