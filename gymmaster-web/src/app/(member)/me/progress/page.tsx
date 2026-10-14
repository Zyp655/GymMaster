import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { ProgressView } from '@/features/portal/components/ProgressView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.progress');
  return { title: t('title') };
}

export default function MemberProgressPage() {
  return <ProgressView />;
}
