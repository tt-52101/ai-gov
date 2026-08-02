package ui_permission

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// sysActionCatalog 是 sys_action_catalogs 表的轻量投影——仅用于 store 层 FK 校验。
// 完整模型定义在 abac 包中，ui_permission 不直接依赖 abac。
type sysActionCatalog struct {
	ID         string `gorm:"type:text;primaryKey"`
	ActionCode string `gorm:"type:varchar(128);uniqueIndex;not null"`
}

// TableName 覆盖 GORM 默认表名。
func (sysActionCatalog) TableName() string { return "sys_action_catalogs" }

// ── 菜单 CRUD ──────────────────────────────────────────────────────────────

// CreateMenu 插入新菜单记录。menu_code 必须全局唯一，label 不可为空。
// parent_id 非 nil 时会校验父菜单是否存在。
func CreateMenu(db *gorm.DB, req CreateMenuRequest) (*SysUIMenu, error) {
	if req.MenuCode == "" {
		return nil, errors.New("ui_permission: menu_code 必填")
	}
	if req.Label == "" {
		return nil, errors.New("ui_permission: label 必填")
	}
	if req.ParentID != nil {
		var parent SysUIMenu
		if err := db.First(&parent, *req.ParentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("ui_permission: 父菜单 %d 不存在", *req.ParentID)
			}
			return nil, err
		}
	}
	m := &SysUIMenu{
		MenuCode:  req.MenuCode,
		ParentID:  req.ParentID,
		Label:     req.Label,
		Icon:      req.Icon,
		SortOrder: req.SortOrder,
	}
	if err := db.Create(m).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 创建菜单失败: %w", err)
	}
	return m, nil
}

// GetMenu 按主键 ID 查询菜单。
func GetMenu(db *gorm.DB, id int64) (*SysUIMenu, error) {
	var m SysUIMenu
	if err := db.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("ui_permission: 菜单 %d 不存在", id)
		}
		return nil, err
	}
	return &m, nil
}

// UpdateMenu 更新菜单的 label、icon、sort_order、parent_id。
// 不可用于修改 menu_code（编码一经创建不可更改）。
func UpdateMenu(db *gorm.DB, id int64, req CreateMenuRequest) (*SysUIMenu, error) {
	m, err := GetMenu(db, id)
	if err != nil {
		return nil, err
	}
	if req.Label != "" {
		m.Label = req.Label
	}
	m.Icon = req.Icon
	m.SortOrder = req.SortOrder
	m.ParentID = req.ParentID
	if err := db.Save(m).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 更新菜单失败: %w", err)
	}
	return m, nil
}

// DeleteMenu 按主键 ID 删除菜单。调用方负责确认菜单下无关联路由和子菜单。
func DeleteMenu(db *gorm.DB, id int64) error {
	result := db.Delete(&SysUIMenu{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("ui_permission: 菜单 %d 不存在", id)
	}
	return nil
}

// ListMenus 返回所有菜单，按 parent_id 和 sort_order 排序。
// 用于构建菜单树——调用方按 parent_id 组装层级结构。
func ListMenus(db *gorm.DB) ([]*SysUIMenu, error) {
	var menus []*SysUIMenu
	if err := db.Order("parent_id ASC, sort_order ASC").Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 查询菜单列表失败: %w", err)
	}
	return menus, nil
}

// ── 路由 CRUD ──────────────────────────────────────────────────────────────

// CreateRoute 插入新路由记录。route_path 必须全局唯一。
// required_action_id 非 nil 时会校验 sys_action_catalogs 中对应记录是否存在。
func CreateRoute(db *gorm.DB, req CreateRouteRequest) (*SysUIRoute, error) {
	if req.RoutePath == "" {
		return nil, errors.New("ui_permission: route_path 必填")
	}
	if req.RequiredActionID != nil {
		if err := validateActionExists(db, *req.RequiredActionID); err != nil {
			return nil, err
		}
	}
	r := &SysUIRoute{
		RoutePath:        req.RoutePath,
		MenuID:           req.MenuID,
		RequiredActionID: req.RequiredActionID,
	}
	if err := db.Create(r).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 创建路由失败: %w", err)
	}
	return r, nil
}

// GetRoute 按主键 ID 查询路由。
func GetRoute(db *gorm.DB, id int64) (*SysUIRoute, error) {
	var r SysUIRoute
	if err := db.First(&r, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("ui_permission: 路由 %d 不存在", id)
		}
		return nil, err
	}
	return &r, nil
}

// UpdateRoute 更新路由的 menu_id 和 required_action_id。
// required_action_id 非 nil 时会校验 sys_action_catalogs 中对应记录是否存在。
func UpdateRoute(db *gorm.DB, id int64, req CreateRouteRequest) (*SysUIRoute, error) {
	r, err := GetRoute(db, id)
	if err != nil {
		return nil, err
	}
	if req.RequiredActionID != nil {
		if err := validateActionExists(db, *req.RequiredActionID); err != nil {
			return nil, err
		}
	}
	r.MenuID = req.MenuID
	r.RequiredActionID = req.RequiredActionID
	if err := db.Save(r).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 更新路由失败: %w", err)
	}
	return r, nil
}

// DeleteRoute 按主键 ID 删除路由。
func DeleteRoute(db *gorm.DB, id int64) error {
	result := db.Delete(&SysUIRoute{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("ui_permission: 路由 %d 不存在", id)
	}
	return nil
}

// ListRoutes 返回所有路由记录。
func ListRoutes(db *gorm.DB) ([]*SysUIRoute, error) {
	var routes []*SysUIRoute
	if err := db.Order("route_path ASC").Find(&routes).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 查询路由列表失败: %w", err)
	}
	return routes, nil
}

// FindRoutesByMenu 返回关联到指定菜单 ID 的所有路由。
func FindRoutesByMenu(db *gorm.DB, menuID int64) ([]*SysUIRoute, error) {
	var routes []*SysUIRoute
	if err := db.Where("menu_id = ?", menuID).Order("route_path ASC").Find(&routes).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 按菜单查询路由失败: %w", err)
	}
	return routes, nil
}

// ── 按钮绑定 CRUD ──────────────────────────────────────────────────────────

// CreateActionBinding 插入新按钮绑定记录。
// button_code + page_route 组合必须唯一（数据库 UNIQUE 约束强制）。
// required_action_id 非 nil 时会校验 sys_action_catalogs 中对应记录是否存在。
func CreateActionBinding(db *gorm.DB, req CreateActionBindingRequest) (*SysUIActionBinding, error) {
	if req.ButtonCode == "" {
		return nil, errors.New("ui_permission: button_code 必填")
	}
	if req.ButtonLabel == "" {
		return nil, errors.New("ui_permission: button_label 必填")
	}
	if req.PageRoute == "" {
		return nil, errors.New("ui_permission: page_route 必填")
	}
	if req.RequiredActionID != nil {
		if err := validateActionExists(db, *req.RequiredActionID); err != nil {
			return nil, err
		}
	}
	b := &SysUIActionBinding{
		ButtonCode:       req.ButtonCode,
		ButtonLabel:      req.ButtonLabel,
		PageRoute:        req.PageRoute,
		RequiredActionID: req.RequiredActionID,
	}
	if err := db.Create(b).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 创建按钮绑定失败: %w", err)
	}
	return b, nil
}

// GetActionBinding 按主键 ID 查询按钮绑定。
func GetActionBinding(db *gorm.DB, id int64) (*SysUIActionBinding, error) {
	var b SysUIActionBinding
	if err := db.First(&b, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("ui_permission: 按钮绑定 %d 不存在", id)
		}
		return nil, err
	}
	return &b, nil
}

// UpdateActionBinding 更新按钮绑定的 label 和 required_action_id。
// required_action_id 非 nil 时会校验 sys_action_catalogs 中对应记录是否存在。
func UpdateActionBinding(db *gorm.DB, id int64, req CreateActionBindingRequest) (*SysUIActionBinding, error) {
	b, err := GetActionBinding(db, id)
	if err != nil {
		return nil, err
	}
	if req.RequiredActionID != nil {
		if err := validateActionExists(db, *req.RequiredActionID); err != nil {
			return nil, err
		}
	}
	if req.ButtonLabel != "" {
		b.ButtonLabel = req.ButtonLabel
	}
	b.RequiredActionID = req.RequiredActionID
	if err := db.Save(b).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 更新按钮绑定失败: %w", err)
	}
	return b, nil
}

// DeleteActionBinding 按主键 ID 删除按钮绑定。
func DeleteActionBinding(db *gorm.DB, id int64) error {
	result := db.Delete(&SysUIActionBinding{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("ui_permission: 按钮绑定 %d 不存在", id)
	}
	return nil
}

// ListActionBindingsByPage 返回指定页面路由下的所有按钮绑定。
func ListActionBindingsByPage(db *gorm.DB, pageRoute string) ([]*SysUIActionBinding, error) {
	var bindings []*SysUIActionBinding
	if err := db.Where("page_route = ?", pageRoute).
		Order("button_code ASC").Find(&bindings).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 按页面查询按钮绑定失败: %w", err)
	}
	return bindings, nil
}

// ListAllActionBindings 返回全部按钮绑定记录。
func ListAllActionBindings(db *gorm.DB) ([]*SysUIActionBinding, error) {
	var bindings []*SysUIActionBinding
	if err := db.Order("page_route ASC, button_code ASC").Find(&bindings).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 查询按钮绑定列表失败: %w", err)
	}
	return bindings, nil
}

// ── 辅助校验 ──────────────────────────────────────────────────────────────

// validateActionExists 校验指定的 action_id 在 sys_action_catalogs 表中存在。
func validateActionExists(db *gorm.DB, actionID string) error {
	var ac sysActionCatalog
	if err := db.First(&ac, "id = ?", actionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("ui_permission: sys_action_catalogs 中 action_id=%s 不存在", actionID)
		}
		return fmt.Errorf("ui_permission: 校验操作目录失败: %w", err)
	}
	return nil
}

// ── 迁移 ───────────────────────────────────────────────────────────────────

// Migrate 执行 GORM AutoMigrate 创建或更新三张 UI 权限表。
// 由 store.go 编排层在 Phase 2 迁移中调用。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&SysUIMenu{}, &SysUIRoute{}, &SysUIActionBinding{}); err != nil {
		return fmt.Errorf("ui_permission 迁移: %w", err)
	}
	return nil
}
