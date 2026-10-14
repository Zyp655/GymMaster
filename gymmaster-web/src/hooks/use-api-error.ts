'use client';

import { useCallback } from 'react';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ApiCode, isApiError } from '@/types/api';

/**
 * Chuyển lỗi API thành câu thông báo tiếng Việt.
 * `overrides` cho phép từng màn hình đặt câu riêng theo mã lỗi (VD: 40900 → "Mã đã tồn tại").
 */
export function useApiErrorMessage() {
  const t = useTranslations('common.errors');

  return useCallback(
    (error: unknown, overrides?: Partial<Record<number, string>>): string => {
      if (!isApiError(error)) return t('unknown');
      const override = overrides?.[error.code];
      if (override) return override;
      // Lỗi validate (code 51) / bad request: backend trả câu tiếng Việt cụ thể.
      if (error.code === ApiCode.VALIDATION || error.code === ApiCode.BAD_REQUEST) {
        return error.message || t('badRequest');
      }
      if (error.code === ApiCode.NETWORK) return t('network');
      switch (error.status) {
        case 400:
          return error.message || t('badRequest');
        case 401:
          return t('unauthorized');
        case 403:
          return t('forbidden');
        case 404:
          return error.message || t('notFound');
        case 409:
          return error.message || t('conflict');
        default:
          return error.status >= 500 ? t('internal') : t('unknown');
      }
    },
    [t],
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
