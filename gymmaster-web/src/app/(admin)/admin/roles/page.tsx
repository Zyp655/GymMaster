import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { Forbidden } from '@/components/common/Forbidden';
import { RolesView } from '@/features/role/components/RolesView';
import { getCachedSession } from '@/guards/getCachedSession';
import { hasPermission, Perm } from '@/lib/permissions';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('role');
  return { title: t('title') };
}

export default async function RolesPage() {
  const session = await getCachedSession();
  if (!hasPermission(session?.user, Perm.ROLE_VIEW)) return <Forbidden />;
  return <RolesView />;
}
