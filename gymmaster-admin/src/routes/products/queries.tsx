import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App } from 'antd';
import inventoryApi from '@/api/inventory';
import posApi from '@/api/pos';
import productApi, { productCategoryApi, supplierApi } from '@/api/product';
import type { ProductCategoryInput, ProductInput, SupplierInput } from '@/api/product';
import { getErrorMessage } from '@/utils/errors';

/** Làm mới mọi dữ liệu liên quan sản phẩm (danh sách, POS, tồn kho). */
function useInvalidateCatalog() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: [productApi.key] }),
      queryClient.invalidateQueries({ queryKey: [productCategoryApi.key] }),
      queryClient.invalidateQueries({ queryKey: [posApi.key] }),
      queryClient.invalidateQueries({ queryKey: [inventoryApi.key] }),
    ]);
}

export function useSaveProduct() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: ({ id, data }: { id?: number; data: ProductInput }) =>
      id ? productApi.update(id, data) : productApi.create(data),
    onSuccess: invalidate,
  });
}

export function useSaveCategory() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: ({ id, data }: { id?: number; data: ProductCategoryInput }) =>
      id ? productCategoryApi.update(id, data) : productCategoryApi.create(data),
    onSuccess: invalidate,
  });
}

export function useSaveSupplier() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id?: number; data: SupplierInput }) =>
      id ? supplierApi.update(id, data) : supplierApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [supplierApi.key] }),
  });
}

type Named = { id: number; name: string };

/** Xóa nhiều mục, báo từng mục lỗi (vd. sản phẩm còn tồn, danh mục còn sản phẩm). */
function useDeleteMany(
  label: string,
  removeFn: (id: number) => Promise<unknown>,
  onDone: () => unknown,
) {
  const { message, modal } = App.useApp();
  return useMutation({
    mutationFn: async (items: Named[]) => {
      const results = await Promise.allSettled(items.map((x) => removeFn(x.id)));
      return results.map((r, i) => ({
        item: items[i],
        error: r.status === 'rejected' ? (r.reason as unknown) : null,
      }));
    },
    onSuccess: (results) => {
      const ok = results.filter((r) => !r.error).length;
      const failed = results.filter((r) => r.error);
      if (ok > 0) message.success(`Đã xóa ${ok} ${label}`);
      if (failed.length > 0) {
        modal.error({
          title: `Không xóa được ${failed.length} ${label}`,
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
      return onDone();
    },
  });
}

export function useDeleteProducts() {
  const invalidate = useInvalidateCatalog();
  return useDeleteMany('sản phẩm', (id) => productApi.remove(id), invalidate);
}

export function useDeleteCategories() {
  const invalidate = useInvalidateCatalog();
  return useDeleteMany('danh mục', (id) => productCategoryApi.remove(id), invalidate);
}

export function useDeleteSuppliers() {
  const queryClient = useQueryClient();
  return useDeleteMany(
    'nhà cung cấp',
    (id) => supplierApi.remove(id),
    () => queryClient.invalidateQueries({ queryKey: [supplierApi.key] }),
  );
}
