package audit

import (
	"time"
)

// ── 审计事件状态常量 ──────────────────────────────────────────────────────

const (
	// StatusSuccess 表示操作成功完成。
	StatusSuccess = "success"
	// StatusFailure 表示操作执行失败。
	StatusFailure = "failure"
)

// ── 审计行动常量（对齐 grants.action 与 PRD §7.3）────────────────────────

const (
	// ActionBudgetCapChange 预算帽变更——修改账户级预算上限金额或告警比例。
	ActionBudgetCapChange = "budget_cap.change"

	// ActionPriceChange 价目变更——修改模型或渠道的双轨价格（cost/sell）。
	ActionPriceChange = "price.change"

	// ActionRouteProfileChange 路由档案变更——修改策略矩阵配置或 δ 值。
	ActionRouteProfileChange = "route_profile.change"

	// ActionAllocate 资金划拨——从源账户向目标账户转移资金。
	ActionAllocate = "fund.allocate"

	// ActionLiquidate 清算操作——启动或推进账户清算状态机。
	ActionLiquidate = "fund.liquidate"

	// ActionGrantChange 四轴授权变更——修改 grants 表的授权记录。
	ActionGrantChange = "grant.change"

	// ActionModelGrantChange 模型授权变更——修改 model_grants 表的授权记录。
	ActionModelGrantChange = "model_grant.change"

	// ActionKeyCreate 密钥创建——创建新的 API Key。
	ActionKeyCreate = "key.create"

	// ActionKeyRevoke 密钥吊销——吊销或轮换已有 API Key。
	ActionKeyRevoke = "key.revoke"
)

// ── 数据模型 ──────────────────────────────────────────────────────────────

// AuditEvent 审计事件——记录系统中每一次管理操作或关键变更的完整上下文。
//
// GORM 表: audit_events
//
// 铁律（D-CON-04 / AU-CON-01）：
//   - 应用层仅允许 INSERT，禁止 UPDATE 与 DELETE。
//   - 所有管理员配置变更必须保存 before_snapshot 和 after_snapshot。
//   - 保留不少于 180 天。
type AuditEvent struct {
	// ID 审计事件主键，UUID v4。
	ID string `json:"id" gorm:"primaryKey"`

	// ActorUserID 执行操作的用户标识。
	ActorUserID string `json:"actor_user_id,omitempty" gorm:"index:idx_audit_events_actor"`

	// ActorName 执行操作的用户显示名，冗余字段便于直接展示。
	ActorName string `json:"actor_name,omitempty"`

	// Action 操作类型，使用本包定义的 Action* 常量。
	Action string `json:"action" gorm:"type:varchar(128);not null;index:idx_audit_events_action"`

	// ResourceType 被操作资源的类型（如 party、account、model_price）。
	ResourceType string `json:"resource_type" gorm:"type:varchar(64);not null;index:idx_audit_events_resource"`

	// ResourceID 被操作资源的唯一标识。
	ResourceID string `json:"resource_id" gorm:"not null;index:idx_audit_events_resource"`

	// Status 操作结果状态——成功（success）或失败（failure）。
	Status string `json:"status,omitempty" gorm:"type:varchar(32)"`

	// Message 附加说明或错误信息。
	Message string `json:"message,omitempty" gorm:"type:text"`

	// BeforeSnapshot 变更前的资源完整快照（JSON 格式）。
	// 对于写操作，此字段为必填；对于只读审计查询可为空。
	BeforeSnapshot string `json:"before_snapshot,omitempty" gorm:"type:jsonb"`

	// AfterSnapshot 变更后的资源完整快照（JSON 格式）。
	// 对于写操作，此字段为必填；对于只读审计查询可为空。
	AfterSnapshot string `json:"after_snapshot,omitempty" gorm:"type:jsonb"`

	// IP 操作发起方的客户端 IP 地址。
	IP string `json:"ip,omitempty" gorm:"type:varchar(64)"`

	// UserAgent 操作发起方的 User-Agent 请求头。
	UserAgent string `json:"user_agent,omitempty" gorm:"type:text"`

	// CreatedAt 事件创建时间戳。
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime;index:idx_audit_events_actor;index:idx_audit_events_action;index:idx_audit_events_resource"`
}

// TableName 覆盖 GORM 默认表名。
func (AuditEvent) TableName() string { return "audit_events" }

// AuditChainAnchor 审计哈希链锚点——将一段连续审计事件锚定为一个不可篡改的
// SHA-256 哈希链节点，用于防篡改验证与合规存档。
//
// GORM 表: audit_chain_anchors
//
// 锚定原理（PRD §7.6）：
//   新锚点的 anchor_hash 由前一锚点哈希、事件 ID 范围、事件计数及时间戳
//   拼接后计算 SHA-256 得到。链式结构确保任意中间节点被篡改后验证失败。
type AuditChainAnchor struct {
	// ID 锚点主键，UUID v4。
	ID string `json:"id" gorm:"primaryKey"`

	// AnchorHash SHA-256 链锚哈希值，唯一约束。
	AnchorHash string `json:"anchor_hash" gorm:"uniqueIndex;not null"`

	// StartEventID 此锚点覆盖的起始审计事件 ID。
	StartEventID string `json:"start_event_id" gorm:"index:idx_audit_chain_anchors_start;not null"`

	// EndEventID 此锚点覆盖的结束审计事件 ID。
	EndEventID string `json:"end_event_id" gorm:"index:idx_audit_chain_anchors_end;not null"`

	// EventCount 此锚点覆盖范围内的审计事件数量。
	EventCount int `json:"event_count" gorm:"not null;default:0"`

	// CreatedAt 锚点创建时间戳。
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime;index:idx_audit_chain_anchors_created"`
}

// TableName 覆盖 GORM 默认表名。
func (AuditChainAnchor) TableName() string { return "audit_chain_anchors" }

// ── 查询类型 ──────────────────────────────────────────────────────────────

// AuditFilter 审计事件检索过滤条件。
// 所有字段均为可选——空值表示不对该维度做过滤。
type AuditFilter struct {
	// ActorUserID 按操作用户筛选。
	ActorUserID string `json:"actor_user_id,omitempty"`

	// Action 按操作类型筛选（使用 Action* 常量）。
	Action string `json:"action,omitempty"`

	// ResourceType 按资源类型筛选。
	ResourceType string `json:"resource_type,omitempty"`

	// ResourceID 按具体资源 ID 筛选。
	ResourceID string `json:"resource_id,omitempty"`

	// Status 按操作结果状态筛选。
	Status string `json:"status,omitempty"`

	// StartTime 事件创建时间的起始区间（含）。
	StartTime *time.Time `json:"start_time,omitempty"`

	// EndTime 事件创建时间的结束区间（含）。
	EndTime *time.Time `json:"end_time,omitempty"`

	// Offset 分页偏移量，从 0 开始。
	Offset int `json:"offset"`

	// Limit 每页最大返回条数，最大 200。
	Limit int `json:"limit"`
}
