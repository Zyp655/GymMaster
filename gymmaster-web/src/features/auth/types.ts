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

export interface RegisterMemberPayload {
  full_name: string;
  phone: string;
  email: string;
  password: string;
  branch_id: number;
}

export interface RegisterMemberResponse {
  requires_otp?: boolean;
  message?: string;
  token?: LoginResponse;
}

export interface VerifyMemberOtpPayload {
  phone: string;
  email: string;
  otp: string;
}

export type VerifyMemberOtpResponse = LoginResponse & {
  message?: string;
};

export interface PublicBranchOption {
  id: number;
  name: string;
  address: string;
  phone: string;
  open_time: string;
  close_time: string;
}
