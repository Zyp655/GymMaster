import type { FormInstance } from 'antd';
import { ErrorCode, isApiError } from './api';
import { feedback } from './feedback';

/** Thông điệp tiếng Việt mặc định theo mã lỗi (khi backend không trả message dễ hiểu). */
const DEFAULT_MESSAGES: Record<number, string> = {
  [ErrorCode.BAD_REQUEST]: 'Dữ liệu không hợp lệ',
  [ErrorCode.UNAUTHORIZED]: 'Bạn cần đăng nhập để tiếp tục',
  [ErrorCode.TOKEN_EXPIRED]: 'Phiên đăng nhập đã hết hạn',
  [ErrorCode.INVALID_CREDENTIALS]: 'Email hoặc mật khẩu không đúng',
  [ErrorCode.REFRESH_TOKEN_INVALID]: 'Phiên đăng nhập đã hết hạn, vui lòng đăng nhập lại',
  [ErrorCode.PERMISSION_DENIED]: 'Bạn không có quyền thực hiện thao tác này',
  [ErrorCode.ACCOUNT_LOCKED]: 'Tài khoản đã bị khóa, vui lòng liên hệ quản trị viên',
  [ErrorCode.NOT_FOUND]: 'Không tìm thấy dữ liệu',
  [ErrorCode.CONFLICT]: 'Dữ liệu bị trùng hoặc đang được sử dụng',
  [ErrorCode.INTERNAL]: 'Máy chủ gặp sự cố, vui lòng thử lại sau',
  [ErrorCode.NETWORK]: 'Không kết nối được máy chủ',
};

/** Message gốc tiếng Anh/mã máy của backend → không hiển thị cho người dùng. */
function looksTechnical(message: string): boolean {
  return /^[A-Z_]+$/.test(message) || /^(OK|Internal Error)$/i.test(message);
}

/** Lấy thông điệp lỗi tiếng Việt để hiển thị. */
export function getErrorMessage(error: unknown, fallback = 'Đã có lỗi xảy ra'): string {
  if (isApiError(error)) {
    if (error.message && !looksTechnical(error.message)) return error.message;
    return DEFAULT_MESSAGES[error.code] ?? fallback;
  }
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

/**
 * Hiện toast lỗi (bỏ qua nếu interceptor đã hiện – lỗi 5xx/mạng).
 * Dùng trong `onError` của mutation: `onError: (e) => showError(e, 'Không thể lưu')`.
 */
export function showError(error: unknown, fallback?: string): void {
  if (isApiError(error) && (error.handled || error.code === ErrorCode.CANCELED)) return;
  feedback.message.error(getErrorMessage(error, fallback));
}

/** Lỗi validate/nghiệp vụ nên hiển thị cạnh form (thay vì toast). */
export function isFormError(error: unknown): boolean {
  return (
    isApiError(error) &&
    (error.code === ErrorCode.VALIDATION ||
      error.code === ErrorCode.BAD_REQUEST ||
      error.code === ErrorCode.CONFLICT)
  );
}

/**
 * Gắn lỗi backend vào đúng ô của antd Form dựa trên từ khóa trong message.
 * Trả `true` nếu đã gắn được (khi đó không cần toast).
 *
 *   onError: (e) => {
 *     if (!setFormError(form, e, { code: /mã/i, email: /email/i })) showError(e, 'Không thể lưu');
 *   }
 */
export function setFormError<V>(
  form: FormInstance<V>,
  error: unknown,
  hints: Partial<Record<keyof V & string, RegExp>>,
): boolean {
  if (!isFormError(error)) return false;
  const msg = getErrorMessage(error);
  const field = (Object.keys(hints) as (keyof V & string)[]).find((name) => hints[name]?.test(msg));
  if (!field) return false;
  form.setFields([{ name: field as never, errors: [msg] }]);
  return true;
}

/** Lỗi validate phía client của antd (`form.validateFields()` reject `{ errorFields }`). */
export function isFieldValidationError(error: unknown): boolean {
  return (
    typeof error === 'object' &&
    error !== null &&
    Array.isArray((error as { errorFields?: unknown }).errorFields)
  );
}

/**
 * `form.validateFields()` an toàn: ô lỗi đã hiện dưới form → trả `null` thay vì reject
 * (tránh "Unhandled promise rejection" khi handler gắn trực tiếp vào onClick/onOk).
 *
 *   const v = await validateForm(form);
 *   if (!v) return;
 */
export async function validateForm<V>(form: FormInstance<V>): Promise<V | null> {
  try {
    return await form.validateFields();
  } catch (error) {
    if (isFieldValidationError(error)) return null;
    throw error;
  }
}
