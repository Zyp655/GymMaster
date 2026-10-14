import { http } from '@/utils/api';
import type { Branch, BranchInput, BranchListParams, BranchOption, Page } from './types';

/** Chi nhánh – permission `branch.*`. */
class BranchApi {
  /** Khóa gốc của TanStack Query cho resource này (dùng khi invalidate). */
  readonly key = 'branches';

  list(params: BranchListParams = {}) {
    return http.get<Page<Branch>>('/admin/branches', { params });
  }

  get(id: number) {
    return http.get<Branch>(`/admin/branches/${id}`);
  }

  create(data: BranchInput) {
    return http.post<Branch>('/admin/branches', data);
  }

  update(id: number, data: BranchInput) {
    return http.put<Branch>(`/admin/branches/${id}`, data);
  }

  remove(id: number) {
    return http.delete(`/admin/branches/${id}`);
  }

  /** Chi nhánh đang hoạt động (public, không cần quyền) – dùng cho ô chọn khi user thiếu `branch.view`. */
  publicList() {
    return http.get<BranchOption[]>('/public/branches', { silent: true });
  }
}

export const branchApi = new BranchApi();
export default branchApi;
