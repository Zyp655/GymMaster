import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { NotificationsView } from '@/features/portal/components/NotificationsView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.notifications');
  return { title: t('title') };
}

export default function MemberNotificationsPage() {
  return <NotificationsView />;
}
