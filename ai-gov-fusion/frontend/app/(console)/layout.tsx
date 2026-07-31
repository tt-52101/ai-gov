import { ConsoleRouter } from "./console-router";
import { runtimeAPIBaseURL } from "@/lib/runtime-config";

export const dynamic = "force-dynamic";

/**
 * ConsoleLayout —— (console) 路由组通用布局（Server Component）。
 *
 * 将 runtimeAPIBaseURL 作为 prop 传递给客户端 ConsoleRouter 组件，
 * 由 ConsoleRouter 根据当前路径决定渲染 AdminConsole SPA 还是 GovLayout。
 */
export default function ConsoleLayout({ children }: { children: React.ReactNode }) {
  return <ConsoleRouter defaultBaseURL={runtimeAPIBaseURL()}>{children}</ConsoleRouter>;
}
