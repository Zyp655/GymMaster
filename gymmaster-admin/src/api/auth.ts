import { http } from '@/utils/api';
import type { AuthUser, LoginResult } from './types';

export interface ForgotPasswordInput {
  email: string;
  portal?: 'ADMIN' | 'MEMBER' | string;
}

export interface ResetPasswordInput {
  token: string;
  new_password: string;
}

class AuthApi {
  login(email: string, password: string) {
    return http.post<LoginResult>(
      '/auth/login',
      { email, password },
      { skipAuthRefresh: true, silent: false },
    );
  }

  me() {
    return http.get<AuthUser>('/auth/me');
  }

  logout(refreshToken: string) {
    return http.post<null>(
      '/auth/logout',
      { refresh_token: refreshToken },
      { skipAuthRefresh: true, silent: true },
    );
  }

  changePassword(oldPassword: string, newPassword: string) {
    return http.put<null>('/auth/change-password', {
      old_password: oldPassword,
      new_password: newPassword,
    });
  }

  forgotPassword(input: ForgotPasswordInput | string) {
    const payload =
      typeof input === 'string'
        ? { email: input, portal: 'ADMIN' }
        : { portal: 'ADMIN', ...input };
    return http.post<{ reset_token?: string } | null>('/auth/forgot-password', payload, {
      skipAuthRefresh: true,
      silent: false,
    });
  }

  resetPassword(input: ResetPasswordInput) {
    return http.post<null>('/auth/reset-password', input, {
      skipAuthRefresh: true,
      silent: false,
    });
  }
}

export const authApi = new AuthApi();
export const forgotPassword = (input: ForgotPasswordInput | string) =>
  authApi.forgotPassword(input);
export const resetPassword = (input: ResetPasswordInput) => authApi.resetPassword(input);
export default authApi;
