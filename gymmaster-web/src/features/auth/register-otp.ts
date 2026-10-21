import { ApiCode, isApiError } from '@/types/api';
import type { VerifyMemberOtpPayload } from './types';

/** Backend chặn yêu cầu OTP mới trong 60 giây sau lần gửi trước (42900). */
export const OTP_RESEND_COOLDOWN_SECONDS = 60;

/**
 * Số giây chờ trước khi được gửi lại OTP, đọc từ câu 429 của server
 * ("...đợi 60 giây...", "...thử lại sau 1 giờ"); không đọc được → `fallback`.
 */
export function parseRetryAfterSeconds(
  message: string | undefined,
  fallback = OTP_RESEND_COOLDOWN_SECONDS,
): number {
  const match = message?.match(/(\d+)\s*(giây|phút|giờ)/i);
  if (!match) return fallback;
  const value = Number(match[1]);
  if (!Number.isFinite(value) || value <= 0) return fallback;
  const unit = match[2].toLowerCase();
  if (unit === 'giờ') return value * 3600;
  if (unit === 'phút') return value * 60;
  return value;
}

/** Đếm ngược dạng m:ss (VD: 59 → "0:59", 3600 → "60:00"). */
export function formatCountdown(totalSeconds: number): string {
  const s = Math.max(0, Math.ceil(totalSeconds));
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}

/** So sánh SĐT bỏ khoảng trắng/dấu chấm/gạch và quy +84 về 0 (giống chuẩn hóa phía backend). */
export function normalizePhone(phone: string): string {
  const compact = phone.replace(/[\s.-]+/g, '');
  return compact.startsWith('+84') ? `0${compact.slice(3)}` : compact;
}

export function isSamePhone(a: string, b: string): boolean {
  return normalizePhone(a) === normalizePhone(b);
}

/**
 * Body POST /auth/verify-member-otp: mật khẩu người dùng đã nhập ở form đăng ký được gửi ở bước này
 * (backend không lưu mật khẩu ở bước register-member); email không gửi – tài khoản dùng email trong hồ sơ.
 */
export function buildVerifyMemberOtpPayload(
  request: { phone: string; password: string },
  otp: string,
): VerifyMemberOtpPayload {
  return { phone: request.phone, otp: otp.trim(), password: request.password };
}

/**
 * Phân loại lỗi POST /auth/verify-member-otp để hiển thị đúng:
 * - `invalid`: 40104 – OTP sai/hết hạn → ở lại màn OTP, nhập lại.
 * - `locked`: 40105 – sai quá 5 lần, mã bị vô hiệu → mời "Gửi lại mã".
 * - `throttled`: 42900/HTTP 429.
 * - `other`: còn lại (409 SĐT đã có tài khoản, 400 mật khẩu không đạt chính sách, mạng…).
 */
export type VerifyOtpErrorKind = 'invalid' | 'locked' | 'throttled' | 'other';

export function classifyVerifyOtpError(error: unknown): VerifyOtpErrorKind {
  if (!isApiError(error)) return 'other';
  if (error.code === ApiCode.RESET_TOKEN_INVALID) return 'invalid';
  if (error.code === ApiCode.TOO_MANY_ATTEMPTS) return 'locked';
  if (error.code === ApiCode.TOO_MANY_REQUESTS || error.status === 429) return 'throttled';
  return 'other';
}

/** Lỗi 429 của register-member (gửi lại OTP quá sớm / quá 5 lần mỗi giờ). */
export function isThrottleError(error: unknown): boolean {
  return isApiError(error) && (error.code === ApiCode.TOO_MANY_REQUESTS || error.status === 429);
}
