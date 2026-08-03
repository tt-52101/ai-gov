import { redirect } from "next/navigation";

/**
 * 根入口 —— GOV 治理控制面为唯一入口。
 * 服务端重定向到 /gov/dashboard，避免旧 AdminConsole SPA 暴露。
 */
export default function LoginPage(): never {
  redirect("/gov/dashboard");
}
