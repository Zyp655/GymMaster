import { apiClient } from '@/lib/http';
import type { AuthUser } from '@/types/auth';
import type {
  ChangePasswordPayload,
  ForgotPasswordPayload,
  ForgotPasswordResponse,
  PublicBranchOption,
  RegisterMemberPayload,
  RegisterMemberResponse,
  ResetPasswordPayload,
  ResetPasswordResponse,
  VerifyMemberOtpPayload,
  VerifyMemberOtpResponse,
} from './types';

/** Lời gọi auth phía client. Login/refresh/logout đi qua next-auth (xem ./server.ts). */
export const authApi = {
  me: () => apiClient.get<AuthUser>('/auth/me'),
  changePassword: (payload: ChangePasswordPayload) =>
    apiClient.put<null>('/auth/change-password', payload),
  forgotPassword: (payload: ForgotPasswordPayload) =>
    apiClient.post<ForgotPasswordResponse>(
      '/auth/forgot-password',
      { portal: 'WEB', ...payload },
      { auth: false },
    ),
  resetPassword: (payload: ResetPasswordPayload) =>
    apiClient.post<ResetPasswordResponse>('/auth/reset-password', payload, { auth: false }),
  registerMember: (payload: RegisterMemberPayload) =>
    apiClient.post<RegisterMemberResponse>('/auth/register-member', payload, { auth: false }),
  verifyMemberOtp: (payload: VerifyMemberOtpPayload) =>
    apiClient.post<VerifyMemberOtpResponse>('/auth/verify-member-otp', payload, { auth: false }),
  publicBranches: () => apiClient.get<PublicBranchOption[]>('/public/branches', { auth: false }),
};
