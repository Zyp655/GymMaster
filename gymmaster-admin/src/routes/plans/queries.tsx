import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App } from 'antd';
import membershipPlanApi from '@/api/plan';
import type { MembershipPlanInput } from '@/api/plan';
import ptPackageApi from '@/api/ptPackage';
import type { PtPackageInput } from '@/api/ptPackage';
import type { ActiveStatus } from '@/api/types';
import { ErrorCode, isApiError } from '@/utils/api';
import { showError } from '@/utils/errors';

export function useSavePlan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id?: number; data: MembershipPlanInput }) =>
      id ? membershipPlanApi.update(id, data) : membershipPlanApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [membershipPlanApi.key] }),
  });
}

export function useSavePtPackage() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id?: number; data: PtPackageInput }) =>
      id ? ptPackageApi.update(id, data) : ptPackageApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [ptPackageApi.key] }),
  });
}

type Kind = 'plan' | 'pt';
interface SaleApi {
  key: string;
  setStatus: (id: number, status: ActiveStatus) => Promise<unknown>;
  remove: (id: number) => Promise<unknown>;
}
const apiOf = (kind: Kind): SaleApi => (kind === 'plan' ? membershipPlanApi : ptPackageApi);

/** Đổi trạng thái nhanh (Switch) – `PUT .../{id}/status`. */
export function useSetSaleStatus(kind: Kind) {
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const api = apiOf(kind);
  return useMutation({
    mutationFn: ({ id, status }: { id: number; status: ActiveStatus }) => api.setStatus(id, status),
    onSuccess: (_, v) => {
      message.success(v.status === 'ACTIVE' ? 'Đã mở bán gói' : 'Đã chuyển sang Ngừng bán');
      return queryClient.invalidateQueries({ queryKey: [api.key] });
    },
    onError: (e) => showError(e, 'Không thể đổi trạng thái'),
  });
}

/**
 * Xóa gói. Gói đã bán (40900) → hỏi chuyển sang Ngừng bán thay vì xóa.
 */
export function useDeleteSalePackage(kind: Kind) {
  const queryClient = useQueryClient();
  const { message, modal } = App.useApp();
  const api = apiOf(kind);
  const setStatus = useSetSaleStatus(kind);
  return useMutation({
    mutationFn: (item: { id: number; name: string; status: ActiveStatus }) => api.remove(item.id),
    onSuccess: () => {
      message.success('Đã xóa gói');
      return queryClient.invalidateQueries({ queryKey: [api.key] });
    },
    onError: (e, item) => {
      if (isApiError(e) && e.code === ErrorCode.CONFLICT) {
        modal.confirm({
          title: `Không thể xóa "${item.name}"`,
          content: e.message,
          okText: item.status === 'ACTIVE' ? 'Chuyển sang Ngừng bán' : 'Đóng',
          cancelText: 'Hủy',
          cancelButtonProps: item.status === 'ACTIVE' ? undefined : { style: { display: 'none' } },
          onOk: () =>
            item.status === 'ACTIVE'
              ? setStatus.mutateAsync({ id: item.id, status: 'INACTIVE' }).catch(() => undefined)
              : undefined,
        });
        return;
      }
      showError(e, 'Không thể xóa gói');
    },
  });
}
