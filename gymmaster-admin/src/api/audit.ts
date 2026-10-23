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

/**
 * Nhãn tiếng Việt cho hành động – đối chiếu đủ các hằng `Audit*`/`audit*`/`ptAct*` trong
 * `gymmaster-api/internal/service` (không có → hiển thị mã gốc).
 */
export const AUDIT_ACTION_LABEL: Record<string, string> = {
  CREATE: 'Tạo mới',
  UPDATE: 'Cập nhật',
  DELETE: 'Xóa',
  ACTIVATE: 'Kích hoạt',
  ADJUST: 'Điều chỉnh tồn kho',
  ASSIGN_ROLES: 'Gán vai trò',
  AUTO_NO_SHOW: 'Tự đánh vắng mặt',
  BRANCH_TRANSFER: 'Chuyển chi nhánh',
  BULK_ACTIVATE: 'Kích hoạt hàng loạt',
  BULK_DEACTIVATE: 'Ngừng hoạt động hàng loạt',
  BULK_NOTIFY: 'Gửi thông báo hàng loạt',
  BULK_TRANSFER_BRANCH: 'Chuyển chi nhánh hàng loạt',
  CANCEL: 'Hủy',
  CHANGE_PASSWORD: 'Đổi mật khẩu',
  CHANGE_TRAINER: 'Đổi HLV',
  CHECKOUT: 'Thanh toán',
  COMPLETE: 'Hoàn thành',
  COPY_WEEK: 'Sao chép lịch tuần',
  EXPIRE: 'Hết hạn',
  EXPORT: 'Xuất dữ liệu',
  FREEZE_APPROVE: 'Duyệt bảo lưu',
  FREEZE_END: 'Kết thúc bảo lưu',
  FREEZE_REJECT: 'Từ chối bảo lưu',
  FREEZE_REQUEST: 'Yêu cầu bảo lưu',
  GRANT_PORTAL: 'Cấp tài khoản cổng',
  IMPORT: 'Nhập dữ liệu',
  LOCK: 'Khóa',
  LOCK_PORTAL: 'Khóa cổng hội viên',
  LOGIN: 'Đăng nhập',
  LOGIN_FAILED: 'Đăng nhập thất bại',
  LOGOUT: 'Đăng xuất',
  NO_SHOW: 'Vắng mặt',
  PAY: 'Chi trả',
  PAYMENT: 'Thu tiền',
  PORTAL_UPDATE_PROFILE: 'Hội viên tự cập nhật hồ sơ',
  PRICE_CHANGE: 'Đổi giá',
  REFUND: 'Hoàn tiền',
  RENEW: 'Gia hạn',
  RESCHEDULE: 'Dời lịch',
  RESET_PASSWORD: 'Đặt lại mật khẩu',
  RESET_PASSWORD_REQUEST: 'Yêu cầu đặt lại mật khẩu',
  RESET_QR: 'Đổi mã QR',
  RETURN: 'Trả hàng',
  SALE: 'Bán hàng',
  SCHEDULE: 'Đặt lịch',
  SELL: 'Bán',
  TRANSFER: 'Chuyển nhượng',
  UNLOCK: 'Mở khóa',
  UPGRADE: 'Nâng cấp',
  VOID: 'Hủy hóa đơn',
  REASSIGN_TRAINER: 'Chuyển HLV dạy thay',
  SUBSTITUTE_TRAINER: 'Phân công HLV dạy thay',
  // Dự phòng cho các thao tác có thể phát sinh.
  APPROVE: 'Duyệt',
  REJECT: 'Từ chối',
  PROCESS: 'Xử lý',
  ENABLE: 'Bật',
  DISABLE: 'Tắt',
  OPEN: 'Mở ca',
  CLOSE: 'Đóng ca',
  CONVERT: 'Chuyển đổi',
};

/** Nhãn theo cặp `ACTION:entity` khi cùng một mã mang nghĩa khác nhau theo đối tượng. */
const AUDIT_ACTION_ENTITY_LABEL: Record<string, string> = {
  'IMPORT:stock_voucher': 'Nhập kho',
  'TRANSFER:stock_voucher': 'Chuyển kho',
  'PAY:trainer_commission': 'Chi hoa hồng',
  'IMPORT:member': 'Nhập hội viên',
  'IMPORT:legacy_import': 'Nhập dữ liệu cũ',
};

/** Nhãn tiếng Việt cho đối tượng. */
export const AUDIT_ENTITY_LABEL: Record<string, string> = {
  access_device: 'Thiết bị ra vào',
  announcement: 'Bảng tin',
  auth: 'Xác thực / đăng nhập',
  badge: 'Huy hiệu',
  body_metric: 'Chỉ số cơ thể',
  booking: 'Đặt lớp',
  branch: 'Chi nhánh',
  branch_holiday: 'Ngày nghỉ chi nhánh',
  cash_session: 'Ca thu ngân',
  challenge: 'Thử thách',
  class_schedule: 'Buổi lớp',
  class_schedule_series: 'Chuỗi lịch lớp',
  class_type: 'Loại lớp',
  consent: 'Đồng ý điều khoản',
  contract: 'Hợp đồng',
  contract_template: 'Mẫu hợp đồng',
  data_request: 'Yêu cầu dữ liệu cá nhân',
  day_pass: 'Vé ngày',
  equipment: 'Thiết bị tập',
  exercise: 'Bài tập',
  expense: 'Chi phí',
  food: 'Món ăn',
  installment_plan: 'Trả góp',
  invoice: 'Hóa đơn',
  job: 'Tác vụ nền',
  lead: 'Khách tiềm năng',
  leave_request: 'Đơn nghỉ phép',
  legacy_import: 'Nhập dữ liệu cũ',
  locker: 'Tủ đồ',
  locker_rental: 'Lượt thuê tủ đồ',
  loyalty_points: 'Điểm thưởng',
  loyalty_reward: 'Quà đổi điểm',
  maintenance_log: 'Nhật ký bảo trì',
  meal_plan: 'Thực đơn',
  member: 'Hội viên',
  member_contract: 'Hợp đồng hội viên',
  membership_plan: 'Gói tập',
  notification: 'Thông báo',
  online_payment_txn: 'Giao dịch thanh toán trực tuyến',
  payroll: 'Bảng lương',
  pos_order: 'Đơn bán hàng (POS)',
  post: 'Bài viết',
  product: 'Sản phẩm',
  product_category: 'Danh mục sản phẩm',
  promotion: 'Khuyến mãi',
  pt_contract: 'Hợp đồng PT',
  pt_package: 'Gói PT',
  pt_session: 'Buổi PT',
  review: 'Đánh giá',
  role: 'Vai trò',
  room: 'Phòng tập',
  sales_target: 'Chỉ tiêu doanh số',
  setting: 'Cấu hình',
  shift: 'Ca làm việc',
  staff: 'Nhân viên',
  stock_voucher: 'Phiếu kho',
  subscription: 'Đăng ký gói',
  subscription_freeze: 'Bảo lưu',
  supplier: 'Nhà cung cấp',
  survey: 'Khảo sát',
  testimonial: 'Cảm nhận hội viên',
  trainer_availability: 'Lịch rảnh HLV',
  trainer_commission: 'Hoa hồng HLV',
  trainer_profile: 'Hồ sơ HLV',
  user: 'Tài khoản',
  user_mfa: 'Xác thực 2 lớp (2FA)',
  user_session: 'Phiên đăng nhập',
  workout_plan: 'Giáo án',
};

const BULK_PREFIX = 'BULK_';

/**
 * Nhãn hành động; truyền thêm `entity` để lấy nghĩa đúng ngữ cảnh (vd `IMPORT` + `stock_voucher`
 * = "Nhập kho"). Mã `BULK_*` chưa biết → "Hàng loạt: <nhãn mã con>". Không biết → mã gốc.
 */
export function auditActionLabel(action: string, entity?: string): string {
  if (!action) return action;
  if (entity) {
    const byEntity = AUDIT_ACTION_ENTITY_LABEL[`${action}:${entity}`];
    if (byEntity) return byEntity;
  }
  const label = AUDIT_ACTION_LABEL[action];
  if (label) return label;
  if (action.startsWith(BULK_PREFIX) && action.length > BULK_PREFIX.length) {
    const inner = action.slice(BULK_PREFIX.length);
    const innerLabel = AUDIT_ACTION_LABEL[inner];
    return innerLabel ? `Hàng loạt: ${innerLabel.toLowerCase()}` : action;
  }
  return action;
}

export const auditEntityLabel = (e: string) => (e ? (AUDIT_ENTITY_LABEL[e] ?? e) : e);

/** Gộp mã có trong dữ liệu (API options) với mã đã biết → option lọc, sắp theo nhãn. */
export function buildAuditOptions(
  fromApi: readonly string[] | undefined,
  known: Record<string, string>,
  labelOf: (code: string) => string,
): { value: string; label: string }[] {
  const codes = new Set<string>([...(fromApi ?? []), ...Object.keys(known)]);
  codes.delete('');
  return [...codes]
    .map((value) => ({ value, label: labelOf(value) }))
    .sort((a, b) => a.label.localeCompare(b.label, 'vi'));
}

export const auditApi = new AuditApi();
export default auditApi;
