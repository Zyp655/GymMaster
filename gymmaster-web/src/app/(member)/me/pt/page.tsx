import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { PTView } from '@/features/portal/components/PTView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.pt');
  return { title: t('title') };
}

export default function MemberPTPage() {
  return <PTView />;
}
