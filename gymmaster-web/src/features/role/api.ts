import { apiClient } from '@/lib/http';
import type { CreateRolePayload, Permission, Role, UpdateRolePayload } from './types';

export const roleApi = {
  list: (signal?: AbortSignal) => apiClient.get<Role[]>('/admin/roles', { signal }),
  get: (id: number) => apiClient.get<Role>(`/admin/roles/${id}`),
  create: (payload: CreateRolePayload) => apiClient.post<Role>('/admin/roles', payload),
  update: (id: number, payload: UpdateRolePayload) =>
    apiClient.put<Role>(`/admin/roles/${id}`, payload),
  remove: (id: number) => apiClient.del<null>(`/admin/roles/${id}`),
  permissions: (signal?: AbortSignal) =>
    apiClient.get<Permission[]>('/admin/permissions', { signal }),
};
