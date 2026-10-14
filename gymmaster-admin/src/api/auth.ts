import { http } from '@/utils/api';
import type { AuthUser, LoginResult } from './types';

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
}

export const authApi = new AuthApi();
export default authApi;
