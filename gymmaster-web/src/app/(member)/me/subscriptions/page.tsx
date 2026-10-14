import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { SubscriptionsView } from '@/features/portal/components/SubscriptionsView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.subscriptions');
  return { title: t('title') };
}

export default function MemberSubscriptionsPage() {
  return <SubscriptionsView />;
}
