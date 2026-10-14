import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { OverviewView } from '@/features/portal/components/OverviewView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.overview');
  return { title: t('title') };
}

export default function MemberOverviewPage() {
  return <OverviewView />;
}
