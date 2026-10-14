'use client';

import { useSession } from 'next-auth/react';
import { hasPermission, type PermCode } from '@/lib/permissions';

/** Người dùng hiện tại + helper kiểm tra quyền phía client. */
export function useAuth() {
  const { data: session, status } = useSession();
  const user = session?.user ?? null;
  return {
    user,
    status,
    isLoading: status === 'loading',
    can: (perm: PermCode) => hasPermission(user, perm),
  };
}
