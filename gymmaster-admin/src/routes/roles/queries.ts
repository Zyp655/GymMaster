import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import roleApi from '@/api/role';
import type { RoleCreateInput, RoleUpdateInput } from '@/api/types';

export function useRoles() {
  return useQuery({ queryKey: [roleApi.key, 'list'], queryFn: () => roleApi.list() });
}

export function usePermissions(enabled = true) {
  return useQuery({
    queryKey: [roleApi.key, 'permissions'],
    queryFn: () => roleApi.permissions(),
    staleTime: 10 * 60_000,
    enabled,
  });
}

export function useSaveRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (v: { id?: number; create?: RoleCreateInput; update?: RoleUpdateInput }) =>
      v.id ? roleApi.update(v.id, v.update!) : roleApi.create(v.create!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [roleApi.key] }),
  });
}

export function useDeleteRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => roleApi.remove(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [roleApi.key] }),
  });
}
