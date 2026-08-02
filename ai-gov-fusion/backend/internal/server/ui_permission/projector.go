package ui_permission

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"gorm.io/gorm"
)

// ABACEngine 定义 ABAC 策略评估接口——ui_permission 只依赖接口，不依赖具体 ABAC 实现。
// Evaluate 返回 nil 表示权限通过，返回 error 表示拒绝（含拒绝原因）。
type ABACEngine interface {
	Evaluate(ctx context.Context, subject Subject, action string, resource Resource) error
}

// Projector 将 ABAC 权限决策投影到 UI 元素——菜单树、路由列表、按钮显隐。
// 每次请求时动态计算，不缓存（角色变更后下一次请求立即生效）。
type Projector struct {
	DB   *gorm.DB
	ABAC ABACEngine
}

// NewProjector 创建 UI 投影器。调用方负责注入已连接的数据库和 ABAC 引擎实例。
func NewProjector(db *gorm.DB, engine ABACEngine) *Projector {
	return &Projector{DB: db, ABAC: engine}
}

// ── 菜单投影 ──────────────────────────────────────────────────────────────

// ProjectMenus 根据主体 ABAC 权限投影可见菜单树。
//
// 投影规则（PRD §7.4.2）：
//  1. 叶子菜单：关联路由的 required_action 通过 ABAC 评估 → visible=true
//  2. 容器菜单（无路由）：至少一个后代菜单 visible=true → 自身 visible=true
//  3. 无权限的菜单不删除，仅设置 visible=false——前端据此隐藏
//  4. 孤立的不可见菜单（整棵子树均不可见）从树中过滤，不返回
//
// 返回已按 sort_order 排序的菜单树根节点列表。
func (p *Projector) ProjectMenus(ctx context.Context, subject Subject) ([]*SysUIMenu, error) {
	menus, err := ListMenus(p.DB)
	if err != nil {
		return nil, fmt.Errorf("ui_permission: 加载菜单失败: %w", err)
	}
	if len(menus) == 0 {
		return nil, nil
	}

	routes, err := ListRoutes(p.DB)
	if err != nil {
		return nil, fmt.Errorf("ui_permission: 加载路由失败: %w", err)
	}

	// 构建 menu_id → 路由列表 映射
	menuRoutes := make(map[int64][]*SysUIRoute)
	for _, r := range routes {
		if r.MenuID != nil {
			menuRoutes[*r.MenuID] = append(menuRoutes[*r.MenuID], r)
		}
	}

	// 收集所有 required_action_id，批量查询 action_code
	actionCodes, err := loadActionCodes(p.DB, routes)
	if err != nil {
		return nil, err
	}

	// 构建 menu_id → 菜单 映射（用于后续树组装）
	menuByID := make(map[int64]*SysUIMenu, len(menus))
	for _, m := range menus {
		menuByID[m.ID] = m
	}

	// 第一遍：对各菜单评估其路由权限
	for _, m := range menus {
		routeList := menuRoutes[m.ID]
		if len(routeList) == 0 {
			// 容器菜单——待第二遍自底向上传播
			m.Visible = false
			continue
		}
		m.Visible = p.evaluateAnyRoute(ctx, subject, routeList, actionCodes)
	}

	// 第二遍：自底向上传播可见性——构建 parent→children 映射
	childrenMap := make(map[int64][]*SysUIMenu)
	var roots []*SysUIMenu
	for _, m := range menus {
		if m.ParentID != nil {
			childrenMap[*m.ParentID] = append(childrenMap[*m.ParentID], m)
		} else {
			roots = append(roots, m)
		}
	}

	// 递归传播可见性并组装树
	result := make([]*SysUIMenu, 0, len(roots))
	for _, root := range roots {
		p.propagateVisibility(root, childrenMap)
		if root.Visible {
			result = append(result, root)
		}
	}

	// 按 sort_order 排序根节点
	sort.Slice(result, func(i, j int) bool {
		return result[i].SortOrder < result[j].SortOrder
	})

	slog.DebugContext(ctx, "菜单投影完成",
		"subject_type", subject.Type,
		"subject_id", subject.ID,
		"total_menus", len(menus),
		"visible_roots", len(result),
	)
	return result, nil
}

// evaluateAnyRoute 对菜单关联的路由列表逐一评估 ABAC——任一通过即返回 true。
// 无 required_action_id 的路由视为公开路由，直接通过。
// 若 ABAC 引擎未配置，则所有路由视为可见（开发模式）。
func (p *Projector) evaluateAnyRoute(ctx context.Context, subject Subject, routes []*SysUIRoute, actionCodes map[string]string) bool {
	// 无 ABAC 引擎 → 开发模式，全部可见。
	if p.ABAC == nil {
		return true
	}
	for _, r := range routes {
		if r.RequiredActionID == nil {
			return true
		}
		actionCode, ok := actionCodes[*r.RequiredActionID]
		if !ok {
			continue
		}
		resource := Resource{Type: "route", ID: r.RoutePath}
		if err := p.ABAC.Evaluate(ctx, subject, actionCode, resource); err == nil {
			return true
		}
	}
	return false
}

// propagateVisibility 递归传播可见性并组装子节点。
// 规则：父菜单 visible = 自身 visible || 至少一个子菜单 visible。
// 不可见的子菜单从 Children 中过滤。
func (p *Projector) propagateVisibility(menu *SysUIMenu, childrenMap map[int64][]*SysUIMenu) {
	children := childrenMap[menu.ID]
	visibleChildren := make([]*SysUIMenu, 0, len(children))
	for _, child := range children {
		p.propagateVisibility(child, childrenMap)
		if child.Visible {
			visibleChildren = append(visibleChildren, child)
			menu.Visible = true
		}
	}
	sort.Slice(visibleChildren, func(i, j int) bool {
		return visibleChildren[i].SortOrder < visibleChildren[j].SortOrder
	})
	menu.Children = visibleChildren
}

// ── 路由投影 ──────────────────────────────────────────────────────────────

// ProjectRoutes 根据主体 ABAC 权限投影可访问路由列表。
// 仅返回 required_action 通过 ABAC 评估的路由；无 required_action 的路由视为公开。
// 若 ABAC 引擎未配置，返回所有路由（开发模式）。
func (p *Projector) ProjectRoutes(ctx context.Context, subject Subject) ([]*SysUIRoute, error) {
	routes, err := ListRoutes(p.DB)
	if err != nil {
		return nil, fmt.Errorf("ui_permission: 加载路由失败: %w", err)
	}

	// 无 ABAC 引擎 → 开发模式，返回所有路由。
	if p.ABAC == nil {
		slog.DebugContext(ctx, "路由投影完成（无ABAC，开发模式）",
			"subject_type", subject.Type,
			"subject_id", subject.ID,
			"total_routes", len(routes),
		)
		return routes, nil
	}

	actionCodes, err := loadActionCodes(p.DB, routes)
	if err != nil {
		return nil, err
	}

	accessible := make([]*SysUIRoute, 0, len(routes))
	for _, r := range routes {
		if r.RequiredActionID == nil {
			accessible = append(accessible, r)
			continue
		}
		actionCode, ok := actionCodes[*r.RequiredActionID]
		if !ok {
			continue
		}
		resource := Resource{Type: "route", ID: r.RoutePath}
		if err := p.ABAC.Evaluate(ctx, subject, actionCode, resource); err == nil {
			accessible = append(accessible, r)
		}
	}

	slog.DebugContext(ctx, "路由投影完成",
		"subject_type", subject.Type,
		"subject_id", subject.ID,
		"total_routes", len(routes),
		"accessible", len(accessible),
	)
	return accessible, nil
}

// ── 按钮投影 ──────────────────────────────────────────────────────────────

// ProjectActions 根据主体 ABAC 权限投影指定页面的按钮显隐状态。
// 返回 map[button_code]bool：true 表示按钮可显示，false 表示隐藏。
// 无 required_action 的按钮视为公开按钮，始终为 true。
// 若 ABAC 引擎未配置，所有按钮显示（开发模式）。
func (p *Projector) ProjectActions(ctx context.Context, subject Subject, pageRoute string) (map[string]bool, error) {
	bindings, err := ListActionBindingsByPage(p.DB, pageRoute)
	if err != nil {
		return nil, fmt.Errorf("ui_permission: 加载按钮绑定失败: %w", err)
	}

	// 无 ABAC 引擎 → 开发模式，所有按钮可见。
	if p.ABAC == nil {
		result := make(map[string]bool, len(bindings))
		for _, b := range bindings {
			result[b.ButtonCode] = true
		}
		return result, nil
	}

	// 收集 action_id → 查询 action_code
	actionCodes, err := loadBindingActionCodes(p.DB, bindings)
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool, len(bindings))
	for _, b := range bindings {
		if b.RequiredActionID == nil {
			result[b.ButtonCode] = true
			continue
		}
		actionCode, ok := actionCodes[*b.RequiredActionID]
		if !ok {
			result[b.ButtonCode] = false
			continue
		}
		resource := Resource{Type: "page", ID: pageRoute}
		if err := p.ABAC.Evaluate(ctx, subject, actionCode, resource); err == nil {
			result[b.ButtonCode] = true
		} else {
			result[b.ButtonCode] = false
		}
	}

	slog.DebugContext(ctx, "按钮投影完成",
		"subject_type", subject.Type,
		"subject_id", subject.ID,
		"page_route", pageRoute,
		"total_buttons", len(bindings),
	)
	return result, nil
}

// ── 辅助：批量加载 action_code ─────────────────────────────────────────────

// loadActionCodes 从路由列表中提取所有 required_action_id，批量查询对应的 action_code。
// 返回 action_id → action_code 的映射。
func loadActionCodes(db *gorm.DB, routes []*SysUIRoute) (map[string]string, error) {
	ids := collectActionIDs(routes)
	if len(ids) == 0 {
		return nil, nil
	}
	var catalogs []sysActionCatalog
	if err := db.Where("id IN ?", ids).Find(&catalogs).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 查询操作目录失败: %w", err)
	}
	m := make(map[string]string, len(catalogs))
	for _, c := range catalogs {
		m[c.ID] = c.ActionCode
	}
	return m, nil
}

// loadBindingActionCodes 从按钮绑定列表中提取所有 required_action_id，批量查询 action_code。
func loadBindingActionCodes(db *gorm.DB, bindings []*SysUIActionBinding) (map[string]string, error) {
	ids := collectBindingActionIDs(bindings)
	if len(ids) == 0 {
		return nil, nil
	}
	var catalogs []sysActionCatalog
	if err := db.Where("id IN ?", ids).Find(&catalogs).Error; err != nil {
		return nil, fmt.Errorf("ui_permission: 查询操作目录失败: %w", err)
	}
	m := make(map[string]string, len(catalogs))
	for _, c := range catalogs {
		m[c.ID] = c.ActionCode
	}
	return m, nil
}

func collectActionIDs(routes []*SysUIRoute) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, r := range routes {
		if r.RequiredActionID != nil && !seen[*r.RequiredActionID] {
			seen[*r.RequiredActionID] = true
			ids = append(ids, *r.RequiredActionID)
		}
	}
	return ids
}

func collectBindingActionIDs(bindings []*SysUIActionBinding) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, b := range bindings {
		if b.RequiredActionID != nil && !seen[*b.RequiredActionID] {
			seen[*b.RequiredActionID] = true
			ids = append(ids, *b.RequiredActionID)
		}
	}
	return ids
}
