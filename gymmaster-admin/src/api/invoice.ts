import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

/** Hóa đơn, thanh toán, hoàn tiền – hợp đồng `docs/api/sales.md` §3. */

export type InvoiceStatus = 'UNPAID' | 'PARTIAL' | 'PAID' | 'VOID' | 'REFUNDED';
export type InvoiceSource = 'COUNTER' | 'POS' | 'PORTAL';
export type InvoiceItemType = 'MEMBERSHIP' | 'PT' | 'PRODUCT' | 'FEE' | 'OTHER';
export type PaymentMethod = 'CASH' | 'TRANSFER' | 'CARD' | 'VNPAY';
export type OnlineTxnStatus =
  | 'PENDING'
  | 'SUCCESS'
  | 'FAILED'
  | 'EXPIRED'
  | 'NEEDS_REFUND'
  | 'REFUNDED';

export interface InvoiceMemberRef {
  id: number;
  member_code: string;
  full_name: string;
  phone: string;
}

/** Dòng danh sách `GET /admin/invoices`. */
export interface Invoice {
  id: number;
  invoice_no: string;
  member_id: number | null;
  /** `null` = khách vãng lai (POS). */
  member: InvoiceMemberRef | null;
  branch_id: number;
  branch_name: string;
  subtotal: number;
  discount: number;
  total: number;
  paid_amount: number;
  refunded_amount: number;
  /** Còn phải thu = total − paid_amount (0 khi VOID). */
  due: number;
  status: InvoiceStatus;
  source: InvoiceSource;
  promotion_id: number | null;
  promotion_code: string;
  note: string;
  issued_by: number | null;
  issued_by_name: string;
  issued_at: string;
  voided_at: string | null;
  void_reason: string;
  created_at: string;
  /** Phần nợ được xóa khi hủy gói (không phải giảm giá): total = subtotal − discount − write_off_amount. */
  write_off_amount?: number;
  write_off_reason?: string;
}

export interface InvoiceItem {
  id: number;
  item_type: InvoiceItemType;
  ref_id: number | null;
  description: string;
  quantity: number;
  unit_price: number;
  discount: number;
  amount: number;
}

export interface Payment {
  id: number;
  invoice_id: number;
  amount: number;
  method: PaymentMethod;
  reference: string;
  paid_at: string;
  received_by: number | null;
  received_by_name: string;
  note: string;
}

export interface Refund {
  id: number;
  invoice_id: number;
  amount: number;
  method: PaymentMethod;
  reason: string;
  refunded_by: number | null;
  refunded_by_name: string;
  created_at: string;
}

export interface InvoiceDetail extends Invoice {
  voided_by_name: string;
  items: InvoiceItem[];
  payments: Payment[];
  refunds: Refund[];
}

export interface InvoicePrintData {
  invoice: InvoiceDetail;
  branch: {
    id: number;
    code: string;
    name: string;
    address: string;
    phone: string;
    email: string;
  };
  member: InvoiceMemberRef | null;
  total_in_words: string;
  due_in_words: string;
  printed_at: string;
  printed_by_name: string;
}

export interface PaymentInput {
  amount: number;
  method: PaymentMethod;
  reference?: string;
  note?: string;
}

export interface RefundInput {
  amount: number;
  method: PaymentMethod;
  reason: string;
}

export interface InvoiceListParams extends ListParams {
  branch_id?: number;
  member_id?: number;
  status?: InvoiceStatus;
  source?: InvoiceSource;
  method?: PaymentMethod;
  /** Ngày lập từ (YYYY-MM-DD). */
  from?: string;
  to?: string;
}

/** Dòng sổ thu `GET /admin/payments`. */
export interface PaymentItem extends Payment {
  invoice_no: string;
  branch_id: number;
  branch_name: string;
  member_id: number | null;
  member_code: string;
  member_name: string;
}

export interface PaymentSummary {
  total_amount: number;
  count: number;
  /** Luôn đủ 3 phương thức CASH, TRANSFER, CARD. */
  by_method: { method: PaymentMethod; amount: number; count: number }[];
}

export interface PaymentListResult extends Page<PaymentItem> {
  summary: PaymentSummary;
}

export interface PaymentListParams extends ListParams {
  branch_id?: number;
  method?: PaymentMethod;
  from?: string;
  to?: string;
  received_by?: number;
}

export interface OnlinePaymentTxn {
  id: number;
  invoice_id: number;
  invoice_no?: string;
  member_id: number;
  member_code?: string;
  member_name?: string;
  branch_id?: number;
  branch_name?: string;
  txn_ref: string;
  amount: number;
  status: OnlineTxnStatus;
  provider_txn_no: string;
  bank_code: string;
  created_at: string;
  paid_at: string | null;
  /** Phần đã ghi vào hóa đơn; amount − recorded_amount (chưa hoàn) là phần cần hoàn cho khách. */
  recorded_amount?: number;
  refunded_amount?: number;
  /** Số còn có thể hoàn = amount − refunded_amount. */
  refundable?: number;
  note?: string;
}

export interface OnlinePaymentListParams extends ListParams {
  branch_id?: number;
  member_id?: number;
  invoice_id?: number;
  status?: OnlineTxnStatus;
  from?: string;
  to?: string;
}

class InvoiceApi {
  /** Khóa gốc – sổ thu dùng `[key, 'payments', ...]` để thu/hoàn tiền xong tự làm mới. */
  readonly key = 'invoices';

  list(params: InvoiceListParams = {}) {
    return http.get<Page<Invoice>>('/admin/invoices', { params });
  }

  get(id: number) {
    return http.get<InvoiceDetail>(`/admin/invoices/${id}`);
  }

  print(id: number) {
    return http.get<InvoicePrintData>(`/admin/invoices/${id}/print`);
  }

  collect(id: number, data: PaymentInput) {
    return http.post<InvoiceDetail>(`/admin/invoices/${id}/payments`, data);
  }

  void(id: number, reason: string) {
    return http.post<InvoiceDetail>(`/admin/invoices/${id}/void`, { reason });
  }

  refund(id: number, data: RefundInput) {
    return http.post<InvoiceDetail>(`/admin/invoices/${id}/refunds`, data);
  }

  payments(params: PaymentListParams = {}) {
    return http.get<PaymentListResult>('/admin/payments', { params });
  }

  onlinePayments(params: OnlinePaymentListParams = {}) {
    return http.get<Page<OnlinePaymentTxn>>('/admin/online-payments', { params });
  }
}

export const invoiceApi = new InvoiceApi();
export default invoiceApi;

export const PAYMENT_METHOD_LABEL: Record<PaymentMethod, string> = {
  CASH: 'Tiền mặt',
  TRANSFER: 'Chuyển khoản',
  CARD: 'Thẻ',
  VNPAY: 'VNPay Online',
};
export const PAYMENT_METHOD_OPTIONS = (Object.keys(PAYMENT_METHOD_LABEL) as PaymentMethod[]).map(
  (value) => ({ value, label: PAYMENT_METHOD_LABEL[value] }),
);
export const COUNTER_PAYMENT_METHOD_OPTIONS = (
  ['CASH', 'TRANSFER', 'CARD'] as PaymentMethod[]
).map((value) => ({ value, label: PAYMENT_METHOD_LABEL[value] }));

export const INVOICE_SOURCE_LABEL: Record<InvoiceSource, string> = {
  COUNTER: 'Quầy',
  POS: 'POS',
  PORTAL: 'Cổng hội viên',
};
export const INVOICE_SOURCE_OPTIONS = (Object.keys(INVOICE_SOURCE_LABEL) as InvoiceSource[]).map(
  (value) => ({ value, label: INVOICE_SOURCE_LABEL[value] }),
);

export const INVOICE_ITEM_TYPE_LABEL: Record<InvoiceItemType, string> = {
  MEMBERSHIP: 'Gói tập',
  PT: 'PT',
  PRODUCT: 'Sản phẩm',
  FEE: 'Phí chuyển nhượng',
  OTHER: 'Khác',
};

/** Đường dẫn trang in (ngoài layout chính). */
export const invoicePrintUrl = (id: number) => `/print/invoices/${id}`;
