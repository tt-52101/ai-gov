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
import { extractErrorMessage } from "@/lib/error-codes";

/** 导航菜单项定义 */
interface NavItem {
  /** 导航路径 */
  href: string;
  /** 显示标签 */
  label: string;
  /** lucide-react 图标 */
  icon: React.ComponentType<{ className?: string }>;
  /** 菜单编码，对应后端 menu_code */
  code: string;
}

/** UI 权限快照中单个菜单的可见性 */
interface MenuVisibility {
  menu_code: string;
  visible: boolean;
}

/**
 * AI 治理网关管理控制台导航菜单配置。
 * 包含全部 8 个功能模块的入口。
 * code 字段对应后端 UI 权限投影中的 menu_code。
 */
const allNavItems: NavItem[] = [
  { href: "/gov/dashboard", label: "仪表盘", icon: LayoutDashboard, code: "dashboard" },
  { href: "/gov/parties", label: "Party 管理", icon: Building2, code: "parties" },
  { href: "/gov/fund", label: "资金操作", icon: Wallet, code: "fund" },
  { href: "/gov/pricing", label: "价目维护", icon: Tags, code: "pricing" },
  { href: "/gov/routes", label: "路由档案", icon: GitBranch, code: "routes" },
  { href: "/gov/abac", label: "ABAC 策略", icon: Shield, code: "abac" },
  { href: "/gov/ui-permissions", label: "UI 权限", icon: Eye, code: "ui_permissions" },
  { href: "/gov/audit", label: "审计日志", icon: ClipboardList, code: "audit" },
];

/**
 * 治理控制台布局 —— 左侧固定导航栏 + 右侧内容区域。
 * 组件挂载时调用 GET /gov/ui-permissions/snapshot 获取当前用户的菜单可见性投影，
 * 根据 visible 字段动态过滤导航项（ABAC 权限隐藏）。
 * 加载期间显示骨架屏占位。
 * 使用 Next.js 路由组 (console) 内的子布局，与 TokenHub 管理台共享路由组。
 */
export default function GovLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  // 权限快照加载状态
  const [permLoading, setPermLoading] = React.useState(true);
  // 根据权限过滤后的可见导航项
  const [visibleItems, setVisibleItems] = React.useState<NavItem[]>([]);

  // 组件挂载时获取 UI 权限投影
  React.useEffect(() => {
    let cancelled = false;

    async function loadPermissions() {
      setPermLoading(true);
      try {
        const res = await fetch("/v1/gov/ui-permissions/snapshot");
        if (!res.ok) {
          // 权限接口失败时回退：显示全部菜单，确保用户不会被锁死
          if (!cancelled) {
            setVisibleItems(allNavItems);
          }
          return;
        }

        const body = await res.json();
        // 期望后端返回 { menus: [{ menu_code, visible }] }
        const menus: MenuVisibility[] = body?.menus ?? [];

        if (menus.length === 0) {
          // 后端未返回菜单数据时回退显示全部
          if (!cancelled) setVisibleItems(allNavItems);
          return;
        }

        // 构建 menu_code -> visible 映射
        const visibleMap = new Map<string, boolean>();
        for (const m of menus) {
          visibleMap.set(m.menu_code, m.visible);
        }

        // 过滤：仅保留 visible !== false 的菜单项（未在投影中出现的视为可见）
        const filtered = allNavItems.filter((item) => {
          const v = visibleMap.get(item.code);
          return v !== false; // undefined 或 true 均视为可见
        });

        if (!cancelled) setVisibleItems(filtered);
      } catch {
        // 网络异常等回退显示全部
        if (!cancelled) setVisibleItems(allNavItems);
      } finally {
        if (!cancelled) setPermLoading(false);
      }
    }

    loadPermissions();
    return () => { cancelled = true; };
  }, []);

  /** 渲染骨架屏导航项 */
  const renderSkeleton = () => (
    <ul className="space-y-1">
      {Array.from({ length: 8 }).map((_, i) => (
        <li key={i} className="flex items-center gap-3 rounded-lg px-3 py-2">
          <div className="h-4.5 w-4.5 flex-shrink-0 rounded bg-gray-200 animate-pulse" />
          <div className="h-4 flex-1 rounded bg-gray-200 animate-pulse" />
        </li>
      ))}
    </ul>
  );

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
          {permLoading ? (
            renderSkeleton()
          ) : (
            <ul className="space-y-1">
              {visibleItems.map((item) => {
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
          )}
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
