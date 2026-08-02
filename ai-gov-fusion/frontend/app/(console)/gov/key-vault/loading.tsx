export default function KeyVaultLoading() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="flex items-center justify-between">
        <div>
          <div className="h-8 w-32 rounded bg-gray-200" />
          <div className="mt-1 h-4 w-64 rounded bg-gray-200" />
        </div>
        <div className="h-10 w-20 rounded-lg bg-gray-200" />
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-28 rounded-lg bg-gray-200" />
        ))}
      </div>
      <div className="h-40 rounded-lg bg-gray-200" />
    </div>
  );
}