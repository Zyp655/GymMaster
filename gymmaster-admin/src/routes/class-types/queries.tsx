import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App } from 'antd';
import classApi from '@/api/class';
import type { ClassType, ClassTypeInput } from '@/api/class';
import { getErrorMessage } from '@/utils/errors';

/** Tạo / cập nhật loại lớp. Lỗi do form tự xử lý. */
export function useSaveClassType() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id?: number; data: ClassTypeInput }) =>
      id ? classApi.updateType(id, data) : classApi.createType(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [classApi.typeKey] }),
  });
}

/** Xóa một hoặc nhiều loại lớp; báo mục lỗi (vd 40900 còn lịch sắp diễn ra). */
export function useDeleteClassTypes() {
  const queryClient = useQueryClient();
  const { message, modal } = App.useApp();

  return useMutation({
    mutationFn: async (items: Pick<ClassType, 'id' | 'name'>[]) => {
      const results = await Promise.allSettled(items.map((t) => classApi.removeType(t.id)));
      return results.map((r, i) => ({
        item: items[i],
        error: r.status === 'rejected' ? r.reason : null,
      }));
    },
    onSuccess: (results) => {
      const ok = results.filter((r) => !r.error).length;
      const failed = results.filter((r) => r.error);
      if (ok > 0) message.success(`Đã xóa ${ok} loại lớp`);
      if (failed.length > 0) {
        modal.error({
          title: `Không xóa được ${failed.length} loại lớp`,
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
      return queryClient.invalidateQueries({ queryKey: [classApi.typeKey] });
    },
  });
}
