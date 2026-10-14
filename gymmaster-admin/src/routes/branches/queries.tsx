import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App } from 'antd';
import branchApi from '@/api/branch';
import type { Branch, BranchInput } from '@/api/types';
import { getErrorMessage } from '@/utils/errors';

/** Tạo / cập nhật chi nhánh. Lỗi do form tự xử lý (onError của nơi gọi). */
export function useSaveBranch() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id?: number; data: BranchInput }) =>
      id ? branchApi.update(id, data) : branchApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [branchApi.key] }),
  });
}

/** Xóa một hoặc nhiều chi nhánh; báo kết quả từng phần nếu có mục lỗi (ví dụ 40900 còn nhân viên). */
export function useDeleteBranches() {
  const queryClient = useQueryClient();
  const { message, modal } = App.useApp();

  return useMutation({
    mutationFn: async (items: Pick<Branch, 'id' | 'name'>[]) => {
      const results = await Promise.allSettled(items.map((b) => branchApi.remove(b.id)));
      return results.map((r, i) => ({
        item: items[i],
        error: r.status === 'rejected' ? r.reason : null,
      }));
    },
    onSuccess: (results) => {
      const ok = results.filter((r) => !r.error).length;
      const failed = results.filter((r) => r.error);
      if (ok > 0) message.success(`Đã xóa ${ok} chi nhánh`);
      if (failed.length > 0) {
        modal.error({
          title: `Không xóa được ${failed.length} chi nhánh`,
          content: (
            <ul style={{ paddingLeft: 18, margin: 0 }}>
              {failed.map((f) => (
                <li key={f.item.id}>
                  <b>{f.item.name}</b>: {getErrorMessage(f.error)}
                </li>
              ))}
            </ul>
          ),
        });
      }
      return queryClient.invalidateQueries({ queryKey: [branchApi.key] });
    },
  });
}
