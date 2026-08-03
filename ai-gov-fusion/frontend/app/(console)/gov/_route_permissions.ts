/**
 * 治理控制台路由权限声明（前端真实代码，非后端硬编码）。
 *
 * 扫描器 scripts/scan-frontend-routes.mjs 会静态读取本文件，
 * 将每个 /gov/* 路由映射到其所需的 ABAC 操作编码（action_code），
 * 回灌到后端 sys_ui_routes 表，供 UI 权限投影与前端路由守卫使用。
 *
 * 约定：
 * - route 必须与 app/(console)/gov/ 下真实存在的 page.tsx 路径一致
 * - action 必须是后端 sys_action_catalogs 中已注册的操作编码
 *   否则扫描器会拒绝回灌（防止恶意注入未授权操作）
 * - 新增治理页面时，必须在此同步声明其路由权限
 */

/** 单条路由权限声明 */
export interface RoutePermission {
  /** 前端路由路径，如 /gov/dashboard */
  route: string;
  /** 访问该路由所需的 ABAC 操作编码（对应 sys_action_catalogs.action_code） */
  action: string;
  /** 菜单编码，与 layout.tsx 中 allNavItems[*].code 对应（可选，用于回灌菜单关联） */
  menuCode?: string;
}

/** 治理控制台全部路由的权限声明 */
export const govRoutePermissions: RoutePermission[] = [
  { route: "/gov/dashboard", action: "data.ui.read", menuCode: "dashboard" },
  { route: "/gov/keys", action: "iam.key.read", menuCode: "keys" },
  { route: "/gov/key-vault", action: "iam.key.read", menuCode: "key_vault" },
  { route: "/gov/parties", action: "data.party.read", menuCode: "parties" },
  { route: "/gov/fund", action: "fund.balance.read", menuCode: "fund" },
  { route: "/gov/pricing", action: "routing.price.read", menuCode: "pricing" },
  { route: "/gov/routes", action: "routing.route_profile.read", menuCode: "routes" },
  { route: "/gov/model-permissions", action: "routing.model_grant.read", menuCode: "model_permissions" },
  { route: "/gov/abac", action: "iam.policy.read", menuCode: "abac" },
  { route: "/gov/security-reports", action: "data.report.read", menuCode: "security_reports" },
  { route: "/gov/tracing", action: "data.usage.read", menuCode: "tracing" },
  { route: "/gov/ui-permissions", action: "data.ui.read", menuCode: "ui_permissions" },
  { route: "/gov/audit", action: "data.audit.read", menuCode: "audit" },
];
