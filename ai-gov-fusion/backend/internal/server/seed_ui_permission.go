package server

import (
	"context"
	"fmt"
	"log/slog"

	"tokenhub/backend/internal/server/abac"
	"tokenhub/backend/internal/server/ui_permission"

	"gorm.io/gorm"
)

// govMenuSeedItem 描述治理控制台侧边栏需要种子化的一个菜单项。
type govMenuSeedItem struct {
	// MenuCode 菜单编码，与前端 allNavItems[*].code 对应。
	MenuCode string
	// Label 菜单显示文本。
	Label string
	// Icon 菜单图标标识（可选）。
	Icon string
	// SortOrder 同级排序权重。
	SortOrder int
	// RoutePath 关联前端路由路径，如 /gov/dashboard。
	RoutePath string
	// ActionCode 路由所需的 ABAC 操作编码；空字符串表示公开路由（无需鉴权）。
	ActionCode string
}

// govMenuSeedItems 定义治理控制台全部 13 个功能模块的菜单种子。
// 顺序、编码与前端 app/(console)/gov/layout.tsx 中的 allNavItems 保持一致。
var govMenuSeedItems = []govMenuSeedItem{
	{MenuCode: "dashboard", Label: "仪表盘", Icon: "LayoutDashboard", SortOrder: 10, RoutePath: "/gov/dashboard", ActionCode: "data.ui.read"},
	{MenuCode: "keys", Label: "Key 管理", Icon: "Key", SortOrder: 20, RoutePath: "/gov/keys", ActionCode: "iam.key.read"},
	{MenuCode: "key_vault", Label: "密钥仓库", Icon: "Lock", SortOrder: 30, RoutePath: "/gov/key-vault", ActionCode: "iam.key.read"},
	{MenuCode: "parties", Label: "Party 管理", Icon: "Building2", SortOrder: 40, RoutePath: "/gov/parties", ActionCode: "data.party.read"},
	{MenuCode: "fund", Label: "资金操作", Icon: "Wallet", SortOrder: 50, RoutePath: "/gov/fund", ActionCode: "fund.balance.read"},
	{MenuCode: "pricing", Label: "价目维护", Icon: "Tags", SortOrder: 60, RoutePath: "/gov/pricing", ActionCode: "routing.price.read"},
	{MenuCode: "routes", Label: "路由档案", Icon: "GitBranch", SortOrder: 70, RoutePath: "/gov/routes", ActionCode: "routing.route_profile.read"},
	{MenuCode: "model_permissions", Label: "模型权限", Icon: "Brain", SortOrder: 80, RoutePath: "/gov/model-permissions", ActionCode: "routing.model_grant.read"},
	{MenuCode: "abac", Label: "ABAC 策略", Icon: "Shield", SortOrder: 90, RoutePath: "/gov/abac", ActionCode: "iam.policy.read"},
	{MenuCode: "security_reports", Label: "安全报表", Icon: "AlertTriangle", SortOrder: 100, RoutePath: "/gov/security-reports", ActionCode: "data.report.read"},
	{MenuCode: "tracing", Label: "调用追踪", Icon: "Search", SortOrder: 110, RoutePath: "/gov/tracing", ActionCode: "data.usage.read"},
	{MenuCode: "ui_permissions", Label: "UI 权限", Icon: "Eye", SortOrder: 120, RoutePath: "/gov/ui-permissions", ActionCode: "data.ui.read"},
	{MenuCode: "audit", Label: "审计日志", Icon: "ClipboardList", SortOrder: 130, RoutePath: "/gov/audit", ActionCode: "data.audit.read"},
}

// seedGovUIMenusAndRoutes 种子化治理控制台菜单与路由数据。
//
// 幂等：按 menu_code 和 route_path 去重，已存在则跳过。该函数依赖
// SeedActionCatalogs 与 SeedAdminRoleAndPermissions 已执行，确保路由所需
// action_code 已注册且超级管理员拥有全部权限。
//
// 写入两张表：
//   - sys_ui_menus：菜单树节点
//   - sys_ui_routes：前端路由到菜单和 ABAC 操作的映射
func seedGovUIMenusAndRoutes(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("seedGovUIMenusAndRoutes: 数据库未配置")
	}

	// 加载 action_code -> id 映射，供路由绑定使用。
	var catalogs []abac.SysActionCatalog
	if err := db.WithContext(ctx).Find(&catalogs).Error; err != nil {
		return fmt.Errorf("seedGovUIMenusAndRoutes: 加载操作目录失败: %w", err)
	}
	actionIDByCode := make(map[string]string, len(catalogs))
	for _, c := range catalogs {
		actionIDByCode[c.ActionCode] = c.ID
	}

	createdMenus := 0
	createdRoutes := 0

	for _, item := range govMenuSeedItems {
		// 1) 幂等创建菜单。
		var menu ui_permission.SysUIMenu
		err := db.WithContext(ctx).
			Where("menu_code = ?", item.MenuCode).
			First(&menu).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("seedGovUIMenusAndRoutes: 查询菜单 %s 失败: %w", item.MenuCode, err)
		}

		if err == gorm.ErrRecordNotFound {
			m, createErr := ui_permission.CreateMenu(db, ui_permission.CreateMenuRequest{
				MenuCode:  item.MenuCode,
				Label:     item.Label,
				Icon:      item.Icon,
				SortOrder: item.SortOrder,
			})
			if createErr != nil {
				return fmt.Errorf("seedGovUIMenusAndRoutes: 创建菜单 %s 失败: %w", item.MenuCode, createErr)
			}
			menu = *m
			createdMenus++
		}

		// 2) 幂等创建路由并绑定菜单与操作。
		var route ui_permission.SysUIRoute
		err = db.WithContext(ctx).
			Where("route_path = ?", item.RoutePath).
			First(&route).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("seedGovUIMenusAndRoutes: 查询路由 %s 失败: %w", item.RoutePath, err)
		}

		if err == gorm.ErrRecordNotFound {
			var requiredActionID *string
			if item.ActionCode != "" {
				id, ok := actionIDByCode[item.ActionCode]
				if !ok {
					slog.WarnContext(ctx, "治理菜单路由所需操作编码未注册，跳过权限绑定",
						"menu_code", item.MenuCode,
						"route_path", item.RoutePath,
						"action_code", item.ActionCode,
					)
				} else {
					requiredActionID = &id
				}
			}

			menuID := menu.ID
			_, createErr := ui_permission.CreateRoute(db, ui_permission.CreateRouteRequest{
				RoutePath:        item.RoutePath,
				MenuID:           &menuID,
				RequiredActionID: requiredActionID,
			})
			if createErr != nil {
				return fmt.Errorf("seedGovUIMenusAndRoutes: 创建路由 %s 失败: %w", item.RoutePath, createErr)
			}
			createdRoutes++
		}
	}

	slog.InfoContext(ctx, "治理控制台 UI 菜单与路由种子完成",
		"menus_created", createdMenus,
		"routes_created", createdRoutes,
		"total_menus", len(govMenuSeedItems),
	)
	return nil
}
