export default function ModelPermissionsLoading() {
  return (
    <div className="space-y-6 animate-pulse">
      <div>
        <div className="h-8 w-40 rounded bg-gray-200" />
        <div className="mt-1 h-4 w-64 rounded bg-gray-200" />
      </div>
      <div className="flex gap-4 border-b border-gray-200 pb-2">
        {Array.from({ length: 2 }).map((_, i) => (
          <div key={i} className="h-8 w-24 rounded bg-gray-200" />
        ))}
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