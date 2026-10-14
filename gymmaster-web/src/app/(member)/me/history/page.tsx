import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { HistoryView } from '@/features/portal/components/HistoryView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.history');
  return { title: t('title') };
}

export default function MemberHistoryPage() {
  return <HistoryView />;
}
