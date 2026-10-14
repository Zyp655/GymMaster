'use client';

import type { ReactNode } from 'react';
import { useAuth } from '@/hooks/use-auth';
import { hasAnyPermission, type PermCode } from '@/lib/permissions';

/**
 * Chỉ render children khi người dùng có quyền `perm` (hoặc một trong `anyOf`).
 * Đây chỉ là cải thiện UX – backend luôn kiểm tra quyền lần cuối.
 */
export function PermissionGate({
  perm,
  anyOf,
  fallback = null,
  children,
}: {
  perm?: PermCode;
  anyOf?: PermCode[];
  fallback?: ReactNode;
  children: ReactNode;
}) {
  const { user } = useAuth();
  const perms = anyOf ?? (perm ? [perm] : []);
  if (perms.length > 0 && !hasAnyPermission(user, perms)) return <>{fallback}</>;
  return <>{children}</>;
}
