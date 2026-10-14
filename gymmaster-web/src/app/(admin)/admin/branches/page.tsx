import { Suspense } from 'react';
import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { Forbidden } from '@/components/common/Forbidden';
import { BranchesView } from '@/features/branch/components/BranchesView';
import { getCachedSession } from '@/guards/getCachedSession';
import { hasPermission, Perm } from '@/lib/permissions';
import AdminLoading from '../loading';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('branch');
  return { title: t('title') };
}

export default async function BranchesPage() {
  const session = await getCachedSession();
  if (!hasPermission(session?.user, Perm.BRANCH_VIEW)) return <Forbidden />;
  return (
    // BranchesView đọc bộ lọc từ URL (useSearchParams) nên cần Suspense.
    <Suspense fallback={<AdminLoading />}>
      <BranchesView />
    </Suspense>
  );
}
