import { useMutation, useQueryClient } from '@tanstack/react-query';
import staffApi from '@/api/staff';
import type { RoleAssignment, StaffCreateInput, StaffInput } from '@/api/staff';

/**
 * Tạo / cập nhật nhân viên.
 * Sửa: PUT thông tin, sau đó (nếu vai trò thay đổi) PUT /roles. Lỗi do form xử lý,
 * `step` cho biết bước lỗi để gắn đúng tab.
 */
export function useSaveStaff() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (
      v:
        | { id?: undefined; create: StaffCreateInput }
        | { id: number; update: StaffInput; roles?: RoleAssignment[] },
    ) => {
      if (!('update' in v)) return staffApi.create(v.create);
      const staff = await staffApi.update(v.id, v.update);
      if (!v.roles) return staff;
      try {
        return await staffApi.assignRoles(v.id, v.roles);
      } catch (error) {
        // Thông tin đã lưu, chỉ gán vai trò lỗi → đánh dấu để form báo đúng chỗ.
        throw Object.assign(error as object, { step: 'roles' });
      }
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: [staffApi.key] }),
  });
}

export function useStaffAction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action }: { id: number; action: 'lock' | 'unlock' | 'delete' }) =>
      action === 'lock'
        ? staffApi.lock(id)
        : action === 'unlock'
          ? staffApi.unlock(id)
          : staffApi.remove(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [staffApi.key] }),
  });
}

export function useResetStaffPassword() {
  return useMutation({
    mutationFn: ({ id, password }: { id: number; password: string }) =>
      staffApi.resetPassword(id, password),
  });
}
