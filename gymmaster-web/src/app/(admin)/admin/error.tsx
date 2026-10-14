'use client';

import { ErrorView } from '@/components/common/ErrorView';

export default function AdminError({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="rounded-2xl border border-line bg-surface">
      <ErrorView reset={reset} />
    </div>
  );
}
