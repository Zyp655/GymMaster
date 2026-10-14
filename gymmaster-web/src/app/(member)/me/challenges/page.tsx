import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { ChallengesView } from '@/features/portal/components/ChallengesView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.challenges');
  return { title: t('title') };
}

export default function ChallengesPage() {
  return <ChallengesView />;
}
