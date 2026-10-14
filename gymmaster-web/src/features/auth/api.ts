import { apiClient } from '@/lib/http';
import type { AuthUser } from '@/types/auth';
import type { ChangePasswordPayload } from './types';

/** Lời gọi auth phía client. Login/refresh/logout đi qua next-auth (xem ./server.ts). */
export const authApi = {
  me: () => apiClient.get<AuthUser>('/auth/me'),
  changePassword: (payload: ChangePasswordPayload) =>
    apiClient.put<null>('/auth/change-password', payload),
};
