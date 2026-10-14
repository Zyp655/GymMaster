import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

/** Nhật ký thao tác – `docs/api/reports-system.md` mục 3. */

export interface AuditLog {
  id: number;
  /** `null` = Hệ thống (job định kỳ). */
  actor_id: number | null;
  actor_name: string;
  actor_email: string;
  action: string;
  entity: string;
  entity_id: number | null;
  ip: string;
  created_at: string;
}

export interface AuditChange {
  field: string;
  before: unknown;
  after: unknown;
}

export interface AuditLogDetail extends AuditLog {
  before: unknown;
  after: unknown;
  changes: AuditChange[];
}

export interface AuditLogListParams extends ListParams {
  actor_id?: number;
  entity?: string;
  entity_id?: number;
  action?: string;
  from?: string;
  to?: string;
}

export interface AuditLogOptions {
  entities: string[];
  actions: string[];
}

class AuditApi {
  readonly key = 'audit-logs';

  list(params: AuditLogListParams = {}) {
    return http.get<Page<AuditLog>>('/admin/audit-logs', { params });
  }

  get(id: number) {
    return http.get<AuditLogDetail>(`/admin/audit-logs/${id}`);
  }

  options() {
    return http.get<AuditLogOptions>('/admin/audit-logs/options');
  }
}

/** Nhãn tiếng Việt cho hành động (không có → hiển thị mã gốc). */
export const AUDIT_ACTION_LABEL: Record<string, string> = {
  CREATE: 'Tạo mới',
  UPDATE: 'Cập nhật',
  DELETE: 'Xóa',
  ACTIVATE: 'Kích hoạt',
  ASSIGN_ROLES: 'Gán vai trò',
  CANCEL: 'Hủy',
  FREEZE_APPROVE: 'Duyệt bảo lưu',
  FREEZE_REJECT: 'Từ chối bảo lưu',
  FREEZE_END: 'Kết thúc bảo lưu',
  FREEZE_REQUEST: 'Yêu cầu bảo lưu',
  GRANT_PORTAL: 'Cấp tài khoản cổng',
  IMPORT: 'Nhập kho',
  LOCK: 'Khóa',
  UNLOCK: 'Mở khóa',
  LOCK_PORTAL: 'Khóa cổng hội viên',
  PAYMENT: 'Thu tiền',
  REFUND: 'Hoàn tiền',
  RENEW: 'Gia hạn',
  RESET_PASSWORD: 'Đặt lại mật khẩu',
  RESET_QR: 'Đổi mã QR',
  SELL: 'Bán',
  TRANSFER: 'Chuyển nhượng',
  UPGRADE: 'Nâng cấp',
  VOID: 'Hủy hóa đơn',
  LOGIN: 'Đăng nhập',
  CHECKOUT: 'Thanh toán',
  ADJUST: 'Điều chỉnh',
  EXPORT: 'Xuất',
};

/** Nhãn tiếng Việt cho đối tượng. */
export const AUDIT_ENTITY_LABEL: Record<string, string> = {
  body_metric: 'Chỉ số cơ thể',
  branch: 'Chi nhánh',
  booking: 'Đặt lớp',
  class_schedule: 'Buổi lớp',
  class_schedule_series: 'Chuỗi lịch lớp',
  class_type: 'Loại lớp',
  invoice: 'Hóa đơn',
  member: 'Hội viên',
  membership_plan: 'Gói tập',
  product: 'Sản phẩm',
  product_category: 'Danh mục sản phẩm',
  promotion: 'Khuyến mãi',
  pt_contract: 'Hợp đồng PT',
  pt_package: 'Gói PT',
  pt_session: 'Buổi PT',
  role: 'Vai trò',
  room: 'Phòng tập',
  setting: 'Cấu hình',
  shift: 'Ca làm việc',
  staff: 'Nhân viên',
  stock_voucher: 'Phiếu kho',
  subscription: 'Đăng ký gói',
  subscription_freeze: 'Bảo lưu',
  supplier: 'Nhà cung cấp',
  trainer_commission: 'Hoa hồng HLV',
  user: 'Tài khoản',
};

export const auditActionLabel = (a: string) => AUDIT_ACTION_LABEL[a] ?? a;
export const auditEntityLabel = (e: string) => AUDIT_ENTITY_LABEL[e] ?? e;

export const auditApi = new AuditApi();
export default auditApi;
