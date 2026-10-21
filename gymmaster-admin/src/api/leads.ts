import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

/** CRM Khách tiềm năng & Chăm sóc hội viên (Sprint 16 – permission `lead.view` / `lead.manage`). */

export type LeadStatus = 'NEW' | 'CONTACTED' | 'TRIAL' | 'CONVERTED' | 'LOST';
export type LeadSource = 'WEBSITE' | 'WALK_IN' | 'FACEBOOK' | 'REFERRAL' | 'OTHER';
export type LeadActivityType = 'CALL' | 'SMS' | 'VISIT' | 'NOTE';

export const LEAD_STATUS_ORDER: LeadStatus[] = [
  'NEW',
  'CONTACTED',
  'TRIAL',
  'CONVERTED',
  'LOST',
];

export const LEAD_SOURCE_LABELS: Record<LeadSource, string> = {
  WEBSITE: 'Website / Landing',
  WALK_IN: 'Khách vãng lai (Walk-in)',
  FACEBOOK: 'Facebook / Mạng xã hội',
  REFERRAL: 'Hội viên giới thiệu',
  OTHER: 'Nguồn khác',
};

export const LEAD_SOURCE_COLORS: Record<LeadSource, string> = {
  WEBSITE: 'blue',
  WALK_IN: 'green',
  FACEBOOK: 'geekblue',
  REFERRAL: 'purple',
  OTHER: 'default',
};

export const LEAD_SOURCE_OPTIONS = (Object.keys(LEAD_SOURCE_LABELS) as LeadSource[]).map(
  (value) => ({ value, label: LEAD_SOURCE_LABELS[value] }),
);

export const LEAD_ACTIVITY_LABELS: Record<LeadActivityType, string> = {
  CALL: 'Gọi điện tư vấn',
  SMS: 'Nhắn tin SMS / Zalo',
  VISIT: 'Khách đến tham quan',
  NOTE: 'Ghi chú nội bộ',
};

export const LEAD_ACTIVITY_OPTIONS = (
  Object.keys(LEAD_ACTIVITY_LABELS) as LeadActivityType[]
).map((value) => ({ value, label: LEAD_ACTIVITY_LABELS[value] }));

export interface LeadActivity {
  id: number;
  lead_id: number;
  type: LeadActivityType;
  content: string;
  created_by?: number | null;
  created_by_name?: string;
  created_at: string;
}

export interface TrialPass {
  id: number;
  lead_id: number;
  branch_id: number;
  valid_date: string;
  qr_token: string;
  used_at?: string | null;
  created_at?: string;
}

export interface Lead {
  id: number;
  branch_id: number;
  branch_name?: string;
  full_name: string;
  phone: string;
  email: string;
  source: LeadSource;
  interest_plan_id?: number | null;
  interest_plan_name?: string;
  status: LeadStatus;
  assigned_to?: number | null;
  assigned_to_name?: string;
  trial_date?: string | null;
  member_id?: number | null;
  note: string;
  activities?: LeadActivity[];
  trial_pass?: TrialPass | null;
  created_at: string;
  updated_at?: string;
}

export interface LeadInput {
  branch_id: number;
  full_name: string;
  phone: string;
  email?: string;
  source?: LeadSource;
  interest_plan_id?: number | null;
  status?: LeadStatus;
  assigned_to?: number | null;
  trial_date?: string | null;
  note?: string;
}

export interface LeadListParams extends ListParams {
  branch_id?: number;
  status?: LeadStatus;
  source?: LeadSource;
  assigned_to?: number;
}

export interface AtRiskMember {
  id: number;
  member_id?: number;
  member_code: string;
  full_name: string;
  phone: string;
  email?: string;
  branch_id?: number;
  branch_name?: string;
  plan_name?: string;
  subscription_end_date?: string;
  end_date?: string;
  last_checkin_at?: string | null;
  days_since_last_checkin?: number;
  absent_days?: number;
  days_inactive?: number;
}

export interface LeadFunnelStatusRow {
  status?: LeadStatus | string;
  key?: string;
  label?: string;
  count: number;
  pct?: number;
}

export interface LeadFunnelSourceRow {
  source?: LeadSource | string;
  key?: string;
  label?: string;
  count: number;
  converted?: number;
  conversion_rate?: number;
  pct?: number;
}

export interface LeadFunnelReport {
  total?: number;
  converted?: number;
  conversion_rate?: number;
  by_status?: LeadFunnelStatusRow[];
  by_source?: LeadFunnelSourceRow[];
}

function normalizePage<T>(res: Page<T> | T[]): Page<T> {
  if (Array.isArray(res)) {
    return {
      items: res,
      total: res.length,
      total_page: 1,
      current_page: 1,
      limit: res.length || 20,
    };
  }
  return res;
}

class LeadsApi {
  readonly key = 'leads';
  readonly atRiskKey = 'members-at-risk';

  async list(params: LeadListParams = {}): Promise<Page<Lead>> {
    const res = await http.get<Page<Lead> | Lead[]>('/admin/leads', { params });
    return normalizePage(res);
  }

  get(id: number) {
    return http.get<Lead>(`/admin/leads/${id}`);
  }

  create(data: LeadInput) {
    return http.post<Lead>('/admin/leads', data);
  }

  update(id: number, data: Partial<LeadInput>) {
    return http.put<Lead>(`/admin/leads/${id}`, data);
  }

  /** Chỉ đổi trạng thái (Kanban) – không cần gửi lại toàn bộ thông tin khách. */
  updateStatus(id: number, status: LeadStatus) {
    return http.patch<Lead>(`/admin/leads/${id}/status`, { status });
  }

  remove(id: number) {
    return http.delete(`/admin/leads/${id}`);
  }

  async listActivities(id: number): Promise<LeadActivity[]> {
    const res = await http.get<LeadActivity[] | Page<LeadActivity>>(
      `/admin/leads/${id}/activities`,
    );
    return Array.isArray(res) ? res : (res.items ?? []);
  }

  addActivity(id: number, data: { type: LeadActivityType; content: string }) {
    return http.post<LeadActivity>(`/admin/leads/${id}/activities`, data);
  }

  issueTrialPass(id: number, validDate?: string) {
    return http.post<TrialPass>(`/admin/leads/${id}/trial-pass`, {
      valid_date: validDate,
    });
  }

  convert(id: number) {
    return http.post<{ member_id?: number; id?: number; lead?: Lead }>(
      `/admin/leads/${id}/convert`,
    );
  }

  async atRiskMembers(params: { branch_id?: number; days?: number; page?: number; limit?: number } = {}): Promise<Page<AtRiskMember>> {
    const res = await http.get<Page<AtRiskMember> | AtRiskMember[]>(
      '/admin/members/at-risk',
      { params },
    );
    return normalizePage(res);
  }

  funnelReport(params: { branch_id?: number; from?: string; to?: string } = {}) {
    return http.get<LeadFunnelReport>('/admin/reports/leads', { params });
  }
}

export const leadsApi = new LeadsApi();
export default leadsApi;
