/**
 * Ô nhập mã 2FA: chế độ `totp` (6 chữ số, bàn phím số) hoặc `recovery` (mã khôi phục dạng
 * `ABCD-1234` gồm chữ + số – backend sinh 8 ký tự hex in hoa, so khớp sau khi `ToUpper`).
 */
export type MfaCodeMode = 'totp' | 'recovery';

export const TOTP_LENGTH = 6;
/** `XXXX-XXXX`. */
export const RECOVERY_CODE_LENGTH = 9;

/** Thuộc tính cho `<Input>` theo chế độ (bàn phím điện thoại + giới hạn độ dài). */
export function mfaInputProps(mode: MfaCodeMode) {
  return mode === 'totp'
    ? {
        inputMode: 'numeric' as const,
        maxLength: TOTP_LENGTH,
        autoComplete: 'one-time-code',
        placeholder: 'Mã 6 số từ ứng dụng xác thực',
      }
    : {
        inputMode: 'text' as const,
        // Rộng hơn 9 để dán được mã có khoảng trắng/xuống dòng – normalize sẽ cắt lại.
        maxLength: 20,
        autoComplete: 'off',
        autoCapitalize: 'characters',
        placeholder: 'Mã khôi phục, vd: ABCD-1234',
      };
}

/** Chuẩn hóa giá trị người dùng nhập/dán theo chế độ. */
export function normalizeMfaCode(raw: string, mode: MfaCodeMode): string {
  if (mode === 'totp') return raw.replace(/\D/g, '').slice(0, TOTP_LENGTH);
  const chars = raw
    .toUpperCase()
    .replace(/[^A-Z0-9]/g, '')
    .slice(0, 8);
  return chars.length > 4 ? `${chars.slice(0, 4)}-${chars.slice(4)}` : chars;
}

/** Mã đã đủ để gửi lên server chưa. */
export function isMfaCodeComplete(code: string, mode: MfaCodeMode): boolean {
  return mode === 'totp'
    ? new RegExp(`^\\d{${TOTP_LENGTH}}$`).test(code)
    : /^[A-Z0-9]{4}-[A-Z0-9]{4}$/.test(code);
}
