import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App } from 'antd';
import memberApi from '@/api/member';
import type { BodyMetricInput, MemberInput } from '@/api/member';
import { showError } from '@/utils/errors';

/** Làm mới mọi dữ liệu hội viên (danh sách desktop/mobile, hồ sơ, options). */
function useInvalidateMembers() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: [memberApi.key] });
}

/** Tạo / cập nhật hội viên. Lỗi do form tự xử lý. */
export function useSaveMember() {
  const invalidate = useInvalidateMembers();
  return useMutation({
    mutationFn: ({ id, data }: { id?: number; data: MemberInput }) =>
      id ? memberApi.update(id, data) : memberApi.create(data),
    onSuccess: invalidate,
  });
}

export function useDeleteMember() {
  const invalidate = useInvalidateMembers();
  const { message } = App.useApp();
  return useMutation({
    mutationFn: (id: number) => memberApi.remove(id),
    onSuccess: () => {
      message.success('Đã xóa hội viên');
      return invalidate();
    },
    onError: (e) => showError(e, 'Không thể xóa hội viên'),
  });
}

export function useGrantPortal(memberId: number) {
  const invalidate = useInvalidateMembers();
  return useMutation({
    mutationFn: (data: { email: string; password: string }) =>
      memberApi.grantPortal(memberId, data),
    onSuccess: invalidate,
  });
}

export function useLockPortal(memberId: number) {
  const invalidate = useInvalidateMembers();
  const { message } = App.useApp();
  return useMutation({
    mutationFn: () => memberApi.lockPortal(memberId),
    onSuccess: () => {
      message.success('Đã khóa tài khoản cổng hội viên');
      return invalidate();
    },
    onError: (e) => showError(e, 'Không thể khóa tài khoản'),
  });
}

export function useResetQr(memberId: number) {
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  return useMutation({
    mutationFn: () => memberApi.resetQr(memberId),
    onSuccess: () => {
      message.success('Đã đặt lại mã QR. Các mã cũ không còn hiệu lực');
      return queryClient.invalidateQueries({ queryKey: [memberApi.key, 'qr', memberId] });
    },
    onError: (e) => showError(e, 'Không thể đặt lại QR'),
  });
}

export function useAddMetric(memberId: number) {
  const invalidate = useInvalidateMembers();
  return useMutation({
    mutationFn: (data: BodyMetricInput) => memberApi.addMetric(memberId, data),
    onSuccess: invalidate,
  });
}

export function useDeleteMetric(memberId: number) {
  const invalidate = useInvalidateMembers();
  const { message } = App.useApp();
  return useMutation({
    mutationFn: (metricId: number) => memberApi.removeMetric(memberId, metricId),
    onSuccess: () => {
      message.success('Đã xóa lần đo');
      return invalidate();
    },
    onError: (e) => showError(e, 'Không thể xóa lần đo'),
  });
}
