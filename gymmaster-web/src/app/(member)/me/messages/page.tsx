import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { MessagesView } from '@/features/portal/components/MessagesView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.nav');
  // Chỉ tiêu đề ngắn – layout gốc tự nối "· GymMaster".
  return { title: t('messages') };
}

export default function MessagesPage() {
  return <MessagesView />;
}
