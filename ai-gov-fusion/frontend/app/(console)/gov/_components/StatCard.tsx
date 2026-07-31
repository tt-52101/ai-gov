"use client";

import { type LucideIcon } from "lucide-react";

/** 统计卡片属性 */
export interface StatCardProps {
  /** 卡片标题 */
  title: string;
  /** 主要数值 */
  value: string | number;
  /** 辅助描述文本 */
  description?: string;
  /** lucide-react 图标组件 */
  icon: LucideIcon;
  /** 数值变化趋势（正数为增长，负数为下降） */
  trend?: number;
  /** 自定义颜色方案（覆盖默认颜色） */
  colorClass?: string;
}

/**
 * 统计卡片组件 —— 展示单个统计指标的卡片。
 * 用于仪表盘、概览页等场景，包含图标、标题、数值和趋势指示。
 */
export function StatCard({
  title,
  value,
  description,
  icon: Icon,
  trend,
  colorClass = "text-blue-600",
}: StatCardProps) {
  // 格式化趋势显示文本
  const trendText = trend !== undefined
    ? `${trend >= 0 ? "+" : ""}${trend.toFixed(1)}%`
    : null;
  const trendColor = trend !== undefined
    ? trend >= 0 ? "text-green-600" : "text-red-600"
    : "";

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm transition-shadow hover:shadow-md">
      {/* 顶部：图标 + 标题 */}
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-gray-500">{title}</span>
        <Icon className={`h-5 w-5 ${colorClass}`} aria-hidden="true" />
      </div>

      {/* 中部：主要数值 */}
      <div className="mt-2 flex items-baseline gap-2">
        <span className="text-2xl font-bold text-gray-900">{value}</span>
        {trendText && (
          <span className={`text-sm font-medium ${trendColor}`}>
            {trendText}
          </span>
        )}
      </div>

      {/* 底部：辅助描述 */}
      {description && (
        <p className="mt-1 text-xs text-gray-400">{description}</p>
      )}
    </div>
  );
}
