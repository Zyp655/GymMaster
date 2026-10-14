import { http } from '@/utils/api';
import type { ActiveStatus, ListParams, Page } from './types';

/** Sản phẩm, danh mục sản phẩm, nhà cung cấp – hợp đồng `docs/api/pos-inventory.md` §1–3. */

export interface ProductCategory {
  id: number;
  name: string;
  sort_order: number;
  product_count: number;
  created_at: string;
  updated_at: string;
}

export interface ProductCategoryInput {
  name: string;
  sort_order: number;
}

export interface Supplier {
  id: number;
  name: string;
  phone: string;
  email: string;
  address: string;
  note: string;
  created_at: string;
  updated_at: string;
}

export type SupplierInput = Omit<Supplier, 'id' | 'created_at' | 'updated_at'>;

export interface SupplierOption {
  id: number;
  name: string;
}

export interface Product {
  id: number;
  sku: string;
  barcode: string;
  name: string;
  category_id: number | null;
  category_name: string;
  unit: string;
  /** Giá vốn bình quân gia quyền (có thể lẻ 2 chữ số). */
  cost_price: number;
  sell_price: number;
  /** `null` = hội viên mua theo giá bán. */
  member_price: number | null;
  image_url: string;
  min_stock: number;
  status: ActiveStatus;
  /** Tổng tồn mọi chi nhánh. */
  total_stock: number;
  /** Chỉ có khi truyền `branch_id`. */
  stock: number | null;
  low_stock: boolean | null;
  created_at: string;
  updated_at: string;
}

export interface ProductBranchStock {
  branch_id: number;
  branch_name: string;
  quantity: number;
  low_stock: boolean;
}

export interface ProductDetail extends Product {
  stocks: ProductBranchStock[];
}

export interface ProductInput {
  sku: string;
  barcode: string;
  name: string;
  category_id: number | null;
  unit: string;
  cost_price: number;
  sell_price: number;
  member_price: number | null;
  image_url: string;
  min_stock: number;
  status: ActiveStatus;
}

export interface ProductListParams extends ListParams {
  category_id?: number;
  status?: ActiveStatus;
  branch_id?: number;
  /** Cần `branch_id`. */
  low_stock?: boolean;
}

export type SupplierListParams = ListParams;

class ProductApi {
  readonly key = 'products';

  list(params: ProductListParams = {}) {
    return http.get<Page<Product>>('/admin/products', { params });
  }

  /** Toàn bộ sản phẩm theo bộ lọc (gộp các trang 100 dòng) – dùng cho ô chọn sản phẩm của phiếu kho. */
  async listAll(params: Omit<ProductListParams, 'page' | 'limit'> = {}) {
    const items: Product[] = [];
    for (let page = 1; page <= 50; page++) {
      const res = await this.list({ order_by: 'name', order_dir: 'ASC', ...params, page, limit: 100 });
      items.push(...res.items);
      if (page >= res.total_page || res.items.length === 0) break;
    }
    return items;
  }

  get(id: number) {
    return http.get<ProductDetail>(`/admin/products/${id}`);
  }

  create(data: ProductInput) {
    return http.post<ProductDetail>('/admin/products', data);
  }

  update(id: number, data: ProductInput) {
    return http.put<ProductDetail>(`/admin/products/${id}`, data);
  }

  remove(id: number) {
    return http.delete(`/admin/products/${id}`);
  }
}

class ProductCategoryApi {
  readonly key = 'product-categories';

  /** Mảng, không phân trang – sắp theo `sort_order, name`. */
  list(keyword?: string) {
    return http.get<ProductCategory[]>('/admin/product-categories', {
      params: { keyword: keyword || undefined },
    });
  }

  create(data: ProductCategoryInput) {
    return http.post<ProductCategory>('/admin/product-categories', data);
  }

  update(id: number, data: ProductCategoryInput) {
    return http.put<ProductCategory>(`/admin/product-categories/${id}`, data);
  }

  remove(id: number) {
    return http.delete(`/admin/product-categories/${id}`);
  }
}

class SupplierApi {
  readonly key = 'suppliers';

  list(params: SupplierListParams = {}) {
    return http.get<Page<Supplier>>('/admin/suppliers', { params });
  }

  options() {
    return http.get<SupplierOption[]>('/admin/suppliers/options');
  }

  get(id: number) {
    return http.get<Supplier>(`/admin/suppliers/${id}`);
  }

  create(data: SupplierInput) {
    return http.post<Supplier>('/admin/suppliers', data);
  }

  update(id: number, data: SupplierInput) {
    return http.put<Supplier>(`/admin/suppliers/${id}`, data);
  }

  remove(id: number) {
    return http.delete(`/admin/suppliers/${id}`);
  }
}

export const productApi = new ProductApi();
export const productCategoryApi = new ProductCategoryApi();
export const supplierApi = new SupplierApi();
export default productApi;
