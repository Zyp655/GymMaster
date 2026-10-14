import { http } from '@/utils/api';
import type { ActiveStatus, ListParams, Page } from './types';

/** Khuyến mãi – permission `promotion.view` / `promotion.manage` (docs/api/catalog.md §3). */

export type PromotionType = 'PERCENT' | 'FIXED';
export type PromotionAppliesTo = 'ALL' | 'MEMBERSHIP' | 'PT' | 'PRODUCT';
export type PromotionValidity = 'active' | 'upcoming' | 'expired' | 'exhausted' | 'inactive';

export interface Promotion {
  id: number;
  code: string;
  name: string;
  type: PromotionType;
  value: number;
  /** Trần giảm cho PERCENT (`null` = không trần). */
  max_discount: number | null;
  min_amount: number;
  applies_to: PromotionAppliesTo;
  start_at: string;
  end_at: string;
  /** `null` = không giới hạn. */
  usage_limit: number | null;
  used_count: number;
  status: ActiveStatus;
  validity: PromotionValidity;
  created_at: string;
  updated_at: string;
}

export interface PromotionInput {
  code: string;
  name: string;
  type: PromotionType;
  value: number;
  max_discount: number | null;
  min_amount: number;
  applies_to: PromotionAppliesTo;
  start_at: string;
  end_at: string;
  usage_limit: number | null;
  status: ActiveStatus;
}

export interface PromotionListParams extends ListParams {
  status?: ActiveStatus;
  applies_to?: PromotionAppliesTo;
  validity?: Exclude<PromotionValidity, 'exhausted'>;
}

export interface PromotionCheckInput {
  code: string;
  applies_to: Exclude<PromotionAppliesTo, 'ALL'>;
  amount: number;
}

export interface PromotionCheckResult {
  promotion: Promotion;
  discount: number;
  final_amount: number;
}

class PromotionApi {
  readonly key = 'promotions';

  list(params: PromotionListParams = {}) {
    return http.get<Page<Promotion>>('/admin/promotions', { params });
  }

  get(id: number) {
    return http.get<Promotion>(`/admin/promotions/${id}`);
  }

  create(data: PromotionInput) {
    return http.post<Promotion>('/admin/promotions', data);
  }

  update(id: number, data: PromotionInput) {
    return http.put<Promotion>(`/admin/promotions/${id}`, data);
  }

  remove(id: number) {
    return http.delete(`/admin/promotions/${id}`);
  }

  /** Kiểm tra mã (không tăng lượt dùng). */
  check(data: PromotionCheckInput) {
    return http.post<PromotionCheckResult>('/admin/promotions/check', data);
  }
}

export const promotionApi = new PromotionApi();
export default promotionApi;
