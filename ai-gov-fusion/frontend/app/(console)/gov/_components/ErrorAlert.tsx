"use client";

import React from "react";
import { AlertTriangle, RefreshCw, X } from "lucide-react";

/** 错误提示属性 */
export interface ErrorAlertProps {
  /** 错误标题 */
  title?: string;
  /** 错误详细描述 */
  message: string;
  /** 重试回调函数 */
  onRetry?: () => void;
  /** 是否可关闭 */
  dismissible?: boolean;
}

/**
 * 错误提示组件 —— 用于显示操作失败或数据加载错误的提示卡片。
 * 支持重试操作和手动关闭。
 */
export function ErrorAlert({
  title = "操作失败",
  message,
  onRetry,
  dismissible = false,
}: ErrorAlertProps) {
  const [dismissed, setDismissed] = React.useState(false);

  if (dismissed) return null;

  return (
    <div
      role="alert"
      className="rounded-lg border border-red-300 bg-red-50 p-4"
    >
      <div className="flex items-start gap-3">
        <AlertTriangle className="mt-0.5 h-5 w-5 flex-shrink-0 text-red-500" />
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold text-red-800">{title}</h3>
          <p className="mt-1 text-sm text-red-700">{message}</p>

          {/* 操作按钮行 */}
          <div className="mt-3 flex gap-2">
            {onRetry && (
              <button
                onClick={onRetry}
                className="inline-flex items-center gap-1 rounded-md bg-red-100 px-3 py-1.5 text-sm font-medium text-red-800 transition-colors hover:bg-red-200"
              >
                <RefreshCw className="h-3.5 w-3.5" />
                重试
              </button>
            )}
            {dismissible && (
              <button
                onClick={() => setDismissed(true)}
                className="inline-flex items-center gap-1 rounded-md px-3 py-1.5 text-sm font-medium text-red-600 transition-colors hover:bg-red-100"
              >
                <X className="h-3.5 w-3.5" />
                关闭
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
