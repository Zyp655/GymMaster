/** Hàm thuần cho Dashboard (tách riêng để unit test). */

export interface BranchSummary {
  total: number;
  active: number;
  inactive: number;
}

/**
 * Đếm chi nhánh trong phạm vi người dùng: `allowedIds` rỗng = toàn hệ thống,
 * ngược lại chỉ đếm các chi nhánh được phân công.
 */
export function summarizeBranches(
  branches: readonly { id: number; status?: string }[],
  allowedIds: readonly number[] | null | undefined,
): BranchSummary {
  const allowed = allowedIds ?? [];
  const scoped = allowed.length === 0 ? branches : branches.filter((b) => allowed.includes(b.id));
  const active = scoped.filter((b) => b.status === 'ACTIVE').length;
  return { total: scoped.length, active, inactive: scoped.length - active };
}
