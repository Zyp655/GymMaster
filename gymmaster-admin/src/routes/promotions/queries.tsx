import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App } from 'antd';
import promotionApi from '@/api/promotion';
import type { PromotionCheckInput, PromotionInput } from '@/api/promotion';
import { ErrorCode, isApiError } from '@/utils/api';
import { getErrorMessage } from '@/utils/errors';

export function useSavePromotion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id?: number; data: PromotionInput }) =>
      id ? promotionApi.update(id, data) : promotionApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [promotionApi.key] }),
  });
}

export function useDeletePromotion() {
  const queryClient = useQueryClient();
  const { message, modal } = App.useApp();
  return useMutation({
    mutationFn: (item: { id: number; code: string }) => promotionApi.remove(item.id),
    onSuccess: () => {
      message.success('Đã xóa khuyến mãi');
      return queryClient.invalidateQueries({ queryKey: [promotionApi.key] });
    },
    onError: (e, item) => {
      if (isApiError(e) && e.code === ErrorCode.CONFLICT) {
        modal.warning({ title: `Không thể xóa mã ${item.code}`, content: e.message });
        return;
      }
      message.error(getErrorMessage(e, 'Không thể xóa khuyến mãi'));
    },
  });
}

export function useCheckPromotion() {
  return useMutation({ mutationFn: (data: PromotionCheckInput) => promotionApi.check(data) });
}
