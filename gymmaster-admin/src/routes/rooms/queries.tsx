import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App } from 'antd';
import roomApi from '@/api/room';
import type { Room, RoomInput } from '@/api/room';
import { getErrorMessage } from '@/utils/errors';

/** Tạo / cập nhật phòng. Lỗi do form xử lý. */
export function useSaveRoom() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id?: number; data: RoomInput }) =>
      id ? roomApi.update(id, data) : roomApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [roomApi.key] }),
  });
}

/** Xóa một hoặc nhiều phòng; báo từng mục lỗi (40900 còn lịch lớp sắp diễn ra). */
export function useDeleteRooms() {
  const queryClient = useQueryClient();
  const { message, modal } = App.useApp();

  return useMutation({
    mutationFn: async (items: Pick<Room, 'id' | 'name'>[]) => {
      const results = await Promise.allSettled(items.map((r) => roomApi.remove(r.id)));
      return results.map((r, i) => ({
        item: items[i],
        error: r.status === 'rejected' ? r.reason : null,
      }));
    },
    onSuccess: (results) => {
      const ok = results.filter((r) => !r.error).length;
      const failed = results.filter((r) => r.error);
      if (ok > 0) message.success(`Đã xóa ${ok} phòng`);
      if (failed.length > 0) {
        modal.error({
          title: `Không xóa được ${failed.length} phòng`,
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
      return queryClient.invalidateQueries({ queryKey: [roomApi.key] });
    },
  });
}
