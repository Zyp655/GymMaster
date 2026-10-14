import { Skeleton } from '@/components/ui/skeleton';

export default function Loading() {
  return (
    <div className="min-h-dvh bg-canvas">
      <div className="h-16 border-b border-line bg-surface" />
      <div className="bg-ink-950">
        <div className="mx-auto grid max-w-6xl gap-6 px-4 py-24 sm:px-6">
          <Skeleton className="h-6 w-48 bg-ink-800" />
          <Skeleton className="h-12 w-full max-w-2xl bg-ink-800" />
          <Skeleton className="h-6 w-full max-w-xl bg-ink-800" />
        </div>
      </div>
      <div className="mx-auto grid max-w-6xl gap-5 px-4 py-16 sm:grid-cols-2 sm:px-6 lg:grid-cols-4">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} className="h-80 w-full rounded-2xl" />
        ))}
      </div>
    </div>
  );
}
