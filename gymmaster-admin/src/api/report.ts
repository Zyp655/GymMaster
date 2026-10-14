import { api, ApiError, ErrorCode, http } from '@/utils/api';
import type { Envelope } from '@/utils/api';

/** Báo cáo & dashboard – hợp đồng `docs/api/reports-system.md` mục 1. */

export type ReportGroupBy = 'day' | 'week' | 'month';

export interface ReportParams {
  branch_id?: number;
  /** YYYY-MM-DD */
  from?: string;
  /** YYYY-MM-DD (tính cả ngày) */
  to?: string;
}

export interface ReportSeriesParams extends ReportParams {
  group_by?: ReportGroupBy;
}

export interface ReportPeriod {
  from: string;
  to: string;
  days: number;
}

/** Phần tử phân bố dạng key/label/value|count/pct. */
export interface ShareItem {
  key: string;
  label: string;
  value: number;
  pct: number;
}

export interface CountShare {
  key: string;
  label: string;
  count: number;
  pct: number;
}

// ------------------------------------------------------------------ overview

export interface LowStockItem {
  product_id: number;
  sku: string;
  name: string;
  branch_id: number;
  branch_name: string;
  quantity: number;
  min_stock: number;
}

export interface ReportOverview {
  date: string;
  revenue: {
    today: number;
    month: number;
    prev_month_to_date: number;
    prev_month: number;
    change_pct: number | null;
  };
  members: {
    active: number;
    new_this_month: number;
    new_prev_month: number;
    new_change_pct: number | null;
    expiring_7d: number;
  };
  checkins: { today: number; present: number };
  classes: { today: number; capacity: number; booked: number; fill_rate: number };
  pt: { today: number; completed: number };
  pending_freezes: number;
  low_stock_count: number;
  low_stock_items: LowStockItem[];
}

// ------------------------------------------------------------------ revenue

export type RevenueType = 'MEMBERSHIP' | 'PT' | 'PRODUCT' | 'FEE' | 'OTHER';
export type RevenueMethod = 'CASH' | 'TRANSFER' | 'CARD';

export interface RevenuePoint {
  period: string;
  label: string;
  total: number;
  membership: number;
  pt: number;
  product: number;
  fee: number;
  other: number;
  payments: number;
  refunds: number;
}

export interface RevenueReport {
  period: ReportPeriod;
  group_by: ReportGroupBy;
  totals: { net: number; payments: number; refunds: number };
  previous: ReportPeriod & { net: number; change_pct: number | null };
  series: RevenuePoint[];
  by_type: (ShareItem & { key: RevenueType })[];
  by_method: (ShareItem & { key: RevenueMethod })[];
  by_branch: { branch_id: number; branch_name: string; amount: number; pct: number }[];
}

// ------------------------------------------------------------------ members

export type MemberPopulation = 'new' | 'all';

export interface MembersReportParams extends ReportSeriesParams {
  population?: MemberPopulation;
}

export interface MembersReport {
  period: ReportPeriod;
  group_by: ReportGroupBy;
  population: MemberPopulation;
  new_total: number;
  new_series: { period: string; label: string; count: number }[];
  by_source: CountShare[];
  by_gender: CountShare[];
  by_age: CountShare[];
  by_branch: CountShare[];
  status: CountShare[];
  total_members: number;
}

// ------------------------------------------------------------------ retention

export interface RetentionParams extends ReportParams {
  window_days?: number;
}

export interface RetentionRow {
  month: string;
  label: string;
  expired: number;
  renewed: number;
  pending: number;
  churned: number;
  renewal_rate: number;
  churn_rate: number;
}

export interface RetentionReport {
  period: ReportPeriod;
  window_days: number;
  summary: RetentionRow;
  months: RetentionRow[];
}

// ------------------------------------------------------------------ check-in heatmap

export interface HeatmapRow {
  /** 1 = Thứ 2 … 7 = Chủ nhật */
  dow: number;
  label: string;
  /** 24 phần tử, index = giờ */
  hours: number[];
  total: number;
  avg_per_day: number;
}

export interface CheckinHeatmap {
  period: ReportPeriod;
  matrix: HeatmapRow[];
  by_hour: number[];
  max: number;
  total: number;
  denied: number;
  unique_members: number;
  avg_per_day: number;
  peak: { dow: number; hour: number; label: string; count: number } | null;
  peak_hours: number[];
  daily: { period: string; label: string; count: number }[];
}

// ------------------------------------------------------------------ class occupancy

export interface OccupancyRow {
  key: string;
  label: string;
  id?: number;
  color?: string;
  sessions: number;
  capacity: number;
  booked: number;
  attended: number;
  no_show: number;
  cancelled: number;
  waitlist: number;
  fill_rate: number;
  attendance_rate: number;
}

export interface ClassOccupancyReport {
  period: ReportPeriod;
  summary: OccupancyRow;
  by_class_type: OccupancyRow[];
  by_trainer: OccupancyRow[];
  by_time_slot: OccupancyRow[];
  by_weekday: OccupancyRow[];
}

// ------------------------------------------------------------------ trainer performance

export interface TrainerPerformanceRow {
  trainer_id: number;
  trainer_name: string;
  level: string;
  pt_completed: number;
  pt_scheduled: number;
  pt_no_show: number;
  pt_cancelled: number;
  classes_taught: number;
  class_attendees: number;
  active_pt_members: number;
  pt_contracts_sold: number;
  pt_revenue: number;
  commission: number;
  commission_paid: number;
}

export interface TrainerPerformanceReport {
  period: ReportPeriod;
  items: TrainerPerformanceRow[];
  totals: TrainerPerformanceRow;
}

// ------------------------------------------------------------------ sales products

export interface SalesProductsParams extends ReportParams {
  limit?: number;
}

export interface SalesProductRow {
  product_id: number;
  sku: string;
  name: string;
  category_name: string;
  quantity: number;
  revenue: number;
  cost: number;
  gross_profit: number;
  margin: number;
}

export interface SalesProductsReport {
  period: ReportPeriod;
  totals: SalesProductRow;
  top: SalesProductRow[];
  by_category: {
    category_name: string;
    quantity: number;
    revenue: number;
    gross_profit: number;
    pct: number;
  }[];
  product_count: number;
}

// ------------------------------------------------------------------ subscriptions

export interface SubscriptionEventCounts {
  new: number;
  renewed: number;
  upgraded: number;
  transferred: number;
  cancelled: number;
  expired: number;
  frozen: number;
}

export interface SubscriptionsReport {
  period: ReportPeriod;
  group_by: ReportGroupBy;
  counts: SubscriptionEventCounts;
  series: (SubscriptionEventCounts & { period: string; label: string })[];
  by_plan: {
    plan_id: number;
    plan_code: string;
    plan_name: string;
    plan_type: 'TIME' | 'SESSION';
    sold: number;
    new_sales: number;
    continued: number;
    sold_amount: number;
    collected: number;
    pct: number;
  }[];
}

// ------------------------------------------------------------------ export

export type ReportExportType = 'revenue' | 'members' | 'checkins' | 'subscriptions' | 'invoices';

export interface ExportParams extends ReportParams {
  type: ReportExportType;
}

export interface DownloadedFile {
  blob: Blob;
  filename: string;
}

/** Lấy tên file từ Content-Disposition (ưu tiên `filename*` UTF-8). */
export function parseFilename(disposition: string, fallback: string): string {
  const star = /filename\*=UTF-8''([^;]+)/i.exec(disposition)?.[1];
  if (star) {
    try {
      return decodeURIComponent(star.trim());
    } catch {
      /* bỏ qua, dùng filename thường */
    }
  }
  return /filename="?([^";]+)"?/i.exec(disposition)?.[1]?.trim() || fallback;
}

class ReportApi {
  readonly key = 'reports';

  overview(params: { branch_id?: number } = {}) {
    return http.get<ReportOverview>('/admin/reports/overview', { params });
  }

  revenue(params: ReportSeriesParams = {}) {
    return http.get<RevenueReport>('/admin/reports/revenue', { params });
  }

  members(params: MembersReportParams = {}) {
    return http.get<MembersReport>('/admin/reports/members', { params });
  }

  retention(params: RetentionParams = {}) {
    return http.get<RetentionReport>('/admin/reports/retention', { params });
  }

  checkinHeatmap(params: ReportParams = {}) {
    return http.get<CheckinHeatmap>('/admin/reports/checkin-heatmap', { params });
  }

  classOccupancy(params: ReportParams = {}) {
    return http.get<ClassOccupancyReport>('/admin/reports/class-occupancy', { params });
  }

  trainerPerformance(params: ReportParams = {}) {
    return http.get<TrainerPerformanceReport>('/admin/reports/trainer-performance', { params });
  }

  salesProducts(params: SalesProductsParams = {}) {
    return http.get<SalesProductsReport>('/admin/reports/sales-products', { params });
  }

  subscriptions(params: ReportSeriesParams = {}) {
    return http.get<SubscriptionsReport>('/admin/reports/subscriptions', { params });
  }

  /**
   * Xuất Excel. Response là file (không bọc envelope); lỗi vẫn là JSON envelope.
   * Dùng `arraybuffer` để đọc được JSON lỗi đồng bộ trong `transformResponse` → interceptor
   * xử lý lỗi như mọi request khác (kể cả refresh token khi 40101).
   */
  async export(params: ExportParams): Promise<DownloadedFile> {
    const data = (await api.get('/admin/reports/export', {
      params,
      responseType: 'arraybuffer',
      transformResponse: (raw: ArrayBuffer, headers): DownloadedFile | Envelope => {
        const type = String(headers?.['content-type'] ?? '');
        if (type.includes('application/json')) {
          try {
            return JSON.parse(new TextDecoder().decode(raw)) as Envelope;
          } catch {
            return { code: ErrorCode.BAD_REQUEST, message: 'Không thể xuất báo cáo', data: null };
          }
        }
        return {
          blob: new Blob([raw], { type: type || 'application/octet-stream' }),
          filename: parseFilename(
            String(headers?.['content-disposition'] ?? ''),
            `bao-cao-${params.type}.xlsx`,
          ),
        };
      },
    })) as unknown as DownloadedFile | Envelope;

    if ('code' in data) {
      throw new ApiError(data.message || 'Không thể xuất báo cáo', data.code, 400, data.data);
    }
    return data;
  }
}

/** Lưu blob thành file tải về. */
export function saveFile({ blob, filename }: DownloadedFile): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export const reportApi = new ReportApi();
export default reportApi;
