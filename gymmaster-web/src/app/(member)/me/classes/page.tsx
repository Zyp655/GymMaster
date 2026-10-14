import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { ClassesView } from '@/features/portal/components/ClassesView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.classes');
  return { title: t('title') };
}

export default function MemberClassesPage() {
  return <ClassesView />;
}
