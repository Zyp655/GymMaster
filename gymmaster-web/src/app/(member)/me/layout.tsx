import type { ReactNode } from 'react';
import { redirect } from 'next/navigation';
import { PortalShell } from '@/features/portal/components/PortalShell';
import { getCachedSession } from '@/guards/getCachedSession';

export default async function MemberPortalLayout({ children }: { children: ReactNode }) {
  const session = await getCachedSession();
  if (!session?.user || session.error) {
    redirect('/login?returnUrl=/me');
  }
  return <PortalShell user={session.user}>{children}</PortalShell>;
}
