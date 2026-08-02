"use client";

import { ErrorAlert } from "../_components/ErrorAlert";

export default function TracingError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">调用追踪</h1>
        <p className="mt-1 text-sm text-gray-500">请求日志查询与调用链追踪</p>
      </div>
      <ErrorAlert
        title="页面加载失败"
        message={error.message}
        onRetry={reset}
      />
    </div>
  );
}