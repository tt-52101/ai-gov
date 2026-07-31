"use client";

import React from "react";
import { AlertTriangle, Loader2 } from "lucide-react";

/** 确认对话框属性 */
export interface ConfirmDialogProps {
  /** 是否显示对话框 */
  open: boolean;
  /** 对话框标题 */
  title: string;
  /** 对话框描述信息 */
  message: string;
  /** 确认按钮文本 */
  confirmLabel?: string;
  /** 取消按钮文本 */
  cancelLabel?: string;
  /** 是否为危险操作（红色按钮） */
  danger?: boolean;
  /** 是否正在执行中 */
  loading?: boolean;
  /** 确认回调 */
  onConfirm: () => void;
  /** 取消回调 */
  onCancel: () => void;
}

/**
 * 危险操作确认弹窗 —— 在执行不可逆操作前要求用户二次确认。
 * 用于资金划拨、清算、删除角色等高危操作场景。
 */
export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = "确认",
  cancelLabel = "取消",
  danger = false,
  loading = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  if (!open) return null;

  // 处理 Esc 键关闭
  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !loading) {
        onCancel();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onCancel, loading]);

  // 确认按钮样式
  const confirmButtonClass = danger
    ? "bg-red-600 text-white hover:bg-red-700 focus:ring-red-500"
    : "bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500";

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
    >
      {/* 半透明遮罩 */}
      <div
        className="fixed inset-0 bg-black/50 transition-opacity"
        onClick={loading ? undefined : onCancel}
      />

      {/* 对话框内容 */}
      <div className="relative z-10 mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
        <div className="flex items-start gap-4">
          {danger && (
            <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-red-100">
              <AlertTriangle className="h-5 w-5 text-red-600" />
            </div>
          )}

          <div className="min-w-0 flex-1">
            <h3
              id="confirm-dialog-title"
              className="text-lg font-semibold text-gray-900"
            >
              {title}
            </h3>
            <p className="mt-2 text-sm text-gray-600">{message}</p>
          </div>
        </div>

        {/* 按钮行 */}
        <div className="mt-6 flex justify-end gap-3">
          <button
            onClick={onCancel}
            disabled={loading}
            className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 disabled:opacity-50"
          >
            {cancelLabel}
          </button>
          <button
            onClick={onConfirm}
            disabled={loading}
            className={`inline-flex items-center gap-1.5 rounded-md px-4 py-2 text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 ${confirmButtonClass}`}
          >
            {loading && <Loader2 className="h-4 w-4 animate-spin" />}
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
