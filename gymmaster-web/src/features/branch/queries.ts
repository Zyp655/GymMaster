'use client';

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { branchApi } from './api';
import type { BranchListParams, BranchPayload } from './types';

export const branchKeys = {
  all: ['branches'] as const,
  lists: () => [...branchKeys.all, 'list'] as const,
  list: (params: BranchListParams) => [...branchKeys.lists(), params] as const,
  details: () => [...branchKeys.all, 'detail'] as const,
  detail: (id: number) => [...branchKeys.details(), id] as const,
};

export function useBranchesQuery(params: BranchListParams) {
  return useQuery({
    queryKey: branchKeys.list(params),
    queryFn: ({ signal }) => branchApi.list(params, signal),
    placeholderData: keepPreviousData,
  });
}

export function useBranchQuery(id: number | undefined) {
  return useQuery({
    queryKey: branchKeys.detail(id ?? 0),
    queryFn: () => branchApi.get(id as number),
    enabled: !!id,
  });
}

export function useCreateBranch() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: BranchPayload) => branchApi.create(payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: branchKeys.lists() }),
  });
}

export function useUpdateBranch() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: BranchPayload }) =>
      branchApi.update(id, payload),
    onSuccess: branch => {
      qc.setQueryData(branchKeys.detail(branch.id), branch);
      return qc.invalidateQueries({ queryKey: branchKeys.lists() });
    },
  });
}

export function useDeleteBranch() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => branchApi.remove(id),
    onSuccess: (_data, id) => {
      qc.removeQueries({ queryKey: branchKeys.detail(id) });
      return qc.invalidateQueries({ queryKey: branchKeys.lists() });
    },
  });
}
