/** Chính sách mật khẩu mới – giống `ValidateNewPassword` ở backend (auth_service.go). */
export const PASSWORD_POLICY_MESSAGE = 'Mật khẩu phải có ít nhất 8 ký tự, gồm cả chữ và số';

/** Trả thông báo lỗi nếu mật khẩu mới không đạt chính sách, ngược lại `null`. */
export function checkNewPassword(value: string | undefined | null): string | null {
  const v = value ?? '';
  if ([...v].length < 8 || !/\p{L}/u.test(v) || !/\p{Nd}/u.test(v)) return PASSWORD_POLICY_MESSAGE;
  if (new TextEncoder().encode(v).length > 72) return 'Mật khẩu tối đa 72 byte';
  return null;
}

/**
 * Lỗi đổi mật khẩu từ server hiển thị dưới ô nào: sai mật khẩu cũ (40102) hoặc thông báo nhắc
 * "hiện tại/cũ" → ô Mật khẩu hiện tại; còn lại (chính sách, trùng mật khẩu cũ...) → ô Mật khẩu mới.
 */
export function passwordErrorField(
  code: number,
  message: string,
  invalidCredentialsCode = 40102,
): 'old_password' | 'new_password' {
  if (code === invalidCredentialsCode) return 'old_password';
  if (/mới|new/i.test(message)) return 'new_password';
  if (/hiện tại|mật khẩu cũ|current|old/i.test(message)) return 'old_password';
  return 'new_password';
}
