import { useQuery } from '@tanstack/react-query';
import branchApi from '@/api/branch';
import type { BranchOption } from '@/api/types';
import { PERM } from '@/constants/permissions';
import { usePermission } from './usePermission';

/**
 * Danh sách chi nhánh mà người dùng được phép làm việc (cho ô chọn chi nhánh).
 * - Có `branch.view`: lấy từ `/admin/branches` (kể cả chi nhánh ngừng hoạt động).
 * - Không có: lấy `/public/branches` (chỉ chi nhánh đang hoạt động).
 * - User có `branch_ids` → chỉ giữ các chi nhánh đó.
 */
export function useBranchOptions() {
  const { user, can } = usePermission();
  const canView = can(PERM.BRANCH_VIEW);
  const allowed = user?.branch_ids ?? [];

  const query = useQuery({
    queryKey: [branchApi.key, 'options', canView],
    enabled: !!user,
    staleTime: 5 * 60_000,
    queryFn: async (): Promise<BranchOption[]> => {
      if (canView) {
        const page = await branchApi.list({ limit: 100, order_by: 'name', order_dir: 'ASC' });
        return page.items.map((b) => ({ id: b.id, name: b.name, code: b.code, status: b.status }));
      }
      return (await branchApi.publicList()) ?? [];
    },
  });

  const options = (query.data ?? []).filter((b) => allowed.length === 0 || allowed.includes(b.id));

  return {
    ...query,
    branches: options,
    /** User được làm việc toàn hệ thống (không bị giới hạn chi nhánh). */
    isGlobal: allowed.length === 0,
  };
}

export default useBranchOptions;
