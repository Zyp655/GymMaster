import 'server-only';
import { getApiBaseUrl } from '@/env';
import { sendRequest } from '@/lib/http.core';
import type { LoginResponse } from '@/types/auth';

/** Các lời gọi auth phía server (dùng trong next-auth callbacks). */
export function loginRequest(email: string, password: string) {
  return sendRequest<LoginResponse>(getApiBaseUrl(), 'POST', '/auth/login', {
    body: { email, password },
  });
}

export function refreshRequest(refreshToken: string) {
  return sendRequest<LoginResponse>(getApiBaseUrl(), 'POST', '/auth/refresh', {
    body: { refresh_token: refreshToken },
  });
}

export function logoutRequest(accessToken: string, refreshToken: string) {
  return sendRequest<null>(getApiBaseUrl(), 'POST', '/auth/logout', {
    accessToken,
    body: { refresh_token: refreshToken },
  });
}
