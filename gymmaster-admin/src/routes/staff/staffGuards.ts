import { ErrorCode, isApiError } from '@/utils/apiClient';

/**
 * Lỗi "chặn thao tác" của API nhân viên: 403 (nhân viên có quyền vượt quyền người thao tác)
 * hoặc 400 thao tác lên chính mình ("...chính mình"). Các lỗi này không thuộc ô nhập nào →
 * phải hiện nguyên thông điệp server (không gắn nhầm vào ô form theo từ khóa).
 */
export function isStaffGuardError(error: unknown): boolean {
  if (!isApiError(error)) return false;
  if (error.code === ErrorCode.PERMISSION_DENIED || error.status === 403) return true;
  const badRequest =
    error.code === ErrorCode.BAD_REQUEST ||
    error.code === ErrorCode.VALIDATION ||
    error.status === 400;
  return badRequest && /chính mình/i.test(error.message ?? '');
}

/** Hàng của chính người đang đăng nhập → không tự khóa/xóa/đặt lại mật khẩu/đổi vai trò (dùng trang Hồ sơ). */
export function isSelfStaff(
  staff: { id: number } | null | undefined,
  userId: number | null | undefined,
): boolean {
  return !!staff && !!userId && staff.id === userId;
}
