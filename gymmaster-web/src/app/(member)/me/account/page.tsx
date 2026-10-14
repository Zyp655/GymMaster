import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { AccountView } from '@/features/portal/components/AccountView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.account');
  return { title: t('title') };
}

export default function MemberAccountPage() {
  return <AccountView />;
}
