import { useCallback, useMemo } from 'react';
import { ROLE } from '@/constants/permissions';
import { useAppSelector } from '@/store';
import type { AuthUser } from '@/api/types';

export type PermInput = string | readonly string[] | undefined | null;

/** Kiểm tra quyền thuần (dùng được ngoài React). OWNER luôn có toàn quyền. */
export function hasPermission(user: AuthUser | null, perm: PermInput, all = false): boolean {
  if (!user) return false;
  if (!perm || (Array.isArray(perm) && perm.length === 0)) return true;
  if (user.roles?.includes(ROLE.OWNER)) return true;
  const list: readonly string[] = typeof perm === 'string' ? [perm] : perm;
  const owned = new Set(user.permissions || []);
  return all ? list.every((p) => owned.has(p)) : list.some((p) => owned.has(p));
}

/**
 * const { can, canAll, isOwner } = usePermission();
 * can('branch.create')                 // có quyền này
 * can(['member.view', 'checkin.scan']) // có ÍT NHẤT một quyền (anyOf)
 * canAll(['a', 'b'])                   // có TẤT CẢ
 */
export function usePermission() {
  const user = useAppSelector((s) => s.auth.user);
  const can = useCallback((perm: PermInput) => hasPermission(user, perm), [user]);
  const canAll = useCallback((perm: PermInput) => hasPermission(user, perm, true), [user]);
  return useMemo(
    () => ({
      user,
      can,
      canAll,
      isOwner: !!user?.roles?.includes(ROLE.OWNER),
      roles: user?.roles ?? [],
    }),
    [user, can, canAll],
  );
}

export default usePermission;
