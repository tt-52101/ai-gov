// Package abac 实现基于属性的访问控制（ABAC）策略引擎。
// 策略评估顺序：显式 deny → 显式 allow → 角色权限 → 默认拒绝。
// ABAC 作为本产品的统一授权引擎，RBAC（角色）作为主体属性之一，不构成独立授权体系。
//
// 本包包含 6 张 ABAC 治理表的数据模型与 GORM 映射：
//   - sys_action_catalogs      原子操作目录（四轴分类）
//   - sys_roles                角色定义
//   - sys_role_permissions     角色→操作 N:M 映射
//   - sys_subject_role_bindings 主体→角色绑定（含作用域）
//   - sys_access_policies      ABAC 策略定义
//   - sys_access_policy_bindings 策略→主体绑定
package abac

import (
	"crypto/rand"
	"fmt"
	"time"
)

// ── 轴线常量（PRD §7.2.4 四轴正交） ──────────────────────────────────────

const (
	// AxisData 数据轴：用量查询、报表读取、成员查看。
	AxisData = "data"
	// AxisFund 资金轴：余额、流水、划拨、清算、预算帽。
	AxisFund = "fund"
	// AxisIAM 身份轴：Key 管理、用户管理、成员管理。
	AxisIAM = "iam"
	// AxisRouting 路由轴：价目、路由档案、上游密钥、模型目录。
	AxisRouting = "routing"
)

// ── 策略效果常量 ──────────────────────────────────────────────────────────

const (
	// EffectAllow 允许访问。
	EffectAllow = "allow"
	// EffectDeny 拒绝访问。deny 优先于 allow。
	EffectDeny = "deny"
)

// ── 主体类型常量 ──────────────────────────────────────────────────────────

const (
	// SubjectTypeUser 自然人主体。
	SubjectTypeUser = "user"
	// SubjectTypeParty 组织/项目主体。
	SubjectTypeParty = "party"
	// SubjectTypeRole 角色主体（策略绑定到角色）。
	SubjectTypeRole = "role"
	// SubjectTypeKey API Key 主体。
	SubjectTypeKey = "key"
)

// ── 请求/上下文类型 ───────────────────────────────────────────────────────

// Subject 表示执行操作的主体。
// 可以是用户（user）或组织/项目（party）。
type Subject struct {
	// Type 为主体类型："user" 或 "party"。
	Type string `json:"type"`
	// ID 为主体的唯一标识。
	ID string `json:"id"`
}

// Resource 表示被操作的目标资源。
type Resource struct {
	// Type 为资源类型，如 "party", "account", "model"。
	Type string `json:"type"`
	// ID 为资源的唯一标识，可为空（表示不限定具体资源）。
	ID string `json:"id"`
}

// ── GORM 数据模型 ────────────────────────────────────────────────────────

// SysActionCatalog 原子操作目录。
// 定义系统中所有可执行动作，按四轴（data/fund/iam/routing）分类。
// 所有 ABAC 权限判定最终收敛到对某个 action_code 的允许或拒绝。
//
// GORM 表: sys_action_catalogs
type SysActionCatalog struct {
	// ID 为操作记录唯一标识（UUID）。
	ID string `json:"id" gorm:"type:text;primaryKey"`
	// ActionCode 为操作编码，如 "balance.read"、"model.invoke"。
	ActionCode string `json:"action_code" gorm:"type:text;uniqueIndex;not null"`
	// ActionName 为操作的中文名称，如 "查看余额"。
	ActionName string `json:"action_name" gorm:"type:text;not null"`
	// Axis 为所属治理轴：data / fund / iam / routing。
	Axis string `json:"axis" gorm:"type:text;not null;index"`
	// ResourceType 为关联资源类型，如 "party"、"account"、"model"。
	ResourceType string `json:"resource_type,omitempty" gorm:"type:text;index"`
	// Description 为操作描述。
	Description string `json:"description,omitempty" gorm:"type:text"`
	// CreatedAt 为创建时间。
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	// UpdatedAt 为最后更新时间。
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (SysActionCatalog) TableName() string { return "sys_action_catalogs" }

// SysRole 角色定义。
// 角色是权限的集合模板，通过 sys_role_permissions 与操作关联，
// 通过 sys_subject_role_bindings 分配给主体。
// 系统角色（is_system=true）不可删除。
//
// GORM 表: sys_roles
type SysRole struct {
	// ID 为角色记录唯一标识（UUID）。
	ID string `json:"id" gorm:"type:text;primaryKey"`
	// RoleCode 为角色编码，如 "super_admin"、"finance_mgr"。
	RoleCode string `json:"role_code" gorm:"type:text;uniqueIndex;not null"`
	// RoleName 为角色中文名称，如 "超级管理员"。
	RoleName string `json:"role_name" gorm:"type:text;not null"`
	// Description 为角色描述。
	Description string `json:"description,omitempty" gorm:"type:text"`
	// IsSystem 标记是否为系统内置角色。系统角色不可删除。
	IsSystem bool `json:"is_system" gorm:"not null;default:false"`
	// CreatedAt 为创建时间。
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	// UpdatedAt 为最后更新时间。
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (SysRole) TableName() string { return "sys_roles" }

// SysRolePermission 角色→操作 N:M 映射。
// 表示某个角色拥有某个操作的权限。
//
// GORM 表: sys_role_permissions
type SysRolePermission struct {
	// ID 为映射记录唯一标识（UUID）。
	ID string `json:"id" gorm:"type:text;primaryKey"`
	// RoleID 为角色外键，关联 sys_roles.id。
	RoleID string `json:"role_id" gorm:"type:text;not null;index;uniqueIndex:idx_role_action,priority:1"`
	// ActionID 为操作外键，关联 sys_action_catalogs.id。
	ActionID string `json:"action_id" gorm:"type:text;not null;index;uniqueIndex:idx_role_action,priority:2"`
	// CreatedAt 为创建时间。
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (SysRolePermission) TableName() string { return "sys_role_permissions" }

// SysSubjectRoleBinding 主体→角色绑定。
// 将角色分配给用户或 Party，支持作用域限制（scope_party_id）。
// scope_party_id 为 NULL 表示全局生效；指定则仅在对应 Party 及其下级生效。
// valid_from / valid_until 控制角色有效期。
//
// GORM 表: sys_subject_role_bindings
type SysSubjectRoleBinding struct {
	// ID 为绑定记录唯一标识（UUID）。
	ID string `json:"id" gorm:"type:text;primaryKey"`
	// SubjectType 为主体类型："user" 或 "party"。
	SubjectType string `json:"subject_type" gorm:"type:text;not null;index:idx_srb_subject,priority:1"`
	// SubjectID 为主体唯一标识。
	SubjectID string `json:"subject_id" gorm:"type:text;not null;index:idx_srb_subject,priority:2"`
	// RoleID 为角色外键，关联 sys_roles.id。
	RoleID string `json:"role_id" gorm:"type:text;not null;index"`
	// ScopePartyID 为角色生效的组织范围。NULL 表示全局。
	ScopePartyID *string `json:"scope_party_id,omitempty" gorm:"type:text;index"`
	// ValidFrom 为绑定生效起始时间。NULL 表示立即生效。
	ValidFrom *time.Time `json:"valid_from,omitempty"`
	// ValidUntil 为绑定失效时间。NULL 表示永久有效。
	ValidUntil *time.Time `json:"valid_until,omitempty"`
	// CreatedAt 为创建时间。
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	// UpdatedAt 为最后更新时间。
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (SysSubjectRoleBinding) TableName() string { return "sys_subject_role_bindings" }

// SysAccessPolicy ABAC 策略定义。
// 策略通过 effect（allow/deny）+ conditions_json 实现细粒度访问控制。
// priority 越高越优先评估；deny 策略始终在 allow 之前评估。
// 系统策略（is_system=true）不可删除。
//
// GORM 表: sys_access_policies
type SysAccessPolicy struct {
	// ID 为策略记录唯一标识（UUID）。
	ID string `json:"id" gorm:"type:text;primaryKey"`
	// PolicyCode 为策略编码，如 "P-DENY-EXTERNAL-MODEL"。
	PolicyCode string `json:"policy_code" gorm:"type:text;uniqueIndex;not null"`
	// PolicyName 为策略中文名称。
	PolicyName string `json:"policy_name" gorm:"type:text;not null"`
	// Effect 为策略效果："allow" 或 "deny"。
	Effect string `json:"effect" gorm:"type:text;not null;default:allow;index"`
	// ConditionsJSON 为 ABAC 条件 JSON，格式如 {"axis":"data","actions":["model.invoke"],"resource_type":"model"}。
	ConditionsJSON string `json:"conditions_json" gorm:"type:text;not null;default:{}"`
	// Priority 为策略优先级，数值越大越优先评估。
	Priority int `json:"priority" gorm:"not null;default:0;index"`
	// IsSystem 标记是否为系统内置策略。系统策略不可删除。
	IsSystem bool `json:"is_system" gorm:"not null;default:false"`
	// Description 为策略描述。
	Description string `json:"description,omitempty" gorm:"type:text"`
	// CreatedAt 为创建时间。
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	// UpdatedAt 为最后更新时间。
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (SysAccessPolicy) TableName() string { return "sys_access_policies" }

// SysAccessPolicyBinding 策略→主体绑定。
// 将 ABAC 策略与用户、Party、角色或 Key 关联。
// 一个策略可绑定多个主体，一个主体可被多个策略约束。
//
// GORM 表: sys_access_policy_bindings
type SysAccessPolicyBinding struct {
	// ID 为绑定记录唯一标识（UUID）。
	ID string `json:"id" gorm:"type:text;primaryKey"`
	// PolicyID 为策略外键，关联 sys_access_policies.id。
	PolicyID string `json:"policy_id" gorm:"type:text;not null;index"`
	// SubjectType 为主体类型："user"、"party"、"role" 或 "key"。
	SubjectType string `json:"subject_type" gorm:"type:text;not null;index:idx_apb_subject,priority:1"`
	// SubjectID 为主体唯一标识。
	SubjectID string `json:"subject_id" gorm:"type:text;not null;index:idx_apb_subject,priority:2"`
	// CreatedAt 为创建时间。
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (SysAccessPolicyBinding) TableName() string { return "sys_access_policy_bindings" }

// ── 辅助函数 ──────────────────────────────────────────────────────────────

// newID 生成简化的唯一标识符（UUID v4 风格）。
// 用于在写入数据库前为模型分配主键。
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read 在 Linux/Windows 上极少失败；
		// 若失败则退化为时间戳 + 回退随机。
		_ = err
		b = make([]byte, 16)
		for i := range b {
			b[i] = byte(time.Now().UnixNano()>>(i*4)) ^ byte(i*37)
		}
	}
	// 设置 UUID v4 变体位。
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Migrate 执行 GORM AutoMigrate 为所有 6 张 ABAC 表创建或更新表结构。
// 由 store.go 编排层在 Phase 2 迁移阶段调用。
func Migrate(db interface{ AutoMigrate(dst ...interface{}) error }) error {
	if err := db.AutoMigrate(
		&SysActionCatalog{},
		&SysRole{},
		&SysRolePermission{},
		&SysSubjectRoleBinding{},
		&SysAccessPolicy{},
		&SysAccessPolicyBinding{},
	); err != nil {
		return fmt.Errorf("abac 迁移失败: %w", err)
	}
	return nil
}
