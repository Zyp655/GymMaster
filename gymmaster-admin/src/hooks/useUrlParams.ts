import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

export type UrlValue = string | number | boolean | null | undefined | (string | number)[];

/**
 * Đọc/ghi query string của URL (nguồn sự thật cho bộ lọc + phân trang của trang danh sách).
 * Giá trị rỗng (undefined/null/''/[]) sẽ bị xóa khỏi URL. Mảng được nối bằng dấu phẩy.
 */
export function useUrlParams() {
  const [searchParams, setSearchParams] = useSearchParams();

  const values = useMemo(() => Object.fromEntries(searchParams.entries()), [searchParams]);

  const get = useCallback((key: string) => searchParams.get(key) ?? undefined, [searchParams]);

  const set = useCallback(
    (patch: Record<string, UrlValue>, options: { resetPage?: boolean } = {}) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          Object.entries(patch).forEach(([key, value]) => {
            const empty =
              value === undefined ||
              value === null ||
              value === '' ||
              (Array.isArray(value) && value.length === 0);
            if (empty) next.delete(key);
            else next.set(key, Array.isArray(value) ? value.join(',') : String(value));
          });
          if (options.resetPage) next.delete('page');
          return next;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  return { values, get, set };
}

export default useUrlParams;
