import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import shiftApi from '@/api/shift';
import type { CopyWeekInput, ShiftInput } from '@/api/shift';
import staffApi from '@/api/staff';

export function useWeekShifts(branchId: number | undefined, from: string, to: string) {
  return useQuery({
    queryKey: [shiftApi.key, 'list', { branch_id: branchId, from, to }],
    queryFn: () => shiftApi.list({ branch_id: branchId, from, to }),
    enabled: !!branchId,
  });
}

export function useStaffOptions(branchId: number | undefined) {
  return useQuery({
    queryKey: [staffApi.key, 'options', branchId],
    queryFn: () => staffApi.options(branchId),
    enabled: !!branchId,
    staleTime: 60_000,
  });
}

export function useSaveShift() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id?: number; data: ShiftInput }) =>
      id ? shiftApi.update(id, data) : shiftApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [shiftApi.key] }),
  });
}

export function useDeleteShift() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => shiftApi.remove(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [shiftApi.key] }),
  });
}

export function useCopyWeek() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CopyWeekInput) => shiftApi.copyWeek(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [shiftApi.key] }),
  });
}
