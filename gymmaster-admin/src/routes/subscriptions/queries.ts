import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import invoiceApi from '@/api/invoice';
import type { PaymentInput, RefundInput } from '@/api/invoice';
import memberApi from '@/api/member';
import subscriptionApi from '@/api/subscription';
import type {
  CancelInput,
  FreezeInput,
  RenewInput,
  SellInput,
  TransferInput,
  UpgradeInput,
} from '@/api/subscription';

/**
 * Mọi thao tác bán gói / hóa đơn đều ảnh hưởng chéo (gói ↔ hóa đơn ↔ sổ thu ↔ gói hiệu lực của
 * hội viên) nên làm mới cả 3 nhóm khóa.
 */
export function invalidateSales(queryClient: QueryClient) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: [subscriptionApi.key] }),
    queryClient.invalidateQueries({ queryKey: [invoiceApi.key] }),
    queryClient.invalidateQueries({ queryKey: [memberApi.key] }),
  ]);
}

function useSalesMutation<V, R>(fn: (v: V) => Promise<R>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => invalidateSales(queryClient),
  });
}

export const useSellSubscription = () =>
  useSalesMutation((data: SellInput) => subscriptionApi.sell(data));

export const useRenewSubscription = () =>
  useSalesMutation(({ id, data }: { id: number; data: RenewInput }) =>
    subscriptionApi.renew(id, data),
  );

export const useUpgradeSubscription = () =>
  useSalesMutation(({ id, data }: { id: number; data: UpgradeInput }) =>
    subscriptionApi.upgrade(id, data),
  );

export const useRequestFreeze = () =>
  useSalesMutation(({ id, data }: { id: number; data: FreezeInput }) =>
    subscriptionApi.requestFreeze(id, data),
  );

export const useReviewFreeze = () =>
  useSalesMutation(
    ({ id, action, note }: { id: number; action: 'approve' | 'reject' | 'end'; note?: string }) =>
      action === 'approve'
        ? subscriptionApi.approveFreeze(id, note)
        : action === 'reject'
          ? subscriptionApi.rejectFreeze(id, note)
          : subscriptionApi.endFreeze(id, note),
  );

export const useTransferSubscription = () =>
  useSalesMutation(({ id, data }: { id: number; data: TransferInput }) =>
    subscriptionApi.transfer(id, data),
  );

export const useCancelSubscription = () =>
  useSalesMutation(({ id, data }: { id: number; data: CancelInput }) =>
    subscriptionApi.cancel(id, data),
  );

export const useCollectPayment = () =>
  useSalesMutation(({ id, data }: { id: number; data: PaymentInput }) =>
    invoiceApi.collect(id, data),
  );

export const useVoidInvoice = () =>
  useSalesMutation(({ id, reason }: { id: number; reason: string }) => invoiceApi.void(id, reason));

export const useRefundInvoice = () =>
  useSalesMutation(({ id, data }: { id: number; data: RefundInput }) =>
    invoiceApi.refund(id, data),
  );
