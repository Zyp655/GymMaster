import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

/** Check-in / ra vào – hợp đồng `docs/api/checkin.md`. */

export type CheckinResult = 'ALLOWED' | 'DENIED';
export type CheckinMethod = 'QR' | 'MANUAL';
export type DenyReason =
  | ''
  | 'QR_INVALID'
  | 'QR_EXPIRED'
  | 'BRANCH_CLOSED'
  | 'MEMBER_INACTIVE'
  | 'DUPLICATE'
  | 'SUBSCRIPTION_FROZEN'
  | 'NO_SESSIONS_LEFT'
  | 'WRONG_BRANCH'
  | 'NO_ACTIVE_SUBSCRIPTION';

export interface ScanMember {
  id: number;
  code: string;
  full_name: string;
  avatar_url: string;
  phone: string;
  status: string;
}

export interface ScanSubscription {
  id: number;
  plan_name: string;
  type: 'TIME' | 'SESSION';
  status: 'ACTIVE' | 'FROZEN';
  end_date: string;
  /** 0 = hết hạn hôm nay. */
  days_left: number;
  /** Số buổi còn lại SAU lần vào này; `null` với gói TIME. */
  remaining_sessions: number | null;
}

export interface ScanCheckin {
  id: number;
  checked_in_at: string;
  branch_id: number;
  branch_name: string;
}

/** Kết quả quét/thủ công. DENIED không phải lỗi (HTTP 200). */
export interface ScanResult {
  result: CheckinResult;
  deny_reason: DenyReason;
  deny_message: string;
  member: ScanMember | null;
  subscription: ScanSubscription | null;
  checkin: ScanCheckin | null;
  /** Chỉ có khi DUPLICATE. */
  previous_checkin: ScanCheckin | null;
  warnings: string[];
}

export interface CheckinItem {
  id: number;
  member_id: number;
  member_code: string;
  member_name: string;
  member_phone: string;
  member_avatar_url: string;
  branch_id: number;
  branch_name: string;
  subscription_id: number | null;
  plan_name: string | null;
  plan_type: 'TIME' | 'SESSION' | null;
  checked_in_at: string;
  checked_out_at: string | null;
  method: CheckinMethod;
  result: CheckinResult;
  deny_reason: DenyReason;
  deny_message: string;
  staff_id: number | null;
  staff_name: string | null;
  note: string;
}

export interface CheckinListParams extends ListParams {
  branch_id?: number;
  member_id?: number;
  result?: CheckinResult;
  method?: CheckinMethod;
  /** YYYY-MM-DD */
  from?: string;
  to?: string;
}

export interface PresentItem {
  checkin_id: number;
  member_id: number;
  member_code: string;
  member_name: string;
  member_phone: string;
  member_avatar_url: string;
  branch_id: number;
  branch_name: string;
  plan_name: string | null;
  checked_in_at: string;
  minutes: number;
}

export interface CheckinStats {
  date: string;
  total: number;
  allowed: number;
  denied: number;
  unique_members: number;
  present: number;
  by_hour: { hour: number; allowed: number; denied: number }[];
  deny_reasons: { reason: DenyReason; message: string; count: number }[];
}

class CheckinApi {
  readonly key = 'checkins';

  scan(token: string, branchId: number) {
    return http.post<ScanResult>('/admin/checkins/scan', { token, branch_id: branchId });
  }

  manual(memberId: number, branchId: number, note?: string) {
    return http.post<ScanResult>('/admin/checkins/manual', {
      member_id: memberId,
      branch_id: branchId,
      note: note || undefined,
    });
  }

  checkout(id: number) {
    return http.post<CheckinItem>(`/admin/checkins/${id}/checkout`);
  }

  list(params: CheckinListParams = {}) {
    return http.get<Page<CheckinItem>>('/admin/checkins', { params });
  }

  present(branchId?: number | null) {
    return http.get<PresentItem[]>('/admin/checkins/present', {
      params: { branch_id: branchId ?? undefined },
    });
  }

  stats(branchId?: number | null, date?: string) {
    return http.get<CheckinStats>('/admin/checkins/stats', {
      params: { branch_id: branchId ?? undefined, date },
    });
  }
}

export const checkinApi = new CheckinApi();
export default checkinApi;

export const CHECKIN_RESULT_MAP: Record<CheckinResult, { label: string; color: string }> = {
  ALLOWED: { label: 'Cho vào', color: 'green' },
  DENIED: { label: 'Từ chối', color: 'red' },
};
export const CHECKIN_RESULT_OPTIONS = [
  { value: 'ALLOWED', label: 'Cho vào' },
  { value: 'DENIED', label: 'Từ chối' },
];
export const CHECKIN_METHOD_LABEL: Record<CheckinMethod, string> = {
  QR: 'Quét QR',
  MANUAL: 'Thủ công',
};
export const CHECKIN_METHOD_OPTIONS = [
  { value: 'QR', label: 'Quét QR' },
  { value: 'MANUAL', label: 'Thủ công' },
];
