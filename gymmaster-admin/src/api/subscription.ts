import { http } from '@/utils/api';
import type { AccessScope, PlanType } from './plan';
import type { InvoiceDetail, InvoiceStatus, PaymentInput, PaymentMethod } from './invoice';
import type { ListParams, Page } from './types';

/** Bán gói, đăng ký gói tập, bảo lưu, chuyển nhượng, hủy – hợp đồng `docs/api/sales.md` §1–2. */

export type SubscriptionStatus =
  'PENDING' | 'ACTIVE' | 'FROZEN' | 'EXPIRED' | 'CANCELLED' | 'TRANSFERRED';
export type FreezeStatus = 'PENDING' | 'APPROVED' | 'REJECTED' | 'ENDED' | 'CANCELLED';
export type SubscriptionEventType =
  | 'CREATED'
  | 'ACTIVATED'
  | 'RENEWED'
  | 'UPGRADED'
  | 'FREEZE_REQUESTED'
  | 'FROZEN'
  | 'UNFROZEN'
  | 'FREEZE_REJECTED'
  | 'TRANSFERRED'
  | 'CANCELLED'
  | 'EXPIRED'
  | 'SESSION_USED';

/** Dòng danh sách `GET /admin/subscriptions`. */
export interface Subscription {
  id: number;
  member_id: number;
  member_code: string;
  member_name: string;
  member_phone: string;
  plan_id: number;
  plan_code: string;
  plan_name: string;
  plan_type: PlanType;
  access_scope: AccessScope;
  branch_id: number;
  branch_name: string;
  invoice_id: number | null;
  invoice_no: string;
  /** Rỗng khi không có hóa đơn (vd. chuyển nhượng phí 0đ). */
  invoice_status: InvoiceStatus | '';
  start_date: string;
  end_date: string;
  total_sessions: number | null;
  remaining_sessions: number | null;
  /** Giá trị gói = số tiền thực bán sau khuyến mãi. */
  price: number;
  status: SubscriptionStatus;
  freeze_days_used: number;
  freeze_times_used: number;
  previous_id: number | null;
  /** Chỉ có với PENDING/ACTIVE/FROZEN. */
  days_left: number | null;
  sold_by: number | null;
  sold_by_name: string;
  note: string;
  created_at: string;
  updated_at: string;
}

export interface PlanRef {
  id: number;
  code: string;
  name: string;
  type: PlanType;
  duration_days: number;
  session_count: number | null;
  price: number;
  access_scope: AccessScope;
  max_freeze_days: number;
  max_freeze_times: number;
  transfer_fee: number;
  status: string;
}

export interface SubscriptionMemberRef {
  id: number;
  member_code: string;
  full_name: string;
  phone: string;
  email: string;
  status: string;
}

export interface Freeze {
  id: number;
  subscription_id: number;
  member_id: number;
  member_code: string;
  member_name: string;
  plan_name: string;
  branch_id: number;
  branch_name: string;
  start_date: string;
  end_date: string;
  days: number;
  reason: string;
  status: FreezeStatus;
  requested_by: number | null;
  requested_by_name: string;
  reviewed_by: number | null;
  reviewed_by_name: string;
  reviewed_at: string | null;
  review_note: string;
  created_at: string;
}

export interface SubscriptionEvent {
  id: number;
  event_type: SubscriptionEventType;
  /** JSON tự do theo loại sự kiện. */
  data: Record<string, unknown> | null;
  /** `null` = hệ thống (cron). */
  actor_id: number | null;
  actor_name: string;
  created_at: string;
}

export interface SubscriptionDetail extends Subscription {
  plan: PlanRef;
  member: SubscriptionMemberRef;
  invoice: InvoiceDetail | null;
  freezes: Freeze[];
  events: SubscriptionEvent[];
}

export interface SellInput {
  member_id: number;
  plan_id: number;
  branch_id: number;
  /** Rỗng = hôm nay hoặc nối tiếp gói đang chạy. */
  start_date?: string;
  promotion_code?: string;
  note?: string;
  payment?: PaymentInput;
}

export interface QuoteResult {
  member_id: number;
  branch_id: number;
  plan: PlanRef;
  start_date: string;
  end_date: string;
  /** true = nối tiếp gói đang chạy. */
  chained: boolean;
  previous_id: number | null;
  total_sessions: number | null;
  subtotal: number;
  discount: number;
  total: number;
  promotion: { id: number; code: string; name: string; discount: number } | null;
}

export interface RenewInput {
  /** 0/bỏ trống = gói hiện tại. */
  plan_id?: number;
  promotion_code?: string;
  note?: string;
  payment?: PaymentInput;
}

export interface UpgradeQuote {
  subscription_id: number;
  current_plan: PlanRef;
  new_plan: PlanRef;
  current_price: number;
  total_days: number;
  remaining_days: number;
  total_sessions: number | null;
  remaining_sessions: number | null;
  remaining_value: number;
  new_price: number;
  amount_due: number;
  start_date: string;
  end_date: string;
}

export interface UpgradeInput {
  plan_id: number;
  note?: string;
  payment?: PaymentInput;
}

export interface FreezeInput {
  start_date: string;
  end_date: string;
  reason?: string;
}

export interface TransferInput {
  to_member_id: number;
  note?: string;
  payment?: PaymentInput;
}

export interface CancelInput {
  reason: string;
  refund?: { amount: number; method: PaymentMethod };
}

export interface SubscriptionListParams extends ListParams {
  branch_id?: number;
  member_id?: number;
  plan_id?: number;
  status?: SubscriptionStatus;
  /** Gói ACTIVE/FROZEN hết hạn trong N ngày tới (0 = bỏ qua). */
  expiring_in_days?: number;
  /** Lọc `end_date` từ/đến (YYYY-MM-DD). */
  from?: string;
  to?: string;
}

export interface FreezeListParams extends ListParams {
  status?: FreezeStatus;
  branch_id?: number;
  subscription_id?: number;
}

class SubscriptionApi {
  /** Khóa gốc; yêu cầu bảo lưu dùng `[key, 'freezes', ...]`. */
  readonly key = 'subscriptions';

  quote(data: Omit<SellInput, 'payment'>) {
    return http.post<QuoteResult>('/admin/subscriptions/quote', data);
  }

  sell(data: SellInput) {
    return http.post<SubscriptionDetail>('/admin/subscriptions', data);
  }

  list(params: SubscriptionListParams = {}) {
    return http.get<Page<Subscription>>('/admin/subscriptions', { params });
  }

  get(id: number) {
    return http.get<SubscriptionDetail>(`/admin/subscriptions/${id}`);
  }

  events(id: number) {
    return http.get<SubscriptionEvent[]>(`/admin/subscriptions/${id}/events`);
  }

  /** Lịch sử gói của hội viên (mới nhất trước). */
  byMember(memberId: number) {
    return http.get<Subscription[]>(`/admin/members/${memberId}/subscriptions`);
  }

  renewQuote(id: number, data: Omit<RenewInput, 'payment'>) {
    return http.post<QuoteResult>(`/admin/subscriptions/${id}/renew/quote`, data);
  }

  renew(id: number, data: RenewInput) {
    return http.post<SubscriptionDetail>(`/admin/subscriptions/${id}/renew`, data);
  }

  upgradeQuote(id: number, planId: number) {
    return http.post<UpgradeQuote>(`/admin/subscriptions/${id}/upgrade/quote`, {
      plan_id: planId,
    });
  }

  upgrade(id: number, data: UpgradeInput) {
    return http.post<SubscriptionDetail>(`/admin/subscriptions/${id}/upgrade`, data);
  }

  requestFreeze(id: number, data: FreezeInput) {
    return http.post<Freeze>(`/admin/subscriptions/${id}/freezes`, data);
  }

  freezes(params: FreezeListParams = {}) {
    return http.get<Page<Freeze>>('/admin/subscription-freezes', { params });
  }

  approveFreeze(id: number, note = '') {
    return http.post<Freeze>(`/admin/subscription-freezes/${id}/approve`, { note });
  }

  rejectFreeze(id: number, note = '') {
    return http.post<Freeze>(`/admin/subscription-freezes/${id}/reject`, { note });
  }

  endFreeze(id: number, note = '') {
    return http.post<Freeze>(`/admin/subscription-freezes/${id}/end-early`, { note });
  }

  transfer(id: number, data: TransferInput) {
    return http.post<SubscriptionDetail>(`/admin/subscriptions/${id}/transfer`, data);
  }

  cancel(id: number, data: CancelInput) {
    return http.post<SubscriptionDetail>(`/admin/subscriptions/${id}/cancel`, data);
  }
}

export const subscriptionApi = new SubscriptionApi();
export default subscriptionApi;

export const SUBSCRIPTION_EVENT_LABEL: Record<SubscriptionEventType, string> = {
  CREATED: 'Tạo gói',
  ACTIVATED: 'Kích hoạt',
  RENEWED: 'Gia hạn',
  UPGRADED: 'Nâng cấp',
  FREEZE_REQUESTED: 'Yêu cầu bảo lưu',
  FROZEN: 'Bắt đầu bảo lưu',
  UNFROZEN: 'Kết thúc bảo lưu',
  FREEZE_REJECTED: 'Từ chối bảo lưu',
  TRANSFERRED: 'Chuyển nhượng',
  CANCELLED: 'Hủy gói',
  EXPIRED: 'Hết hạn',
  SESSION_USED: 'Dùng buổi tập',
};

export const PLAN_TYPE_LABEL: Record<PlanType, string> = {
  TIME: 'Theo thời gian',
  SESSION: 'Theo buổi',
};

export const ACCESS_SCOPE_LABEL: Record<AccessScope, string> = {
  SINGLE_BRANCH: 'Một chi nhánh',
  ALL_BRANCHES: 'Mọi chi nhánh',
};
