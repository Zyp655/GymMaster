import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { WorkoutView } from '@/features/portal/components/WorkoutView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.workout');
  return { title: t('title') };
}

export default function WorkoutPage() {
  return <WorkoutView />;
}
