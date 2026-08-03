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

/** 唯一需要认证保护的入口前缀 —— GOV 治理控制面 */
const PROTECTED_PREFIXES = ["/gov"];

/** GOV 内不需要认证的公开路径（登录/落地页）—— 防止重定向死循环 */
const PUBLIC_GOV_PATHS = new Set<string>(["/gov/dashboard"]);

/** 旧 TokenHub 路径前缀 —— 不再做认证拦截，直接落到对应 page.tsx 由服务端 redirect 跳转到 /gov/* */
const LEGACY_REDIRECT_PREFIXES = [
  "/overview", "/models", "/providers", "/api-keys",
  "/users", "/usage", "/billing", "/settings",
  "/security-policies", "/alerts", "/reports", "/approvals",
  "/projects", "/teams",
  "/gateway", "/audit", "/audit-log", "/monitors",
  "/proxies", "/quota-policies", "/playground",
  "/notification-channels", "/identity-providers",
  "/cost-centers", "/chargebacks", "/invoices",
  "/database-status", "/project-members",
  "/alert-events", "/alert-deliveries", "/announcements",
  "/sqlite-backups", "/budgets", "/approval-flows",
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

  // 旧 TokenHub 路径：放行，让对应 page.tsx 的服务端 redirect 接管
  // 这样 /projects → /gov/parties（精准重定向）才能生效
  for (const prefix of LEGACY_REDIRECT_PREFIXES) {
    if (pathname === prefix || pathname.startsWith(prefix + "/")) {
      return NextResponse.next();
    }
  }

  // 只处理 /gov/* 这一个需要保护的管理台路由族
  if (!isProtected(pathname)) {
    return NextResponse.next();
  }

  // 排除 Next.js 内部资源和静态文件
  if (pathname.startsWith("/_next") || pathname.includes(".")) {
    return NextResponse.next();
  }

  // 公开 GOV 路径（如 /gov/dashboard 落地页）：允许未认证访问，由页面自身处理登录引导
  if (PUBLIC_GOV_PATHS.has(pathname)) {
    return NextResponse.next();
  }

  // 开发/联调模式放行：前端 API 调用已由 gov-api.ts 通过 Authorization: Bearer
  // 注入管理 Token，后端 ABAC 策略引擎对所有 /v1/gov/* 接口做二次鉴权兜底。
  // 此时前端路由守卫若仍强制检查 session cookie（前端从未写入），会把所有导航
  // 弹回 dashboard，导致治理控制台无法跳转。故 dev 环境直接放行全部 /gov/*，
  // 让路由真正挂载到真实页面。生产环境（NODE_ENV=production）保持严格 cookie 认证。
  if (process.env.NODE_ENV === "development") {
    return NextResponse.next();
  }

  // 检查认证 cookie
  const sessionCookie =
    request.cookies.get("gov_session")?.value ||
    request.cookies.get("tokenhub_session")?.value;

  if (!sessionCookie) {
    // 未登录直接落仪表盘（仪表盘自身处理登录引导）
    const loginUrl = new URL("/gov/dashboard", request.url);
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
    "/projects/:path*", "/teams/:path*", "/gateway/:path*", "/audit/:path*", "/audit-log/:path*", "/monitors/:path*",
    "/proxies/:path*", "/quota-policies/:path*", "/playground/:path*",
    "/notification-channels/:path*", "/identity-providers/:path*",
    "/cost-centers/:path*", "/chargebacks/:path*", "/invoices/:path*",
    "/database-status/:path*", "/project-members/:path*",
    "/alert-events/:path*", "/alert-deliveries/:path*", "/announcements/:path*",
    "/sqlite-backups/:path*", "/budgets/:path*", "/approval-flows/:path*",
  ],
};
