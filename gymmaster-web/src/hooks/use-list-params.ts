'use client';

import { useCallback, useMemo } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import type { ListParams } from '@/types/api';

export const PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const;

export type ListState<F extends string> = Required<Pick<ListParams, 'page' | 'limit'>> &
  Pick<ListParams, 'order_by' | 'order_dir'> & {
    keyword: string;
    filters: Partial<Record<F, string>>;
  };

/**
 * Trạng thái danh sách (trang, số dòng, tìm kiếm, sắp xếp, bộ lọc) lưu trên URL.
 * → F5 không mất bộ lọc, nút Back/Forward của trình duyệt hoạt động đúng, có thể chia sẻ link.
 * Component dùng hook này phải nằm trong <Suspense> (yêu cầu của useSearchParams).
 */
export function useListParams<F extends string = never>(filterKeys: readonly F[] = []) {
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();

  const state = useMemo<ListState<F>>(() => {
    const page = Math.max(1, Number(searchParams.get('page')) || 1);
    const rawLimit = Number(searchParams.get('limit'));
    const limit = (PAGE_SIZE_OPTIONS as readonly number[]).includes(rawLimit) ? rawLimit : 10;
    const dir = searchParams.get('order_dir');
    const filters: Partial<Record<F, string>> = {};
    for (const key of filterKeys) {
      const value = searchParams.get(key);
      if (value) filters[key] = value;
    }
    return {
      page,
      limit,
      keyword: searchParams.get('keyword') ?? '',
      order_by: searchParams.get('order_by') ?? undefined,
      order_dir: dir === 'ASC' || dir === 'DESC' ? dir : undefined,
      filters,
    };
    // filterKeys là hằng số do màn hình truyền vào
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  const update = useCallback(
    (patch: Record<string, string | number | undefined | null>, resetPage = true) => {
      const next = new URLSearchParams(searchParams.toString());
      for (const [key, value] of Object.entries(patch)) {
        if (value === undefined || value === null || value === '') next.delete(key);
        else next.set(key, String(value));
      }
      if (resetPage && !('page' in patch)) next.delete('page');
      const query = next.toString();
      router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false });
    },
    [pathname, router, searchParams],
  );

  const setFilter = useCallback(
    (key: F, value: string | undefined) => update({ [key]: value }),
    [update],
  );

  const clearFilters = useCallback(() => {
    const patch: Record<string, undefined> = { keyword: undefined };
    for (const key of filterKeys) patch[key] = undefined;
    update(patch);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [update]);

  const hasActiveFilters = state.keyword !== '' || Object.keys(state.filters).length > 0;

  /** Tham số gửi lên API. */
  const apiParams = useMemo(
    () => ({
      page: state.page,
      limit: state.limit,
      keyword: state.keyword || undefined,
      order_by: state.order_by,
      order_dir: state.order_dir,
      ...state.filters,
    }),
    [state],
  );

  return {
    state,
    apiParams,
    hasActiveFilters,
    setPage: (page: number) => update({ page: page <= 1 ? undefined : page }, false),
    setLimit: (limit: number) => update({ limit: limit === 10 ? undefined : limit }),
    setKeyword: (keyword: string) => update({ keyword }),
    setSort: (order_by?: string, order_dir?: 'ASC' | 'DESC') => update({ order_by, order_dir }),
    setFilter,
    clearFilters,
  };
}
