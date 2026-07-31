// Package routing 实现模型请求的智能路由引擎，核心为可插拔策略矩阵（12 种策略）。
//
// 策略管道按固定顺序执行：S-COMPLIANCE（合规过滤）→ ModelGrant（模型授权）→
// 价格帽（δ 过滤）→ S-CLASSIFY（智能分类）→ 其余策略 → 选取最优候选。
//
// 每个策略必须实现 Strategy 接口的 Filter（剔除不合格候选）和 Score（打分排序）
// 两个方法。策略通过全局注册表管理，路由档案通过策略绑定组合多种策略。
package routing

import (
	"context"
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ── 健康状态常量 ──────────────────────────────────────────────────────────

const (
	// HealthUp 表示渠道运行正常。
	HealthUp = "up"
	// HealthDegraded 表示渠道降级运行，仍可用但性能或可靠性下降。
	HealthDegraded = "degraded"
	// HealthDown 表示渠道不可用，应从候选集中剔除。
	HealthDown = "down"
)

// ── 策略代码常量 ──────────────────────────────────────────────────────────

const (
	StrategyCompliance = "S-COMPLIANCE" // 合规网络——硬过滤，不可关闭
	StrategyHealth     = "S-HEALTH"     // 健康与熔断——三态熔断
	StrategyPriority   = "S-PRI"        // 优先级分组——主备硬分组
	StrategyWeight     = "S-WEIGHT"     // 权重与负载——按配置权重分配
	StrategyCost       = "S-COST"       // 成本感知——低价优先
	StrategyLatency    = "S-LATENCY"    // 延迟感知——EWMA 延迟越低越好
	StrategyError      = "S-ERROR"      // 错误率感知——近期成功率惩罚
	StrategyRate       = "S-RATE"       // 限流感知——降低 429 概率
	StrategyAffinity   = "S-AFFINITY"   // 会话亲和——同会话优先同渠道
	StrategyTag        = "S-TAG"        // 业务标签——按标签定向路由
	StrategyCache      = "S-CACHE"      // 缓存兜底——最后手段降级
	StrategyClassify   = "S-CLASSIFY"   // 智能分类——任务复杂度预判
)

// HealthState 表示渠道健康状态。
type HealthState = string

// ── 核心接口 ──────────────────────────────────────────────────────────────

// Strategy 路由策略接口——所有 12 种策略必须实现。
//
// Filter：剔除不合格候选（返回过滤后的切片）。硬策略（如 S-COMPLIANCE）在此阶段
// 强制移除不符合条件的候选，软策略可在 Filter 中不做任何操作。
//
// Score：为合格候选打分（修改 Candidate.Score 字段）。分数越高优先级越高。
// 多个策略的分数累加，最终按总分降序排列选取最优候选。
//
// 每个策略通过 ID() 返回唯一标识符，用于注册表和审计日志。
type Strategy interface {
	// ID 返回策略唯一标识符（如 "S-COST"），用于注册表和审计日志。
	ID() string

	// Filter 剔除不合格候选。返回过滤后的切片（可为空切片）。
	// 不会修改原始切片——调用方应使用返回值。
	Filter(ctx context.Context, candidates []Candidate) []Candidate

	// Score 为合格候选打分，直接修改每个 Candidate.Score 字段。
	// 调用方负责在管道开始前将 Score 归零。
	Score(ctx context.Context, candidates []Candidate) []Candidate
}

// Candidate 路由候选——一个可用的上游渠道+模型组合。
//
// 每个候选携带身份标识、健康状态、价格预估、延迟与错误率等维度数据，
// 供各策略进行过滤和打分。Score 字段在管道执行过程中累加。
// Eliminated 标记为 true 的候选不会进入最终排序。
type Candidate struct {
	// ChannelID 标识上游渠道（对应 provider_resources.id）。
	ChannelID int64 `json:"channel_id"`

	// ModelID 标识逻辑模型名（对应 models.name）。
	ModelID string `json:"model_id"`

	// Priority 静态优先级，数值越大优先级越高（S-PRI 使用）。
	Priority int `json:"priority"`

	// Weight 配置权重，表示分配概率的相对比重（S-WEIGHT 使用）。
	Weight float64 `json:"weight"`

	// Health 健康状态：up / degraded / down（S-HEALTH 使用）。
	Health HealthState `json:"health"`

	// EstSell 预估内部结算价格（用于 δ 价格帽过滤和 S-COST 打分）。
	EstSell decimal.Decimal `json:"est_sell"`

	// EstCost 预估上游成本（参考值，不参与价格帽计算）。
	EstCost decimal.Decimal `json:"est_cost"`

	// LatencyEWMA 指数加权移动平均延迟（S-LATENCY 使用）。
	LatencyEWMA time.Duration `json:"latency_ewma"`

	// ErrorRate 近期错误率（0.0–1.0），S-ERROR 使用。
	ErrorRate float64 `json:"error_rate"`

	// Score 综合评分，管道执行过程中由各策略累加。越高越优先。
	Score float64 `json:"score"`

	// Eliminated 标记是否已被剔除。被剔除的候选不参与最终排序。
	Eliminated bool `json:"eliminated"`

	// ElimReason 剔除原因（仅当 Eliminated 为 true 时有意义）。
	ElimReason string `json:"elim_reason,omitempty"`

	// Metadata 扩展元数据，策略可在此存储临时数据（如 S-AFFINITY 的命中标记）。
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ── 路由档案与策略绑定 ──────────────────────────────────────────────────

// StrategyBinding 策略绑定——将策略代码与启用状态关联。
// 在路由档案中配置，决定管道中包含哪些策略。
type StrategyBinding struct {
	// Code 策略代码，如 "S-COST"、"S-HEALTH"。
	Code string `json:"code"`

	// Enabled 是否启用该策略。
	Enabled bool `json:"enabled"`

	// Priority 策略在管道中的排序位置（数值越小越先执行）。
	Priority int `json:"priority"`

	// Config 策略级配置 JSON，由各策略自行解析。
	Config json.RawMessage `json:"config,omitempty"`
}

// RouteProfile 路由档案——策略组合 + δ 价格帽 + 影子模式。
//
// 路由档案定义了一组策略的绑定和执行参数。δ 价格帽约束候选内部价不得
// 超过锚定价 × (1+δ)。影子模式（Shadow=true）下仅记录路由决策而不实际
// 执行路由，用于灰度验证新的策略组合。
//
// GORM 表: route_profiles
type RouteProfile struct {
	ID          int64            `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string           `json:"name" gorm:"uniqueIndex;not null"`
	Description string           `json:"description,omitempty" gorm:"type:text"`
	Strategies  []StrategyBinding `json:"strategies" gorm:"serializer:json;column:strategies_json;not null;default:'[]'"`
	DeltaCap    decimal.Decimal  `json:"delta_cap" gorm:"type:numeric(18,6);not null;default:0"`
	MaxAttempts int              `json:"max_attempts" gorm:"default:3"`
	Shadow      bool             `json:"shadow" gorm:"default:false"`
	Status      string           `json:"status" gorm:"default:active"`
	CreatedAt   time.Time        `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time        `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (RouteProfile) TableName() string { return "route_profiles" }

// ── 档案状态常量 ──────────────────────────────────────────────────────────

const (
	// ProfileStatusActive 表示档案处于活跃状态，可被路由引擎使用。
	ProfileStatusActive = "active"
	// ProfileStatusInactive 表示档案已停用。
	ProfileStatusInactive = "inactive"
)

// ── 管道执行约束 ──────────────────────────────────────────────────────────

const (
	// MaxDeltaCap δ 价格帽硬上限（20%），超过此值拒绝保存。
	MaxDeltaCap = 0.20

	// MaxAttemptsDefault 默认最大重试次数。
	MaxAttemptsDefault = 3

	// MaxAttemptsHardLimit 最大重试次数硬上限。
	MaxAttemptsHardLimit = 10
)

// ── 迁移入口 ──────────────────────────────────────────────────────────────

// Migrate 执行路由包所有 GORM 模型的自动迁移。
// 在 store.go 协调层按阶段 3 调用。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&RouteProfile{})
}
