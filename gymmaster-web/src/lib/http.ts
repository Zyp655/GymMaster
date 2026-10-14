'use client';

import { getCsrfToken, getSession, signOut } from 'next-auth/react';
import { getApiBaseUrl } from '@/env';
import { ApiCode, ApiError } from '@/types/api';
import { getClientAccessToken, setClientAccessToken } from './session-token';
import { sendRequest, type HttpMethod, type RequestOptions } from './http.core';

let refreshPromise: Promise<string | null> | null = null;
let signingOut = false;

async function currentAccessToken(): Promise<string | null> {
  const cached = getClientAccessToken();
  if (cached) return cached;
  const session = await getSession();
  const token = session?.accessToken ?? null;
  setClientAccessToken(token);
  return token;
}

/**
 * Ép next-auth refresh token: POST /api/auth/session với { forceRefresh: true }
 * (đúng cơ chế mà useSession().update() dùng). jwt callback sẽ gọi /auth/refresh của backend.
 * Nhiều request cùng lỗi 40101 chỉ kích hoạt MỘT lần refresh.
 */
function forceRefreshSession(): Promise<string | null> {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      try {
        const csrfToken = await getCsrfToken();
        const res = await fetch('/api/auth/session', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ csrfToken, data: { forceRefresh: true } }),
        });
        if (!res.ok) return null;
        const session = (await res.json()) as { accessToken?: string; error?: string } | null;
        if (!session?.accessToken || session.error) return null;
        setClientAccessToken(session.accessToken);
        return session.accessToken;
      } catch {
        return null;
      } finally {
        setTimeout(() => {
          refreshPromise = null;
        }, 0);
      }
    })();
  }
  return refreshPromise;
}

async function forceSignOut() {
  if (signingOut) return;
  signingOut = true;
  setClientAccessToken(null);
  const returnUrl = window.location.pathname + window.location.search;
  await signOut({ callbackUrl: `/login?returnUrl=${encodeURIComponent(returnUrl)}` });
}

async function request<T>(method: HttpMethod, path: string, options: RequestOptions = {}) {
  const baseUrl = getApiBaseUrl();
  const useAuth = options.auth !== false;
  const accessToken = useAuth
    ? (options.token ?? (await currentAccessToken()) ?? undefined)
    : undefined;

  try {
    return await sendRequest<T>(baseUrl, method, path, { ...options, accessToken });
  } catch (error) {
    if (!useAuth || !(error instanceof ApiError)) throw error;

    if (error.code === ApiCode.TOKEN_EXPIRED) {
      const newToken = await forceRefreshSession();
      if (!newToken) {
        await forceSignOut();
        throw error;
      }
      return sendRequest<T>(baseUrl, method, path, { ...options, accessToken: newToken });
    }
    if (error.code === ApiCode.UNAUTHORIZED || error.code === ApiCode.REFRESH_TOKEN_INVALID) {
      await forceSignOut();
    }
    throw error;
  }
}

/** HTTP client dùng trong Client Components / TanStack Query. */
export const apiClient = {
  get: <T>(path: string, options?: Omit<RequestOptions, 'body'>) =>
    request<T>('GET', path, options),
  post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>('POST', path, { ...options, body }),
  put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>('PUT', path, { ...options, body }),
  patch: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>('PATCH', path, { ...options, body }),
  del: <T>(path: string, options?: RequestOptions) => request<T>('DELETE', path, options),
};
