import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { NutritionView } from '@/features/portal/components/NutritionView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.nutrition');
  return { title: t('title') };
}

export default function NutritionPage() {
  return <NutritionView />;
}
