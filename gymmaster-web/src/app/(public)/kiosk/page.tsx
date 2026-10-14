import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { KioskView } from '@/features/kiosk/KioskView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.kiosk');
  return { title: t('title') };
}

export default function KioskPage() {
  return <KioskView />;
}
