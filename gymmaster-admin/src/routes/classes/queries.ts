import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import classApi from '@/api/class';
import type {
  AttendanceItem,
  ClassScheduleCancelInput,
  ClassScheduleCreateInput,
  ClassScheduleRecurringInput,
  ClassScheduleUpdateInput,
} from '@/api/class';

const OPTIONS_STALE = 5 * 60_000;

/* ------------------------------------------------------------------ Dropdown */

export function useClassTypeOptions() {
  return useQuery({
    queryKey: [classApi.typeKey, 'options'],
    queryFn: () => classApi.typeOptions(),
    staleTime: OPTIONS_STALE,
  });
}

export function useTrainerOptions(branchId?: number | null, enabled = true) {
  return useQuery({
    queryKey: ['class-options', 'trainers', branchId ?? null],
    queryFn: () => classApi.trainerOptions(branchId ?? undefined),
    staleTime: OPTIONS_STALE,
    enabled,
  });
}

export function useRoomOptions(branchId?: number | null, enabled = true) {
  return useQuery({
    queryKey: ['class-options', 'rooms', branchId ?? null],
    queryFn: () => classApi.roomOptions(branchId ?? undefined),
    staleTime: OPTIONS_STALE,
    enabled,
  });
}

export function useBranchHours() {
  return useQuery({
    queryKey: ['class-options', 'branch-hours'],
    queryFn: async () => (await classApi.branchHours()) ?? [],
    staleTime: OPTIONS_STALE,
  });
}

/* ------------------------------------------------------------------ Dữ liệu */

export function useScheduleBookings(scheduleId?: number) {
  return useQuery({
    queryKey: [classApi.key, 'bookings', scheduleId],
    queryFn: () => classApi.listBookings(scheduleId as number),
    enabled: !!scheduleId,
  });
}

export function useSchedule(scheduleId?: number) {
  return useQuery({
    queryKey: [classApi.key, 'detail', scheduleId],
    queryFn: () => classApi.getSchedule(scheduleId as number),
    enabled: !!scheduleId,
  });
}

/* ------------------------------------------------------------------ Mutation */

/** Mọi thay đổi lịch/đặt chỗ làm mới toàn bộ `[class-schedules]` (lịch tuần, chi tiết, đặt chỗ). */
function useInvalidateSchedules() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: [classApi.key] }),
      queryClient.invalidateQueries({ queryKey: ['class-bookings'] }),
    ]);
}

export function useCreateSchedule() {
  const invalidate = useInvalidateSchedules();
  return useMutation({
    mutationFn: (data: ClassScheduleCreateInput) => classApi.createSchedule(data),
    onSuccess: invalidate,
  });
}

export function useCreateRecurring() {
  const invalidate = useInvalidateSchedules();
  return useMutation({
    mutationFn: (data: ClassScheduleRecurringInput) => classApi.createRecurring(data),
    onSuccess: invalidate,
  });
}

export function useUpdateSchedule() {
  const invalidate = useInvalidateSchedules();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: ClassScheduleUpdateInput }) =>
      classApi.updateSchedule(id, data),
    onSuccess: invalidate,
  });
}

export function useCancelSchedule() {
  const invalidate = useInvalidateSchedules();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: ClassScheduleCancelInput }) =>
      classApi.cancelSchedule(id, data),
    onSuccess: invalidate,
  });
}

export function useBookMember() {
  const invalidate = useInvalidateSchedules();
  return useMutation({
    mutationFn: ({ scheduleId, memberId }: { scheduleId: number; memberId: number }) =>
      classApi.book(scheduleId, memberId),
    onSuccess: invalidate,
  });
}

export function useCancelBooking() {
  const invalidate = useInvalidateSchedules();
  return useMutation({
    mutationFn: (bookingId: number) => classApi.cancelBooking(bookingId),
    onSuccess: invalidate,
  });
}

export function useMarkAttendance() {
  const invalidate = useInvalidateSchedules();
  return useMutation({
    mutationFn: ({ scheduleId, items }: { scheduleId: number; items: AttendanceItem[] }) =>
      classApi.markAttendance(scheduleId, items),
    onSuccess: invalidate,
  });
}
