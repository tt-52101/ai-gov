// Package authz 实现四轴正交授权——grants 表直接授权模型，作为 ABAC 策略引擎的补充。
// 四轴（data/fund/iam/routing）权限独立判定，禁止一轴推导另一轴特权（A-CON-01）。
// 未显式授予即拒绝（A-CON-02：最小权限默认）。
package authz

import "time"

// ── 主体类型常量 ────────────────────────────────────────────────────────────

const (
	// PrincipalUser 实体人。
	PrincipalUser = "user"
	// PrincipalParty 组织或项目。
	PrincipalParty = "party"
	// PrincipalKey API 密钥。
	PrincipalKey = "key"
	// PrincipalRole 角色。
	PrincipalRole = "role"
)

// ── 治理轴常量（PRD §7.2.4）────────────────────────────────────────────────

const (
	// AxisData 数据轴——用量查看、报表查看、成员查看。
	AxisData = "data"
	// AxisFund 资金轴——余额查看、流水查看、划拨、清算、预算配置。
	AxisFund = "fund"
	// AxisIAM 身份轴——Key 创建/吊销/轮换、用户禁用、成员管理。
	AxisIAM = "iam"
	// AxisRouting 路由轴——价格配置、路由档案配置、渠道配置、上游密钥配置、模型目录配置、模型授权配置。
	AxisRouting = "routing"
)

// ── 效果常量 ────────────────────────────────────────────────────────────────

const (
	// EffectAllow 显式允许。
	EffectAllow = "allow"
	// EffectDeny 显式拒绝（优先于 ALLOW）。
	EffectDeny = "deny"
)

// ── 操作常量（PRD §7.2.4 四轴正交）─────────────────────────────────────────

// data 轴操作
const (
	ActionUsageRead  = "usage.read"
	ActionReportRead = "report.read"
	ActionMemberRead = "member.read"
)

// fund 轴操作
const (
	ActionBalanceRead  = "balance.read"
	ActionLedgerRead   = "ledger.read"
	ActionAllocate     = "allocate"
	ActionLiquidate    = "liquidate"
	ActionBudgetWrite  = "budget.write"
)

// iam 轴操作
const (
	ActionKeyCreate  = "key.create"
	ActionKeyRevoke  = "key.revoke"
	ActionKeyRotate  = "key.rotate"
	ActionUserDisable = "user.disable"
	ActionMemberAdd  = "member.add"
	ActionMemberRemove = "member.remove"
)

// routing 轴操作
const (
	ActionPriceWrite       = "price.write"
	ActionRouteProfileWrite = "route_profile.write"
	ActionChannelWrite     = "channel.write"
	ActionUpstreamSecretWrite = "upstream_secret.write"
	ActionModelCatalogWrite = "model_catalog.write"
	ActionModelGrantWrite  = "model_grant.write"
)

// AllActions 返回本包定义的所有操作常量的合集，供 UI 投影及策略评估使用。
func AllActions() []string {
	return []string{
		ActionUsageRead, ActionReportRead, ActionMemberRead,
		ActionBalanceRead, ActionLedgerRead, ActionAllocate, ActionLiquidate, ActionBudgetWrite,
		ActionKeyCreate, ActionKeyRevoke, ActionKeyRotate, ActionUserDisable, ActionMemberAdd, ActionMemberRemove,
		ActionPriceWrite, ActionRouteProfileWrite, ActionChannelWrite,
		ActionUpstreamSecretWrite, ActionModelCatalogWrite, ActionModelGrantWrite,
	}
}

// ── 资源类型常量 ────────────────────────────────────────────────────────────

const (
	ResourceParty   = "party"
	ResourceAccount = "account"
	ResourceModel   = "model"
	ResourceAny     = "*"
)

// ── 域模型 ──────────────────────────────────────────────────────────────────

// Grant 表示一条四轴正交授权记录。
// 每条记录定义「哪个主体在哪个轴上可以对哪个资源执行什么操作」。
// Grant 是 ABAC 策略引擎的补充——ABAC 处理策略化规则（"所有部门 Leader 均可…"），
// Grant 处理一次性直接指派（"张三可以对 AI 研发部账本查看余额"）。
//
// GORM 表: grants
type Grant struct {
	// ID 授权记录唯一标识。
	ID string `json:"id" gorm:"type:text;primaryKey"`

	// PrincipalType 主体类型：user / party / key / role。
	PrincipalType string `json:"principal_type" gorm:"type:text;not null;index:idx_grants_principal"`

	// PrincipalID 主体唯一标识。
	PrincipalID string `json:"principal_id" gorm:"type:text;not null;index:idx_grants_principal"`

	// Axis 治理轴：data / fund / iam / routing。
	Axis string `json:"axis" gorm:"type:text;not null;index:idx_grants_axis"`

	// Action 操作标识，如 balance.read / allocate / price.write。
	Action string `json:"action" gorm:"type:text;not null;index:idx_grants_axis"`

	// ResourceType 资源类型：party / account / model，NULL 表示不限。
	ResourceType *string `json:"resource_type,omitempty" gorm:"type:text;index:idx_grants_resource"`

	// ResourceID 具体资源 ID 或 '*' 通配，NULL 表示不限。
	ResourceID *string `json:"resource_id,omitempty" gorm:"type:text;index:idx_grants_resource"`

	// Effect 效果：allow / deny（deny 优先）。
	Effect string `json:"effect" gorm:"type:text;not null;default:allow"`

	// Conditions 附加条件 JSON（预留 ABAC 条件集成）。
	Conditions string `json:"conditions,omitempty" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (Grant) TableName() string { return "grants" }
