import { ConsoleRouter } from "./console-router";
import { runtimeAPIBaseURL } from "@/lib/runtime-config";

export const dynamic = "force-dynamic";

/**
 * ConsoleLayout —— (console) 路由组通用布局（Server Component）。
 *
 * GOV 治理控制面为唯一入口，AdminConsole 旧 SPA 已废弃；
 * ConsoleRouter 仅作为可观测的客户端节点保留以备审计追踪。
 */
export default function ConsoleLayout({ children }: { children: React.ReactNode }) {
  return <ConsoleRouter defaultBaseURL={runtimeAPIBaseURL()}>{children}</ConsoleRouter>;
}
