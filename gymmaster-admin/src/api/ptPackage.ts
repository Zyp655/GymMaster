import { http } from '@/utils/api';
import type { ActiveStatus, ListParams, Page } from './types';

/** Gói PT – permission `plan.view` / `plan.manage` (docs/api/catalog.md §2). */

export interface PtPackage {
  id: number;
  code: string;
  name: string;
  description: string;
  session_count: number;
  session_minutes: number;
  validity_days: number;
  price: number;
  /** `round(price / session_count)` – tham khảo. */
  price_per_session: number;
  status: ActiveStatus;
  created_at: string;
  updated_at: string;
}

export type PtPackageInput = Omit<
  PtPackage,
  'id' | 'price_per_session' | 'created_at' | 'updated_at'
>;

export interface PtPackageListParams extends ListParams {
  status?: ActiveStatus;
}

class PtPackageApi {
  readonly key = 'pt-packages';

  list(params: PtPackageListParams = {}) {
    return http.get<Page<PtPackage>>('/admin/pt-packages', { params });
  }

  /** Gói ACTIVE – dropdown bán PT. */
  options() {
    return http.get<PtPackage[]>('/admin/pt-packages/options');
  }

  get(id: number) {
    return http.get<PtPackage>(`/admin/pt-packages/${id}`);
  }

  create(data: PtPackageInput) {
    return http.post<PtPackage>('/admin/pt-packages', data);
  }

  update(id: number, data: PtPackageInput) {
    return http.put<PtPackage>(`/admin/pt-packages/${id}`, data);
  }

  setStatus(id: number, status: ActiveStatus) {
    return http.put<PtPackage>(`/admin/pt-packages/${id}/status`, { status });
  }

  remove(id: number) {
    return http.delete(`/admin/pt-packages/${id}`);
  }
}

export const ptPackageApi = new PtPackageApi();
export default ptPackageApi;
