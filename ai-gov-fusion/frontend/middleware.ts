import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/**
 * 前端路由守卫 middleware —— 对 /gov/* 路径检查认证状态。
 *
 * 认证策略：
 *   1. 检查 cookie 中的 gov_session token（后端登录后设置）。
 *   2. 若未认证，重定向到根路径 /（TokenHub 管理台登录页）。
 *   3. /gov 内嵌页面（如 _next/static）和 API 路由放行。
 *
 * 注意：本 middleware 为前端层面的路由守卫。后端 API 层面通过 ABAC 策略引擎
 * 在 gov_handlers.go 中的 requireGovAuth/requireGovItemAuth 执行二次鉴权。
 */
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // 仅对 /gov 治理控制台路径执行认证检查。
  if (!pathname.startsWith("/gov")) {
    return NextResponse.next();
  }

  // 排除 Next.js 内部资源和 API 路由（由后端 ABAC 保护）。
  if (
    pathname.startsWith("/gov/_next") ||
    pathname.startsWith("/gov/api") ||
    pathname.includes(".")
  ) {
    return NextResponse.next();
  }

  // 检查认证 cookie —— gov_session 由登录接口在响应中设置。
  // 如果后端使用 Bearer token 而不是 cookie，则此检查为软守卫，
  // GovLayout 的 fetch /v1/gov/ui-permissions/snapshot 会执行实际鉴权。
  const sessionCookie =
    request.cookies.get("gov_session")?.value ||
    request.cookies.get("tokenhub_session")?.value;

  if (!sessionCookie) {
    // 未登录——重定向到根路径（TokenHub 管理台登录页）。
    // 登录成功后可通过 returnUrl 参数跳回原路径。
    const loginUrl = new URL("/", request.url);
    loginUrl.searchParams.set("returnUrl", pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

/**
 * 路由匹配配置 —— 仅对 /gov 路径及其子路径执行 middleware。
 * 其他管理台路由（如 /overview、/settings 等）由 AdminConsole SPA 自行保护。
 */
export const config = {
  matcher: ["/gov/:path*"],
};
