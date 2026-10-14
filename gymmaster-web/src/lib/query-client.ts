import { QueryClient } from '@tanstack/react-query';
import { isApiError } from '@/types/api';

export function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        refetchOnWindowFocus: false,
        // Không retry lỗi 4xx (sai quyền, không tồn tại...) – chỉ retry lỗi mạng/5xx.
        retry: (failureCount, error) => {
          if (isApiError(error) && error.status >= 400 && error.status < 500) return false;
          return failureCount < 2;
        },
      },
      mutations: { retry: false },
    },
  });
}
