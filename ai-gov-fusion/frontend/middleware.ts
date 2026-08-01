import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/**
 * 前端全量路由守卫 middleware —— 覆盖 gov 8大模块 + 旧系统 30+ 页面。
 *
 * 认证策略：
 *   1. 检查 cookie 中的 gov_session 或 tokenhub_session token。
 *   2. 若未认证，重定向到根路径 /（TokenHub 管理台登录页）。
 *   3. Next.js 内部资源（_next/static）和静态文件放行。
 *   4. API 路由放行（由后端 ABAC 策略引擎二次鉴权）。
 */

/** 所有需要认证保护的管理台路由前缀 */
const PROTECTED_PREFIXES = [
  "/gov",
  "/overview", "/models", "/providers", "/api-keys",
  "/users", "/usage", "/billing", "/settings",
  "/security-policies", "/alerts", "/reports", "/approvals",
  "/projects", "/gateway", "/audit-log", "/monitors",
  "/proxies", "/quota-policies", "/playground",
  "/notification-channels", "/identity-providers",
  "/cost-centers", "/chargebacks", "/invoices",
  "/database-status", "/project-members",
  "/alert-events", "/alert-deliveries", "/announcements",
  "/sqlite-backups", "/budgets",
];

/** 检测路径是否需要认证保护 */
function isProtected(pathname: string): boolean {
  for (const prefix of PROTECTED_PREFIXES) {
    if (pathname.startsWith(prefix)) return true;
  }
  return false;
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // 只处理需要保护的管理台路由
  if (!isProtected(pathname)) {
    return NextResponse.next();
  }

  // 排除 Next.js 内部资源和静态文件
  if (pathname.startsWith("/_next") || pathname.includes(".")) {
    return NextResponse.next();
  }

  // 检查认证 cookie
  const sessionCookie =
    request.cookies.get("gov_session")?.value ||
    request.cookies.get("tokenhub_session")?.value;

  if (!sessionCookie) {
    const loginUrl = new URL("/", request.url);
    loginUrl.searchParams.set("returnUrl", pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

/**
 * 路由匹配配置 —— 覆盖全部管理台路由。
 * 后端 API 通过 ABAC 策略引擎在 gov_handlers.go 中执行二次鉴权。
 */
export const config = {
  matcher: [
    "/gov/:path*",
    "/overview/:path*", "/models/:path*", "/providers/:path*", "/api-keys/:path*",
    "/users/:path*", "/usage/:path*", "/billing/:path*", "/settings/:path*",
    "/security-policies/:path*", "/alerts/:path*", "/reports/:path*", "/approvals/:path*",
    "/projects/:path*", "/gateway/:path*", "/audit-log/:path*", "/monitors/:path*",
    "/proxies/:path*", "/quota-policies/:path*", "/playground/:path*",
    "/notification-channels/:path*", "/identity-providers/:path*",
    "/cost-centers/:path*", "/chargebacks/:path*", "/invoices/:path*",
    "/database-status/:path*", "/project-members/:path*",
    "/alert-events/:path*", "/alert-deliveries/:path*", "/announcements/:path*",
    "/sqlite-backups/:path*", "/budgets/:path*",
  ],
};
