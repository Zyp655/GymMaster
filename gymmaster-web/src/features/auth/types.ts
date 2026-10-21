import type { LoginResponse } from '@/types/auth';

export type { AuthUser, LoginResponse } from '@/types/auth';

export interface ChangePasswordPayload {
  old_password: string;
  new_password: string;
}

export interface ForgotPasswordPayload {
  email: string;
  portal?: 'WEB' | 'ADMIN';
}

export interface ForgotPasswordResponse {
  message?: string;
}

export interface ResetPasswordPayload {
  token: string;
  new_password: string;
}

export interface ResetPasswordResponse {
  message?: string;
}

/**
 * POST /auth/register-member. `email`/`password` bắt buộc khi tạo hội viên mới; khi SĐT trùng hồ sơ
 * có sẵn (luồng liên kết) backend bỏ qua cả hai – OTP gửi tới email trong hồ sơ, mật khẩu gửi ở bước verify.
 */
export interface RegisterMemberPayload {
  full_name: string;
  phone: string;
  email?: string;
  password?: string;
  branch_id?: number;
}

export interface RegisterMemberResponse {
  requires_otp: boolean;
  message: string;
  /** Email trong hồ sơ hội viên đã nhận OTP, đã che bớt (VD: "ng*******@gmail.com"). */
  masked_email?: string;
  /** Có khi tạo hội viên mới (đã đăng nhập). */
  token?: LoginResponse;
}

/** POST /auth/verify-member-otp – mật khẩu tài khoản mới đặt ở bước này (theo chính sách mật khẩu mới). */
export interface VerifyMemberOtpPayload {
  phone: string;
  otp: string;
  password: string;
}

/** TokenRes: tài khoản đã được tạo + liên kết hồ sơ, email đăng nhập = `user.email` (email trong hồ sơ). */
export type VerifyMemberOtpResponse = LoginResponse;

export interface PublicBranchOption {
  id: number;
  name: string;
  address: string;
  phone: string;
  open_time: string;
  close_time: string;
}
