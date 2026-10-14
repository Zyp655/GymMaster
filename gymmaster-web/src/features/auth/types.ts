export type { AuthUser, LoginResponse } from '@/types/auth';

export interface ChangePasswordPayload {
  old_password: string;
  new_password: string;
}
