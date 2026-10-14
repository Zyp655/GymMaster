import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { RewardsView } from '@/features/portal/components/RewardsView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.rewards');
  return { title: t('title') };
}

export default function RewardsPage() {
  return <RewardsView />;
}
