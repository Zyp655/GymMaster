/**
 * Lõi HTTP client (thuần, không phụ thuộc React/antd/Vite) để có thể kiểm thử bằng Node.
 * Ứng dụng dùng instance đã cấu hình trong `src/utils/api.ts`.
 *
 * - Unwrap envelope `{code,message,data}` → trả thẳng `data`.
 * - Lỗi → ném `ApiError{code,status,message}`.
 * - 40101 (TOKEN_EXPIRED) → refresh token đúng MỘT lần; các request đồng thời chờ chung một lần
 *   refresh (backend xoay vòng refresh token nên gọi refresh 2 lần song song sẽ làm hỏng phiên).
 * - 40100 / 40103 → hết phiên → `onSessionExpired`.
 * - 5xx / mất kết nối → `onServerError` (trừ khi request đặt `silent: true`).
 */
import axios, { AxiosError } from 'axios';
import type { AxiosInstance, AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios';
import type { TokenStore } from './token';

export interface Envelope<T = unknown> {
  code: number;
  message: string;
  data: T;
}

/** Mã lỗi nghiệp vụ backend (trường `code` trong body; HTTP status chuẩn hóa theo nhóm mã – xem rescode). */
export const ErrorCode = {
  OK: 0,
  /** @deprecated Backend đã chuẩn hóa lỗi validate về 40000; giữ lại để tương thích. */
  VALIDATION: 51,
  BAD_REQUEST: 40000,
  UNAUTHORIZED: 40100,
  TOKEN_EXPIRED: 40101,
  INVALID_CREDENTIALS: 40102,
  REFRESH_TOKEN_INVALID: 40103,
  PERMISSION_DENIED: 40300,
  ACCOUNT_LOCKED: 40301,
  NOT_FOUND: 40400,
  CONFLICT: 40900,
  INTERNAL: 50000,
  /** Mã nội bộ phía FE: không kết nối được máy chủ. */
  NETWORK: -1,
  /** Mã nội bộ phía FE: request bị hủy chủ động (AbortController) – không phải lỗi. */
  CANCELED: -2,
} as const;

export class ApiError extends Error {
  readonly code: number;
  readonly status: number;
  readonly data: unknown;
  /** Đã được hiển thị bởi interceptor (tránh toast trùng). */
  handled = false;

  constructor(message: string, code: number, status: number, data?: unknown) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
    this.data = data;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

export interface AuthTokens<U = unknown> {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: U;
}

export interface ApiClientOptions<U = unknown> {
  baseURL: string;
  tokens: TokenStore;
  timeout?: number;
  /** Chi nhánh làm việc hiện tại → header `X-Branch-Id`. */
  getBranchId?: () => number | null;
  /** Hết phiên (refresh thất bại, 40100, 40103). */
  onSessionExpired?: (error: ApiError) => void;
  /** Refresh thành công (backend trả kèm user mới nhất). */
  onTokensRefreshed?: (data: AuthTokens<U>) => void;
  /** Lỗi 5xx hoặc mất kết nối. */
  onServerError?: (error: ApiError) => void;
}

export interface HttpClient {
  get<T>(url: string, config?: AxiosRequestConfig): Promise<T>;
  delete<T = null>(url: string, config?: AxiosRequestConfig): Promise<T>;
  post<T>(url: string, body?: unknown, config?: AxiosRequestConfig): Promise<T>;
  put<T>(url: string, body?: unknown, config?: AxiosRequestConfig): Promise<T>;
  patch<T>(url: string, body?: unknown, config?: AxiosRequestConfig): Promise<T>;
}

export interface ApiClient<U = unknown> {
  /** Axios instance (interceptor đã unwrap envelope: kết quả là `data`). */
  instance: AxiosInstance;
  /** Wrapper có kiểu cho instance. */
  http: HttpClient;
  /** Refresh access token (single-flight). Trả access token mới. */
  refreshAccessToken: () => Promise<string>;
  /** Số lần thực sự gọi /auth/refresh (phục vụ kiểm thử). */
  stats: { refreshCalls: number };
  /** Kiểu user để TS suy luận. */
  readonly _user?: U;
}

function isEnvelope(body: unknown): body is Envelope {
  return (
    typeof body === 'object' &&
    body !== null &&
    'code' in body &&
    typeof (body as { code: unknown }).code === 'number'
  );
}

export function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) return error;
  if (error instanceof AxiosError) {
    const res = error.response;
    if (res) {
      const body = res.data;
      if (isEnvelope(body)) {
        return new ApiError(body.message || 'Đã có lỗi xảy ra', body.code, res.status, body.data);
      }
      return new ApiError(
        res.status >= 500 ? 'Máy chủ gặp sự cố, vui lòng thử lại sau' : 'Đã có lỗi xảy ra',
        res.status * 100,
        res.status,
      );
    }
    if (error.code === AxiosError.ERR_CANCELED) {
      return new ApiError('Yêu cầu đã bị hủy', ErrorCode.CANCELED, 0);
    }
    if (error.code === AxiosError.ECONNABORTED || error.code === 'ETIMEDOUT') {
      return new ApiError('Máy chủ phản hồi quá lâu, vui lòng thử lại', ErrorCode.NETWORK, 0);
    }
    return new ApiError('Không kết nối được máy chủ', ErrorCode.NETWORK, 0);
  }
  if (error instanceof Error) return new ApiError(error.message, ErrorCode.NETWORK, 0);
  return new ApiError('Đã có lỗi xảy ra', ErrorCode.NETWORK, 0);
}

const SESSION_END_CODES: number[] = [ErrorCode.UNAUTHORIZED, ErrorCode.REFRESH_TOKEN_INVALID];

export function createApiClient<U = unknown>(options: ApiClientOptions<U>): ApiClient<U> {
  const { baseURL, tokens, timeout = 30000 } = options;
  const stats = { refreshCalls: 0 };

  const instance = axios.create({
    baseURL,
    timeout,
    headers: { Accept: 'application/json' },
  });

  // ---------------------------------------------------------------- refresh (single-flight)
  let refreshing: Promise<string> | null = null;

  const refreshAccessToken = (): Promise<string> => {
    if (refreshing) return refreshing;
    const refreshToken = tokens.getRefreshToken();
    if (!refreshToken) {
      return Promise.reject(
        new ApiError('Phiên đăng nhập đã hết hạn', ErrorCode.REFRESH_TOKEN_INVALID, 401),
      );
    }
    stats.refreshCalls += 1;
    refreshing = axios
      .post<Envelope<AuthTokens<U>>>(
        `${baseURL}/auth/refresh`,
        { refresh_token: refreshToken },
        { timeout, headers: { Accept: 'application/json' } },
      )
      .then((res) => {
        const body = res.data;
        if (!isEnvelope(body) || body.code !== ErrorCode.OK || !body.data?.access_token) {
          throw new ApiError(
            body?.message || 'Phiên đăng nhập đã hết hạn',
            body?.code ?? ErrorCode.REFRESH_TOKEN_INVALID,
            res.status,
          );
        }
        tokens.setTokens(body.data.access_token, body.data.refresh_token);
        options.onTokensRefreshed?.(body.data);
        return body.data.access_token;
      })
      .catch((error: unknown) => {
        const apiError = toApiError(error);
        // Lỗi xác thực khi refresh → coi như refresh token không còn hiệu lực.
        if (apiError.status === 401 || apiError.status === 400) {
          throw new ApiError(apiError.message, ErrorCode.REFRESH_TOKEN_INVALID, 401);
        }
        throw apiError;
      })
      .finally(() => {
        refreshing = null;
      });
    return refreshing;
  };

  const endSession = (error: ApiError) => {
    tokens.clear();
    options.onSessionExpired?.(error);
  };

  // ---------------------------------------------------------------- request
  instance.interceptors.request.use((config: InternalAxiosRequestConfig) => {
    const token = tokens.getAccessToken();
    if (token && !config.headers.has('Authorization')) {
      config.headers.set('Authorization', `Bearer ${token}`);
    }
    const branchId = options.getBranchId?.();
    if (branchId && !config.headers.has('X-Branch-Id')) {
      config.headers.set('X-Branch-Id', String(branchId));
    }
    return config;
  });

  // ---------------------------------------------------------------- response
  instance.interceptors.response.use(
    (response) => {
      const rt = response.config.responseType;
      if (rt === 'blob' || rt === 'arraybuffer' || rt === 'stream') return response.data;
      const body = response.data;
      if (isEnvelope(body)) {
        if (body.code !== ErrorCode.OK) {
          throw new ApiError(body.message, body.code, response.status, body.data);
        }
        return body.data;
      }
      return body;
    },
    async (rawError: unknown) => {
      const error = toApiError(rawError);
      const config = rawError instanceof AxiosError ? rawError.config : undefined;

      if (config && !config.skipAuthRefresh) {
        if (error.code === ErrorCode.TOKEN_EXPIRED && !config._retry) {
          config._retry = true;
          try {
            const usedHeader = String(config.headers?.get?.('Authorization') ?? '');
            const current = tokens.getAccessToken();
            // Một request khác đã refresh xong trong lúc request này đang bay → dùng luôn token mới.
            const token =
              current && usedHeader && usedHeader !== `Bearer ${current}`
                ? current
                : await refreshAccessToken();
            config.headers.set('Authorization', `Bearer ${token}`);
            return instance.request(config);
          } catch (refreshError) {
            const apiError = toApiError(refreshError);
            if (apiError.code === ErrorCode.REFRESH_TOKEN_INVALID || apiError.status === 401) {
              endSession(apiError);
            }
            throw apiError;
          }
        }
        if (
          SESSION_END_CODES.includes(error.code) ||
          (error.code === ErrorCode.TOKEN_EXPIRED && config._retry)
        ) {
          endSession(error);
          throw error;
        }
      }

      if ((error.status >= 500 || error.code === ErrorCode.NETWORK) && !config?.silent) {
        options.onServerError?.(error);
        error.handled = true;
      }
      throw error;
    },
  );

  const http: HttpClient = {
    get: (url, config) => instance.get(url, config) as never,
    delete: (url, config) => instance.delete(url, config) as never,
    post: (url, body, config) => instance.post(url, body, config) as never,
    put: (url, body, config) => instance.put(url, body, config) as never,
    patch: (url, body, config) => instance.patch(url, body, config) as never,
  };

  return { instance, http, refreshAccessToken, stats };
}
