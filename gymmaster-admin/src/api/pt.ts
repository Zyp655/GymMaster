import { http } from '@/utils/api';
import type { InvoiceDetail, PaymentMethod } from './invoice';
import type { ListParams, Page } from './types';

/**
 * Huấn luyện cá nhân (PT): hợp đồng, buổi PT, lịch dạy HLV, hoa hồng – docs/api/pt.md.
 * Quyền: `pt.view`, `pt.sell`, `pt.schedule`, `pt.complete`, `pt.cancel_contract`,
 * `commission.view`, `commission.pay`.
 */

export type PtContractStatus = 'PENDING' | 'ACTIVE' | 'COMPLETED' | 'EXPIRED' | 'CANCELLED';
export type PtSessionStatus = 'SCHEDULED' | 'COMPLETED' | 'CANCELLED' | 'NO_SHOW';
export type PtSessionSource = 'COUNTER' | 'PORTAL' | 'TRAINER';
export type CommissionStatus = 'PENDING' | 'PAID';
export type CommissionSourceType = 'PT_SESSION' | 'CLASS';

/* ------------------------------------------------------------------ Hợp đồng */

export interface PtContract {
  id: number;
  member_id: number;
  member_code: string;
  member_name: string;
  member_phone: string;
  trainer_id: number;
  trainer_name: string;
  pt_package_id: number;
  package_code: string;
  package_name: string;
  branch_id: number;
  branch_name: string;
  invoice_id: number | null;
  invoice_no: string;
  invoice_status: string;
  total_sessions: number;
  /** Đã trừ (hoàn thành + vắng mặt/hủy muộn bị tính). */
  used_sessions: number;
  /** Đang đặt lịch (SCHEDULED). */
  scheduled_sessions: number;
  /** = total − used. */
  remaining_sessions: number;
  /** Còn đặt được = total − used − scheduled. */
  bookable_sessions: number;
  session_minutes: number;
  /** `YYYY-MM-DD`. */
  start_date: string;
  end_date: string;
  /** Giá trị hợp đồng (số tiền thực bán sau khuyến mãi). */
  price: number;
  price_per_session: number;
  status: PtContractStatus;
  /** Chỉ có với PENDING/ACTIVE. */
  days_left: number | null;
  sold_by: number | null;
  sold_by_name: string;
  note: string;
  created_at: string;
  updated_at: string;
}

export interface PtPackageRef {
  id: number;
  code: string;
  name: string;
  session_count: number;
  session_minutes: number;
  validity_days: number;
  price: number;
  status: string;
}

export interface PtMemberRef {
  id: number;
  member_code: string;
  full_name: string;
  phone: string;
  email: string;
  status: string;
}

export interface PtContractDetail extends PtContract {
  package: PtPackageRef;
  member: PtMemberRef;
  invoice: InvoiceDetail | null;
  /** Mọi buổi của hợp đồng, sắp theo giờ bắt đầu. */
  sessions: PtSession[];
}

export interface PtSellInput {
  member_id: number;
  pt_package_id: number;
  trainer_id: number;
  branch_id: number;
  /** `YYYY-MM-DD`; rỗng = hôm nay. */
  start_date?: string;
  promotion_code?: string;
  note?: string;
}

export interface PtPaymentInput {
  amount: number;
  method: PaymentMethod;
  reference?: string;
  note?: string;
}

export interface PtSellRequest extends PtSellInput {
  payment?: PtPaymentInput;
}

export interface PtPromotionApplied {
  id: number;
  code: string;
  name: string;
  discount: number;
}

export interface PtQuote {
  member_id: number;
  branch_id: number;
  trainer_id: number;
  trainer_name: string;
  package: PtPackageRef;
  start_date: string;
  end_date: string;
  total_sessions: number;
  session_minutes: number;
  subtotal: number;
  discount: number;
  total: number;
  price_per_session: number;
  promotion: PtPromotionApplied | null;
}

export interface PtContractListParams extends ListParams {
  branch_id?: number;
  trainer_id?: number;
  member_id?: number;
  pt_package_id?: number;
  status?: PtContractStatus;
}

export interface ChangeTrainerInput {
  trainer_id: number;
  note?: string;
}

export interface SessionConflict {
  session_id: number;
  start_at: string;
  end_at: string;
  reason: string;
}

export interface ChangeTrainerCheck {
  ok: boolean;
  trainer_id: number;
  trainer_name: string;
  /** Số buổi SCHEDULED tương lai sẽ chuyển sang HLV mới. */
  moved_sessions: number;
  conflicts: SessionConflict[];
}

export interface CancelContractInput {
  reason: string;
  refund?: { amount: number; method: PaymentMethod };
}

/* ------------------------------------------------------------------ Buổi PT */

export interface PtSession {
  id: number;
  contract_id: number;
  trainer_id: number;
  trainer_name: string;
  member_id: number;
  member_code: string;
  member_name: string;
  member_phone: string;
  branch_id: number;
  branch_name: string;
  package_name: string;
  start_at: string;
  end_at: string;
  status: PtSessionStatus;
  /** true = đã trừ vào hợp đồng (COMPLETED, hoặc NO_SHOW đã chốt). */
  counted: boolean;
  workout_note: string;
  trainer_note: string;
  completed_at: string | null;
  cancelled_at: string | null;
  cancel_reason: string;
  source: PtSessionSource;
  created_by: number | null;
  created_by_name: string;
  created_at: string;
  updated_at: string;
}

export interface PtSessionListParams {
  /** `YYYY-MM-DD`, bắt buộc. */
  from: string;
  /** `YYYY-MM-DD`, bắt buộc (bao gồm), tối đa 62 ngày. */
  to: string;
  branch_id?: number;
  trainer_id?: number;
  member_id?: number;
  contract_id?: number;
  status?: PtSessionStatus;
}

export interface PtSessionCancelInput {
  reason: string;
  /** Chỉ có tác dụng khi hủy muộn: true = tính buổi (NO_SHOW). */
  count_as_used?: boolean;
}

export interface PtSessionCompleteInput {
  workout_note?: string;
  trainer_note?: string;
}

/** Mục trong lịch dạy HLV (buổi PT hoặc buổi lớp nhóm). */
export interface TrainerScheduleItem {
  type: 'PT' | 'CLASS';
  /** id buổi PT hoặc id lịch lớp. */
  id: number;
  title: string;
  start_at: string;
  end_at: string;
  status: string;
  branch_id: number;
  branch_name: string;
  trainer_id: number;
  trainer_name: string;
  contract_id: number | null;
  member_id: number | null;
  room_name: string;
  color: string;
  booked_count: number | null;
  capacity: number | null;
}

export interface TrainerScheduleParams {
  /** Bỏ trống = chính mình. */
  trainer_id?: number;
  from: string;
  to: string;
}

/* ------------------------------------------------------------------ Hoa hồng */

export interface Commission {
  id: number;
  trainer_id: number;
  trainer_name: string;
  branch_id: number;
  branch_name: string;
  source_type: CommissionSourceType;
  source_id: number;
  session_start_at: string | null;
  contract_id: number | null;
  member_name: string;
  base_amount: number;
  /** % hoa hồng. */
  rate: number;
  amount: number;
  /** `YYYY-MM`. */
  period: string;
  status: CommissionStatus;
  paid_at: string | null;
  paid_by: number | null;
  paid_by_name: string;
  created_at: string;
}

export interface CommissionListParams extends ListParams {
  trainer_id?: number;
  branch_id?: number;
  period?: string;
  status?: CommissionStatus;
  source_type?: CommissionSourceType;
}

export interface CommissionSummaryItem {
  trainer_id: number;
  trainer_name: string;
  staff_code: string;
  sessions: number;
  base_total: number;
  total_amount: number;
  paid_amount: number;
  pending_amount: number;
}

export interface CommissionSummary {
  period: string;
  branch_id: number;
  items: CommissionSummaryItem[];
  sessions: number;
  total_amount: number;
  paid_amount: number;
  pending_amount: number;
}

export interface CommissionPayInput {
  trainer_id: number;
  period: string;
  branch_id?: number;
}

export interface CommissionPayResult {
  trainer_id: number;
  trainer_name: string;
  period: string;
  count: number;
  amount: number;
  paid_at: string;
}

class PtApi {
  /** Khóa gốc hợp đồng + buổi PT (thao tác buổi ảnh hưởng hợp đồng nên dùng chung). */
  readonly key = 'pt';
  readonly commissionKey = 'commissions';

  // Hợp đồng
  quote(data: PtSellInput) {
    return http.post<PtQuote>('/admin/pt-contracts/quote', data);
  }
  sell(data: PtSellRequest) {
    return http.post<PtContractDetail>('/admin/pt-contracts', data);
  }
  listContracts(params: PtContractListParams = {}) {
    return http.get<Page<PtContract>>('/admin/pt-contracts', { params });
  }
  getContract(id: number) {
    return http.get<PtContractDetail>(`/admin/pt-contracts/${id}`);
  }
  memberContracts(memberId: number) {
    return http.get<PtContract[]>(`/admin/members/${memberId}/pt-contracts`);
  }
  checkChangeTrainer(id: number, data: ChangeTrainerInput) {
    return http.post<ChangeTrainerCheck>(`/admin/pt-contracts/${id}/change-trainer/check`, data);
  }
  changeTrainer(id: number, data: ChangeTrainerInput) {
    return http.post<PtContractDetail>(`/admin/pt-contracts/${id}/change-trainer`, data);
  }
  cancelContract(id: number, data: CancelContractInput) {
    return http.post<PtContractDetail>(`/admin/pt-contracts/${id}/cancel`, data);
  }

  // Buổi PT
  listSessions(params: PtSessionListParams) {
    return http.get<PtSession[]>('/admin/pt-sessions', { params });
  }
  getSession(id: number) {
    return http.get<PtSession>(`/admin/pt-sessions/${id}`);
  }
  /** `start_at`: RFC3339 hoặc `YYYY-MM-DDTHH:mm` (giờ VN). */
  createSession(contractId: number, startAt: string) {
    return http.post<PtSession>('/admin/pt-sessions', {
      contract_id: contractId,
      start_at: startAt,
    });
  }
  reschedule(id: number, startAt: string) {
    return http.put<PtSession>(`/admin/pt-sessions/${id}`, { start_at: startAt });
  }
  cancelSession(id: number, data: PtSessionCancelInput) {
    return http.post<PtSession>(`/admin/pt-sessions/${id}/cancel`, data);
  }
  completeSession(id: number, data: PtSessionCompleteInput) {
    return http.post<PtSession>(`/admin/pt-sessions/${id}/complete`, data);
  }
  noShow(id: number, note?: string) {
    return http.post<PtSession>(`/admin/pt-sessions/${id}/no-show`, { note: note ?? '' });
  }
  trainerSchedule(params: TrainerScheduleParams) {
    return http.get<TrainerScheduleItem[]>('/admin/trainer-schedule', { params });
  }

  // Hoa hồng
  listCommissions(params: CommissionListParams = {}) {
    return http.get<Page<Commission>>('/admin/commissions', { params });
  }
  commissionSummary(period: string, branchId?: number) {
    return http.get<CommissionSummary>('/admin/commissions/summary', {
      params: { period, branch_id: branchId },
    });
  }
  payCommission(data: CommissionPayInput) {
    return http.post<CommissionPayResult>('/admin/commissions/pay', data);
  }
}

export const ptApi = new PtApi();
export default ptApi;
