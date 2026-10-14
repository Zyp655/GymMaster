import { useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { setWorkingBranch } from '@/store/authSlice';
import { useAppDispatch, useAppSelector } from '@/store';

/**
 * Chi nhánh làm việc đang chọn ở header.
 *
 *   const { branchId } = useWorkingBranch();
 *   // gửi kèm khi gọi API danh sách: { ...params, branch_id: branchId ?? undefined }
 *   // và đưa branchId vào queryKey để tự tải lại khi đổi chi nhánh.
 *
 * `branchId = null` nghĩa là "Tất cả chi nhánh" (chỉ có với user không bị giới hạn chi nhánh).
 * Axios cũng tự gửi header `X-Branch-Id` theo giá trị này.
 */
export function useWorkingBranch() {
  const branchId = useAppSelector((s) => s.auth.workingBranchId);
  const dispatch = useAppDispatch();
  const queryClient = useQueryClient();

  const setBranchId = useCallback(
    (id: number | null) => {
      dispatch(setWorkingBranch(id));
      // Dữ liệu phụ thuộc chi nhánh → tải lại.
      void queryClient.invalidateQueries();
    },
    [dispatch, queryClient],
  );

  return { branchId, setBranchId };
}

export default useWorkingBranch;
