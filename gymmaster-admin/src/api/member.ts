import { http } from '@/utils/api';
import { ApiError, ErrorCode, isApiError } from '@/utils/api';
import type { ListParams, Page } from './types';

/** Hội viên, chỉ số cơ thể, nhập Excel – hợp đồng `docs/api/members.md`. */

export type Gender = 'MALE' | 'FEMALE' | 'OTHER';
export type MemberSource = 'WALK_IN' | 'REFERRAL' | 'FACEBOOK' | 'WEBSITE' | 'OTHER';
export type MemberStatus = 'ACTIVE' | 'INACTIVE' | 'BANNED';
export type SubscriptionFilter = 'active' | 'expiring' | 'none';

/** Gói đang hiệu lực (ACTIVE/FROZEN, `end_date` xa nhất). */
export interface MemberActiveSubscription {
  id: number;
  plan_id: number;
  plan_name: string;
  status: 'ACTIVE' | 'FROZEN';
  start_date: string;
  end_date: string;
  /** Chỉ có với gói theo buổi. */
  remaining_sessions: number | null;
  /** `end_date` − hôm nay (giờ VN). */
  days_left: number;
}

/** Dòng danh sách `GET /admin/members`. */
export interface MemberItem {
  id: number;
  member_code: string;
  full_name: string;
  gender: Gender;
  dob: string | null;
  phone: string;
  email: string;
  avatar_url: string;
  home_branch_id: number;
  home_branch_name: string;
  source: MemberSource;
  status: MemberStatus;
  has_portal_account: boolean;
  active_subscription: MemberActiveSubscription | null;
  created_at: string;
}

export interface PortalAccount {
  user_id: number;
  email: string;
  status: 'ACTIVE' | 'LOCKED';
  last_login_at: string | null;
}

export interface BodyMetric {
  id: number;
  member_id: number;
  measured_at: string;
  weight_kg: number;
  height_cm: number;
  body_fat_pct: number | null;
  muscle_mass_kg: number | null;
  waist_cm: number | null;
  chest_cm: number | null;
  hip_cm: number | null;
  bmi: number;
  note: string;
  measured_by: number;
  measured_by_name: string;
  created_at: string;
}

export interface BodyMetricInput {
  /** RFC3339, mặc định hiện tại, không ở tương lai. */
  measured_at?: string;
  weight_kg: number;
  height_cm: number;
  body_fat_pct?: number | null;
  muscle_mass_kg?: number | null;
  waist_cm?: number | null;
  chest_cm?: number | null;
  hip_cm?: number | null;
  note?: string;
}

/** Hồ sơ chi tiết `GET /admin/members/{id}`. */
export interface Member {
  id: number;
  member_code: string;
  full_name: string;
  gender: Gender;
  dob: string | null;
  phone: string;
  email: string;
  id_card: string;
  address: string;
  avatar_url: string;
  emergency_contact_name: string;
  emergency_contact_phone: string;
  home_branch_id: number;
  home_branch: { id: number; code: string; name: string } | null;
  source: MemberSource;
  status: MemberStatus;
  note: string;
  booking_blocked_until: string | null;
  portal_account: PortalAccount | null;
  latest_metric: BodyMetric | null;
  active_subscription: MemberActiveSubscription | null;
  stats: { total_checkins: number; last_checkin_at: string | null };
  created_at: string;
  updated_at: string;
}

/** Body POST/PUT (PUT gửi đầy đủ trường). */
export interface MemberInput {
  full_name: string;
  phone: string;
  gender?: Gender;
  dob?: string | null;
  email?: string;
  id_card?: string;
  address?: string;
  avatar_url?: string;
  emergency_contact_name?: string;
  emergency_contact_phone?: string;
  home_branch_id: number;
  source?: MemberSource;
  status?: MemberStatus;
  note?: string;
}

export interface MemberListParams extends ListParams {
  branch_id?: number;
  status?: MemberStatus;
  gender?: Gender;
  source?: MemberSource;
  subscription_status?: SubscriptionFilter;
}

/** Phần tử `GET /admin/members/options` (tối đa 20). */
export interface MemberOption {
  id: number;
  member_code: string;
  full_name: string;
  phone: string;
  avatar_url: string;
  status: MemberStatus;
}

export interface MemberImportResult {
  created: number;
  skipped: number;
  errors: { row: number; message: string }[];
}

export interface MemberQr {
  member_id: number;
  /** `GM1.<memberID>.<expUnix>.<sig>` */
  token: string;
  expires_at: string;
  ttl_seconds: number;
}

class MemberApi {
  readonly key = 'members';

  list(params: MemberListParams = {}) {
    return http.get<Page<MemberItem>>('/admin/members', { params });
  }

  get(id: number) {
    return http.get<Member>(`/admin/members/${id}`);
  }

  create(data: MemberInput) {
    return http.post<Member>('/admin/members', data);
  }

  update(id: number, data: MemberInput) {
    return http.put<Member>(`/admin/members/${id}`, data);
  }

  remove(id: number) {
    return http.delete(`/admin/members/${id}`);
  }

  /** Tìm nhanh cho ô chọn hội viên (quyền: member.view | subscription.sell | checkin.scan | ...). */
  options(keyword?: string, branchId?: number | null) {
    return http.get<MemberOption[]>('/admin/members/options', {
      params: { keyword: keyword || undefined, branch_id: branchId ?? undefined },
    });
  }

  grantPortal(id: number, data: { email: string; password: string }) {
    return http.post<PortalAccount>(`/admin/members/${id}/portal-account`, data);
  }

  lockPortal(id: number) {
    return http.delete(`/admin/members/${id}/portal-account`);
  }

  resetQr(id: number) {
    return http.post<null>(`/admin/members/${id}/reset-qr`);
  }

  /** Mã QR động để lễ tân hiển thị (quyền `checkin.scan`). */
  qr(id: number) {
    return http.get<MemberQr>(`/admin/members/${id}/qr`);
  }

  metrics(id: number) {
    return http.get<BodyMetric[]>(`/admin/members/${id}/body-metrics`);
  }

  addMetric(id: number, data: BodyMetricInput) {
    return http.post<BodyMetric>(`/admin/members/${id}/body-metrics`, data);
  }

  removeMetric(id: number, metricId: number) {
    return http.delete(`/admin/members/${id}/body-metrics/${metricId}`);
  }

  /**
   * File mẫu nhập Excel (Blob, không bọc envelope). Body lỗi là Blob nên interceptor không đọc
   * được message → thay bằng câu tiếng Việt theo HTTP status.
   */
  async importTemplate(): Promise<Blob> {
    try {
      return await http.get<Blob>('/admin/members/import-template', { responseType: 'blob' });
    } catch (error) {
      if (isApiError(error) && error.status === 403) {
        throw new ApiError('Bạn không có quyền tải file mẫu', ErrorCode.PERMISSION_DENIED, 403);
      }
      throw error;
    }
  }

  import(file: File, branchId: number) {
    const form = new FormData();
    form.append('file', file);
    form.append('branch_id', String(branchId));
    return http.post<MemberImportResult>('/admin/members/import', form, { timeout: 120000 });
  }
}

export const memberApi = new MemberApi();
export default memberApi;

export const GENDER_LABEL: Record<Gender, string> = { MALE: 'Nam', FEMALE: 'Nữ', OTHER: 'Khác' };
export const GENDER_OPTIONS = (Object.keys(GENDER_LABEL) as Gender[]).map((value) => ({
  value,
  label: GENDER_LABEL[value],
}));

export const SOURCE_LABEL: Record<MemberSource, string> = {
  WALK_IN: 'Vãng lai',
  REFERRAL: 'Giới thiệu',
  FACEBOOK: 'Facebook',
  WEBSITE: 'Website',
  OTHER: 'Khác',
};
export const SOURCE_OPTIONS = (Object.keys(SOURCE_LABEL) as MemberSource[]).map((value) => ({
  value,
  label: SOURCE_LABEL[value],
}));

export const MEMBER_STATUS_MAP: Record<MemberStatus, { label: string; color: string }> = {
  ACTIVE: { label: 'Đang hoạt động', color: 'green' },
  INACTIVE: { label: 'Ngừng', color: 'default' },
  BANNED: { label: 'Cấm', color: 'red' },
};
export const MEMBER_STATUS_OPTIONS = (Object.keys(MEMBER_STATUS_MAP) as MemberStatus[]).map(
  (value) => ({ value, label: MEMBER_STATUS_MAP[value].label }),
);

export const SUBSCRIPTION_FILTER_OPTIONS: { value: SubscriptionFilter; label: string }[] = [
  { value: 'active', label: 'Có gói hiệu lực' },
  { value: 'expiring', label: 'Sắp hết hạn (≤ 7 ngày)' },
  { value: 'none', label: 'Chưa có gói' },
];
