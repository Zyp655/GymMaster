import { http } from '@/utils/api';
import type {
  Invoice,
  InvoiceDetail,
  InvoiceListParams,
  InvoicePrintData,
  PaymentMethod,
} from './invoice';
import type { Page } from './types';

/** Bán hàng tại quầy (POS) – hợp đồng `docs/api/pos-inventory.md` §5. */

export interface PosProduct {
  id: number;
  sku: string;
  barcode: string;
  name: string;
  category_id: number | null;
  category_name: string;
  unit: string;
  sell_price: number;
  /** `null` = hội viên mua theo giá bán. */
  member_price: number | null;
  image_url: string;
  stock: number;
  min_stock: number;
  low_stock: boolean;
}

export interface PosProductParams {
  branch_id: number;
  keyword?: string;
  category_id?: number;
  in_stock?: boolean;
}

export interface PosPaymentInput {
  amount: number;
  method: PaymentMethod;
  reference?: string;
}

export interface PosOrderInput {
  branch_id: number;
  member_id?: number | null;
  items: { product_id: number; quantity: number }[];
  promotion_code?: string;
  payments: PosPaymentInput[];
  note?: string;
}

export interface PosOrderResult {
  invoice: InvoiceDetail;
  received_amount: number;
  change_amount: number;
  print: InvoicePrintData;
}

export interface PosOrderLine {
  product_id: number;
  description: string;
  sold_quantity: number;
  returned_quantity: number;
  returnable_quantity: number;
  amount: number;
  refunded_value: number;
}

export interface PosOrderDetail {
  invoice: InvoiceDetail;
  lines: PosOrderLine[];
}

export interface PosReturnInput {
  items: { product_id: number; quantity: number }[];
  refund_method: PaymentMethod;
  reason: string;
}

export interface PosReturnResult extends PosOrderDetail {
  voucher_no: string;
  refund_amount: number;
}

export type PosOrderListParams = Omit<InvoiceListParams, 'source'>;

export interface PosSummary {
  date: string;
  branch_id: number;
  order_count: number;
  items_sold: number;
  gross_revenue: number;
  discount: number;
  refund_amount: number;
  net_revenue: number;
  cost_of_goods: number;
  gross_profit: number;
  by_method: { method: PaymentMethod; amount: number; count: number }[];
  top_products: { product_id: number; sku: string; name: string; quantity: number; revenue: number }[];
}

class PosApi {
  readonly key = 'pos';

  products(params: PosProductParams) {
    return http.get<PosProduct[]>('/admin/pos/products', { params });
  }

  byBarcode(code: string, branchId: number) {
    return http.get<PosProduct>(`/admin/pos/products/by-barcode/${encodeURIComponent(code)}`, {
      params: { branch_id: branchId },
    });
  }

  createOrder(data: PosOrderInput) {
    return http.post<PosOrderResult>('/admin/pos/orders', data);
  }

  orders(params: PosOrderListParams = {}) {
    return http.get<Page<Invoice>>('/admin/pos/orders', { params });
  }

  order(invoiceId: number) {
    return http.get<PosOrderDetail>(`/admin/pos/orders/${invoiceId}`);
  }

  returnItems(invoiceId: number, data: PosReturnInput) {
    return http.post<PosReturnResult>(`/admin/pos/orders/${invoiceId}/returns`, data);
  }

  summary(params: { branch_id?: number; date?: string } = {}) {
    return http.get<PosSummary>('/admin/pos/summary', { params });
  }
}

export const posApi = new PosApi();
export default posApi;
