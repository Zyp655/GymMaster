import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

function normalizePage<T>(res: Page<T> | T[] | null | undefined): Page<T> {
  if (!res) {
    return { items: [], total: 0, total_page: 1, current_page: 1, limit: 20 };
  }
  if (Array.isArray(res)) {
    return {
      items: res,
      total: res.length,
      total_page: 1,
      current_page: 1,
      limit: res.length || 20,
    };
  }
  return {
    items: res.items ?? [],
    total: res.total ?? (res.items?.length || 0),
    total_page: res.total_page ?? 1,
    current_page: res.current_page ?? 1,
    limit: res.limit ?? 20,
  };
}

/* ============================================================================
 * SPRINT 22 – Điểm thưởng, Huy hiệu & Thử thách
 * ============================================================================ */

export type RewardType = 'VOUCHER' | 'PRODUCT';
export type ChallengeMetric = 'CHECKINS' | 'CLASSES' | 'PT_SESSIONS';
export type ChallengeStatus = 'ACTIVE' | 'ENDED' | 'DRAFT';
export type LoyaltyReason =
  | 'CHECKIN'
  | 'RENEW'
  | 'REFERRAL'
  | 'REVIEW'
  | 'REDEEM'
  | 'CHALLENGE'
  | 'ADJUST';

export const REWARD_TYPE_LABELS: Record<RewardType, string> = {
  VOUCHER: 'Mã giảm giá (Voucher)',
  PRODUCT: 'Quà tặng hiện vật (Tại quầy)',
};

export const REWARD_TYPE_OPTIONS = (Object.keys(REWARD_TYPE_LABELS) as RewardType[]).map(
  (value) => ({ value, label: REWARD_TYPE_LABELS[value] }),
);

export const CHALLENGE_METRIC_LABELS: Record<ChallengeMetric, string> = {
  CHECKINS: 'Số lượt Check-in phòng tập',
  CLASSES: 'Số buổi tham gia lớp nhóm',
  PT_SESSIONS: 'Số buổi hoàn thành PT 1-1',
};

export const CHALLENGE_METRIC_OPTIONS = (
  Object.keys(CHALLENGE_METRIC_LABELS) as ChallengeMetric[]
).map((value) => ({ value, label: CHALLENGE_METRIC_LABELS[value] }));

export const LOYALTY_REASON_LABELS: Record<LoyaltyReason, string> = {
  CHECKIN: 'Điểm danh Check-in',
  RENEW: 'Gia hạn gói tập',
  REFERRAL: 'Giới thiệu bạn bè',
  REVIEW: 'Đánh giá buổi tập',
  REDEEM: 'Đổi quà thưởng',
  CHALLENGE: 'Hoàn thành thử thách',
  ADJUST: 'Điều chỉnh thủ công',
};

export interface LoyaltyReward {
  id: number;
  name: string;
  description: string;
  cost_points: number;
  reward_type: RewardType;
  ref_id?: number | null;
  discount_pct: number;
  stock: number;
  status: 'ACTIVE' | 'INACTIVE';
  created_at?: string;
}

export interface LoyaltyRewardInput {
  name: string;
  description?: string;
  cost_points: number;
  reward_type: RewardType;
  ref_id?: number | null;
  discount_pct?: number;
  stock: number;
  status?: 'ACTIVE' | 'INACTIVE';
}

export interface Challenge {
  id: number;
  branch_id?: number | null;
  branch_name?: string;
  name: string;
  description: string;
  metric: ChallengeMetric;
  target: number;
  start_date: string;
  end_date: string;
  reward_points: number;
  status: ChallengeStatus;
  participants_count?: number;
  completed_count?: number;
  created_at?: string;
}

export interface ChallengeInput {
  branch_id?: number | null;
  name: string;
  description?: string;
  metric: ChallengeMetric;
  target: number;
  start_date: string;
  end_date: string;
  reward_points: number;
  status?: ChallengeStatus;
}

export interface BadgeItem {
  id: number;
  code: string;
  name: string;
  description: string;
  icon: string;
  rule: {
    metric?: ChallengeMetric | string;
    target?: number;
    [key: string]: unknown;
  };
  earned_count?: number;
  created_at?: string;
}

export interface BadgeInput {
  code: string;
  name: string;
  description?: string;
  icon?: string;
  rule: {
    metric: ChallengeMetric | string;
    target: number;
  };
}

/** Dòng lịch sử điểm – GET /admin/loyalty-ledgers (lọc theo chi nhánh được phân quyền). */
export interface LoyaltyLedger {
  id: number;
  member_id: number;
  member_name: string;
  member_code: string;
  branch_id: number;
  branch_name: string;
  points: number;
  reason: LoyaltyReason;
  ref_type: string;
  ref_id: number | null;
  note: string;
  created_by: number | null;
  created_by_name: string;
  created_at: string;
}

export interface LoyaltyLedgerListParams extends ListParams {
  member_id?: number;
  branch_id?: number;
  reason?: LoyaltyReason;
}

class LoyaltyApi {
  readonly rewardKey = 'loyalty-rewards';
  readonly challengeKey = 'challenges';
  readonly badgeKey = 'badges';
  readonly ledgerKey = 'loyalty-ledgers';

  async listRewards(params: ListParams & { status?: string; reward_type?: RewardType } = {}): Promise<Page<LoyaltyReward>> {
    const res = await http.get<Page<LoyaltyReward> | LoyaltyReward[]>('/admin/loyalty-rewards', {
      params,
    });
    return normalizePage(res);
  }

  createReward(data: LoyaltyRewardInput) {
    return http.post<LoyaltyReward>('/admin/loyalty-rewards', data);
  }

  updateReward(id: number, data: Partial<LoyaltyRewardInput>) {
    return http.put<LoyaltyReward>(`/admin/loyalty-rewards/${id}`, data);
  }

  deleteReward(id: number) {
    return http.delete(`/admin/loyalty-rewards/${id}`);
  }

  async listChallenges(
    params: ListParams & { branch_id?: number; status?: ChallengeStatus; metric?: ChallengeMetric } = {},
  ): Promise<Page<Challenge>> {
    const res = await http.get<Page<Challenge> | Challenge[]>('/admin/challenges', { params });
    return normalizePage(res);
  }

  createChallenge(data: ChallengeInput) {
    return http.post<Challenge>('/admin/challenges', data);
  }

  updateChallenge(id: number, data: Partial<ChallengeInput>) {
    return http.put<Challenge>(`/admin/challenges/${id}`, data);
  }

  deleteChallenge(id: number) {
    return http.delete(`/admin/challenges/${id}`);
  }

  async listBadges(params: ListParams = {}): Promise<Page<BadgeItem>> {
    const res = await http.get<Page<BadgeItem> | BadgeItem[]>('/admin/badges', { params });
    return normalizePage(res);
  }

  createBadge(data: BadgeInput) {
    return http.post<BadgeItem>('/admin/badges', data);
  }

  updateBadge(id: number, data: Partial<BadgeInput>) {
    return http.put<BadgeItem>(`/admin/badges/${id}`, data);
  }

  deleteBadge(id: number) {
    return http.delete(`/admin/badges/${id}`);
  }

  adjustPoints(memberId: number, data: { points: number; note: string }) {
    return http.post<{ loyalty_points?: number; ledger?: LoyaltyLedger }>(
      `/admin/members/${memberId}/points/adjust`,
      data,
    );
  }

  /** Lỗi được ném lên để DataTable hiển thị trạng thái lỗi + "Thử lại" (không nuốt lỗi thành danh sách rỗng). */
  listLedgers(params: LoyaltyLedgerListParams = {}): Promise<Page<LoyaltyLedger>> {
    return http.get<Page<LoyaltyLedger>>('/admin/loyalty-ledgers', { params });
  }
}

export const loyaltyApi = new LoyaltyApi();

/* ============================================================================
 * SPRINT 23 – Dinh dưỡng, Bảng tin & Khảo sát NPS
 * ============================================================================ */

export type MealType = 'BREAKFAST' | 'LUNCH' | 'DINNER' | 'SNACK';
export type SurveyType = 'NPS' | 'CUSTOM';
export type SurveyStatus = 'ACTIVE' | 'CLOSED';

export const MEAL_TYPE_LABELS: Record<MealType, string> = {
  BREAKFAST: 'Bữa sáng',
  LUNCH: 'Bữa trưa',
  DINNER: 'Bữa tối',
  SNACK: 'Bữa phụ (Snack)',
};

export const MEAL_TYPE_OPTIONS = (Object.keys(MEAL_TYPE_LABELS) as MealType[]).map((value) => ({
  value,
  label: MEAL_TYPE_LABELS[value],
}));

export interface Food {
  id: number;
  name: string;
  unit: string;
  calories: number;
  protein: number;
  carbs: number;
  fat: number;
  created_at?: string;
}

export interface FoodInput {
  name: string;
  unit: string;
  calories: number;
  protein: number;
  carbs: number;
  fat: number;
}

export interface MealPlanItem {
  id?: number;
  plan_id?: number;
  day_index: number;
  meal: MealType;
  food_id: number;
  food_name?: string;
  unit?: string;
  calories?: number;
  qty: number;
  note?: string;
}

export interface MealPlan {
  id: number;
  member_id: number;
  member_name?: string;
  member_code?: string;
  trainer_id?: number | null;
  trainer_name?: string;
  name: string;
  goal: string;
  target_calories: number;
  start_date?: string | null;
  end_date?: string | null;
  status: 'ACTIVE' | 'ARCHIVED';
  items?: MealPlanItem[];
  created_at: string;
}

export interface MealPlanInput {
  member_id: number;
  trainer_id?: number | null;
  name: string;
  goal?: string;
  target_calories: number;
  start_date?: string | null;
  end_date?: string | null;
  status?: 'ACTIVE' | 'ARCHIVED';
  items?: Omit<MealPlanItem, 'id' | 'plan_id'>[];
}

export interface Announcement {
  id: number;
  branch_id?: number | null;
  branch_name?: string;
  title: string;
  body: string;
  image_url?: string;
  publish_at: string;
  expire_at?: string | null;
  pinned: boolean;
  created_by?: number | null;
  created_by_name?: string;
  created_at: string;
}

export interface AnnouncementInput {
  branch_id?: number | null;
  title: string;
  body: string;
  image_url?: string;
  publish_at?: string;
  expire_at?: string | null;
  pinned?: boolean;
}

export interface Survey {
  id: number;
  title: string;
  description: string;
  type: SurveyType;
  start_at: string;
  end_at?: string | null;
  status: SurveyStatus;
  responses_count?: number;
  avg_score?: number;
  created_at: string;
}

export interface SurveyInput {
  title: string;
  description?: string;
  type?: SurveyType;
  start_at?: string;
  end_at?: string | null;
  status?: SurveyStatus;
}

export interface SurveyResponseItem {
  id: number;
  survey_id: number;
  member_id: number;
  member_name?: string;
  member_code?: string;
  member_phone?: string;
  score: number;
  comment: string;
  created_at: string;
}

export interface NPSReport {
  nps_score: number;
  total_responses: number;
  promoters_count: number;
  passives_count: number;
  detractors_count: number;
  promoters_pct: number;
  passives_pct: number;
  detractors_pct: number;
  avg_score: number;
  recent_detractors?: SurveyResponseItem[];
  responses?: SurveyResponseItem[];
}

class EngagementApi {
  readonly foodKey = 'foods';
  readonly mealPlanKey = 'meal-plans';
  readonly announcementKey = 'announcements';
  readonly surveyKey = 'surveys';
  readonly npsKey = 'nps-report';

  async listFoods(params: ListParams = {}): Promise<Page<Food>> {
    const res = await http.get<Page<Food> | Food[]>('/admin/foods', { params });
    return normalizePage(res);
  }

  createFood(data: FoodInput) {
    return http.post<Food>('/admin/foods', data);
  }

  updateFood(id: number, data: Partial<FoodInput>) {
    return http.put<Food>(`/admin/foods/${id}`, data);
  }

  deleteFood(id: number) {
    return http.delete(`/admin/foods/${id}`);
  }

  async listMealPlans(
    params: ListParams & { member_id?: number; trainer_id?: number; status?: string } = {},
  ): Promise<Page<MealPlan>> {
    const res = await http.get<Page<MealPlan> | MealPlan[]>('/admin/meal-plans', { params });
    return normalizePage(res);
  }

  getMealPlan(id: number) {
    return http.get<MealPlan>(`/admin/meal-plans/${id}`);
  }

  createMealPlan(data: MealPlanInput) {
    return http.post<MealPlan>('/admin/meal-plans', data);
  }

  updateMealPlan(id: number, data: Partial<MealPlanInput>) {
    return http.put<MealPlan>(`/admin/meal-plans/${id}`, data);
  }

  deleteMealPlan(id: number) {
    return http.delete(`/admin/meal-plans/${id}`);
  }

  async listAnnouncements(
    params: ListParams & { branch_id?: number; pinned?: boolean } = {},
  ): Promise<Page<Announcement>> {
    const res = await http.get<Page<Announcement> | Announcement[]>('/admin/announcements', {
      params,
    });
    return normalizePage(res);
  }

  createAnnouncement(data: AnnouncementInput) {
    return http.post<Announcement>('/admin/announcements', data);
  }

  updateAnnouncement(id: number, data: Partial<AnnouncementInput>) {
    return http.put<Announcement>(`/admin/announcements/${id}`, data);
  }

  deleteAnnouncement(id: number) {
    return http.delete(`/admin/announcements/${id}`);
  }

  async listSurveys(params: ListParams & { status?: SurveyStatus; type?: SurveyType } = {}): Promise<Page<Survey>> {
    const res = await http.get<Page<Survey> | Survey[]>('/admin/surveys', { params });
    return normalizePage(res);
  }

  createSurvey(data: SurveyInput) {
    return http.post<Survey>('/admin/surveys', data);
  }

  updateSurvey(id: number, data: Partial<SurveyInput>) {
    return http.put<Survey>(`/admin/surveys/${id}`, data);
  }

  deleteSurvey(id: number) {
    return http.delete(`/admin/surveys/${id}`);
  }

  async npsReport(params: { branch_id?: number; survey_id?: number; from?: string; to?: string } = {}): Promise<NPSReport> {
    const raw = await http.get<Record<string, unknown>>('/admin/reports/nps', { params });
    const total = Number(raw?.total_responses ?? raw?.total ?? 0);
    const promotersCount = Number(raw?.promoters_count ?? raw?.promoters ?? 0);
    const passivesCount = Number(raw?.passives_count ?? raw?.passives ?? 0);
    const detractorsCount = Number(raw?.detractors_count ?? raw?.detractors ?? 0);
    const promotersPct = Number(
      raw?.promoters_pct ?? (total > 0 ? Math.round((promotersCount / total) * 1000) / 10 : 0),
    );
    const passivesPct = Number(
      raw?.passives_pct ?? (total > 0 ? Math.round((passivesCount / total) * 1000) / 10 : 0),
    );
    const detractorsPct = Number(
      raw?.detractors_pct ?? (total > 0 ? Math.round((detractorsCount / total) * 1000) / 10 : 0),
    );
    const npsScore = Number(
      raw?.nps_score ?? raw?.nps ?? Math.round(promotersPct - detractorsPct),
    );

    return {
      nps_score: npsScore,
      total_responses: total,
      promoters_count: promotersCount,
      passives_count: passivesCount,
      detractors_count: detractorsCount,
      promoters_pct: promotersPct,
      passives_pct: passivesPct,
      detractors_pct: detractorsPct,
      avg_score: Number(raw?.avg_score ?? 0),
      recent_detractors: (raw?.recent_detractors ?? raw?.detractor_responses ?? []) as SurveyResponseItem[],
      responses: (raw?.responses ?? []) as SurveyResponseItem[],
    };
  }
}

export const engagementApi = new EngagementApi();
export default loyaltyApi;
