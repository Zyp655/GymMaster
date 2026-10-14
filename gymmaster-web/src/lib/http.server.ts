import 'server-only';
import { getApiBaseUrl } from '@/env';
import { getCachedSession } from '@/guards/getCachedSession';
import { sendRequest, type HttpMethod, type RequestOptions } from './http.core';

async function request<T>(method: HttpMethod, path: string, options: RequestOptions = {}) {
  const accessToken =
    options.auth === false
      ? undefined
      : (options.token ?? (await getCachedSession())?.accessToken ?? undefined);
  return sendRequest<T>(getApiBaseUrl(), method, path, { ...options, accessToken });
}

/** HTTP client dùng trong Server Components / Route Handlers. */
export const serverApi = {
  get: <T>(path: string, options?: Omit<RequestOptions, 'body'>) =>
    request<T>('GET', path, options),
  post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>('POST', path, { ...options, body }),
  put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>('PUT', path, { ...options, body }),
  del: <T>(path: string, options?: RequestOptions) => request<T>('DELETE', path, options),
};
