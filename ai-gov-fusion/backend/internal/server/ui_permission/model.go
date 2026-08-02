package ui_permission

import "time"

// ── 主体与资源类型（ABAC 引擎参数） ──────────────────────────────────────────

// Subject 描述 ABAC 评估的主体——访问者身份及其属性。
// Type 为 "user" 或 "party"，ID 为对应的唯一标识，Attributes 承载附加属性（如角色列表）。
type Subject struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// Resource 描述 ABAC 评估的资源——被访问的目标及其属性。
type Resource struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// ── UI 权限模型（对应 DDL 表 sys_ui_menus / sys_ui_routes / sys_ui_action_bindings） ──

// SysUIMenu 动态菜单树节点，支持 self-ref parent_id 无限层级嵌套。
// Visible 字段为 ABAC 投影结果，不存储在数据库中——前端据此决定显示或隐藏。
//
// GORM 表: sys_ui_menus
type SysUIMenu struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	MenuCode  string    `json:"menu_code" gorm:"type:varchar(128);uniqueIndex;not null"`
	ParentID  *int64    `json:"parent_id,omitempty" gorm:"index"`
	Label     string    `json:"label" gorm:"type:varchar(128);not null"`
	Icon      string    `json:"icon,omitempty" gorm:"type:varchar(64)"`
	SortOrder int       `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Visible 为 ABAC 投影结果——当前主体是否有权看到此菜单。
	// gorm:"-" 标记不入库，每次请求时由 Projector 动态计算。
	Visible  bool         `json:"visible" gorm:"-"`
	Children []*SysUIMenu `json:"children,omitempty" gorm:"-"`
}

// TableName 覆盖 GORM 默认表名。
func (SysUIMenu) TableName() string { return "sys_ui_menus" }

// SysUIRoute 前端路由到菜单和操作的权限映射。
// 访问路由需要对应操作权限——无权限时路由守卫拦截。
//
// GORM 表: sys_ui_routes
type SysUIRoute struct {
	ID               int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	RoutePath        string    `json:"route_path" gorm:"type:varchar(256);uniqueIndex;not null"`
	MenuID           *int64    `json:"menu_id,omitempty" gorm:"index"`
	RequiredActionID *string   `json:"required_action_id,omitempty" gorm:"type:text;index"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (SysUIRoute) TableName() string { return "sys_ui_routes" }

// SysUIActionBinding 页面按钮到操作权限的映射。
// 每个按钮通过 required_action_id 关联 sys_action_catalogs 中的操作——
// ABAC 评估通过则按钮显示，否则隐藏。
//
// GORM 表: sys_ui_action_bindings
type SysUIActionBinding struct {
	ID               int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ButtonCode       string    `json:"button_code" gorm:"type:varchar(128);not null;index"`
	ButtonLabel      string    `json:"button_label" gorm:"type:varchar(128);not null"`
	PageRoute        string    `json:"page_route" gorm:"type:varchar(256);not null;index"`
	RequiredActionID *string   `json:"required_action_id,omitempty" gorm:"type:text;index"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (SysUIActionBinding) TableName() string { return "sys_ui_action_bindings" }

// ── 请求类型 ──────────────────────────────────────────────────────────────

// CreateMenuRequest 创建菜单的请求参数。
type CreateMenuRequest struct {
	MenuCode  string `json:"menu_code"`            // 菜单编码，全局唯一
	ParentID  *int64 `json:"parent_id,omitempty"`  // 父菜单 ID，nil 为顶级菜单
	Label     string `json:"label"`                // 显示文本
	Icon      string `json:"icon,omitempty"`       // 图标标识
	SortOrder int    `json:"sort_order,omitempty"` // 同级排序，默认 0
}

// CreateRouteRequest 创建路由的请求参数。
// RequiredActionID 非 nil 时会在 store 层校验对应的 sys_action_catalogs 记录是否存在。
type CreateRouteRequest struct {
	RoutePath        string  `json:"route_path"`                   // 前端路由路径，如 /dashboard/usage
	MenuID           *int64  `json:"menu_id,omitempty"`            // 关联菜单
	RequiredActionID *string `json:"required_action_id,omitempty"` // 关联操作 UUID，校验是否存在
}

// CreateActionBindingRequest 创建按钮绑定的请求参数。
// RequiredActionID 非 nil 时会在 store 层校验对应的 sys_action_catalogs 记录是否存在。
type CreateActionBindingRequest struct {
	ButtonCode       string  `json:"button_code"`                 // 按钮编码，如 btn-create-key
	ButtonLabel      string  `json:"button_label"`                // 按钮显示文本
	PageRoute        string  `json:"page_route"`                  // 所属页面路由
	RequiredActionID *string `json:"required_action_id,omitempty"` // 关联操作 UUID，校验是否存在
}
