export interface AuthUser {
  id: number;
  email: string;
  full_name: string;
  phone: string;
  avatar_url: string;
  roles: string[];
  permissions: string[];
  /** Rỗng = làm việc trên toàn hệ thống. */
  branch_ids: number[];
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  /** Số giây access token còn hiệu lực. */
  expires_in: number;
  user: AuthUser;
}
