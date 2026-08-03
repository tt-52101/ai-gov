package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"tokenhub/backend/internal/server/abac"
	"tokenhub/backend/internal/server/ui_permission"

	"gorm.io/gorm"
)

// frontendManifestJSON 由 scripts/scan-frontend-routes.mjs 扫描前端真实路由后生成，
// 通过 go:embed 编译进二进制，避免运行时依赖外部文件路径。
//
// 该 manifest 的内容全部来自前端源码的静态声明（layout.tsx 的 allNavItems、
// _route_permissions.ts 的 govRoutePermissions、页面中的 data-permission 标记），
// 不包含任何后端硬编码的菜单/路由/按钮——从根上杜绝恶意注入越界权限。
//
//go:embed ui_permission/frontend_manifest.json
var frontendManifestJSON []byte

// frontendManifest 是扫描器输出的 UI 权限清单结构。
type frontendManifest struct {
	ScannedAt string `json:"scannedAt"`
	// Menus 菜单项（来自前端 layout.tsx 的 allNavItems）。
	Menus []struct {
		Code  string `json:"code"`
		Label string `json:"label"`
		Icon  string `json:"icon"`
		Href  string `json:"href"`
	} `json:"menus"`
	// Routes 路由项（来自前端 _route_permissions.ts + 真实 page.tsx 扫描）。
	Routes []struct {
		Path           string `json:"path"`
		MenuCode       string `json:"menuCode"`
		RequiredAction string `json:"requiredAction"`
	} `json:"routes"`
	// Buttons 按钮授权项（来自页面中的 data-permission / data-action 标记）。
	Buttons []struct {
		Code           string `json:"code"`
		Page           string `json:"page"`
		RequiredAction string `json:"requiredAction"`
	} `json:"buttons"`
	// Warnings 扫描器发现的异常（孤儿路由、悬挂权限声明）。
	Warnings struct {
		OrphanRoutes      []string `json:"orphanRoutes"`
		DanglingPerms     []any    `json:"danglingPermissions"`
	} `json:"warnings"`
}

// loadFrontendManifest 解析嵌入的前端权限 manifest。
func loadFrontendManifest() (*frontendManifest, error) {
	var m frontendManifest
	if err := json.Unmarshal(frontendManifestJSON, &m); err != nil {
		return nil, fmt.Errorf("解析前端 manifest 失败: %w", err)
	}
	return &m, nil
}

// seedGovUIMenusAndRoutes 将前端扫描得到的真实路由/菜单/按钮回灌到数据库。
//
// 该函数由 RunStartupBootstrap 在 SeedActionCatalogs 与 SeedAdminRoleAndPermissions
// 之后调用，确保权限投影所需的 action_code 已注册且管理员拥有全部权限。
//
// 防恶意注入（双重校验，与扫描器前端校验互补）：
//  1. 每个路由路径必须以 /gov/ 开头，且不含 ".." 或 "//" 越界片段。
//  2. required_action 必须已在 sys_action_catalogs 注册，否则拒绝回灌。
//  3. 菜单的 href 必须与某个已扫描到的真实路由一致。
//
// 写入三张表：sys_ui_menus / sys_ui_routes / sys_ui_action_bindings，全部幂等。
func seedGovUIMenusAndRoutes(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("seedGovUIMenusAndRoutes: 数据库未配置")
	}

	manifest, err := loadFrontendManifest()
	if err != nil {
		return err
	}

	// 加载 action_code -> id 映射，供路由/按钮绑定校验。
	var catalogs []abac.SysActionCatalog
	if err := db.WithContext(ctx).Find(&catalogs).Error; err != nil {
		return fmt.Errorf("seedGovUIMenusAndRoutes: 加载操作目录失败: %w", err)
	}
	actionIDByCode := make(map[string]string, len(catalogs))
	for _, c := range catalogs {
		actionIDByCode[c.ActionCode] = c.ID
	}

	// 收集已扫描到的真实路由集合，用于校验菜单 href 是否与真实页面一致。
	realRoutes := make(map[string]bool, len(manifest.Routes))
	for _, r := range manifest.Routes {
		realRoutes[r.Path] = true
	}

	createdMenus, createdRoutes, createdButtons := 0, 0, 0

	// 1) 回灌菜单（仅来自前端 layout.tsx 的 allNavItems）。
	for _, m := range manifest.Menus {
		// 防恶意注入：菜单 href 必须对应真实存在的 /gov/* 页面。
		if !realRoutes[m.Href] {
			slog.WarnContext(ctx, "菜单 href 无对应真实路由，跳过回灌（防止注入未挂载菜单）",
				"menu_code", m.Code, "href", m.Href)
			continue
		}

		var existing ui_permission.SysUIMenu
		err := db.WithContext(ctx).Where("menu_code = ?", m.Code).First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("seedGovUIMenusAndRoutes: 查询菜单 %s 失败: %w", m.Code, err)
		}
		if err == gorm.ErrRecordNotFound {
			if _, createErr := ui_permission.CreateMenu(db, ui_permission.CreateMenuRequest{
				MenuCode:  m.Code,
				Label:     m.Label,
				Icon:      m.Icon,
				SortOrder: menuSortOrder(m.Code, manifest.Menus),
			}); createErr != nil {
				return fmt.Errorf("seedGovUIMenusAndRoutes: 创建菜单 %s 失败: %w", m.Code, createErr)
			}
			createdMenus++
		}
	}

	// 2) 回灌路由（仅来自前端 _route_permissions.ts + 真实 page.tsx）。
	for _, r := range manifest.Routes {
		// 防恶意注入：路由路径必须位于 /gov/ 命名空间。
		if !strings.HasPrefix(r.Path, "/gov/") || strings.Contains(r.Path, "..") || strings.Contains(r.Path, "//") {
			slog.WarnContext(ctx, "拒绝回灌非法路由路径（防止注入越界路由）",
				"route", r.Path, "menu_code", r.MenuCode)
			continue
		}

		// 防恶意注入：required_action 必须已在 sys_action_catalogs 注册。
		actionID, ok := actionIDByCode[r.RequiredAction]
		if !ok {
			slog.WarnContext(ctx, "路由所需 action 未注册，跳过回灌（防止注入未授权操作）",
				"route", r.Path, "action", r.RequiredAction)
			continue
		}

		// 查找关联菜单 ID。
		var menu ui_permission.SysUIMenu
		menuErr := db.WithContext(ctx).Where("menu_code = ?", r.MenuCode).First(&menu).Error
		if menuErr != nil {
			if menuErr != gorm.ErrRecordNotFound {
				return fmt.Errorf("seedGovUIMenusAndRoutes: 查询菜单 %s 失败: %w", r.MenuCode, menuErr)
			}
			// 菜单未创建（可能上一步因校验被跳过），仅记录但不阻断路由写入。
			slog.WarnContext(ctx, "路由关联菜单不存在，路由无菜单归属",
				"route", r.Path, "menu_code", r.MenuCode)
		}

		var existing ui_permission.SysUIRoute
		err := db.WithContext(ctx).Where("route_path = ?", r.Path).First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("seedGovUIMenusAndRoutes: 查询路由 %s 失败: %w", r.Path, err)
		}
		if err == gorm.ErrRecordNotFound {
			menuID := menu.ID
			var menuIDPtr *int64
			if menuErr == nil {
				menuIDPtr = &menuID
			}
			if _, createErr := ui_permission.CreateRoute(db, ui_permission.CreateRouteRequest{
				RoutePath:        r.Path,
				MenuID:           menuIDPtr,
				RequiredActionID: &actionID,
			}); createErr != nil {
				return fmt.Errorf("seedGovUIMenusAndRoutes: 创建路由 %s 失败: %w", r.Path, createErr)
			}
			createdRoutes++
		}
	}

	// 3) 回灌按钮授权（仅来自页面中的 data-permission 标记）。
	for _, b := range manifest.Buttons {
		// 防恶意注入：按钮所在页面必须是真实路由。
		if !realRoutes[b.Page] {
			slog.WarnContext(ctx, "按钮所在页面无对应路由，跳过回灌",
				"button", b.Code, "page", b.Page)
			continue
		}

		// 防恶意注入：按钮所需 action 必须已注册（允许为空——公开按钮）。
		var actionIDPtr *string
		if b.RequiredAction != "" {
			id, ok := actionIDByCode[b.RequiredAction]
			if !ok {
				slog.WarnContext(ctx, "按钮所需 action 未注册，跳过回灌",
					"button", b.Code, "action", b.RequiredAction)
				continue
			}
			actionIDPtr = &id
		}

		var existing ui_permission.SysUIActionBinding
		err := db.WithContext(ctx).
			Where("button_code = ? AND page_route = ?", b.Code, b.Page).
			First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("seedGovUIMenusAndRoutes: 查询按钮 %s 失败: %w", b.Code, err)
		}
		if err == gorm.ErrRecordNotFound {
			if _, createErr := ui_permission.CreateActionBinding(db, ui_permission.CreateActionBindingRequest{
				ButtonCode:       b.Code,
				ButtonLabel:      b.Code,
				PageRoute:        b.Page,
				RequiredActionID: actionIDPtr,
			}); createErr != nil {
				return fmt.Errorf("seedGovUIMenusAndRoutes: 创建按钮 %s 失败: %w", b.Code, createErr)
			}
			createdButtons++
		}
	}

	slog.InfoContext(ctx, "治理控制台 UI 菜单/路由/按钮种子完成（前端扫描回灌）",
		"menus_created", createdMenus,
		"routes_created", createdRoutes,
		"buttons_created", createdButtons,
		"total_menus", len(manifest.Menus),
		"total_routes", len(manifest.Routes),
		"total_buttons", len(manifest.Buttons),
		"orphan_routes", len(manifest.Warnings.OrphanRoutes),
		"dangling_perms", len(manifest.Warnings.DanglingPerms),
	)
	return nil
}

// menuSortOrder 根据菜单在 manifest 中的声明顺序生成排序权重（每 10 一档）。
func menuSortOrder(code string, menus []struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
	Href  string `json:"href"`
}) int {
	for i, m := range menus {
		if m.Code == code {
			return (i + 1) * 10
		}
	}
	return 0
}
