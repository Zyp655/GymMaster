import { QueryClient } from '@tanstack/react-query';
import { isApiError } from '@/utils/api';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnWindowFocus: false,
      // Không thử lại lỗi nghiệp vụ 4xx; lỗi mạng/5xx thử lại 1 lần.
      retry: (count, error) => {
        if (isApiError(error) && error.status >= 400 && error.status < 500) return false;
        return count < 1;
      },
    },
    mutations: { retry: false },
  },
});
