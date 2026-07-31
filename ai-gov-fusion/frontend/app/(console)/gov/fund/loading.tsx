/**
 * 资金操作页面加载状态。
 */
export default function FundLoading() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="h-7 w-24 rounded bg-gray-200" />
      <div className="grid grid-cols-4 gap-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-24 rounded-lg bg-gray-200" />
        ))}
      </div>
      <div className="h-64 rounded-lg bg-gray-200" />
    </div>
  );
}
