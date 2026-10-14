import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { QrCheckinView } from '@/features/portal/components/QrCheckinView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.qr');
  return { title: t('title') };
}

export default function MemberQrPage() {
  return <QrCheckinView />;
}
