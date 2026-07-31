/**
 * Party 管理页面加载状态 —— 展示骨架屏占位。
 */
export default function PartiesLoading() {
  return (
    <div className="space-y-6 animate-pulse">
      {/* 标题栏骨架 */}
      <div className="flex items-center justify-between">
        <div>
          <div className="h-7 w-36 rounded bg-gray-200" />
          <div className="mt-2 h-4 w-64 rounded bg-gray-200" />
        </div>
        <div className="h-9 w-32 rounded bg-gray-200" />
      </div>
      {/* 表格骨架 */}
      <div className="rounded-lg border border-gray-200 bg-white p-6">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="mb-4 flex gap-4">
            <div className="h-4 flex-1 rounded bg-gray-200" />
            <div className="h-4 w-16 rounded bg-gray-200" />
            <div className="h-4 w-16 rounded bg-gray-200" />
            <div className="h-4 w-20 rounded bg-gray-200" />
          </div>
        ))}
      </div>
    </div>
  );
}
