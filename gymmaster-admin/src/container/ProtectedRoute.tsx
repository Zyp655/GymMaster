import type { ReactNode } from 'react';
import Forbidden from '@/components/Forbidden';
import { usePermission } from '@/hooks/usePermission';

/** Chặn trang theo quyền (anyOf). Việc kiểm tra đăng nhập nằm ở AppRoutes. */
export function ProtectedRoute({ perm, children }: { perm?: string[]; children: ReactNode }) {
  const { can } = usePermission();
  return <>{can(perm) ? children : <Forbidden />}</>;
}

export default ProtectedRoute;
