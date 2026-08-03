"use client";

/**
 * ConsoleRouter —— 简化后的路由节点。
 *
 * 旧版本根据路径在 GOV 治理控制面 与 TokenHub 旧 AdminConsole SPA 之间分流。
 * 自 A-02 起，AdminConsole SPA 入口已废弃，根入口 (app/page.tsx) 已直接重定向至 /gov/*。
 *
 * 此组件保留为 GOV 子树的轻量客户端包装节点：
 *   - 旧路径（已通过 page.tsx redirect 跳转）不会到达此处
 *   - /gov/* 子树直接渲染 children，由 gov/layout.tsx 提供 ABAC 投影
 *
 * 不再调用 AdminConsole 组件，不再执行 ABAC 路由白名单检查，
 * 不再执行 FAIL-CLOSED 阻断逻辑——所有权限检查由 GOV 内部 layout 统一负责。
 */
export function ConsoleRouter({
  children,
}: {
  defaultBaseURL: string;
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
