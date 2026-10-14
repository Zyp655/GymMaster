'use client';

import { ErrorView } from '@/components/common/ErrorView';

export default function RootError({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return <ErrorView reset={reset} className="min-h-dvh" />;
}
