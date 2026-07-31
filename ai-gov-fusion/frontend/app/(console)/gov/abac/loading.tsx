export default function AbacLoading() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="h-7 w-36 rounded bg-gray-200" />
      <div className="flex gap-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-8 w-20 rounded bg-gray-200" />
        ))}
      </div>
      <div className="h-64 rounded-lg bg-gray-200" />
    </div>
  );
}
