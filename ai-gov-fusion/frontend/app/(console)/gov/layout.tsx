"use client";

import React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Building2,
  Wallet,
  Tags,
  GitBranch,
  Shield,
  Eye,
  ClipboardList,
  LayoutDashboard,
  ChevronRight,
} from "lucide-react";

/** 导航菜单项定义 */
interface NavItem {
  /** 导航路径 */
  href: string;
  /** 显示标签 */
  label: string;
  /** lucide-react 图标 */
  icon: React.ComponentType<{ className?: string }>;
}

/**
 * AI 治理网关管理控制台导航菜单配置。
 * 包含全部 8 个功能模块的入口。
 */
const navItems: NavItem[] = [
  { href: "/gov/dashboard", label: "仪表盘", icon: LayoutDashboard },
  { href: "/gov/parties", label: "Party 管理", icon: Building2 },
  { href: "/gov/fund", label: "资金操作", icon: Wallet },
  { href: "/gov/pricing", label: "价目维护", icon: Tags },
  { href: "/gov/routes", label: "路由档案", icon: GitBranch },
  { href: "/gov/abac", label: "ABAC 策略", icon: Shield },
  { href: "/gov/ui-permissions", label: "UI 权限", icon: Eye },
  { href: "/gov/audit", label: "审计日志", icon: ClipboardList },
];

/**
 * 治理控制台布局 —— 左侧固定导航栏 + 右侧内容区域。
 * 使用 Next.js 路由组 (console) 内的子布局，与 TokenHub 管理台共享路由组。
 */
export default function GovLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  return (
    <div className="flex h-screen bg-gray-50">
      {/* 左侧导航栏 */}
      <aside className="flex w-56 flex-shrink-0 flex-col border-r border-gray-200 bg-white">
        {/* 品牌区域 */}
        <div className="border-b border-gray-200 px-5 py-4">
          <Link href="/gov/dashboard" className="block">
            <h1 className="text-lg font-bold text-gray-900">
              AI 治理控制台
            </h1>
            <p className="mt-0.5 text-xs text-gray-500">
              Governance Console
            </p>
          </Link>
        </div>

        {/* 导航菜单列表 */}
        <nav className="flex-1 overflow-y-auto px-3 py-4" aria-label="主导航">
          <ul className="space-y-1">
            {navItems.map((item) => {
              // 判断当前是否处于该导航项的路径下
              const isActive =
                pathname === item.href || pathname.startsWith(item.href + "/");
              const IconComponent = item.icon;

              return (
                <li key={item.href}>
                  <Link
                    href={item.href}
                    className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                      isActive
                        ? "bg-blue-50 text-blue-700"
                        : "text-gray-600 hover:bg-gray-100 hover:text-gray-900"
                    }`}
                  >
                    <IconComponent
                      className={`h-4.5 w-4.5 flex-shrink-0 ${
                        isActive ? "text-blue-600" : "text-gray-400"
                      }`}
                    />
                    <span>{item.label}</span>
                    {isActive && (
                      <ChevronRight className="ml-auto h-3.5 w-3.5 text-blue-400" />
                    )}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>

        {/* 底部版本信息 */}
        <div className="border-t border-gray-200 px-5 py-3">
          <p className="text-xs text-gray-400">v3.2.0</p>
        </div>
      </aside>

      {/* 右侧内容区域 */}
      <main className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-7xl px-6 py-8">{children}</div>
      </main>
    </div>
  );
}
