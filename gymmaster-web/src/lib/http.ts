'use client';

import { getCsrfToken, getSession, signOut } from 'next-auth/react';
import { getApiBaseUrl } from '@/env';
import { ApiCode, ApiError } from '@/types/api';
import { getClientAccessToken, setClientAccessToken } from './session-token';
import { sendRawRequest, sendRequest, type HttpMethod, type RequestOptions } from './http.core';

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

/** Gắn access token, tự refresh khi 40101 rồi thử lại một lần; 40100/40103 → đăng xuất. */
async function withAuth<R>(
  options: Pick<RequestOptions, 'auth' | 'token'>,
  run: (accessToken: string | undefined) => Promise<R>,
): Promise<R> {
  const useAuth = options.auth !== false;
  const accessToken = useAuth
    ? (options.token ?? (await currentAccessToken()) ?? undefined)
    : undefined;

  try {
    return await run(accessToken);
  } catch (error) {
    if (!useAuth || !(error instanceof ApiError)) throw error;

    if (error.code === ApiCode.TOKEN_EXPIRED) {
      const newToken = await forceRefreshSession();
      if (!newToken) {
        await forceSignOut();
        throw error;
      }
      return run(newToken);
    }
    if (error.code === ApiCode.UNAUTHORIZED || error.code === ApiCode.REFRESH_TOKEN_INVALID) {
      await forceSignOut();
    }
    throw error;
  }
}

function request<T>(method: HttpMethod, path: string, options: RequestOptions = {}) {
  const baseUrl = getApiBaseUrl();
  return withAuth(options, accessToken =>
    sendRequest<T>(baseUrl, method, path, { ...options, accessToken }),
  );
}

/** Tải file (GET, có xác thực) → Blob + tên file từ Content-Disposition nếu đọc được. */
async function download(
  path: string,
  options: Omit<RequestOptions, 'body'> = {},
): Promise<{ blob: Blob; filename: string | null }> {
  const baseUrl = getApiBaseUrl();
  const response = await withAuth(options, accessToken =>
    sendRawRequest(baseUrl, 'GET', path, { ...options, accessToken }),
  );
  const disposition = response.headers.get('Content-Disposition') ?? '';
  const match = /filename\*?=(?:UTF-8'')?"?([^";]+)"?/i.exec(disposition);
  return { blob: await response.blob(), filename: match ? decodeURIComponent(match[1]) : null };
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
  download,
};
