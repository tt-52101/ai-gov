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
  const [abacUnavailable, setAbacUnavailable] = useState(false);

  // /gov/* 路由使用独立的治理控制台布局（GovLayout，自带 ABAC）
  const isGovRoute = pathname.startsWith("/gov");

  // 非 gov 路由：调用 ABAC 权限投影检查当前页面是否可见
  useEffect(() => {
    if (isGovRoute) {
      setAbacReady(true);
      return;
    }

    let cancelled = false;
    const MAX_RETRIES = 2;

    /** 带重试的 fetch，失败后自动重试 */
    async function fetchWithRetry(
      url: string,
      attempts: number,
    ): Promise<Response> {
      let lastErr: unknown;
      for (let i = 0; i <= attempts; i++) {
        try {
          const res = await fetch(url);
          if (res.ok) return res;
          // 非 200 但不抛异常（如 500），同样视为失败
          lastErr = new Error(`权限检查失败 (HTTP ${res.status})`);
        } catch (e) {
          lastErr = e;
        }
        if (i < attempts && !cancelled) {
          // 指数退避：第 1 次重试等 500ms，第 2 次等 1000ms
          await new Promise((r) => setTimeout(r, 500 * (i + 1)));
        }
      }
      throw lastErr ?? new Error("权限检查不可用");
    }

    fetchWithRetry("/v1/gov/ui-permissions/snapshot", MAX_RETRIES)
      .then((res) => res.json())
      .then((data: { routes?: Array<{ route_path: string }> }) => {
        if (cancelled) return;
        // 检查当前路径是否在 ABAC 路由白名单中
        const allowed = data.routes || [];
        const isAllowed = allowed.some(
          (r) => r.route_path === pathname || pathname.startsWith(r.route_path + "/"),
        );
        if (!isAllowed && allowed.length > 0) {
          setAbacDenied(true);
        }
        setAbacReady(true);
      })
      .catch(() => {
        // FAIL-CLOSED: 权限 API 不可用时，仅允许访问 /gov/dashboard
        // 其他页面提示权限服务不可用，避免静默放行所有页面
        if (cancelled) return;
        if (pathname === "/gov/dashboard") {
          setAbacReady(true);
        } else {
          setAbacUnavailable(true);
        }
      });

    return () => { cancelled = true; };
  }, [pathname, isGovRoute]);

  // 加载中
  if (!abacReady && !abacUnavailable) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full" />
        <span className="ml-3 text-gray-500">权限校验中...</span>
      </div>
    );
  }

  // 权限服务不可用 —— FAIL-CLOSED：仅 dashboard 可访问
  if (abacUnavailable) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen gap-4">
        <div className="text-center">
          <h2 className="text-xl font-semibold text-red-600 mb-2">
            权限服务暂不可用
          </h2>
          <p className="text-gray-500 max-w-md">
            权限校验服务当前无法响应，已自动阻断非白名单页面访问。
            请稍后刷新重试，或联系系统管理员。
          </p>
        </div>
        <button
          onClick={() => {
            if (typeof window !== "undefined") {
              window.location.href = "/gov/dashboard";
            }
          }}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
        >
          前往仪表盘
        </button>
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
