"use client";

import { ErrorAlert } from "../_components/ErrorAlert";

export default function SecurityReportsError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">安全报表</h1>
        <p className="mt-1 text-sm text-gray-500">安全事件汇总与密钥轮换审计</p>
      </div>
      <ErrorAlert
        title="页面加载失败"
        message={error.message}
        onRetry={reset}
      />
    </div>
  );
}