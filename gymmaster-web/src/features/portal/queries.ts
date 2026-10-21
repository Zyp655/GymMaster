'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { portalApi } from './api';
import type {
  CheckoutSubscriptionPayload,
  CheckPromotionParams,
  CreateDataRequestPayload,
  CreateFoodLogPayload,
  UpdateFoodLogPayload,
  CreateGuestPassPayload,
  CreateMemberReviewPayload,
  CreateWorkoutLogPayload,
  FreezePayload,
  MFAEnablePayload,
  NotificationPrefItem,
  ProfilePayload,
  SignContractPayload,
  SubmitSurveyPayload,
  UpdateConsentPayload,
} from './types';

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
  publicPlans: () => [...portalKeys.all, 'public-plans'] as const,
  publicBranches: () => [...portalKeys.all, 'public-branches'] as const,
  vnpayReturn: (queryString: string) => [...portalKeys.all, 'vnpay-return', queryString] as const,
  workoutPlans: () => [...portalKeys.all, 'workout-plans'] as const,
  exercises: () => [...portalKeys.all, 'exercises'] as const,
  workoutLogsAll: () => [...portalKeys.all, 'workout-logs'] as const,
  workoutLogs: (exerciseId?: number) =>
    [...portalKeys.workoutLogsAll(), exerciseId ?? 'all'] as const,
  reviews: () => [...portalKeys.all, 'reviews'] as const,
  locker: () => [...portalKeys.all, 'locker'] as const,
  occupancy: () => [...portalKeys.all, 'occupancy'] as const,
  publicOccupancy: () => [...portalKeys.all, 'public-occupancy'] as const,
  announcements: () => [...portalKeys.all, 'announcements'] as const,
  surveys: () => [...portalKeys.all, 'surveys'] as const,
  installments: () => [...portalKeys.all, 'installments'] as const,
  loyalty: () => [...portalKeys.all, 'loyalty'] as const,
  badges: () => [...portalKeys.all, 'badges'] as const,
  challenges: () => [...portalKeys.all, 'challenges'] as const,
  leaderboard: (limit?: number) => [...portalKeys.all, 'leaderboard', limit ?? 10] as const,
  mealPlans: () => [...portalKeys.all, 'meal-plans'] as const,
  foodLogsAll: () => [...portalKeys.all, 'food-logs'] as const,
  foodLogs: (date?: string) => [...portalKeys.foodLogsAll(), date ?? 'today'] as const,
  contracts: () => [...portalKeys.all, 'contracts'] as const,
  guestPasses: () => [...portalKeys.all, 'guest-passes'] as const,
  sessions: () => [...portalKeys.all, 'sessions'] as const,
  consents: () => [...portalKeys.all, 'consents'] as const,
  dataRequests: () => [...portalKeys.all, 'data-requests'] as const,
  conversations: () => [...portalKeys.all, 'conversations'] as const,
  messages: (conversationId: number) =>
    [...portalKeys.all, 'messages', conversationId] as const,
  notificationPreferences: () => [...portalKeys.all, 'notification-preferences'] as const,
  calendarToken: () => [...portalKeys.all, 'calendar-token'] as const,
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

// ---- Sprint 13: Thanh toán VNPay & Tự mua gói tập -------------------------------

export function usePublicPlansQuery() {
  return useQuery({
    queryKey: portalKeys.publicPlans(),
    queryFn: () => portalApi.publicPlans(),
  });
}

export function usePublicBranchesQuery() {
  return useQuery({
    queryKey: portalKeys.publicBranches(),
    queryFn: () => portalApi.publicBranches(),
  });
}

export function useCreateInvoiceVNPayMutation() {
  return useMutation({
    mutationFn: (invoiceId: number) => portalApi.createInvoiceVNPay(invoiceId),
  });
}

export function useCheckPromotionMutation() {
  return useMutation({
    mutationFn: (params: CheckPromotionParams) => portalApi.checkPromotion(params),
  });
}

export function useCheckoutSubscriptionMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CheckoutSubscriptionPayload) => portalApi.checkoutSubscription(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.subscriptions() });
      void qc.invalidateQueries({ queryKey: portalKeys.all });
    },
  });
}

export function useVerifyVNPayReturnQuery(queryString: string, enabled = true) {
  return useQuery({
    queryKey: portalKeys.vnpayReturn(queryString),
    queryFn: () => portalApi.verifyVNPayReturn(queryString),
    enabled,
    retry: false,
  });
}

// ---- Sprint 14: Giáo án & Nhật ký tập luyện -------------------------------------

export function useWorkoutPlansQuery() {
  return useQuery({
    queryKey: portalKeys.workoutPlans(),
    queryFn: () => portalApi.getWorkoutPlans(),
  });
}

export function useExercisesQuery() {
  return useQuery({
    queryKey: portalKeys.exercises(),
    queryFn: () => portalApi.getExercises(),
  });
}

export function useWorkoutLogsQuery(exerciseId?: number, enabled = true) {
  return useQuery({
    queryKey: portalKeys.workoutLogs(exerciseId),
    queryFn: () => portalApi.getWorkoutLogs(exerciseId),
    enabled,
  });
}

export function useCreateWorkoutLogMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateWorkoutLogPayload) => portalApi.createWorkoutLog(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.workoutLogsAll() });
    },
  });
}

// ---- Sprint 15: Đánh giá HLV & Lớp học ------------------------------------------

export function useMemberReviewsQuery() {
  return useQuery({
    queryKey: portalKeys.reviews(),
    queryFn: () => portalApi.getMemberReviews(),
  });
}

export function useCreateMemberReviewMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateMemberReviewPayload) => portalApi.createMemberReview(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.reviews() });
      void qc.invalidateQueries({ queryKey: portalKeys.ptSessionsAll() });
      void qc.invalidateQueries({ queryKey: portalKeys.bookingsAll() });
    },
  });
}

// ---- Sprint 19, 23, 26: Tủ đồ, Mật độ phòng tập, Bảng tin & Khảo sát NPS --------

export function useMemberLockerQuery() {
  return useQuery({
    queryKey: portalKeys.locker(),
    queryFn: () => portalApi.getMemberLocker(),
  });
}

export function useBranchOccupancyQuery() {
  return useQuery({
    queryKey: portalKeys.occupancy(),
    queryFn: () => portalApi.getBranchOccupancy(),
    refetchInterval: 60_000,
  });
}

export function usePublicBranchOccupancyQuery() {
  return useQuery({
    queryKey: portalKeys.publicOccupancy(),
    queryFn: () => portalApi.getPublicBranchOccupancy(),
    refetchInterval: 60_000,
  });
}

export function useMemberAnnouncementsQuery() {
  return useQuery({
    queryKey: portalKeys.announcements(),
    queryFn: () => portalApi.getMemberAnnouncements(),
  });
}

export function useMemberSurveysQuery() {
  return useQuery({
    queryKey: portalKeys.surveys(),
    queryFn: () => portalApi.getMemberSurveys(),
  });
}

export function useSubmitSurveyResponseMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: SubmitSurveyPayload }) =>
      portalApi.submitSurveyResponse(id, payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.surveys() });
    },
  });
}

// ---- Sprint 21: Lịch trả góp hóa đơn --------------------------------------------

export function useMemberInstallmentsQuery() {
  return useQuery({
    queryKey: portalKeys.installments(),
    queryFn: () => portalApi.getMemberInstallments(),
  });
}

// ---- Sprint 22: Điểm thưởng, Đổi quà, Huy hiệu & Thử thách ---------------------

export function useMemberLoyaltyQuery() {
  return useQuery({
    queryKey: portalKeys.loyalty(),
    queryFn: () => portalApi.getMemberLoyalty(),
  });
}

export function useRedeemLoyaltyRewardMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (rewardId: number) => portalApi.redeemLoyaltyReward(rewardId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.loyalty() });
      void qc.invalidateQueries({ queryKey: portalKeys.profile() });
    },
  });
}

export function useMemberBadgesQuery() {
  return useQuery({
    queryKey: portalKeys.badges(),
    queryFn: () => portalApi.getMemberBadges(),
  });
}

export function useMemberChallengesQuery() {
  return useQuery({
    queryKey: portalKeys.challenges(),
    queryFn: () => portalApi.getMemberChallenges(),
  });
}

export function useJoinChallengeMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => portalApi.joinChallenge(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.challenges() });
    },
  });
}

export function useMemberLeaderboardQuery(limit = 10) {
  return useQuery({
    queryKey: portalKeys.leaderboard(limit),
    queryFn: () => portalApi.getMemberLeaderboard(limit),
  });
}

// ---- Sprint 23: Dinh dưỡng & Thực đơn -------------------------------------------

export function useMemberMealPlansQuery() {
  return useQuery({
    queryKey: portalKeys.mealPlans(),
    queryFn: () => portalApi.getMemberMealPlans(),
  });
}

export function useMemberFoodLogsQuery(date?: string) {
  return useQuery({
    queryKey: portalKeys.foodLogs(date),
    queryFn: () => portalApi.getMemberFoodLogs(date),
  });
}

export function useUpdateMemberFoodLogMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: UpdateFoodLogPayload }) =>
      portalApi.updateMemberFoodLog(id, payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.foodLogsAll() });
    },
  });
}

export function useDeleteMemberFoodLogMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => portalApi.deleteMemberFoodLog(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.foodLogsAll() });
    },
  });
}

export function useCreateMemberFoodLogMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateFoodLogPayload) => portalApi.createMemberFoodLog(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.foodLogsAll() });
    },
  });
}

// ---- Sprint 24: Hợp đồng điện tử & Vé khách mời ---------------------------------

export function useMemberContractsQuery() {
  return useQuery({
    queryKey: portalKeys.contracts(),
    queryFn: () => portalApi.getMemberContracts(),
  });
}

export function useSignMemberContractMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: SignContractPayload }) =>
      portalApi.signMemberContract(id, payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.contracts() });
    },
  });
}

export function useMemberGuestPassesQuery() {
  return useQuery({
    queryKey: portalKeys.guestPasses(),
    queryFn: () => portalApi.getMemberGuestPasses(),
  });
}

export function useCreateMemberGuestPassMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateGuestPassPayload) => portalApi.createMemberGuestPass(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.guestPasses() });
    },
  });
}

// ---- Sprint 27: Bảo mật 2FA, Phiên đăng nhập & Quyền riêng tư -------------------

export function useSetupMemberMFAMutation() {
  return useMutation({
    mutationFn: () => portalApi.setupMemberMFA(),
  });
}

export function useEnableMemberMFAMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: MFAEnablePayload) => portalApi.enableMemberMFA(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.profile() });
    },
  });
}

export function useDisableMemberMFAMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (code?: string) => portalApi.disableMemberMFA(code),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.profile() });
    },
  });
}

export function useMemberSessionsQuery() {
  return useQuery({
    queryKey: portalKeys.sessions(),
    queryFn: () => portalApi.getMemberSessions(),
  });
}

export function useRevokeOtherMemberSessionsMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (currentSessionId?: number) =>
      portalApi.revokeOtherMemberSessions(currentSessionId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.sessions() });
    },
  });
}

export function useRevokeMemberSessionMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => portalApi.revokeMemberSession(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.sessions() });
    },
  });
}

export function useMemberConsentsQuery() {
  return useQuery({
    queryKey: portalKeys.consents(),
    queryFn: () => portalApi.getMemberConsents(),
  });
}

export function useUpdateMemberConsentsMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: UpdateConsentPayload) => portalApi.updateMemberConsents(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.consents() });
    },
  });
}

export function useMemberDataRequestsQuery() {
  return useQuery({
    queryKey: portalKeys.dataRequests(),
    queryFn: () => portalApi.getMemberDataRequests(),
  });
}

export function useCreateMemberDataRequestMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateDataRequestPayload) => portalApi.createMemberDataRequest(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.dataRequests() });
    },
  });
}

// ---- Sprint 29: Nhắn tin HLV ↔ Hội viên, Tùy chọn thông báo, Lịch .ics -----------

export function useMemberConversationsQuery() {
  return useQuery({
    queryKey: portalKeys.conversations(),
    queryFn: () => portalApi.getMemberConversations(),
  });
}

export function useMemberMessagesQuery(conversationId: number | null) {
  return useQuery({
    queryKey: portalKeys.messages(conversationId ?? 0),
    queryFn: () => portalApi.getMemberMessages(conversationId!),
    enabled: conversationId !== null && conversationId > 0,
  });
}

export function useSendMemberMessageMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ conversationId, content }: { conversationId: number; content: string }) =>
      portalApi.sendMemberMessage(conversationId, content),
    onSuccess: (_, variables) => {
      void qc.invalidateQueries({ queryKey: portalKeys.messages(variables.conversationId) });
      void qc.invalidateQueries({ queryKey: portalKeys.conversations() });
    },
  });
}

export function useMemberNotificationPreferencesQuery() {
  return useQuery({
    queryKey: portalKeys.notificationPreferences(),
    queryFn: () => portalApi.getMemberNotificationPreferences(),
  });
}

export function useSaveMemberNotificationPreferencesMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (items: NotificationPrefItem[]) => portalApi.saveMemberNotificationPreferences(items),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.notificationPreferences() });
    },
  });
}

export function useMemberCalendarTokenQuery() {
  return useQuery({
    queryKey: portalKeys.calendarToken(),
    queryFn: () => portalApi.getMemberCalendarToken(),
  });
}

export function useRotateMemberCalendarTokenMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => portalApi.rotateMemberCalendarToken(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: portalKeys.calendarToken() });
    },
  });
}


