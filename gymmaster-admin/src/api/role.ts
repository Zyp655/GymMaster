import { http } from '@/utils/api';
import type { Permission, Role, RoleCreateInput, RoleUpdateInput } from './types';

/** Vai trò & quyền – permission `role.*`. Danh sách KHÔNG phân trang. */
class RoleApi {
  readonly key = 'roles';

  list() {
    return http.get<Role[]>('/admin/roles');
  }

  get(id: number) {
    return http.get<Role>(`/admin/roles/${id}`);
  }

  create(data: RoleCreateInput) {
    return http.post<Role>('/admin/roles', data);
  }

  update(id: number, data: RoleUpdateInput) {
    return http.put<Role>(`/admin/roles/${id}`, data);
  }

  remove(id: number) {
    return http.delete(`/admin/roles/${id}`);
  }

  permissions() {
    return http.get<Permission[]>('/admin/permissions');
  }
}

export const roleApi = new RoleApi();
export default roleApi;
