import { apiClient } from '@/lib/http';
import type { PublicBranch, PublicPlansRes } from '@/features/landing/types';
import type { Paginated } from '@/types/api';
import type {
  AnnouncementItem,
  Availability,
  BadgeItem,
  BodyMetric,
  Booking,
  Branch,
  BranchOccupancy,
  CalendarTokenRes,
  ChallengeItem,
  CheckinItem,
  CheckoutSubscriptionPayload,
  CheckoutSubscriptionRes,
  CheckPromotionParams,
  CheckPromotionRes,
  ClassesResponse,
  ConsentItem,
  ConversationItem,
  CreateDataRequestPayload,
  CreateFoodLogPayload,
  UpdateFoodLogPayload,
  CreateGuestPassPayload,
  CreateMemberReviewPayload,
  CreateWorkoutLogPayload,
  DataRequestItem,
  DeviceCheckinRes,
  DeviceHeartbeatRes,
  Exercise,
  FoodItem,
  FoodLogItem,
  FoodLogsResponse,
  Freeze,
  FreezePayload,
  GuestPassItem,
  InstallmentPlanItem,
  Invoice,
  InvoiceDetail,
  LeaderboardEntry,
  MealPlanItem,
  MemberContractItem,
  MemberLeaderboardRes,
  MemberLocker,
  MemberLoyaltyRes,
  MemberNutritionOverviewRes,
  MemberReview,
  MessageItem,
  MFAEnablePayload,
  MFAEnableRes,
  MFASetupRes,
  Notification,
  NotificationPrefItem,
  PaymentURLRes,
  Profile,
  ProfilePayload,
  PTContract,
  PTSession,
  PTSessionsResponse,
  QRToken,
  RedeemRewardRes,
  SessionItem,
  SignContractPayload,
  SubmitSurveyPayload,
  Subscription,
  SurveyItem,
  UpdateConsentPayload,
  VNPayReturnRes,
  WorkoutLog,
  WorkoutLogsResponse,
  WorkoutPlan,
} from './types';

function unwrapArray<T>(res: T[] | { items?: T[] } | null | undefined): T[] {
  if (!res) return [];
  if (Array.isArray(res)) return res;
  if (Array.isArray(res.items)) return res.items;
  return [];
}

export function extractWorkoutLogs(res: WorkoutLogsResponse | null | undefined): WorkoutLog[] {
  if (!res) return [];
  if (Array.isArray(res)) return res;
  if (Array.isArray(res.items)) return res.items;
  if (Array.isArray(res.logs)) return res.logs;
  return [];
}

export { pickDefaultExerciseId } from './workout';

export function extractWorkoutProgress(
  res: WorkoutLogsResponse | null | undefined,
): { date: string; max_weight: number; total_volume: number }[] {
  if (!res) return [];
  if (!Array.isArray(res)) {
    const rawSeries = res.progress ?? res.series;
    if (Array.isArray(rawSeries) && rawSeries.length > 0) {
      return rawSeries.map(p => ({
        date: p.date,
        max_weight: p.max_weight ?? p.MaxWeight ?? 0,
        total_volume: p.total_volume ?? p.TotalVolume ?? 0,
      }));
    }
  }
  const logs = extractWorkoutLogs(res);
  const byDate = new Map<string, { max_weight: number; total_volume: number }>();
  for (const log of logs) {
    const dateKey = (log.performed_at ?? '').slice(0, 10);
    if (!dateKey) continue;
    const prev = byDate.get(dateKey) ?? { max_weight: 0, total_volume: 0 };
    const weight = Number(log.weight) || 0;
    const volume = Number(log.volume ?? log.sets * log.reps * weight) || 0;
    byDate.set(dateKey, {
      max_weight: Math.max(prev.max_weight, weight),
      total_volume: prev.total_volume + volume,
    });
  }
  return Array.from(byDate.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([date, val]) => ({
      date,
      max_weight: val.max_weight,
      total_volume: Math.round(val.total_volume * 100) / 100,
    }));
}

export const createInvoiceVNPay = (invoiceId: number) =>
  apiClient.post<PaymentURLRes>(`/member/invoices/${invoiceId}/vnpay`);

export const checkPromotion = (params: CheckPromotionParams) =>
  apiClient.get<CheckPromotionRes>('/member/promotions/check', {
    query: {
      code: params.code,
      plan_id: params.plan_id,
      amount: params.amount,
    },
  });

export const checkoutSubscription = (payload: CheckoutSubscriptionPayload) =>
  apiClient.post<CheckoutSubscriptionRes>('/member/subscriptions/checkout', payload);

export const verifyVNPayReturn = (queryString: string) => {
  const cleanQuery = queryString.replace(/^\?/, '');
  const path = cleanQuery
    ? `/public/payments/vnpay/return?${cleanQuery}`
    : '/public/payments/vnpay/return';
  return apiClient.get<VNPayReturnRes>(path, { auth: false });
};

export const getWorkoutPlans = async (): Promise<WorkoutPlan[]> => {
  const res = await apiClient.get<WorkoutPlan[] | { items?: WorkoutPlan[] }>(
    '/member/workout-plans',
  );
  return unwrapArray(res);
};

export const getExercises = async (): Promise<Exercise[]> => {
  const res = await apiClient.get<Exercise[] | { items?: Exercise[] }>('/member/exercises');
  return unwrapArray(res);
};

export const getWorkoutLogs = (exerciseId?: number) =>
  apiClient.get<WorkoutLogsResponse>('/member/workout-logs', {
    query: exerciseId ? { exercise_id: exerciseId } : undefined,
  });

export const createWorkoutLog = (input: CreateWorkoutLogPayload) =>
  apiClient.post<WorkoutLog>('/member/workout-logs', input);

export const getMemberReviews = async (): Promise<MemberReview[]> => {
  const res = await apiClient.get<MemberReview[] | { items?: MemberReview[] }>('/member/reviews');
  return unwrapArray(res);
};

export const createMemberReview = (payload: CreateMemberReviewPayload) =>
  apiClient.post<MemberReview>('/member/reviews', payload);

// ---- Sprint 18: Kiosk tự check-in toàn màn hình ---------------------------------

export const deviceCheckin = (qrToken: string, deviceKey: string) =>
  apiClient.post<DeviceCheckinRes>(
    '/public/device/checkin',
    {
      qr_token: qrToken,
      token: qrToken,
      code: qrToken,
      device_key: deviceKey,
    },
    {
      auth: false,
      headers: deviceKey ? { 'X-Device-Key': deviceKey } : undefined,
    },
  );

export const deviceHeartbeat = (deviceKey: string) =>
  apiClient.post<DeviceHeartbeatRes>(
    '/public/device/heartbeat',
    { device_key: deviceKey },
    {
      auth: false,
      headers: deviceKey ? { 'X-Device-Key': deviceKey } : undefined,
    },
  );

// ---- Sprint 19, 23, 26: Tủ đồ, Mật độ phòng tập, Bảng tin, Khảo sát NPS --------

export const getMemberLocker = async (): Promise<MemberLocker[]> => {
  const res = await apiClient.get<
    | MemberLocker[]
    | MemberLocker
    | { items?: MemberLocker[]; locker?: MemberLocker | null; rental?: MemberLocker | null }
    | null
  >('/member/locker');
  if (!res) return [];
  if (Array.isArray(res)) return res;
  if ('items' in res && Array.isArray(res.items)) return res.items;
  if ('rental' in res && res.rental) return [res.rental];
  if ('locker' in res && res.locker) return [res.locker];
  if ('code' in res || 'locker_code' in res) return [res as MemberLocker];
  return [];
};

export const getBranchOccupancy = async (): Promise<BranchOccupancy[]> => {
  const res = await apiClient.get<
    BranchOccupancy[] | BranchOccupancy | { items?: BranchOccupancy[] } | null
  >('/member/occupancy');
  if (!res) return [];
  if (Array.isArray(res)) return res;
  if ('items' in res && Array.isArray(res.items)) return res.items;
  if ('branch_id' in res || 'capacity' in res) return [res as BranchOccupancy];
  return [];
};

export const getPublicBranchOccupancy = async (): Promise<BranchOccupancy[]> => {
  const res = await apiClient.get<
    BranchOccupancy[] | BranchOccupancy | { items?: BranchOccupancy[] } | null
  >('/public/branches/occupancy', { auth: false });
  if (!res) return [];
  if (Array.isArray(res)) return res;
  if ('items' in res && Array.isArray(res.items)) return res.items;
  if ('branch_id' in res || 'capacity' in res) return [res as BranchOccupancy];
  return [];
};

export const getMemberAnnouncements = async (): Promise<AnnouncementItem[]> => {
  const res = await apiClient.get<AnnouncementItem[] | { items?: AnnouncementItem[] }>(
    '/member/announcements',
  );
  return unwrapArray(res);
};

export const getMemberSurveys = async (): Promise<SurveyItem[]> => {
  const res = await apiClient.get<SurveyItem[] | { items?: SurveyItem[] }>('/member/surveys');
  return unwrapArray(res);
};

export const submitSurveyResponse = (id: number, payload: SubmitSurveyPayload) =>
  apiClient.post(`/member/surveys/${id}/responses`, payload);

// ---- Sprint 21: Lịch trả góp hóa đơn --------------------------------------------

export const getMemberInstallments = async (): Promise<InstallmentPlanItem[]> => {
  const res = await apiClient.get<InstallmentPlanItem[] | { items?: InstallmentPlanItem[] }>(
    '/member/installments',
  );
  return unwrapArray(res);
};

// ---- Sprint 22: Điểm thưởng, Đổi quà, Huy hiệu & Thử thách ---------------------

export const getMemberLoyalty = () => apiClient.get<MemberLoyaltyRes>('/member/loyalty');

export const redeemLoyaltyReward = (rewardId: number) =>
  apiClient.post<RedeemRewardRes>('/member/loyalty/redeem', { reward_id: rewardId });

export const getMemberBadges = async (): Promise<BadgeItem[]> => {
  const res = await apiClient.get<BadgeItem[] | { items?: BadgeItem[] }>('/member/badges');
  return unwrapArray(res);
};

export const getMemberChallenges = async (): Promise<ChallengeItem[]> => {
  const res = await apiClient.get<ChallengeItem[] | { items?: ChallengeItem[] }>(
    '/member/challenges',
  );
  return unwrapArray(res);
};

export const joinChallenge = (id: number) =>
  apiClient.post<ChallengeItem>(`/member/challenges/${id}/join`);

export function extractLeaderboardEntries(
  res: MemberLeaderboardRes | null | undefined,
): LeaderboardEntry[] {
  if (!res) return [];
  if (Array.isArray(res)) return res;
  if (Array.isArray(res.items) && res.items.length > 0) return res.items;
  if (Array.isArray(res.by_points) && res.by_points.length > 0) return res.by_points;
  if (Array.isArray(res.by_checkins) && res.by_checkins.length > 0) return res.by_checkins;
  return [];
}

export const getMemberLeaderboard = (limit = 10) =>
  apiClient.get<MemberLeaderboardRes>('/member/leaderboard', { query: { limit } });

// ---- Sprint 23: Dinh dưỡng & Thực đơn -------------------------------------------

export const DEFAULT_VIETNAMESE_FOODS: FoodItem[] = [
  { id: 1, name: 'Ức gà áp chảo (100g)', unit: '100g', calories: 165, protein: 31, carbs: 0, fat: 3.6 },
  { id: 2, name: 'Cơm trắng (1 bát vừa)', unit: '1 bát', calories: 200, protein: 4, carbs: 44, fat: 0.5 },
  { id: 3, name: 'Khoai lang luộc (100g)', unit: '100g', calories: 86, protein: 1.6, carbs: 20.1, fat: 0.1 },
  { id: 4, name: 'Trứng gà luộc (1 quả)', unit: '1 quả', calories: 78, protein: 6.3, carbs: 0.6, fat: 5.3 },
  { id: 5, name: 'Thịt bò thăn (100g)', unit: '100g', calories: 250, protein: 26, carbs: 0, fat: 15 },
  { id: 6, name: 'Cá hồi phi lê (100g)', unit: '100g', calories: 208, protein: 20.4, carbs: 0, fat: 13.4 },
  { id: 7, name: 'Phở bò tái (1 tô)', unit: '1 tô', calories: 450, protein: 24, carbs: 58, fat: 12 },
  { id: 8, name: 'Bún bò Huế (1 tô)', unit: '1 tô', calories: 520, protein: 26, carbs: 62, fat: 16 },
  { id: 9, name: 'Chuối chín (1 quả)', unit: '1 quả', calories: 105, protein: 1.3, carbs: 27, fat: 0.4 },
  { id: 10, name: 'Sữa chua Hy Lạp không đường', unit: '1 hộp', calories: 100, protein: 10, carbs: 6, fat: 0.7 },
  { id: 11, name: 'Whey Protein Isolate (1 muỗng)', unit: '1 muỗng', calories: 120, protein: 25, carbs: 2, fat: 1 },
  { id: 12, name: 'Yến mạch cán dẹt (50g)', unit: '50g', calories: 190, protein: 6.5, carbs: 33, fat: 3.5 },
  { id: 13, name: 'Bông cải xanh luộc (100g)', unit: '100g', calories: 35, protein: 2.4, carbs: 7.2, fat: 0.4 },
  { id: 14, name: 'Đậu phụ trắng (100g)', unit: '100g', calories: 76, protein: 8, carbs: 1.9, fat: 4.8 },
];

export const getMemberMealPlans = async (): Promise<MemberNutritionOverviewRes> => {
  const res = await apiClient.get<
    MemberNutritionOverviewRes | MealPlanItem[] | { items?: MealPlanItem[] }
  >('/member/meal-plans');
  if (Array.isArray(res)) {
    return { plans: res, foods: DEFAULT_VIETNAMESE_FOODS };
  }
  if (res && 'items' in res && Array.isArray(res.items)) {
    return { plans: res.items, foods: DEFAULT_VIETNAMESE_FOODS };
  }
  return (res as MemberNutritionOverviewRes) ?? { plans: [], foods: DEFAULT_VIETNAMESE_FOODS };
};

export const getMemberFoodLogs = (date?: string) =>
  apiClient.get<FoodLogsResponse | FoodLogItem[]>('/member/food-logs', {
    query: date ? { date } : undefined,
  });

export const createMemberFoodLog = (payload: CreateFoodLogPayload) =>
  apiClient.post<FoodLogItem>('/member/food-logs', payload);

export const updateMemberFoodLog = (id: number, payload: UpdateFoodLogPayload) =>
  apiClient.put<FoodLogItem>(`/member/food-logs/${id}`, payload);

export const deleteMemberFoodLog = (id: number) =>
  apiClient.del<null>(`/member/food-logs/${id}`);

// ---- Sprint 24: Hợp đồng điện tử & Vé khách mời ---------------------------------

export const getMemberContracts = async (): Promise<MemberContractItem[]> => {
  const res = await apiClient.get<
    MemberContractItem[] | { items?: MemberContractItem[]; contracts?: MemberContractItem[] }
  >('/member/contracts');
  if (!res) return [];
  if (Array.isArray(res)) return res;
  if (Array.isArray(res.items)) return res.items;
  if (Array.isArray(res.contracts)) return res.contracts;
  return [];
};

export const signMemberContract = (id: number, payload: SignContractPayload) =>
  apiClient.post<MemberContractItem>(`/member/contracts/${id}/sign`, payload);

export const getMemberGuestPasses = async (): Promise<GuestPassItem[]> => {
  const res = await apiClient.get<
    GuestPassItem[] | { items?: GuestPassItem[]; passes?: GuestPassItem[] }
  >('/member/guest-passes');
  if (!res) return [];
  if (Array.isArray(res)) return res;
  if (Array.isArray(res.items)) return res.items;
  if (Array.isArray(res.passes)) return res.passes;
  return [];
};

export const createMemberGuestPass = (payload: CreateGuestPassPayload) =>
  apiClient.post<GuestPassItem>('/member/guest-passes', payload);

// ---- Sprint 27: Bảo mật 2FA, Phiên đăng nhập & Quyền riêng tư -------------------

export const setupMemberMFA = () => apiClient.post<MFASetupRes>('/member/security/mfa/setup');

export const enableMemberMFA = (payload: MFAEnablePayload) =>
  apiClient.post<MFAEnableRes>('/member/security/mfa/enable', payload);

export const disableMemberMFA = (code?: string) =>
  apiClient.post<{ disabled?: boolean; message?: string }>(
    '/member/security/mfa/disable',
    code ? { code, otp: code } : {},
  );

export const getMemberSessions = async (): Promise<SessionItem[]> => {
  const res = await apiClient.get<
    SessionItem[] | { items?: SessionItem[]; sessions?: SessionItem[] }
  >('/member/security/sessions');
  if (!res) return [];
  if (Array.isArray(res)) return res;
  if (Array.isArray(res.items)) return res.items;
  if (Array.isArray(res.sessions)) return res.sessions;
  return [];
};

/**
 * Đăng xuất mọi thiết bị khác, GIỮ phiên hiện tại. API xác định phiên hiện tại qua claim `sid`
 * của access token; gửi kèm `current_session_id` (nếu biết từ danh sách) cho token cũ không có sid.
 */
export const revokeOtherMemberSessions = (currentSessionId?: number) =>
  apiClient.post<{ revoked_count?: number }>(
    '/member/security/sessions/revoke-others',
    currentSessionId ? { current_session_id: currentSessionId } : {},
  );

/** Thu hồi một phiên đăng nhập cụ thể (không áp dụng cho phiên hiện tại). */
export const revokeMemberSession = (id: number) =>
  apiClient.del<null>(`/member/security/sessions/${id}`);

/** Tải bản trích xuất dữ liệu cá nhân (JSON) của yêu cầu EXPORT đã hoàn tất. */
export const downloadMemberDataExport = (id: number) =>
  apiClient.download(`/member/data-requests/${id}/download`);

export const getMemberConsents = async (): Promise<ConsentItem[]> => {
  const res = await apiClient.get<
    ConsentItem[] | { items?: ConsentItem[]; consents?: ConsentItem[] }
  >('/member/consents');
  if (!res) return [];
  if (Array.isArray(res)) return res;
  if (Array.isArray(res.items)) return res.items;
  if (Array.isArray(res.consents)) return res.consents;
  return [];
};

export const updateMemberConsents = (payload: UpdateConsentPayload) =>
  apiClient.put<ConsentItem[] | { consents?: ConsentItem[] }>('/member/consents', payload);

export const getMemberDataRequests = async (): Promise<DataRequestItem[]> => {
  const res = await apiClient.get<
    DataRequestItem[] | { items?: DataRequestItem[]; requests?: DataRequestItem[] }
  >('/member/data-requests');
  if (!res) return [];
  if (Array.isArray(res)) return res;
  if (Array.isArray(res.items)) return res.items;
  if (Array.isArray(res.requests)) return res.requests;
  return [];
};

export const createMemberDataRequest = (payload: CreateDataRequestPayload) =>
  apiClient.post<DataRequestItem>('/member/data-requests', payload);

// ---- Sprint 29: Nhắn tin HLV ↔ Hội viên, Tùy chọn thông báo, Lịch .ics -----------

export const getMemberConversations = async (): Promise<ConversationItem[]> => {
  const res = await apiClient.get<ConversationItem[] | { items?: ConversationItem[] }>(
    '/member/conversations',
  );
  return unwrapArray(res);
};

export const getMemberMessages = async (conversationId: number): Promise<MessageItem[]> => {
  const res = await apiClient.get<MessageItem[] | { items?: MessageItem[] }>(
    `/member/conversations/${conversationId}/messages`,
  );
  return unwrapArray(res);
};

export const sendMemberMessage = (conversationId: number, content: string) =>
  apiClient.post<MessageItem>(`/member/conversations/${conversationId}/messages`, { content });

export const getMemberNotificationPreferences = async (): Promise<NotificationPrefItem[]> => {
  const res = await apiClient.get<NotificationPrefItem[] | { items?: NotificationPrefItem[] }>(
    '/member/notification-preferences',
  );
  return unwrapArray(res);
};

export const saveMemberNotificationPreferences = (items: NotificationPrefItem[]) =>
  apiClient.put<NotificationPrefItem[]>('/member/notification-preferences', { items });

export const getMemberCalendarToken = () =>
  apiClient.get<CalendarTokenRes>('/member/calendar-token');

export const rotateMemberCalendarToken = () =>
  apiClient.post<CalendarTokenRes>('/member/calendar-token/rotate');

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

  // Sprint 13: Thanh toán VNPay & Mua gói tập
  publicPlans: () => apiClient.get<PublicPlansRes>('/public/plans', { auth: false }),
  publicBranches: () => apiClient.get<PublicBranch[]>('/public/branches', { auth: false }),
  createInvoiceVNPay,
  checkPromotion,
  checkoutSubscription,
  verifyVNPayReturn,

  // Sprint 14: Giáo án & Nhật ký tập luyện
  getWorkoutPlans,
  getExercises,
  getWorkoutLogs,
  createWorkoutLog,

  // Sprint 15: Đánh giá HLV & Lớp học
  getMemberReviews,
  createMemberReview,

  // Sprint 18–27: Kiosk, Tủ đồ, Mật độ, Bảng tin, NPS, Trả góp, Loyalty, Dinh dưỡng, Hợp đồng, Bảo mật
  deviceCheckin,
  deviceHeartbeat,
  getMemberLocker,
  getBranchOccupancy,
  getPublicBranchOccupancy,
  getMemberAnnouncements,
  getMemberSurveys,
  submitSurveyResponse,
  getMemberInstallments,
  getMemberLoyalty,
  redeemLoyaltyReward,
  getMemberBadges,
  getMemberChallenges,
  joinChallenge,
  getMemberLeaderboard,
  getMemberMealPlans,
  getMemberFoodLogs,
  createMemberFoodLog,
  updateMemberFoodLog,
  deleteMemberFoodLog,
  getMemberContracts,
  signMemberContract,
  getMemberGuestPasses,
  createMemberGuestPass,
  setupMemberMFA,
  enableMemberMFA,
  disableMemberMFA,
  getMemberSessions,
  revokeOtherMemberSessions,
  revokeMemberSession,
  downloadMemberDataExport,
  getMemberConsents,
  updateMemberConsents,
  getMemberDataRequests,
  createMemberDataRequest,
  getMemberConversations,
  getMemberMessages,
  sendMemberMessage,
  getMemberNotificationPreferences,
  saveMemberNotificationPreferences,
  getMemberCalendarToken,
  rotateMemberCalendarToken,
};
