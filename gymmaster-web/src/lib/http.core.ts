import { ApiCode, ApiError, type ApiEnvelope } from '@/types/api';

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
export type QueryValue = string | number | boolean | null | undefined;

export interface RequestOptions {
  query?: Record<string, QueryValue>;
  body?: unknown;
  /** false = endpoint public, không gắn Bearer token. */
  auth?: boolean;
  /** Ghi đè access token (dùng trong next-auth callbacks). */
  token?: string;
  signal?: AbortSignal;
  headers?: Record<string, string>;
}

export function buildUrl(baseUrl: string, path: string, query?: Record<string, QueryValue>) {
  const url = new URL(baseUrl.replace(/\/$/, '') + (path.startsWith('/') ? path : `/${path}`));
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value === undefined || value === null || value === '') continue;
      url.searchParams.set(key, String(value));
    }
  }
  return url.toString();
}

/**
 * Gửi một request tới backend và bóc envelope {code, message, data}.
 * Ném ApiError nếu code != 0 hoặc không kết nối được.
 */
export async function sendRequest<T>(
  baseUrl: string,
  method: HttpMethod,
  path: string,
  options: RequestOptions & { accessToken?: string } = {},
): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json', ...options.headers };
  const isFormData = typeof FormData !== 'undefined' && options.body instanceof FormData;
  if (options.body !== undefined && !isFormData) headers['Content-Type'] = 'application/json';
  if (options.accessToken) headers.Authorization = `Bearer ${options.accessToken}`;

  let response: Response;
  try {
    response = await fetch(buildUrl(baseUrl, path, options.query), {
      method,
      headers,
      body:
        options.body === undefined
          ? undefined
          : isFormData
            ? (options.body as FormData)
            : JSON.stringify(options.body),
      signal: options.signal,
      cache: 'no-store',
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error;
    throw new ApiError(ApiCode.NETWORK, 0, 'NETWORK_ERROR');
  }

  let envelope: ApiEnvelope<T> | null = null;
  try {
    envelope = (await response.json()) as ApiEnvelope<T>;
  } catch {
    envelope = null;
  }

  if (!envelope || typeof envelope.code !== 'number') {
    if (!response.ok) {
      throw new ApiError(response.status * 100, response.status, response.statusText);
    }
    throw new ApiError(ApiCode.INTERNAL, response.status, 'INVALID_RESPONSE');
  }

  if (envelope.code !== ApiCode.OK) {
    throw new ApiError(envelope.code, response.status, envelope.message);
  }
  return envelope.data;
}
