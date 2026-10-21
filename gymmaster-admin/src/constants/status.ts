/** Nhãn + màu trạng thái dùng chung (StatusTag, FilterBar, MobileList). */
export interface StatusMeta {
  label: string;
  color: string;
}

/**
 * Bảng trạng thái dùng chung. Module mới có trạng thái riêng thì truyền `map`
 * hoặc bổ sung vào đây (chỉ THÊM, không đổi nhãn có sẵn).
 */
export const STATUS_PRESETS: Record<string, StatusMeta> = {
  ACTIVE: { label: 'Đang hoạt động', color: 'green' },
  INACTIVE: { label: 'Ngừng hoạt động', color: 'default' },
  PENDING: { label: 'Chờ xử lý', color: 'gold' },
  APPROVED: { label: 'Đã duyệt', color: 'green' },
  REJECTED: { label: 'Từ chối', color: 'red' },
  DRAFT: { label: 'Nháp', color: 'default' },
  PAID: { label: 'Đã thanh toán', color: 'green' },
  UNPAID: { label: 'Chưa thanh toán', color: 'orange' },
  PARTIAL: { label: 'Thanh toán một phần', color: 'gold' },
  VOID: { label: 'Đã hủy', color: 'red' },
  REFUNDED: { label: 'Đã hoàn tiền', color: 'purple' },
  CANCELLED: { label: 'Đã hủy', color: 'red' },
  EXPIRED: { label: 'Hết hạn', color: 'default' },
  FROZEN: { label: 'Bảo lưu', color: 'blue' },
  COMPLETED: { label: 'Hoàn thành', color: 'green' },
  SCHEDULED: { label: 'Đã lên lịch', color: 'blue' },
  LOCKED: { label: 'Đã khóa', color: 'red' },
  TRANSFERRED: { label: 'Đã chuyển nhượng', color: 'purple' },
  ENDED: { label: 'Đã kết thúc', color: 'default' },
  VNPAY: { label: 'VNPay Online', color: 'geekblue' },
  SUCCESS: { label: 'Thành công', color: 'green' },
  FAILED: { label: 'Thất bại', color: 'red' },
  ARCHIVED: { label: 'Lưu trữ', color: 'default' },
  BROKEN: { label: 'Đang hỏng', color: 'red' },
  MAINTENANCE: { label: 'Đang bảo trì', color: 'gold' },
  RETIRED: { label: 'Ngừng sử dụng', color: 'default' },
  OPEN: { label: 'Chờ xử lý', color: 'orange' },
  IN_PROGRESS: { label: 'Đang thực hiện', color: 'blue' },
  DONE: { label: 'Hoàn tất', color: 'green' },
  NEW: { label: 'Mới', color: 'blue' },
  CONTACTED: { label: 'Đã liên hệ', color: 'cyan' },
  TRIAL: { label: 'Tập thử', color: 'gold' },
  CONVERTED: { label: 'Đã thành HV', color: 'green' },
  LOST: { label: 'Thất bại', color: 'default' },
  OFFLINE: { label: 'Mất kết nối', color: 'red' },
  DISABLED: { label: 'Vô hiệu hóa', color: 'default' },
  AVAILABLE: { label: 'Trống', color: 'green' },
  OCCUPIED: { label: 'Theo lượt', color: 'orange' },
  RENTED: { label: 'Thuê tháng', color: 'blue' },
  CLOSED: { label: 'Đã đóng', color: 'default' },
  OVERDUE: { label: 'Quá hạn', color: 'red' },
  SIGNED: { label: 'Đã ký', color: 'green' },
  PROCESSING: { label: 'Đang xử lý', color: 'blue' },
};

/** Tùy chọn trạng thái hoạt động cho FilterBar/Select. */
export const ACTIVE_STATUS_OPTIONS = [
  { label: STATUS_PRESETS.ACTIVE.label, value: 'ACTIVE' },
  { label: STATUS_PRESETS.INACTIVE.label, value: 'INACTIVE' },
];

/* ------------------------------------------------ Bán gói – đăng ký – hóa đơn (docs/api/sales.md) */

/** Trạng thái gói tập của hội viên (`subscriptions.status`). */
export const SUBSCRIPTION_STATUS_MAP: Record<string, StatusMeta> = {
  PENDING: { label: 'Chờ thanh toán', color: 'gold' },
  ACTIVE: { label: 'Đang hoạt động', color: 'green' },
  FROZEN: { label: 'Bảo lưu', color: 'blue' },
  EXPIRED: { label: 'Hết hạn', color: 'default' },
  CANCELLED: { label: 'Đã hủy', color: 'red' },
  TRANSFERRED: { label: 'Đã chuyển nhượng', color: 'purple' },
};

/** Trạng thái yêu cầu bảo lưu. */
export const FREEZE_STATUS_MAP: Record<string, StatusMeta> = {
  PENDING: { label: 'Chờ duyệt', color: 'gold' },
  APPROVED: { label: 'Đã duyệt', color: 'green' },
  REJECTED: { label: 'Từ chối', color: 'red' },
  ENDED: { label: 'Đã kết thúc', color: 'default' },
  CANCELLED: { label: 'Đã hủy', color: 'default' },
};

/** Trạng thái hóa đơn. */
export const INVOICE_STATUS_MAP: Record<string, StatusMeta> = {
  UNPAID: { label: 'Chưa thu', color: 'orange' },
  PARTIAL: { label: 'Thu một phần', color: 'gold' },
  PAID: { label: 'Đã thanh toán', color: 'green' },
  VOID: { label: 'Đã hủy', color: 'red' },
  REFUNDED: { label: 'Đã hoàn tiền', color: 'purple' },
};

/** Phương thức thanh toán. */
export const PAYMENT_METHOD_MAP: Record<string, StatusMeta> = {
  CASH: { label: 'Tiền mặt', color: 'green' },
  TRANSFER: { label: 'Chuyển khoản', color: 'blue' },
  CARD: { label: 'Thẻ', color: 'cyan' },
  VNPAY: { label: 'VNPay Online', color: 'geekblue' },
};

/** Trạng thái giao dịch thanh toán trực tuyến (`online_payment_txns.status`). */
export const ONLINE_TXN_STATUS_MAP: Record<string, StatusMeta> = {
  PENDING: { label: 'Chờ thanh toán', color: 'gold' },
  SUCCESS: { label: 'Thành công', color: 'green' },
  FAILED: { label: 'Thất bại', color: 'red' },
  EXPIRED: { label: 'Hết hạn', color: 'default' },
  NEEDS_REFUND: { label: 'Đã nhận tiền – cần hoàn', color: 'volcano' },
  REFUNDED: { label: 'Đã hoàn tiền', color: 'purple' },
};

/** Phương thức thu tại quầy (không gồm VNPay – VNPay chỉ do hội viên tự thanh toán trên cổng). */
export const COUNTER_PAYMENT_METHODS = ['CASH', 'TRANSFER', 'CARD'] as const;

const toOptions = (map: Record<string, StatusMeta>) =>
  Object.entries(map).map(([value, meta]) => ({ value, label: meta.label }));

export const SUBSCRIPTION_STATUS_OPTIONS = toOptions(SUBSCRIPTION_STATUS_MAP);
export const FREEZE_STATUS_OPTIONS = toOptions(FREEZE_STATUS_MAP);
export const INVOICE_STATUS_OPTIONS = toOptions(INVOICE_STATUS_MAP);
export const PAYMENT_METHOD_STATUS_OPTIONS = toOptions(PAYMENT_METHOD_MAP);
export const ONLINE_TXN_STATUS_OPTIONS = toOptions(ONLINE_TXN_STATUS_MAP);
export const COUNTER_PAYMENT_METHOD_OPTIONS = COUNTER_PAYMENT_METHODS.map((value) => ({
  value,
  label: PAYMENT_METHOD_MAP[value].label,
}));

/* ------------------------------------------------ POS & kho (docs/api/pos-inventory.md) */

/** Trạng thái sản phẩm. */
export const PRODUCT_STATUS_MAP: Record<string, StatusMeta> = {
  ACTIVE: { label: 'Đang kinh doanh', color: 'green' },
  INACTIVE: { label: 'Ngừng kinh doanh', color: 'default' },
};

/** Loại biến động thẻ kho (`stock_transactions.type`). */
export const STOCK_TX_TYPE_MAP: Record<string, StatusMeta> = {
  IMPORT: { label: 'Nhập hàng', color: 'green' },
  SALE: { label: 'Bán hàng', color: 'blue' },
  SALE_RETURN: { label: 'Trả hàng/hủy đơn', color: 'purple' },
  TRANSFER_IN: { label: 'Nhận điều chuyển', color: 'cyan' },
  TRANSFER_OUT: { label: 'Xuất điều chuyển', color: 'orange' },
  ADJUST: { label: 'Kiểm kê', color: 'gold' },
};

/** Loại phiếu kho (`voucher.kind`). */
export const VOUCHER_KIND_MAP: Record<string, StatusMeta> = {
  IMPORT: { label: 'Phiếu nhập', color: 'green' },
  TRANSFER: { label: 'Phiếu điều chuyển', color: 'cyan' },
  ADJUST: { label: 'Phiếu kiểm kê', color: 'gold' },
  SALE: { label: 'Hóa đơn bán', color: 'blue' },
  SALE_RETURN: { label: 'Phiếu trả hàng', color: 'purple' },
};

export const PRODUCT_STATUS_OPTIONS = toOptions(PRODUCT_STATUS_MAP);
export const STOCK_TX_TYPE_OPTIONS = toOptions(STOCK_TX_TYPE_MAP);

/* ------------------------------------------------ Giáo án – Thiết bị – CRM (Sprint 14–16) */

/** Trạng thái giáo án tập luyện (`workout_plans.status`). */
export const WORKOUT_PLAN_STATUS_MAP: Record<string, StatusMeta> = {
  DRAFT: { label: 'Nháp', color: 'default' },
  ACTIVE: { label: 'Đang áp dụng', color: 'green' },
  COMPLETED: { label: 'Hoàn thành', color: 'blue' },
  ARCHIVED: { label: 'Lưu trữ', color: 'orange' },
};

/** Trạng thái thiết bị phòng tập (`equipments.status`). */
export const EQUIPMENT_STATUS_MAP: Record<string, StatusMeta> = {
  ACTIVE: { label: 'Đang hoạt động', color: 'green' },
  BROKEN: { label: 'Đang hỏng', color: 'red' },
  MAINTENANCE: { label: 'Đang bảo trì', color: 'gold' },
  RETIRED: { label: 'Ngừng sử dụng', color: 'default' },
};

/** Trạng thái phiếu bảo trì (`maintenance_logs.status`). */
export const MAINTENANCE_STATUS_MAP: Record<string, StatusMeta> = {
  OPEN: { label: 'Chờ xử lý', color: 'orange' },
  IN_PROGRESS: { label: 'Đang thực hiện', color: 'blue' },
  DONE: { label: 'Hoàn tất', color: 'green' },
};

/** Trạng thái khách tiềm năng CRM (`leads.status`). */
export const LEAD_STATUS_MAP: Record<string, StatusMeta> = {
  NEW: { label: 'Mới', color: 'blue' },
  CONTACTED: { label: 'Đã liên hệ', color: 'cyan' },
  TRIAL: { label: 'Tập thử', color: 'gold' },
  CONVERTED: { label: 'Đã thành HV', color: 'green' },
  LOST: { label: 'Thất bại', color: 'default' },
};

export const WORKOUT_PLAN_STATUS_OPTIONS = toOptions(WORKOUT_PLAN_STATUS_MAP);
export const EQUIPMENT_STATUS_OPTIONS = toOptions(EQUIPMENT_STATUS_MAP);
export const MAINTENANCE_STATUS_OPTIONS = toOptions(MAINTENANCE_STATUS_MAP);
export const LEAD_STATUS_OPTIONS = toOptions(LEAD_STATUS_MAP);

/* ------------------------------------------------ Giai đoạn 3 & 4 (Sprint 18–27) */

/** Trạng thái thiết bị kiểm soát ra vào & Kiosk (`access_devices.status`). */
export const ACCESS_DEVICE_STATUS_MAP: Record<string, StatusMeta> = {
  ACTIVE: { label: 'Trực tuyến (Active)', color: 'green' },
  OFFLINE: { label: 'Mất kết nối (Offline)', color: 'red' },
  DISABLED: { label: 'Vô hiệu hóa', color: 'default' },
};

/** Trạng thái tủ đồ (`lockers.status`). */
export const LOCKER_STATUS_MAP: Record<string, StatusMeta> = {
  AVAILABLE: { label: 'Trống', color: 'green' },
  OCCUPIED: { label: 'Theo lượt', color: 'orange' },
  RENTED: { label: 'Thuê tháng', color: 'blue' },
  BROKEN: { label: 'Đang hỏng', color: 'red' },
};

/** Trạng thái bảng lương (`payrolls.status`). */
export const PAYROLL_STATUS_MAP: Record<string, StatusMeta> = {
  DRAFT: { label: 'Nháp (Đang tính)', color: 'gold' },
  LOCKED: { label: 'Đã khóa sổ', color: 'blue' },
  PAID: { label: 'Đã chi lương', color: 'green' },
};

/** Trạng thái khoản chi phí (`expenses.status`). */
export const EXPENSE_STATUS_MAP: Record<string, StatusMeta> = {
  PENDING: { label: 'Chờ duyệt', color: 'gold' },
  APPROVED: { label: 'Đã duyệt', color: 'green' },
  REJECTED: { label: 'Từ chối', color: 'red' },
};

/** Trạng thái ca thu ngân (`cash_sessions.status`). */
export const CASH_SESSION_STATUS_MAP: Record<string, StatusMeta> = {
  OPEN: { label: 'Đang mở ca', color: 'green' },
  CLOSED: { label: 'Đã chốt ca', color: 'default' },
};

/** Trạng thái kỳ trả góp (`installment_schedules.status`). */
export const INSTALLMENT_STATUS_MAP: Record<string, StatusMeta> = {
  PENDING: { label: 'Chờ thu', color: 'gold' },
  PAID: { label: 'Đã thu tiền', color: 'green' },
  OVERDUE: { label: 'Quá hạn', color: 'red' },
  SETTLED: { label: 'Đã tất toán', color: 'cyan' },
  CANCELLED: { label: 'Đã hủy', color: 'default' },
};

/** Trạng thái hợp đồng hội viên (`member_contracts.status`). */
export const CONTRACT_STATUS_MAP: Record<string, StatusMeta> = {
  PENDING: { label: 'Chờ ký', color: 'gold' },
  SIGNED: { label: 'Đã ký điện tử', color: 'green' },
};

/** Trạng thái đơn xin nghỉ phép (`staff_leave_requests.status`). */
export const LEAVE_STATUS_MAP: Record<string, StatusMeta> = {
  PENDING: { label: 'Chờ duyệt', color: 'gold' },
  APPROVED: { label: 'Đã duyệt', color: 'green' },
  REJECTED: { label: 'Từ chối', color: 'red' },
};

/** Trạng thái yêu cầu dữ liệu cá nhân (`data_requests.status`). */
export const DATA_REQUEST_STATUS_MAP: Record<string, StatusMeta> = {
  PENDING: { label: 'Chờ xử lý', color: 'gold' },
  PROCESSING: { label: 'Đang xử lý', color: 'blue' },
  DONE: { label: 'Hoàn tất', color: 'green' },
  REJECTED: { label: 'Từ chối', color: 'red' },
};

export const ACCESS_DEVICE_STATUS_OPTIONS = toOptions(ACCESS_DEVICE_STATUS_MAP);
export const LOCKER_STATUS_OPTIONS = toOptions(LOCKER_STATUS_MAP);
export const PAYROLL_STATUS_OPTIONS = toOptions(PAYROLL_STATUS_MAP);
export const EXPENSE_STATUS_OPTIONS = toOptions(EXPENSE_STATUS_MAP);
export const CASH_SESSION_STATUS_OPTIONS = toOptions(CASH_SESSION_STATUS_MAP);
export const INSTALLMENT_STATUS_OPTIONS = toOptions(INSTALLMENT_STATUS_MAP);
export const CONTRACT_STATUS_OPTIONS = toOptions(CONTRACT_STATUS_MAP);
export const LEAVE_STATUS_OPTIONS = toOptions(LEAVE_STATUS_MAP);
export const DATA_REQUEST_STATUS_OPTIONS = toOptions(DATA_REQUEST_STATUS_MAP);

