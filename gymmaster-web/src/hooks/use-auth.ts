'use client';

import { useSession } from 'next-auth/react';

/** Người dùng hiện tại trên cổng hội viên. */
export function useAuth() {
  const { data: session, status } = useSession();
  const user = session?.user ?? null;
  return {
    user,
    status,
    isLoading: status === 'loading',
    isMember: Boolean(user?.roles?.includes('MEMBER')),
  };
}
