"use client";

import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { AdminConsole } from "@/features/admin/shell/admin-console";

/**
 * ConsoleRouter —— 根据当前路径决定渲染哪个布局，并注入 ABAC 权限检查。
 *
 *   - /gov/* 路径：渲染 children（由 GovLayout 包裹，自带 ABAC 动态菜单）
 *   - 其他路径：渲染 TokenHub 原有 AdminConsole SPA
 *     → 注入 ABAC 权限数据（菜单可见性 + 路由白名单）
 *     → 若当前路由在 ABAC 投影中不可见，重定向到 /gov/dashboard
 *
 * 此组件为 Client Component —— 使用 usePathname() 做客户端路由判断。
 */
export function ConsoleRouter({
  defaultBaseURL,
  children,
}: {
  defaultBaseURL: string;
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const [abacReady, setAbacReady] = useState(false);
  const [abacDenied, setAbacDenied] = useState(false);

  // /gov/* 路由使用独立的治理控制台布局（GovLayout，自带 ABAC）
  const isGovRoute = pathname.startsWith("/gov");

  // 非 gov 路由：调用 ABAC 权限投影检查当前页面是否可见
  useEffect(() => {
    if (isGovRoute) {
      setAbacReady(true);
      return;
    }

    let cancelled = false;
    fetch("/v1/gov/ui-permissions/snapshot")
      .then((res) => {
        if (!res.ok) throw new Error("权限检查失败");
        return res.json();
      })
      .then((data: { routes?: Array<{ route_path: string }> }) => {
        if (cancelled) return;
        // 检查当前路径是否在 ABAC 路由白名单中
        const allowed = data.routes || [];
        const isAllowed = allowed.some(
          (r) => r.route_path === pathname || pathname.startsWith(r.route_path + "/")
        );
        if (!isAllowed && allowed.length > 0) {
          setAbacDenied(true);
        }
        setAbacReady(true);
      })
      .catch(() => {
        // API 不可用时回退——允许访问（避免锁死）
        if (!cancelled) setAbacReady(true);
      });

    return () => { cancelled = true; };
  }, [pathname, isGovRoute]);

  // 加载中
  if (!abacReady) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full" />
        <span className="ml-3 text-gray-500">权限校验中...</span>
      </div>
    );
  }

  // ABAC 拒绝 —— 重定向到仪表盘
  if (abacDenied) {
    if (typeof window !== "undefined") {
      window.location.href = "/gov/dashboard";
    }
    return (
      <div className="flex items-center justify-center min-h-screen">
        <p className="text-red-500">权限不足，正在跳转到仪表盘...</p>
      </div>
    );
  }

  // gov 路由 → GovLayout children
  if (isGovRoute) {
    return <>{children}</>;
  }

  // 旧系统路由 → AdminConsole SPA（已通过 ABAC 路由检查）
  return <AdminConsole defaultBaseURL={defaultBaseURL} />;
}
