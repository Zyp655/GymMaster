import type { ReactNode } from 'react';
import { usePermission } from '@/hooks/usePermission';
import type { PermInput } from '@/hooks/usePermission';

export interface PermissionGateProps {
  /** Một quyền hoặc mảng quyền. Mặc định: chỉ cần MỘT quyền (anyOf). */
  perm: PermInput;
  /** Yêu cầu có TẤT CẢ quyền trong mảng. */
  all?: boolean;
  /** Hiển thị khi không đủ quyền (mặc định: không hiển thị gì). */
  fallback?: ReactNode;
  children: ReactNode;
}

/** Ẩn/hiện nội dung theo quyền: `<PermissionGate perm="branch.create">...</PermissionGate>` */
export function PermissionGate({ perm, all, fallback = null, children }: PermissionGateProps) {
  const { can, canAll } = usePermission();
  const ok = all ? canAll(perm) : can(perm);
  return <>{ok ? children : fallback}</>;
}

export default PermissionGate;
