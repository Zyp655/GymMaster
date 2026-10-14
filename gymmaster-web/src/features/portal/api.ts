import { apiClient } from '@/lib/http';
import type { Paginated } from '@/types/api';
import type {
  Availability,
  BodyMetric,
  Booking,
  Branch,
  CheckinItem,
  ClassesResponse,
  Freeze,
  FreezePayload,
  Invoice,
  InvoiceDetail,
  Notification,
  Profile,
  ProfilePayload,
  PTContract,
  PTSession,
  PTSessionsResponse,
  QRToken,
  Subscription,
} from './types';

/** API cổng hội viên (/v1/member/*). Hội viên được xác định từ token – không truyền member_id. */
export const portalApi = {
  profile: () => apiClient.get<Profile>('/member/profile'),
  updateProfile: (payload: ProfilePayload) => apiClient.put<Profile>('/member/profile', payload),
  uploadAvatar: (file: File) => {
    const form = new FormData();
    form.append('file', file);
    return apiClient.post<{ url: string }>('/member/uploads', form);
  },
  qr: () => apiClient.get<QRToken>('/member/qr'),

  subscriptions: () => apiClient.get<Subscription[]>('/member/subscriptions'),
  requestFreeze: (subscriptionId: number, payload: FreezePayload) =>
    apiClient.post<Freeze>(`/member/subscriptions/${subscriptionId}/freezes`, payload),

  checkins: (page: number, limit = 20) =>
    apiClient.get<Paginated<CheckinItem>>('/member/checkins', { query: { page, limit } }),

  branches: () => apiClient.get<Branch[]>('/member/branches'),
  classes: (params: { branch_id?: number; from?: string; to?: string }) =>
    apiClient.get<ClassesResponse>('/member/classes', { query: params }),
  bookClass: (scheduleId: number) => apiClient.post<Booking>(`/member/classes/${scheduleId}/book`),
  bookings: (params: { from?: string; to?: string } = {}) =>
    apiClient.get<Booking[]>('/member/bookings', { query: params }),
  cancelBooking: (id: number) => apiClient.post<Booking>(`/member/bookings/${id}/cancel`),

  ptContracts: () => apiClient.get<PTContract[]>('/member/pt-contracts'),
  ptSessions: (params: { from?: string; to?: string } = {}) =>
    apiClient.get<PTSessionsResponse>('/member/pt-sessions', { query: params }),
  bookPTSession: (payload: { contract_id: number; start_at: string }) =>
    apiClient.post<PTSession>('/member/pt-sessions', payload),
  cancelPTSession: (id: number, reason: string) =>
    apiClient.post<PTSession>(`/member/pt-sessions/${id}/cancel`, { reason }),
  availability: (contractId: number, date: string) =>
    apiClient.get<Availability>(`/member/pt-contracts/${contractId}/trainer-availability`, {
      query: { date },
    }),

  bodyMetrics: () => apiClient.get<BodyMetric[]>('/member/body-metrics'),
  invoices: (page: number, limit = 20) =>
    apiClient.get<Paginated<Invoice>>('/member/invoices', { query: { page, limit } }),
  invoice: (id: number) => apiClient.get<InvoiceDetail>(`/member/invoices/${id}`),

  notifications: (limit = 20) =>
    apiClient.get<Paginated<Notification>>('/member/notifications', { query: { page: 1, limit } }),
  unreadCount: () => apiClient.get<{ count: number }>('/member/notifications/unread-count'),
  readNotification: (id: number) => apiClient.post<Notification>(`/member/notifications/${id}/read`),
  readAllNotifications: () => apiClient.post<{ updated: number }>('/member/notifications/read-all'),
};
