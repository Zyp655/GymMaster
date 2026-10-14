import { Suspense } from 'react';
import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { Skeleton } from '@/components/ui/skeleton';
import { PaymentResultView } from '@/features/portal/components/PaymentResultView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.paymentResult');
  return { title: t('title') };
}

export default function PaymentResultPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-dvh items-center justify-center bg-canvas p-6">
          <Skeleton className="h-80 w-full max-w-lg rounded-2xl" />
        </div>
      }
    >
      <PaymentResultView />
    </Suspense>
  );
}
