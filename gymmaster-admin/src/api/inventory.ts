import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

/** Kho theo chi nhánh – hợp đồng `docs/api/pos-inventory.md` §4. */

export type StockTxType =
  | 'IMPORT'
  | 'SALE'
  | 'SALE_RETURN'
  | 'TRANSFER_IN'
  | 'TRANSFER_OUT'
  | 'ADJUST';
export type VoucherKind = 'IMPORT' | 'TRANSFER' | 'ADJUST' | 'SALE' | 'SALE_RETURN';

/** Dòng thẻ kho. */
export interface StockTransaction {
  id: number;
  product_id: number;
  product_sku: string;
  product_name: string;
  unit: string;
  branch_id: number;
  branch_name: string;
  type: StockTxType;
  /** Có dấu: + nhập, − xuất. */
  quantity: number;
  /** Tồn tại chi nhánh đó sau biến động. */
  balance_after: number;
  unit_cost: number;
  reference_type: string;
  reference_id: number | null;
  supplier_id: number | null;
  supplier_name: string;
  voucher_no: string;
  note: string;
  created_by: number | null;
  created_by_name: string;
  created_at: string;
}

export interface Voucher {
  voucher_no: string;
  kind: VoucherKind;
  branch_id: number;
  branch_name: string;
  to_branch_id: number | null;
  to_branch_name: string;
  supplier_id: number | null;
  supplier_name: string;
  reference_type: string;
  reference_id: number | null;
  note: string;
  total_quantity: number;
  total_value: number;
  created_by: number | null;
  created_by_name: string;
  created_at: string;
  lines: StockTransaction[];
}

export interface ImportInput {
  branch_id: number;
  supplier_id?: number | null;
  note?: string;
  items: { product_id: number; quantity: number; unit_cost: number }[];
}

export interface TransferInput {
  from_branch_id: number;
  to_branch_id: number;
  note?: string;
  items: { product_id: number; quantity: number }[];
}

export interface AdjustmentInput {
  branch_id: number;
  note?: string;
  items: { product_id: number; actual_quantity: number }[];
}

export interface StockRow {
  product_id: number;
  sku: string;
  barcode: string;
  product_name: string;
  category_id: number | null;
  category_name: string;
  unit: string;
  image_url: string;
  branch_id: number;
  branch_name: string;
  quantity: number;
  min_stock: number;
  low_stock: boolean;
  cost_price: number;
  stock_value: number;
  sell_price: number;
}

export interface StockSummary {
  total_quantity: number;
  total_value: number;
  low_stock_count: number;
}

export interface StockListResult extends Page<StockRow> {
  summary: StockSummary;
}

export interface StockListParams extends ListParams {
  branch_id?: number;
  category_id?: number;
  low_stock?: boolean;
}

export interface TransactionListParams extends ListParams {
  product_id?: number;
  branch_id?: number;
  type?: StockTxType;
  voucher_no?: string;
  from?: string;
  to?: string;
}

export interface LowStockRow {
  product_id: number;
  sku: string;
  product_name: string;
  unit: string;
  branch_id: number;
  branch_name: string;
  quantity: number;
  min_stock: number;
}

class InventoryApi {
  readonly key = 'inventory';

  stock(params: StockListParams = {}) {
    return http.get<StockListResult>('/admin/inventory/stock', { params });
  }

  /** Toàn bộ tồn của bộ lọc (gộp các trang 100 dòng) – dùng cho phiếu kiểm kê/điều chuyển. */
  async stockAll(params: Omit<StockListParams, 'page' | 'limit'> = {}) {
    const items: StockRow[] = [];
    for (let page = 1; page <= 50; page++) {
      const res = await this.stock({ ...params, page, limit: 100 });
      items.push(...res.items);
      if (page >= res.total_page || res.items.length === 0) break;
    }
    return items;
  }

  transactions(params: TransactionListParams = {}) {
    return http.get<Page<StockTransaction>>('/admin/inventory/transactions', { params });
  }

  voucher(voucherNo: string) {
    return http.get<Voucher>(`/admin/inventory/vouchers/${encodeURIComponent(voucherNo)}`);
  }

  lowStock(branchId?: number | null) {
    return http.get<LowStockRow[]>('/admin/inventory/low-stock', {
      params: { branch_id: branchId ?? undefined },
    });
  }

  import(data: ImportInput) {
    return http.post<Voucher>('/admin/inventory/imports', data);
  }

  transfer(data: TransferInput) {
    return http.post<Voucher>('/admin/inventory/transfers', data);
  }

  adjust(data: AdjustmentInput) {
    return http.post<Voucher>('/admin/inventory/adjustments', data);
  }
}

export const inventoryApi = new InventoryApi();
export default inventoryApi;
