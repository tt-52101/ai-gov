export default function DashboardLoading() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="flex justify-between">
        <div className="h-7 w-20 rounded bg-gray-200" />
        <div className="h-9 w-24 rounded bg-gray-200" />
      </div>
      <div className="grid grid-cols-4 gap-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-24 rounded-lg bg-gray-200" />
        ))}
      </div>
      <div className="h-48 rounded-lg bg-gray-200" />
      <div className="grid grid-cols-3 gap-4">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="h-40 rounded-lg bg-gray-200" />
        ))}
      </div>
    </div>
  );
}
