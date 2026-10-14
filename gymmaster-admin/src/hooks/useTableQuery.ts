import { useCallback, useMemo } from 'react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import type { QueryKey, UseQueryResult } from '@tanstack/react-query';
import type { SortOrder } from 'antd/es/table/interface';
import type { ListParams, Page, SortDir } from '@/api/types';
import { appConfig } from '@/config';
import { useUrlParams } from './useUrlParams';
import type { UrlValue } from './useUrlParams';

export interface UseTableQueryOptions<T, P extends ListParams> {
  /** Khóa gốc, ví dụ `[branchApi.key, 'list']`. Tham số được nối thêm tự động. */
  queryKey: QueryKey;
  /** Hàm gọi API danh sách, ví dụ `(p) => branchApi.list(p)`. */
  queryFn: (params: P) => Promise<Page<T>>;
  /** Các khóa lọc đọc từ URL ngoài page/limit/keyword/order_by/order_dir. */
  filterKeys?: readonly string[];
  /** Khóa lọc cần ép sang number (ví dụ `branch_id`, `room_id`). */
  numberKeys?: readonly string[];
  /** Giá trị mặc định khi URL chưa có. */
  defaults?: Partial<ListParams>;
  /** Tham số không nằm trên URL (ví dụ chi nhánh làm việc) – vẫn đưa vào queryKey. */
  extraParams?: Partial<P>;
  enabled?: boolean;
}

function toInt(value: string | undefined, fallback: number): number {
  const n = Number(value);
  return Number.isInteger(n) && n > 0 ? n : fallback;
}

/**
 * Xương sống của trang danh sách: đồng bộ phân trang + sắp xếp + lọc với URL
 * (`?page=2&limit=20&keyword=abc&order_by=name&order_dir=ASC&status=ACTIVE`)
 * và tải dữ liệu bằng TanStack Query. Dùng cùng `<DataTable table={table} />` và `<FilterBar />`.
 */
export interface TableQuery<T, P extends ListParams = ListParams> {
  /** Tham số đang áp dụng (đọc từ URL + mặc định + extraParams). */
  params: P;
  query: UseQueryResult<Page<T>, Error>;
  items: T[];
  total: number;
  isLoading: boolean;
  isFetching: boolean;
  isError: boolean;
  error: Error | null;
  refetch: () => unknown;
  /** Đặt bộ lọc (về trang 1). Giá trị rỗng = bỏ lọc. */
  setFilters: (patch: Record<string, UrlValue>) => void;
  setKeyword: (keyword: string) => void;
  setPage: (page: number, limit?: number) => void;
  setSort: (field?: string, dir?: SortDir) => void;
  /** Giá trị `sortOrder` cho cột antd (controlled). */
  sortOrderOf: (field: string) => SortOrder;
}

export function useTableQuery<T, P extends ListParams = ListParams>(
  options: UseTableQueryOptions<T, P>,
): TableQuery<T, P> {
  const {
    queryKey,
    queryFn,
    filterKeys = [],
    numberKeys = [],
    defaults = {},
    extraParams,
    enabled = true,
  } = options;
  const url = useUrlParams();
  const v = url.values;
  // Khóa ổn định cho mảng/obj do trang truyền vào (tránh tính lại mỗi render).
  const filterKeysKey = filterKeys.join(',');
  const numberKeysKey = numberKeys.join(',');
  const extraKey = JSON.stringify(extraParams ?? {});

  const params = useMemo(() => {
    const p: Record<string, unknown> = {
      page: toInt(v.page, 1),
      limit: toInt(v.limit, defaults.limit ?? appConfig.defaultPageSize),
      keyword: v.keyword || undefined,
      order_by: v.order_by || defaults.order_by,
      order_dir: (v.order_dir as SortDir | undefined) || defaults.order_dir,
    };
    filterKeys.forEach((key) => {
      const raw = v[key];
      if (raw === undefined || raw === '') return;
      p[key] = numberKeys.includes(key) ? Number(raw) : raw;
    });
    Object.entries(extraParams ?? {}).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') p[key] = value;
    });
    return p as P;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    v,
    filterKeysKey,
    numberKeysKey,
    extraKey,
    defaults.limit,
    defaults.order_by,
    defaults.order_dir,
  ]);

  const query = useQuery({
    queryKey: [...queryKey, params],
    queryFn: () => queryFn(params),
    placeholderData: keepPreviousData,
    enabled,
  });

  const setFilters = useCallback(
    (patch: Record<string, UrlValue>) => url.set(patch, { resetPage: true }),
    [url],
  );

  const setPage = useCallback(
    (page: number, limit?: number) =>
      url.set({ page: page > 1 ? page : undefined, limit: limit ?? undefined }),
    [url],
  );

  const setSort = useCallback(
    (field?: string, dir?: SortDir) =>
      url.set({ order_by: dir ? field : undefined, order_dir: dir }, { resetPage: true }),
    [url],
  );

  const sortOrderOf = useCallback(
    (field: string): SortOrder => {
      if (params.order_by !== field || !v.order_by) return null;
      return params.order_dir === 'ASC' ? 'ascend' : 'descend';
    },
    [params.order_by, params.order_dir, v.order_by],
  );

  const items = query.data?.items ?? [];
  const total = query.data?.total ?? 0;

  return {
    params,
    query,
    items,
    total,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
    setFilters,
    setKeyword: (keyword: string) => setFilters({ keyword }),
    setPage,
    setSort,
    sortOrderOf,
  };
}

export default useTableQuery;
