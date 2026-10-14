'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { roleApi } from './api';
import type { CreateRolePayload, UpdateRolePayload } from './types';

export const roleKeys = {
  all: ['roles'] as const,
  list: () => [...roleKeys.all, 'list'] as const,
  detail: (id: number) => [...roleKeys.all, 'detail', id] as const,
  permissions: ['permissions'] as const,
};

export function useRolesQuery() {
  return useQuery({ queryKey: roleKeys.list(), queryFn: ({ signal }) => roleApi.list(signal) });
}

export function usePermissionsQuery(enabled = true) {
  return useQuery({
    queryKey: roleKeys.permissions,
    queryFn: ({ signal }) => roleApi.permissions(signal),
    // Danh sách quyền chỉ đổi khi deploy migration mới.
    staleTime: 10 * 60_000,
    enabled,
  });
}

export function useCreateRole() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateRolePayload) => roleApi.create(payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: roleKeys.all }),
  });
}

export function useUpdateRole() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: UpdateRolePayload }) =>
      roleApi.update(id, payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: roleKeys.all }),
  });
}

export function useDeleteRole() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => roleApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: roleKeys.all }),
  });
}
