'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { portalApi } from './api';
import type { FreezePayload, ProfilePayload } from './types';

export const portalKeys = {
  all: ['portal'] as const,
  profile: () => [...portalKeys.all, 'profile'] as const,
  qr: () => [...portalKeys.all, 'qr'] as const,
  subscriptions: () => [...portalKeys.all, 'subscriptions'] as const,
  checkins: (page: number, limit: number) =>
    [...portalKeys.all, 'checkins', { page, limit }] as const,
  branches: () => [...portalKeys.all, 'branches'] as const,
  classesAll: () => [...portalKeys.all, 'classes'] as const,
  classes: (params: { branch_id?: number; from?: string; to?: string }) =>
    [...portalKeys.classesAll(), params] as const,
  bookingsAll: () => [...portalKeys.all, 'bookings'] as const,
  bookings: (params: { from?: string; to?: string }) =>
    [...portalKeys.bookingsAll(), params] as const,
  ptContracts: () => [...portalKeys.all, 'pt-contracts'] as const,
  ptSessionsAll: () => [...portalKeys.all, 'pt-sessions'] as const,
  ptSessions: (params: { from?: string; to?: string }) =>
    [...portalKeys.ptSessionsAll(), params] as const,
  availabilityAll: () => [...portalKeys.all, 'availability'] as const,
  availability: (contractId: number, date: string) =>
    [...portalKeys.availabilityAll(), contractId, date] as const,
  bodyMetrics: () => [...portalKeys.all, 'body-metrics'] as const,
  invoices: (page: number, limit: number) =>
    [...portalKeys.all, 'invoices', { page, limit }] as const,
  invoice: (id: number) => [...portalKeys.all, 'invoice', id] as const,
  notificationsAll: () => [...portalKeys.all, 'notifications'] as const,
  notifications: (limit: number) => [...portalKeys.notificationsAll(), limit] as const,
  unreadCount: () => [...portalKeys.all, 'unread-count'] as const,
};

export function useProfileQuery() {
  return useQuery({
    queryKey: portalKeys.profile(),
    queryFn: () => portalApi.profile(),
  });
}

export function useUpdateProfileMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: ProfilePayload) => portalApi.updateProfile(payload),
    onSuccess: data => {
      qc.setQueryData(portalKeys.profile(), data);
    },
  });
}

export function useUploadAvatarMutation() {
  return useMutation({
    mutationFn: (file: File) => portalApi.uploadAvatar(file),
  });
}

export function useQrQuery() {
  return useQuery({
    queryKey: portalKeys.qr(),
    queryFn: () => portalApi.qr(),
    staleTime: 0,
    refetchOnWindowFocus: true,
  });
}

export function useSubscriptionsQuery() {
  return useQuery({
    queryKey: portalKeys.subscriptions(),
    queryFn: () => portalApi.subscriptions(),
  });
}

export function useRequestFreezeMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ subscriptionId, payload }: { subscriptionId: number; payload: FreezePayload }) =>
      portalApi.requestFreeze(subscriptionId, payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.subscriptions() });
      void qc.invalidateQueries({ queryKey: portalKeys.profile() });
    },
  });
}

export function useCheckinsQuery(page: number, limit = 10) {
  return useQuery({
    queryKey: portalKeys.checkins(page, limit),
    queryFn: () => portalApi.checkins(page, limit),
  });
}

export function useBranchesQuery() {
  return useQuery({
    queryKey: portalKeys.branches(),
    queryFn: () => portalApi.branches(),
  });
}

export function useClassesQuery(
  params: { branch_id?: number; from?: string; to?: string },
  enabled = true,
) {
  return useQuery({
    queryKey: portalKeys.classes(params),
    queryFn: () => portalApi.classes(params),
    enabled,
  });
}

export function useBookingsQuery(params: { from?: string; to?: string } = {}) {
  return useQuery({
    queryKey: portalKeys.bookings(params),
    queryFn: () => portalApi.bookings(params),
  });
}

export function useBookClassMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (scheduleId: number) => portalApi.bookClass(scheduleId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.classesAll() });
      void qc.invalidateQueries({ queryKey: portalKeys.bookingsAll() });
    },
  });
}

export function useCancelBookingMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => portalApi.cancelBooking(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.classesAll() });
      void qc.invalidateQueries({ queryKey: portalKeys.bookingsAll() });
    },
  });
}

export function usePTContractsQuery() {
  return useQuery({
    queryKey: portalKeys.ptContracts(),
    queryFn: () => portalApi.ptContracts(),
  });
}

export function usePTSessionsQuery(params: { from?: string; to?: string } = {}) {
  return useQuery({
    queryKey: portalKeys.ptSessions(params),
    queryFn: () => portalApi.ptSessions(params),
  });
}

export function useAvailabilityQuery(contractId: number | null, date: string) {
  return useQuery({
    queryKey: portalKeys.availability(contractId ?? 0, date),
    queryFn: () => portalApi.availability(contractId!, date),
    enabled: Boolean(contractId && date),
  });
}

export function useBookPTSessionMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { contract_id: number; start_at: string }) =>
      portalApi.bookPTSession(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.ptContracts() });
      void qc.invalidateQueries({ queryKey: portalKeys.ptSessionsAll() });
      void qc.invalidateQueries({ queryKey: portalKeys.availabilityAll() });
    },
  });
}

export function useCancelPTSessionMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) =>
      portalApi.cancelPTSession(id, reason),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.ptContracts() });
      void qc.invalidateQueries({ queryKey: portalKeys.ptSessionsAll() });
      void qc.invalidateQueries({ queryKey: portalKeys.availabilityAll() });
    },
  });
}

export function useBodyMetricsQuery() {
  return useQuery({
    queryKey: portalKeys.bodyMetrics(),
    queryFn: () => portalApi.bodyMetrics(),
  });
}

export function useInvoicesQuery(page: number, limit = 10) {
  return useQuery({
    queryKey: portalKeys.invoices(page, limit),
    queryFn: () => portalApi.invoices(page, limit),
  });
}

export function useInvoiceDetailQuery(id: number | null) {
  return useQuery({
    queryKey: portalKeys.invoice(id ?? 0),
    queryFn: () => portalApi.invoice(id!),
    enabled: id !== null,
  });
}

export function useNotificationsQuery(limit = 50) {
  return useQuery({
    queryKey: portalKeys.notifications(limit),
    queryFn: () => portalApi.notifications(limit),
  });
}

export function useUnreadCountQuery(enabled = true) {
  return useQuery({
    queryKey: portalKeys.unreadCount(),
    queryFn: () => portalApi.unreadCount(),
    enabled,
    refetchInterval: 60_000,
  });
}

export function useReadNotificationMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => portalApi.readNotification(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.notificationsAll() });
      void qc.invalidateQueries({ queryKey: portalKeys.unreadCount() });
    },
  });
}

export function useReadAllNotificationsMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => portalApi.readAllNotifications(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.notificationsAll() });
      void qc.invalidateQueries({ queryKey: portalKeys.unreadCount() });
    },
  });
}
