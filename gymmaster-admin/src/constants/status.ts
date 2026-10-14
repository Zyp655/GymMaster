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

const toOptions = (map: Record<string, StatusMeta>) =>
  Object.entries(map).map(([value, meta]) => ({ value, label: meta.label }));

export const SUBSCRIPTION_STATUS_OPTIONS = toOptions(SUBSCRIPTION_STATUS_MAP);
export const FREEZE_STATUS_OPTIONS = toOptions(FREEZE_STATUS_MAP);
export const INVOICE_STATUS_OPTIONS = toOptions(INVOICE_STATUS_MAP);

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
