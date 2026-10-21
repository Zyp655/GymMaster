import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

function normalizePage<T>(res: Page<T> | T[] | null | undefined): Page<T> {
  if (!res) {
    return { items: [], total: 0, total_page: 1, current_page: 1, limit: 20 };
  }
  if (Array.isArray(res)) {
    return {
      items: res,
      total: res.length,
      total_page: 1,
      current_page: 1,
      limit: res.length || 20,
    };
  }
  return {
    items: res.items ?? [],
    total: res.total ?? (res.items?.length || 0),
    total_page: res.total_page ?? 1,
    current_page: res.current_page ?? 1,
    limit: res.limit ?? 20,
  };
}

/* ============================================================================
 * SPRINT 18 – Thiết bị kiểm soát ra vào & Kiosk (/admin/access-devices)
 * ============================================================================ */

export type AccessDeviceType = 'TURNSTILE' | 'DOOR' | 'KIOSK';
export type AccessDeviceStatus = 'ACTIVE' | 'OFFLINE' | 'DISABLED';

export const DEVICE_TYPE_LABELS: Record<AccessDeviceType, string> = {
  TURNSTILE: 'Cổng xoay (Turnstile)',
  DOOR: 'Cửa từ (Door)',
  KIOSK: 'Màn hình Kiosk QR',
};

export const DEVICE_TYPE_COLORS: Record<AccessDeviceType, string> = {
  TURNSTILE: 'purple',
  DOOR: 'cyan',
  KIOSK: 'geekblue',
};

export const DEVICE_TYPE_OPTIONS = (Object.keys(DEVICE_TYPE_LABELS) as AccessDeviceType[]).map(
  (value) => ({ value, label: DEVICE_TYPE_LABELS[value] }),
);

export interface AccessDevice {
  id: number;
  branch_id: number;
  branch_name?: string;
  code: string;
  name: string;
  type: AccessDeviceType;
  last_seen_at?: string | null;
  status: AccessDeviceStatus;
  raw_api_key?: string;
  api_key?: string;
  created_at?: string;
  updated_at?: string;
}

export interface AccessDeviceInput {
  branch_id: number;
  code: string;
  name: string;
  type: AccessDeviceType;
  status?: AccessDeviceStatus;
}

export interface AccessDeviceListParams extends ListParams {
  branch_id?: number;
  type?: AccessDeviceType;
  status?: AccessDeviceStatus;
}

export interface RotateDeviceKeyResponse {
  id?: number;
  code?: string;
  name?: string;
  raw_api_key?: string;
  api_key?: string;
  key?: string;
  device?: AccessDevice;
}

class AccessDevicesApi {
  readonly key = 'access-devices';

  async list(params: AccessDeviceListParams = {}): Promise<Page<AccessDevice>> {
    const res = await http.get<Page<AccessDevice> | AccessDevice[]>('/admin/access-devices', {
      params,
    });
    return normalizePage(res);
  }

  get(id: number) {
    return http.get<AccessDevice>(`/admin/access-devices/${id}`);
  }

  create(data: AccessDeviceInput) {
    return http.post<AccessDevice & RotateDeviceKeyResponse>('/admin/access-devices', data);
  }

  update(id: number, data: Partial<AccessDeviceInput>) {
    return http.put<AccessDevice>(`/admin/access-devices/${id}`, data);
  }

  remove(id: number) {
    return http.delete(`/admin/access-devices/${id}`);
  }

  rotateKey(id: number) {
    return http.post<RotateDeviceKeyResponse>(`/admin/access-devices/${id}/rotate-key`);
  }
}

export const accessDevicesApi = new AccessDevicesApi();

/* ============================================================================
 * SPRINT 19 – Sơ đồ tủ đồ & Thuê tủ (/admin/lockers)
 * ============================================================================ */

export type LockerSize = 'SMALL' | 'STANDARD' | 'LARGE';
export type LockerStatus = 'AVAILABLE' | 'OCCUPIED' | 'RENTED' | 'BROKEN';
export type LockerRentalType = 'DAILY' | 'MONTHLY';
export type LockerRentalStatus = 'ACTIVE' | 'ENDED' | 'EXPIRED';

export const LOCKER_SIZE_LABELS: Record<LockerSize, string> = {
  SMALL: 'Nhỏ (Small)',
  STANDARD: 'Tiêu chuẩn (Standard)',
  LARGE: 'Lớn (Large)',
};

export const LOCKER_SIZE_OPTIONS = (Object.keys(LOCKER_SIZE_LABELS) as LockerSize[]).map(
  (value) => ({ value, label: LOCKER_SIZE_LABELS[value] }),
);

export const LOCKER_RENTAL_TYPE_LABELS: Record<LockerRentalType, string> = {
  DAILY: 'Gán tủ theo lượt (Trong ngày)',
  MONTHLY: 'Thuê tủ theo tháng',
};

export const LOCKER_RENTAL_TYPE_OPTIONS = (
  Object.keys(LOCKER_RENTAL_TYPE_LABELS) as LockerRentalType[]
).map((value) => ({ value, label: LOCKER_RENTAL_TYPE_LABELS[value] }));

export interface LockerRental {
  id: number;
  locker_id: number;
  locker_code?: string;
  member_id: number;
  member_name?: string;
  member_code?: string;
  member_phone?: string;
  type: LockerRentalType;
  start_at: string;
  end_at?: string | null;
  invoice_id?: number | null;
  price: number;
  status: LockerRentalStatus;
  note?: string;
  created_at?: string;
}

export interface Locker {
  id: number;
  branch_id: number;
  branch_name?: string;
  code: string;
  zone: string;
  size: LockerSize;
  monthly_fee: number;
  status: LockerStatus;
  note: string;
  active_rental?: LockerRental | null;
  current_rental?: LockerRental | null;
  member_id?: number | null;
  member_name?: string;
  member_code?: string;
  member_phone?: string;
  rental_type?: LockerRentalType;
  start_at?: string | null;
  end_at?: string | null;
  created_at?: string;
  updated_at?: string;
}

export interface LockerBoardZone {
  zone: string;
  lockers: Locker[];
}

export interface LockerBoardSummary {
  total: number;
  available: number;
  occupied: number;
  rented: number;
  broken: number;
}

export interface LockerBoardData {
  zones: LockerBoardZone[];
  lockers: Locker[];
  summary: LockerBoardSummary;
}

export interface LockerInput {
  branch_id: number;
  code: string;
  zone: string;
  size?: LockerSize;
  monthly_fee?: number;
  status?: LockerStatus;
  note?: string;
}

export interface LockerAssignInput {
  member_id: number;
  type: LockerRentalType;
  months?: number;
  end_at?: string | null;
  price?: number;
  payment_method?: string;
  note?: string;
}

export interface LockerListParams extends ListParams {
  branch_id?: number;
  zone?: string;
  size?: LockerSize;
  status?: LockerStatus;
}

class LockersApi {
  readonly key = 'lockers';
  readonly rentalKey = 'locker-rentals';

  async board(branchId?: number | null): Promise<LockerBoardData> {
    const raw = await http.get<
      | Locker[]
      | {
          zones?: LockerBoardZone[] | Record<string, Locker[]>;
          lockers?: Locker[];
          items?: Locker[];
          summary?: Partial<LockerBoardSummary>;
        }
    >('/admin/lockers/board', {
      params: branchId ? { branch_id: branchId } : {},
    });

    let allLockers: Locker[] = [];
    let zones: LockerBoardZone[] = [];

    if (Array.isArray(raw)) {
      allLockers = raw;
    } else if (raw) {
      if (Array.isArray(raw.lockers)) {
        allLockers = raw.lockers;
      } else if (Array.isArray(raw.items)) {
        allLockers = raw.items;
      }
      if (Array.isArray(raw.zones)) {
        zones = raw.zones;
        if (allLockers.length === 0) {
          allLockers = raw.zones.flatMap((z) => z.lockers ?? []);
        }
      } else if (raw.zones && typeof raw.zones === 'object') {
        zones = Object.entries(raw.zones).map(([zone, list]) => ({
          zone,
          lockers: Array.isArray(list) ? list : [],
        }));
        if (allLockers.length === 0) {
          allLockers = zones.flatMap((z) => z.lockers);
        }
      }
    }

    if (zones.length === 0 && allLockers.length > 0) {
      const map = new Map<string, Locker[]>();
      for (const lk of allLockers) {
        const z = lk.zone || 'Khu A';
        if (!map.has(z)) map.set(z, []);
        map.get(z)!.push(lk);
      }
      zones = Array.from(map.entries()).map(([zone, lockers]) => ({ zone, lockers }));
    }

    const summary: LockerBoardSummary = {
      total: allLockers.length,
      available: allLockers.filter((l) => l.status === 'AVAILABLE').length,
      occupied: allLockers.filter((l) => l.status === 'OCCUPIED').length,
      rented: allLockers.filter((l) => l.status === 'RENTED').length,
      broken: allLockers.filter((l) => l.status === 'BROKEN').length,
    };

    return { zones, lockers: allLockers, summary };
  }

  async list(params: LockerListParams = {}): Promise<Page<Locker>> {
    const res = await http.get<Page<Locker> | Locker[]>('/admin/lockers', { params });
    return normalizePage(res);
  }

  create(data: LockerInput) {
    return http.post<Locker>('/admin/lockers', data);
  }

  update(id: number, data: Partial<LockerInput>) {
    return http.put<Locker>(`/admin/lockers/${id}`, data);
  }

  remove(id: number) {
    return http.delete(`/admin/lockers/${id}`);
  }

  assign(id: number, data: LockerAssignInput) {
    return http.post<LockerRental>(`/admin/lockers/${id}/assign`, data);
  }

  release(id: number) {
    return http.post(`/admin/lockers/${id}/release`);
  }

  async listRentals(
    params: ListParams & { branch_id?: number; locker_id?: number; member_id?: number; status?: string } = {},
  ): Promise<Page<LockerRental>> {
    const res = await http.get<Page<LockerRental> | LockerRental[]>('/admin/locker-rentals', {
      params,
    });
    return normalizePage(res);
  }
}

export const lockersApi = new LockersApi();

/* ============================================================================
 * SPRINT 24 – Hợp đồng điện tử & Vé ngày (/admin/contract-templates, /admin/day-passes)
 * ============================================================================ */

export type ContractStatus = 'PENDING' | 'SIGNED';
export type DayPassType = 'DAY' | 'GUEST';

export const DAY_PASS_TYPE_LABELS: Record<DayPassType, string> = {
  DAY: 'Vé tập ngày (Day Pass)',
  GUEST: 'Vé khách mời HV (Guest Pass)',
};

export const DAY_PASS_TYPE_OPTIONS = (Object.keys(DAY_PASS_TYPE_LABELS) as DayPassType[]).map(
  (value) => ({ value, label: DAY_PASS_TYPE_LABELS[value] }),
);

export interface ContractTemplate {
  id: number;
  code: string;
  name: string;
  body_html: string;
  version: number;
  status: 'ACTIVE' | 'INACTIVE';
  created_at?: string;
  updated_at?: string;
}

export interface ContractTemplateInput {
  code: string;
  name: string;
  body_html: string;
  version?: number;
  status?: 'ACTIVE' | 'INACTIVE';
}

export interface MemberContract {
  id: number;
  member_id: number;
  member_name?: string;
  member_code?: string;
  member_phone?: string;
  template_id: number;
  template_code?: string;
  template_name?: string;
  template_version: number;
  subscription_id?: number | null;
  rendered_html: string;
  status: ContractStatus;
  signed_at?: string | null;
  signature_url?: string;
  pdf_url?: string;
  ip?: string;
  user_agent?: string;
  created_at: string;
  updated_at?: string;
}

export interface DayPass {
  id: number;
  branch_id: number;
  branch_name?: string;
  buyer_name: string;
  phone: string;
  member_id?: number | null;
  sponsor_member_id?: number | null;
  sponsor_member_name?: string;
  type: DayPassType;
  valid_date: string;
  qr_token: string;
  price: number;
  invoice_id?: number | null;
  used_at?: string | null;
  created_by?: number | null;
  created_at: string;
}

export interface DayPassInput {
  branch_id: number;
  buyer_name: string;
  phone: string;
  type: DayPassType;
  valid_date: string;
  price: number;
  sponsor_member_id?: number | null;
  payment_method?: string;
}

class ContractsPassesApi {
  readonly templateKey = 'contract-templates';
  readonly memberContractKey = 'member-contracts';
  readonly dayPassKey = 'day-passes';

  async listTemplates(params: ListParams & { status?: string } = {}): Promise<Page<ContractTemplate>> {
    const res = await http.get<Page<ContractTemplate> | ContractTemplate[]>(
      '/admin/contract-templates',
      { params },
    );
    return normalizePage(res);
  }

  createTemplate(data: ContractTemplateInput) {
    return http.post<ContractTemplate>('/admin/contract-templates', data);
  }

  updateTemplate(id: number, data: Partial<ContractTemplateInput>) {
    return http.put<ContractTemplate>(`/admin/contract-templates/${id}`, data);
  }

  deleteTemplate(id: number) {
    return http.delete(`/admin/contract-templates/${id}`);
  }

  async listMemberContracts(
    params: ListParams & { member_id?: number; status?: ContractStatus } = {},
  ): Promise<Page<MemberContract>> {
    try {
      const res = await http.get<Page<MemberContract> | MemberContract[]>(
        '/admin/member-contracts',
        { params },
      );
      return normalizePage(res);
    } catch (err) {
      if (params.member_id) {
        const fallback = await http.get<Page<MemberContract> | MemberContract[]>(
          `/admin/members/${params.member_id}/contracts`,
          { params },
        );
        return normalizePage(fallback);
      }
      throw err;
    }
  }

  createMemberContract(data: {
    member_id: number;
    template_id: number;
    subscription_id?: number | null;
  }) {
    return http.post<MemberContract>('/admin/member-contracts', data);
  }

  async listDayPasses(
    params: ListParams & { branch_id?: number; type?: DayPassType; valid_date?: string } = {},
  ): Promise<Page<DayPass>> {
    const res = await http.get<Page<DayPass> | DayPass[]>('/admin/day-passes', { params });
    return normalizePage(res);
  }

  createDayPass(data: DayPassInput) {
    return http.post<DayPass>('/admin/day-passes', data);
  }
}

export const contractsPassesApi = new ContractsPassesApi();

/* ============================================================================
 * SPRINT 25 – Lịch rảnh HLV, Nghỉ phép & Ngày lễ (/admin/trainers/:id/availability, /admin/leave-requests, /admin/branch-holidays)
 * ============================================================================ */

export type LeaveRequestStatus = 'PENDING' | 'APPROVED' | 'REJECTED';

export const DOW_LABELS: Record<number, string> = {
  1: 'Thứ 2',
  2: 'Thứ 3',
  3: 'Thứ 4',
  4: 'Thứ 5',
  5: 'Thứ 6',
  6: 'Thứ 7',
  0: 'Chủ nhật',
};

export const DOW_ORDER: number[] = [1, 2, 3, 4, 5, 6, 0];

export interface TrainerAvailabilitySlot {
  id?: number;
  trainer_id?: number;
  branch_id: number;
  branch_name?: string;
  dow: number;
  start_time: string;
  end_time: string;
  effective_from?: string | null;
  effective_to?: string | null;
}

export interface LeaveRequest {
  id: number;
  staff_id: number;
  staff_name?: string;
  staff_code?: string;
  branch_id?: number | null;
  branch_name?: string;
  from_at: string;
  to_at: string;
  reason: string;
  status: LeaveRequestStatus;
  approved_by?: number | null;
  approved_by_name?: string;
  approved_at?: string | null;
  review_note?: string;
  created_at: string;
}

export interface LeaveRequestInput {
  staff_id?: number;
  branch_id?: number | null;
  from_at: string;
  to_at: string;
  reason: string;
}

export interface LeaveImpactClass {
  id: number;
  class_type_name?: string;
  title?: string;
  room_name?: string;
  start_at: string;
  end_at: string;
  booked_count?: number;
  substitute_trainer_id?: number | null;
  substitute_trainer_name?: string;
}

export interface LeaveImpactPTSession {
  id: number;
  member_id?: number;
  member_name?: string;
  start_at: string;
  end_at: string;
  status?: string;
  trainer_id?: number;
  original_trainer_id?: number | null;
}

export interface LeaveImpactsData {
  classes: LeaveImpactClass[];
  pt_sessions: LeaveImpactPTSession[];
}

export interface BranchHoliday {
  id: number;
  branch_id?: number | null;
  branch_name?: string;
  date: string;
  name: string;
  closed: boolean;
  open_time?: string | null;
  close_time?: string | null;
  extend_subs: boolean;
  created_at?: string;
  /** Chỉ có trong phản hồi thêm/sửa: lớp / buổi PT đã xếp rơi vào ngày nghỉ (hoặc ngoài giờ đặc biệt) – KHÔNG tự hủy. */
  affected_class_schedules?: LeaveImpactClass[];
  affected_pt_sessions?: LeaveImpactPTSession[];
}

export interface BranchHolidayInput {
  branch_id?: number | null;
  date: string;
  name: string;
  closed: boolean;
  open_time?: string | null;
  close_time?: string | null;
  extend_subs?: boolean;
}

class TrainerSchedulesApi {
  readonly availabilityKey = 'trainer-availabilities';
  readonly leaveKey = 'leave-requests';
  readonly holidayKey = 'branch-holidays';

  async getAvailability(trainerId: number, branchId?: number | null): Promise<TrainerAvailabilitySlot[]> {
    const res = await http.get<
      TrainerAvailabilitySlot[] | { items?: TrainerAvailabilitySlot[]; slots?: TrainerAvailabilitySlot[] }
    >(`/admin/trainers/${trainerId}/availability`, {
      params: branchId ? { branch_id: branchId } : {},
    });
    if (Array.isArray(res)) return res;
    return res?.slots ?? res?.items ?? [];
  }

  /** Thay toàn bộ lịch rảnh của HLV tại MỘT chi nhánh (branchId bắt buộc). Danh sách rỗng = không giới hạn giờ nhận PT. */
  updateAvailability(
    trainerId: number,
    slots: Omit<TrainerAvailabilitySlot, 'id' | 'trainer_id'>[],
    branchId: number,
  ) {
    return http.put(`/admin/trainers/${trainerId}/availability`, {
      branch_id: branchId,
      slots,
    });
  }

  async listLeaveRequests(
    params: ListParams & { branch_id?: number; staff_id?: number; status?: LeaveRequestStatus } = {},
  ): Promise<Page<LeaveRequest>> {
    const res = await http.get<Page<LeaveRequest> | LeaveRequest[]>('/admin/leave-requests', {
      params,
    });
    return normalizePage(res);
  }

  createLeaveRequest(data: LeaveRequestInput) {
    return http.post<LeaveRequest>('/admin/leave-requests', data);
  }

  async getLeaveImpacts(id: number): Promise<LeaveImpactsData> {
    const res = await http.get<{
      classes?: LeaveImpactClass[];
      class_schedules?: LeaveImpactClass[];
      pt_sessions?: LeaveImpactPTSession[];
    }>(`/admin/leave-requests/${id}/impacts`);
    return {
      classes: res?.classes ?? res?.class_schedules ?? [],
      pt_sessions: res?.pt_sessions ?? [],
    };
  }

  approveLeaveRequest(id: number, reviewNote?: string) {
    return http.post<LeaveRequest>(`/admin/leave-requests/${id}/approve`, {
      review_note: reviewNote ?? '',
    });
  }

  rejectLeaveRequest(id: number, reviewNote?: string) {
    return http.post<LeaveRequest>(`/admin/leave-requests/${id}/reject`, {
      review_note: reviewNote ?? '',
    });
  }

  substituteClass(scheduleId: number, substituteTrainerId: number) {
    return http.post(`/admin/class-schedules/${scheduleId}/substitute`, {
      substitute_trainer_id: substituteTrainerId,
      trainer_id: substituteTrainerId,
    });
  }

  reassignPTSession(sessionId: number, trainerId: number) {
    return http.post(`/admin/pt-sessions/${sessionId}/reassign`, {
      trainer_id: trainerId,
      new_trainer_id: trainerId,
    });
  }

  async listHolidays(params: ListParams & { branch_id?: number } = {}): Promise<Page<BranchHoliday>> {
    const res = await http.get<Page<BranchHoliday> | BranchHoliday[]>('/admin/branch-holidays', {
      params,
    });
    return normalizePage(res);
  }

  createHoliday(data: BranchHolidayInput) {
    return http.post<BranchHoliday>('/admin/branch-holidays', data);
  }

  updateHoliday(id: number, data: Partial<BranchHolidayInput>) {
    return http.put<BranchHoliday>(`/admin/branch-holidays/${id}`, data);
  }

  deleteHoliday(id: number) {
    return http.delete(`/admin/branch-holidays/${id}`);
  }
}

export const trainerSchedulesApi = new TrainerSchedulesApi();

/* ============================================================================
 * SPRINT 26 – Mật độ phòng tập, Chỉ tiêu KPI & Cohort Retention (/public/branches/occupancy, /admin/sales-targets, /admin/reports/kpi, /admin/reports/cohort)
 * ============================================================================ */

export type OccupancyLevel = 'QUIET' | 'NORMAL' | 'BUSY' | 'FULL';
export type KPIMetric = 'REVENUE' | 'NEW_MEMBERS' | 'RENEWALS' | 'PT_SALES';

export const OCCUPANCY_LEVEL_META: Record<
  OccupancyLevel,
  { label: string; color: string; strokeColor: string }
> = {
  QUIET: { label: 'Thông thoáng', color: 'green', strokeColor: '#16a34a' },
  NORMAL: { label: 'Bình thường', color: 'blue', strokeColor: '#2563eb' },
  BUSY: { label: 'Đông khách', color: 'orange', strokeColor: '#ea580c' },
  FULL: { label: 'Quá tải / Đầy', color: 'red', strokeColor: '#dc2626' },
};

export const KPI_METRIC_LABELS: Record<KPIMetric, string> = {
  REVENUE: 'Doanh thu thuần (VND)',
  NEW_MEMBERS: 'Hội viên mới',
  RENEWALS: 'Lượt gia hạn gói tập',
  PT_SALES: 'Doanh số bán PT (VND)',
};

export const KPI_METRIC_OPTIONS = (Object.keys(KPI_METRIC_LABELS) as KPIMetric[]).map((value) => ({
  value,
  label: KPI_METRIC_LABELS[value],
}));

export interface BranchOccupancyItem {
  branch_id: number;
  branch_name: string;
  branch_code?: string;
  present_count: number;
  capacity: number;
  occupancy_pct: number;
  level: OccupancyLevel;
}

export interface SalesTarget {
  id: number;
  branch_id: number;
  branch_name?: string;
  staff_id?: number | null;
  staff_name?: string;
  period: string;
  metric: KPIMetric;
  target_value: number;
  created_at?: string;
}

export interface SalesTargetInput {
  branch_id: number;
  staff_id?: number | null;
  period: string;
  metric: KPIMetric;
  target_value: number;
}

/** Một chỉ số trong GET /admin/reports/kpi: thực đạt vs chỉ tiêu cấp chi nhánh (staff_id trống). */
export interface KPIReportItem {
  metric: KPIMetric;
  label: string;
  actual: number;
  target_value: number;
  achievement_pct: number;
  /** false → chưa đặt chỉ tiêu (hiện "Chưa đặt chỉ tiêu", không hiện 0%). */
  has_target?: boolean;
  /** Số chi nhánh có chỉ tiêu; xem toàn hệ thống thì `actual` chỉ cộng các chi nhánh này. */
  target_branch_count?: number;
}

export interface KPIReportData {
  period: string;
  /** 0 = toàn hệ thống */
  branch_id: number;
  items: KPIReportItem[];
}

/** Một dòng cohort: size = số HV gia nhập trong tháng; retention_by_month[i] = % còn hoạt động ở tháng +i. */
export interface CohortRow {
  cohort_month: string;
  size: number;
  retention_by_month: number[];
  active_counts: number[];
}

export interface CohortReportData {
  cohorts: CohortRow[];
  max_months: number;
}

class KpiAnalyticsApi {
  readonly occupancyKey = 'branch-occupancy';
  readonly targetKey = 'sales-targets';
  readonly kpiReportKey = 'kpi-report';
  readonly cohortKey = 'cohort-report';

  async occupancy(branchId?: number | null): Promise<BranchOccupancyItem[]> {
    const res = await http.get<BranchOccupancyItem[]>('/public/branches/occupancy');
    const rows = Array.isArray(res) ? res : [];
    return branchId ? rows.filter((r) => r.branch_id === branchId) : rows;
  }

  async listTargets(
    params: ListParams & { branch_id?: number; period?: string; metric?: KPIMetric } = {},
  ): Promise<Page<SalesTarget>> {
    const res = await http.get<Page<SalesTarget> | SalesTarget[]>('/admin/sales-targets', {
      params,
    });
    return normalizePage(res);
  }

  /** POST /admin/sales-targets là upsert theo (chi nhánh, nhân sự, kỳ, chỉ số). */
  createTarget(data: SalesTargetInput) {
    return http.post<SalesTarget>('/admin/sales-targets', data);
  }

  deleteTarget(id: number) {
    return http.delete(`/admin/sales-targets/${id}`);
  }

  async kpiReport(params: { branch_id?: number; period?: string } = {}): Promise<KPIReportData> {
    const res = await http.get<{
      period?: string;
      branch_id?: number;
      metrics?: KPIReportItem[];
      items?: KPIReportItem[];
    }>('/admin/reports/kpi', { params });
    return {
      period: res?.period ?? params.period ?? '',
      branch_id: Number(res?.branch_id ?? 0),
      items: res?.metrics ?? res?.items ?? [],
    };
  }

  async cohortReport(params: { branch_id?: number; months?: number } = {}): Promise<CohortReportData> {
    const res = await http.get<{ months?: number; cohorts?: CohortRow[]; items?: CohortRow[] }>(
      '/admin/reports/cohort',
      { params },
    );
    const rows = res?.cohorts ?? res?.items ?? [];
    const maxMonths =
      res?.months ?? rows.reduce((max, r) => Math.max(max, r.retention_by_month?.length ?? 0), 0);
    return { cohorts: rows, max_months: maxMonths || 6 };
  }
}

export const kpiAnalyticsApi = new KpiAnalyticsApi();

/* ============================================================================
 * SPRINT 27 – Bảo mật 2FA, Phiên đăng nhập & Dữ liệu cá nhân (/admin/security/mfa, /admin/security/sessions, /admin/data-requests)
 * ============================================================================ */

export type DataRequestType = 'EXPORT' | 'DELETE';
export type DataRequestStatus = 'PENDING' | 'PROCESSING' | 'DONE' | 'REJECTED';

export const DATA_REQUEST_TYPE_LABELS: Record<DataRequestType, string> = {
  EXPORT: 'Trích xuất dữ liệu cá nhân (JSON/ZIP)',
  DELETE: 'Yêu cầu xóa / Ẩn danh hóa dữ liệu',
};

export const DATA_REQUEST_TYPE_OPTIONS = (
  Object.keys(DATA_REQUEST_TYPE_LABELS) as DataRequestType[]
).map((value) => ({ value, label: DATA_REQUEST_TYPE_LABELS[value] }));

export interface MFAStatus {
  enabled: boolean;
  enabled_at?: string | null;
  secret?: string;
  otpauth_url?: string;
  qr_url?: string;
  recovery_codes?: string[];
}

export interface MFASetupData {
  secret: string;
  otpauth_url: string;
  qr_url?: string;
  recovery_codes?: string[];
}

export interface ActiveSession {
  id: number | string;
  user_id?: number;
  device_name: string;
  ip?: string;
  user_agent?: string;
  is_current?: boolean;
  current?: boolean;
  last_used_at?: string | null;
  created_at: string;
  expires_at?: string;
}

export interface DataRequest {
  id: number;
  member_id: number;
  member_name?: string;
  member_code?: string;
  member_phone?: string;
  type: DataRequestType;
  status: DataRequestStatus;
  reason: string;
  file_url?: string;
  handled_by?: number | null;
  handled_by_name?: string;
  note?: string;
  created_at: string;
  done_at?: string | null;
}

class SecurityPrivacyApi {
  readonly mfaKey = 'security-mfa';
  readonly sessionKey = 'security-sessions';
  readonly dataRequestKey = 'data-requests';

  async getMfaStatus(): Promise<MFAStatus> {
    try {
      return await http.get<MFAStatus>('/admin/security/mfa');
    } catch {
      try {
        return await http.get<MFAStatus>('/auth/mfa');
      } catch {
        return { enabled: false };
      }
    }
  }

  async setupMfa(): Promise<MFASetupData> {
    try {
      return await http.post<MFASetupData>('/admin/security/mfa/setup');
    } catch {
      return await http.post<MFASetupData>('/auth/mfa/setup');
    }
  }

  async enableMfa(code: string): Promise<MFAStatus> {
    const payload = { code, totp_code: code };
    try {
      return await http.post<MFAStatus>('/admin/security/mfa/enable', payload);
    } catch {
      return await http.post<MFAStatus>('/auth/mfa/enable', payload);
    }
  }

  async disableMfa(code?: string): Promise<void> {
    const payload = { code: code ?? '', totp_code: code ?? '' };
    try {
      await http.post('/admin/security/mfa/disable', payload);
    } catch {
      await http.post('/auth/mfa/disable', payload);
    }
  }

  async listSessions(): Promise<ActiveSession[]> {
    try {
      const res = await http.get<ActiveSession[] | Page<ActiveSession>>('/admin/security/sessions');
      return Array.isArray(res) ? res : (res?.items ?? []);
    } catch {
      const res = await http.get<ActiveSession[] | Page<ActiveSession>>('/auth/sessions');
      return Array.isArray(res) ? res : (res?.items ?? []);
    }
  }

  async revokeSession(id: number | string): Promise<void> {
    try {
      await http.delete(`/admin/security/sessions/${id}`);
    } catch {
      await http.delete(`/auth/sessions/${id}`);
    }
  }

  async revokeOtherSessions(): Promise<void> {
    try {
      await http.post('/admin/security/sessions/revoke-others');
    } catch {
      await http.post('/auth/sessions/revoke-others');
    }
  }

  async listDataRequests(
    params: ListParams & { type?: DataRequestType; status?: DataRequestStatus } = {},
  ): Promise<Page<DataRequest>> {
    const res = await http.get<Page<DataRequest> | DataRequest[]>('/admin/data-requests', {
      params,
    });
    return normalizePage(res);
  }

  async processDataRequest(
    id: number,
    data: { status: 'DONE' | 'REJECTED'; note?: string },
  ): Promise<DataRequest> {
    const payload = {
      status: data.status,
      action: data.status === 'REJECTED' ? 'REJECT' : 'APPROVE',
      note: data.note ?? '',
    };
    try {
      return await http.post<DataRequest>(`/admin/data-requests/${id}/process`, payload);
    } catch {
      return await http.put<DataRequest>(`/admin/data-requests/${id}`, payload);
    }
  }
}

export const securityPrivacyApi = new SecurityPrivacyApi();
