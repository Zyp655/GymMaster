import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { ContractsView } from '@/features/portal/components/ContractsView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.contracts');
  return { title: t('title') };
}

export default function ContractsPage() {
  return <ContractsView />;
}
