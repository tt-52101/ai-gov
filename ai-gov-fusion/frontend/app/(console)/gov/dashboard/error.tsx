"use client";

export default function DashboardError({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-20">
      <h2 className="text-lg font-semibold text-gray-900">仪表盘加载失败</h2>
      <p className="mt-2 text-sm text-gray-500">{error.message}</p>
      <button onClick={reset} className="mt-4 rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">重试</button>
    </div>
  );
}
