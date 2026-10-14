import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { BuySubscriptionView } from '@/features/portal/components/BuySubscriptionView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.buySubscription');
  return { title: t('title') };
}

export default function BuySubscriptionPage() {
  return <BuySubscriptionView />;
}
