"use client";

import React from "react";
import { Copy, Check } from "lucide-react";

/** 代码块属性 */
export interface CodeBlockProps {
  /** JSON 数据对象 */
  data: unknown;
  /** 最大显示高度（超出则滚动） */
  maxHeight?: string;
  /** 是否默认折叠 */
  collapsible?: boolean;
  /** 折叠时的标题 */
  title?: string;
}

/**
 * JSON/代码展示组件 —— 以格式化 JSON 形式展示数据。
 * 支持复制到剪贴板和可折叠显示。
 */
export function CodeBlock({
  data,
  maxHeight = "400px",
  collapsible = false,
  title,
}: CodeBlockProps) {
  const [collapsed, setCollapsed] = React.useState(collapsible);
  const [copied, setCopied] = React.useState(false);

  // 格式化 JSON 字符串
  const jsonString = React.useMemo(
    () => JSON.stringify(data, null, 2),
    [data]
  );

  // 复制到剪贴板
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(jsonString);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // 剪贴板 API 不可用时静默失败
    }
  };

  return (
    <div className="rounded-lg border border-gray-200 bg-gray-50">
      {/* 工具栏：标题 + 操作按钮 */}
      <div className="flex items-center justify-between border-b border-gray-200 px-4 py-2">
        <div className="flex items-center gap-2">
          {collapsible && (
            <button
              onClick={() => setCollapsed(!collapsed)}
              className="text-xs font-medium text-gray-500 hover:text-gray-700"
            >
              {collapsed ? "展开" : "折叠"}
            </button>
          )}
          {title && (
            <span className="text-xs font-medium text-gray-600">{title}</span>
          )}
        </div>
        <button
          onClick={handleCopy}
          className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs text-gray-500 transition-colors hover:bg-gray-200 hover:text-gray-700"
          title="复制到剪贴板"
        >
          {copied ? (
            <Check className="h-3.5 w-3.5 text-green-600" />
          ) : (
            <Copy className="h-3.5 w-3.5" />
          )}
          {copied ? "已复制" : "复制"}
        </button>
      </div>

      {/* JSON 内容 */}
      {!collapsed && (
        <pre
          className="overflow-auto p-4 text-xs leading-relaxed text-gray-800"
          style={{ maxHeight }}
        >
          <code>{jsonString}</code>
        </pre>
      )}
    </div>
  );
}
