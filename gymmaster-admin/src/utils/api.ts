/**
 * Axios instance DUY NHẤT của ứng dụng. Mọi file `src/api/*.ts` dùng `http` từ đây.
 *
 *   const page = await http.get<Page<Branch>>('/admin/branches', { params });
 *
 * Interceptor đã unwrap envelope nên kết quả là `data`; lỗi luôn là `ApiError`.
 * Các callback phiên (hết phiên, refresh) được gắn ở `src/store/session.ts` qua `configureSession`.
 */
import { appConfig } from '@/config';
import { createApiClient } from './apiClient';
import type { AuthTokens } from './apiClient';
import { tokenStore } from './token';
import { feedback } from './feedback';
import type { AuthUser } from '@/api/types';

export { ApiError, ErrorCode, isApiError } from './apiClient';
export type { Envelope } from './apiClient';

interface SessionHandlers {
  getBranchId: () => number | null;
  onSessionExpired: () => void;
  onTokensRefreshed: (data: AuthTokens<AuthUser>) => void;
}

const handlers: SessionHandlers = {
  getBranchId: () => null,
  onSessionExpired: () => undefined,
  onTokensRefreshed: () => undefined,
};

/** Gắn các callback phụ thuộc store (gọi một lần ở main.tsx, tránh import vòng). */
export function configureSession(next: Partial<SessionHandlers>): void {
  Object.assign(handlers, next);
}

const client = createApiClient<AuthUser>({
  baseURL: appConfig.apiUrl,
  tokens: tokenStore,
  getBranchId: () => handlers.getBranchId(),
  onSessionExpired: () => handlers.onSessionExpired(),
  onTokensRefreshed: (data) => handlers.onTokensRefreshed(data),
  onServerError: (error) => {
    feedback.message.error({ content: error.message, key: 'server-error' });
  },
});

export const api = client.instance;
export const http = client.http;
export const refreshAccessToken = client.refreshAccessToken;
export default api;
