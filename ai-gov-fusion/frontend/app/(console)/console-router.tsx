"use client";

import { usePathname } from "next/navigation";
import { AdminConsole } from "@/features/admin/shell/admin-console";

/**
 * ConsoleRouter —— 根据当前路径决定渲染哪个布局。
 *
 *   - /gov/* 路径：渲染 children（由 GovLayout 包裹的治理控制台页面）
 *   - 其他路径：渲染 TokenHub 原有的 AdminConsole SPA 布局
 *
 * 此组件为 Client Component —— 使用 usePathname() 做客户端路由判断。
 * defaultBaseURL 由 Server Component 的 ConsoleLayout 注入，避免客户端
 * 直接导入 server-only 的 runtime-config 模块。
 */
export function ConsoleRouter({
  defaultBaseURL,
  children,
}: {
  defaultBaseURL: string;
  children: React.ReactNode;
}) {
  const pathname = usePathname();

  // /gov/* 路由使用独立的治理控制台布局（GovLayout）
  if (pathname.startsWith("/gov")) {
    return <>{children}</>;
  }

  // 其他路径回退到 TokenHub 原有的 AdminConsole SPA 布局
  return <AdminConsole defaultBaseURL={defaultBaseURL} />;
}
