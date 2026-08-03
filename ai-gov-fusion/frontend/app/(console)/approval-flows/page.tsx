import { redirect } from "next/navigation";

/** 旧 TokenHub 路径 —— 商业 GA 阶段重定向至 GOV 治理控制面。 */
export default function ApprovalFlowsPage(): never {
  redirect("/gov/fund");
}
