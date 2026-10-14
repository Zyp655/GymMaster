import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import inventoryApi from '@/api/inventory';
import type { AdjustmentInput, ImportInput, TransferInput } from '@/api/inventory';
import posApi from '@/api/pos';
import productApi from '@/api/product';

/** Sau mọi phiếu kho: làm mới tồn, thẻ kho, sản phẩm (giá vốn), POS. */
function useInvalidateStock() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: [inventoryApi.key] }),
      queryClient.invalidateQueries({ queryKey: [productApi.key] }),
      queryClient.invalidateQueries({ queryKey: [posApi.key] }),
    ]);
}

export function useImportStock() {
  const invalidate = useInvalidateStock();
  return useMutation({
    mutationFn: (data: ImportInput) => inventoryApi.import(data),
    onSuccess: invalidate,
  });
}

export function useTransferStock() {
  const invalidate = useInvalidateStock();
  return useMutation({
    mutationFn: (data: TransferInput) => inventoryApi.transfer(data),
    onSuccess: invalidate,
  });
}

export function useAdjustStock() {
  const invalidate = useInvalidateStock();
  return useMutation({
    mutationFn: (data: AdjustmentInput) => inventoryApi.adjust(data),
    onSuccess: invalidate,
  });
}

/** Sản phẩm cho ô chọn của phiếu kho (kèm tồn tại `branchId` nếu có). */
export function useProductOptions(branchId: number | null | undefined, enabled = true) {
  return useQuery({
    queryKey: [productApi.key, 'all', branchId ?? null],
    queryFn: () => productApi.listAll({ branch_id: branchId ?? undefined }),
    enabled,
    staleTime: 10_000,
  });
}
