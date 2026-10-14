import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

/** Nhân viên & HLV – permission `staff.*` (docs/api/staff-rooms-shifts.md §1). */

export type StaffStatus = 'ACTIVE' | 'LOCKED';
export type StaffPosition = 'MANAGER' | 'RECEPTIONIST' | 'TRAINER' | 'OTHER';
export type Gender = 'MALE' | 'FEMALE' | 'OTHER';
export type TrainerLevel = 'JUNIOR' | 'SENIOR' | 'MASTER';

export interface StaffProfile {
  staff_code: string;
  position: StaffPosition;
  gender: Gender;
  dob: string | null;
  id_card: string;
  address: string;
  hire_date: string;
  base_salary: number;
  commission_rate: number;
  note: string;
}

export interface StaffRole {
  role_id: number;
  role_code: string;
  role_name: string;
  /** `null` = vai trò toàn hệ thống. */
  branch_id: number | null;
  branch_name: string;
}

export interface TrainerProfile {
  level: TrainerLevel;
  specialties: string[];
  bio: string;
  certifications: string;
  years_experience: number;
}

export interface Staff {
  id: number;
  email: string;
  phone: string;
  full_name: string;
  avatar_url: string;
  status: StaffStatus;
  last_login_at: string | null;
  created_at: string;
  updated_at: string;
  /** `null` với tài khoản quản trị cũ chưa có hồ sơ (VD OWNER seed). */
  staff_profile: StaffProfile | null;
  roles: StaffRole[];
  trainer_profile: TrainerProfile | null;
}

export interface RoleAssignment {
  role_code: string;
  /** `null` = toàn hệ thống (chỉ OWNER được gán). */
  branch_id: number | null;
}

export interface StaffInput {
  full_name: string;
  email: string;
  phone?: string;
  position: StaffPosition;
  gender?: Gender;
  dob?: string;
  id_card?: string;
  address?: string;
  hire_date?: string;
  base_salary?: number;
  commission_rate?: number;
  note?: string;
  trainer_profile?: TrainerProfile | null;
}

export interface StaffCreateInput extends StaffInput {
  password: string;
  roles: RoleAssignment[];
}

export interface StaffListParams extends ListParams {
  branch_id?: number;
  position?: StaffPosition;
  status?: StaffStatus;
}

export interface StaffOption {
  id: number;
  full_name: string;
  staff_code: string;
  position: StaffPosition;
}

export interface TrainerOption {
  id: number;
  full_name: string;
  level: TrainerLevel;
  specialties: string[];
}

class StaffApi {
  readonly key = 'staff';

  list(params: StaffListParams = {}) {
    return http.get<Page<Staff>>('/admin/staff', { params });
  }

  get(id: number) {
    return http.get<Staff>(`/admin/staff/${id}`);
  }

  create(data: StaffCreateInput) {
    return http.post<Staff>('/admin/staff', data);
  }

  update(id: number, data: StaffInput) {
    return http.put<Staff>(`/admin/staff/${id}`, data);
  }

  assignRoles(id: number, roles: RoleAssignment[]) {
    return http.put<Staff>(`/admin/staff/${id}/roles`, { roles });
  }

  lock(id: number) {
    return http.post<Staff>(`/admin/staff/${id}/lock`);
  }

  unlock(id: number) {
    return http.post<Staff>(`/admin/staff/${id}/unlock`);
  }

  resetPassword(id: number, newPassword: string) {
    return http.post<null>(`/admin/staff/${id}/reset-password`, { new_password: newPassword });
  }

  remove(id: number) {
    return http.delete(`/admin/staff/${id}`);
  }

  /** Nhân viên ACTIVE có hồ sơ (tối đa 500) – dropdown xếp ca. */
  options(branchId?: number | null) {
    return http.get<StaffOption[]>('/admin/staff/options', {
      params: { branch_id: branchId ?? undefined },
    });
  }

  /** HLV ACTIVE – dropdown lớp học / PT. */
  trainerOptions(branchId?: number | null) {
    return http.get<TrainerOption[]>('/admin/trainers/options', {
      params: { branch_id: branchId ?? undefined },
    });
  }
}

export const staffApi = new StaffApi();
export default staffApi;
