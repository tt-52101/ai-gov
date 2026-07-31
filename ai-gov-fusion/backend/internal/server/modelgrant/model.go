// Package modelgrant 实现模型访问治理——控制哪些主体可以访问哪些模型。
//
// 核心机制：
//   - ALLOW/DENY 规则：DENY 优先于 ALLOW（A-CON-04）
//   - 级联顺序：Key > Person > Party > 全局默认
//   - 模型级配额（quota_limit）：双层预算第二层，与 Account 预算帽取交集
//   - 禁止仅因 Leader 头衔自动拥有全平台模型权（A-CON-05）
//
// 本包是 ABAC 权限治理体系的数据面组件，在数据面 Pipeline 第 4 步执行。
package modelgrant

import (
	"time"

	"github.com/shopspring/decimal"
)

// ── 主体类型常量 ────────────────────────────────────────────────────────────

const (
	// PrincipalParty 组织或项目主体。
	PrincipalParty = "party"
	// PrincipalPerson 实体人主体。
	PrincipalPerson = "person"
	// PrincipalKey API 密钥主体。
	PrincipalKey = "key"
	// PrincipalRole 角色主体。
	PrincipalRole = "role"
)

// ── 效果常量 ────────────────────────────────────────────────────────────────

const (
	// EffectAllow 显式允许访问该模型。
	EffectAllow = "allow"
	// EffectDeny 显式拒绝访问该模型（优先于 ALLOW）。
	EffectDeny = "deny"
)

// ── 域模型 ──────────────────────────────────────────────────────────────────

// Principal 表示请求访问模型的主体。
// 级联评估时按 Key > Person > Party > 全局默认的顺序逐级查找匹配规则。
type Principal struct {
	// Type 主体类型：party / person / key / role。
	Type string

	// ID 主体唯一标识。
	ID string
}

// ModelGrant 表示一条模型访问授权记录。
// 每条记录定义「哪个主体对哪个模型（或模型标签组）拥有何种访问权限」。
// DENY 规则在任何优先级下均优先于 ALLOW（A-CON-04）。
//
// GORM 表: model_grants
type ModelGrant struct {
	// ID 授权记录唯一标识。
	ID string `json:"id" gorm:"type:text;primaryKey"`

	// PrincipalType 主体类型：party / person / key / role。
	PrincipalType string `json:"principal_type" gorm:"type:text;not null;index:idx_mg_principal"`

	// PrincipalID 主体唯一标识。
	PrincipalID string `json:"principal_id" gorm:"type:text;not null;index:idx_mg_principal"`

	// ModelID 单个逻辑模型 ID（可为空，与 ModelTag 二选一或同时为空表示全局默认）。
	ModelID *string `json:"model_id,omitempty" gorm:"type:text;index:idx_mg_model"`

	// ModelTag 模型标签组（可为空，与 ModelID 二选一）。
	ModelTag *string `json:"model_tag,omitempty" gorm:"type:text"`

	// Effect 效果：allow / deny（deny 优先）。
	Effect string `json:"effect" gorm:"type:text;not null;index:idx_mg_effect"`

	// Priority 冲突解析优先级，数值越大越优先。
	Priority int `json:"priority" gorm:"default:0"`

	// QuotaLimit 模型级预算上限（NULL 表示不限制）。
	// 与 Account.budget_limit_amount 取交集——两者中最严格的生效。
	QuotaLimit *decimal.Decimal `json:"quota_limit,omitempty" gorm:"type:decimal(18,6)"`

	// QuotaConsumed 本授权下已累计消费金额。
	// 由 ConsumeQuota 更新，用于 CheckQuotaLimit 预算判定。
	QuotaConsumed decimal.Decimal `json:"quota_consumed" gorm:"type:decimal(18,6);not null;default:0"`

	// Conditions 附加条件 JSON（预留扩展）。
	Conditions string `json:"conditions,omitempty" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (ModelGrant) TableName() string { return "model_grants" }
