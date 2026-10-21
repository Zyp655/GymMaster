'use client';

import { useCallback } from 'react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ApiCode, isApiError } from '@/types/api';

/** Khóa câu thông báo mặc định (common.errors.* + auth.errors.tooManyRequests). */
export type ApiErrorMessageKey =
  | 'network'
  | 'badRequest'
  | 'unauthorized'
  | 'forbidden'
  | 'notFound'
  | 'conflict'
  | 'internal'
  | 'unknown'
  | 'tooManyRequests';

/**
 * Mã 401 nghĩa là phiên đăng nhập hỏng (chưa đăng nhập / token hết hạn / refresh token sai).
 * Các mã 401 khác (40102 sai mật khẩu, 40104 OTP/token đặt lại sai, 40105 quá số lần thử,
 * 40107/40108 2FA…) là lỗi nghiệp vụ → hiển thị câu của server.
 */
const SESSION_ERROR_CODES: ReadonlySet<number> = new Set([
  ApiCode.UNAUTHORIZED,
  ApiCode.TOKEN_EXPIRED,
  ApiCode.REFRESH_TOKEN_INVALID,
]);

/**
 * Câu thông báo của server chỉ dùng khi là câu cho người đọc. Response không có envelope
 * (proxy/gateway trả 401/403/429) mang `statusText` tiếng Anh ("Too Many Requests") hoặc mã máy
 * ("NETWORK_ERROR") → bỏ qua để dùng câu tiếng Việt mặc định. Mọi câu của gymmaster-api đều là
 * tiếng Việt có dấu nên phép thử "có chữ cái có dấu" (Latin-1 → Latin Extended Additional) là đủ.
 */
function serverMessage(message: string | undefined): string | null {
  const text = message?.trim();
  if (!text) return null;
  return /[À-ỹ]/.test(text) ? text : null;
}

/**
 * Hàm thuần (dễ unit test): chuyển lỗi API thành câu thông báo tiếng Việt.
 * `t` trả câu mặc định theo khóa; `overrides` cho phép từng màn hình đặt câu riêng theo mã lỗi.
 */
export function resolveApiErrorMessage(
  error: unknown,
  t: (key: ApiErrorMessageKey) => string,
  overrides?: Partial<Record<number, string>>,
): string {
  if (!isApiError(error)) return t('unknown');
  const override = overrides?.[error.code];
  if (override) return override;
  // Lỗi validate (code 51) / bad request: backend trả câu tiếng Việt cụ thể.
  if (error.code === ApiCode.VALIDATION || error.code === ApiCode.BAD_REQUEST) {
    return error.message || t('badRequest');
  }
  if (error.code === ApiCode.NETWORK) return t('network');
  if (error.code === ApiCode.TOO_MANY_REQUESTS || error.status === 429) {
    return serverMessage(error.message) ?? t('tooManyRequests');
  }
  switch (error.status) {
    case 400:
      return error.message || t('badRequest');
    case 401:
      if (SESSION_ERROR_CODES.has(error.code)) return t('unauthorized');
      return serverMessage(error.message) ?? t('unknown');
    case 403:
      return serverMessage(error.message) ?? t('forbidden');
    case 404:
      return error.message || t('notFound');
    case 409:
      return error.message || t('conflict');
    default:
      return error.status >= 500 ? t('internal') : t('unknown');
  }
}

/**
 * Chuyển lỗi API thành câu thông báo tiếng Việt.
 * `overrides` cho phép từng màn hình đặt câu riêng theo mã lỗi (VD: 40900 → "Mã đã tồn tại").
 */
export function useApiErrorMessage() {
  const t = useTranslations('common.errors');
  const ta = useTranslations('auth.errors');

  return useCallback(
    (error: unknown, overrides?: Partial<Record<number, string>>): string =>
      resolveApiErrorMessage(
        error,
        key => (key === 'tooManyRequests' ? ta('tooManyRequests') : t(key)),
        overrides,
      ),
    [t, ta],
  );
}

/** Hiển thị toast lỗi cho một mutation. */
export function useToastApiError() {
  const getMessage = useApiErrorMessage();
  return useCallback(
    (error: unknown, overrides?: Partial<Record<number, string>>) => {
      toast.error(getMessage(error, overrides));
    },
    [getMessage],
  );
}
