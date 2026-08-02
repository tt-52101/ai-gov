export default function KeysLoading() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="flex items-center justify-between">
        <div>
          <div className="h-8 w-32 rounded bg-gray-200" />
          <div className="mt-1 h-4 w-64 rounded bg-gray-200" />
        </div>
        <div className="h-10 w-28 rounded-lg bg-gray-200" />
      </div>
      <div className="rounded-lg border border-gray-200 bg-white">
        <div className="space-y-1 p-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-12 rounded bg-gray-100" />
          ))}
        </div>
      </div>
    </div>
  );
}