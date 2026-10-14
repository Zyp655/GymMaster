import { http } from '@/utils/api';
import type { ActiveStatus, ListParams, Page } from './types';

/** Gói tập – permission `plan.view` / `plan.manage` (docs/api/catalog.md §1). */

export type PlanType = 'TIME' | 'SESSION';
export type AccessScope = 'SINGLE_BRANCH' | 'ALL_BRANCHES';

export interface MembershipPlan {
  id: number;
  code: string;
  name: string;
  description: string;
  type: PlanType;
  duration_days: number;
  /** `null` với gói TIME. */
  session_count: number | null;
  price: number;
  access_scope: AccessScope;
  max_freeze_days: number;
  max_freeze_times: number;
  includes_classes: boolean;
  transfer_fee: number;
  status: ActiveStatus;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export type MembershipPlanInput = Omit<MembershipPlan, 'id' | 'created_at' | 'updated_at'>;

export interface MembershipPlanListParams extends ListParams {
  status?: ActiveStatus;
  type?: PlanType;
}

class MembershipPlanApi {
  readonly key = 'membership-plans';

  list(params: MembershipPlanListParams = {}) {
    return http.get<Page<MembershipPlan>>('/admin/membership-plans', { params });
  }

  /** Gói ACTIVE – dropdown bán gói. */
  options() {
    return http.get<MembershipPlan[]>('/admin/membership-plans/options');
  }

  get(id: number) {
    return http.get<MembershipPlan>(`/admin/membership-plans/${id}`);
  }

  create(data: MembershipPlanInput) {
    return http.post<MembershipPlan>('/admin/membership-plans', data);
  }

  update(id: number, data: MembershipPlanInput) {
    return http.put<MembershipPlan>(`/admin/membership-plans/${id}`, data);
  }

  setStatus(id: number, status: ActiveStatus) {
    return http.put<MembershipPlan>(`/admin/membership-plans/${id}/status`, { status });
  }

  remove(id: number) {
    return http.delete(`/admin/membership-plans/${id}`);
  }
}

export const membershipPlanApi = new MembershipPlanApi();
export default membershipPlanApi;
