/** Kiểu dữ liệu cổng hội viên – khớp docs/api/member-portal.md. */

export type SubscriptionStatus =
  | 'PENDING'
  | 'ACTIVE'
  | 'FROZEN'
  | 'EXPIRED'
  | 'CANCELLED'
  | 'TRANSFERRED';
export type FreezeStatus = 'PENDING' | 'APPROVED' | 'REJECTED' | 'ENDED' | 'CANCELLED';
export type BookingStatus = 'BOOKED' | 'WAITLIST' | 'CANCELLED' | 'ATTENDED' | 'NO_SHOW';
export type PTSessionStatus = 'SCHEDULED' | 'COMPLETED' | 'CANCELLED' | 'NO_SHOW';
export type PTContractStatus = 'PENDING' | 'ACTIVE' | 'COMPLETED' | 'EXPIRED' | 'CANCELLED';
export type InvoiceStatus = 'UNPAID' | 'PARTIAL' | 'PAID' | 'VOID' | 'REFUNDED';
export type SlotReason = '' | 'PAST' | 'TRAINER_BUSY' | 'MEMBER_BUSY' | 'OUT_OF_CONTRACT';

export interface BodyMetric {
  id: number;
  member_id: number;
  measured_at: string;
  weight_kg: number;
  height_cm: number;
  body_fat_pct: number | null;
  muscle_mass_kg: number | null;
  waist_cm: number | null;
  chest_cm: number | null;
  hip_cm: number | null;
  bmi: number | null;
  note: string;
}

export interface Member {
  id: number;
  member_code: string;
  full_name: string;
  gender: 'MALE' | 'FEMALE' | 'OTHER';
  dob: string | null;
  phone: string;
  email: string;
  id_card: string;
  address: string;
  avatar_url: string;
  emergency_contact_name: string;
  emergency_contact_phone: string;
  home_branch_id: number;
  status: 'ACTIVE' | 'INACTIVE' | 'BANNED';
  booking_blocked_until: string | null;
  latest_metric: BodyMetric | null;
  stats: { total_checkins: number; last_checkin_at: string | null };
  loyalty_points?: number;
  referral_code?: string;
  mfa_enabled?: boolean;
}

export interface Branch {
  id: number;
  code: string;
  name: string;
  address: string;
  phone: string;
  open_time: string;
  close_time: string;
  is_home: boolean;
}

export interface Freeze {
  id: number;
  subscription_id: number;
  start_date: string;
  end_date: string;
  days: number;
  reason: string;
  status: FreezeStatus;
  review_note: string;
  reviewed_at: string | null;
  created_at: string;
}

export interface Subscription {
  id: number;
  plan_id: number;
  plan_code: string;
  plan_name: string;
  plan_type: 'TIME' | 'SESSION';
  access_scope: 'SINGLE_BRANCH' | 'ALL_BRANCHES';
  branch_id: number;
  branch_name: string;
  invoice_id: number | null;
  invoice_no: string;
  invoice_status: InvoiceStatus | '';
  start_date: string;
  end_date: string;
  total_sessions: number | null;
  remaining_sessions: number | null;
  price: number;
  status: SubscriptionStatus;
  freeze_days_used: number;
  freeze_times_used: number;
  days_left: number | null;
  includes_classes: boolean;
  max_freeze_days: number;
  max_freeze_times: number;
  freezes: Freeze[];
}

export interface Profile {
  member: Member;
  branch: Branch | null;
  current_subscription: Subscription | null;
  has_valid_subscription: boolean;
}

export interface ProfilePayload {
  email: string;
  address: string;
  emergency_contact_name: string;
  emergency_contact_phone: string;
  avatar_url: string;
}

export interface QRToken {
  token: string;
  expires_at: string;
  ttl: number;
  server_time: string;
  has_valid_subscription: boolean;
}

export interface CheckinItem {
  id: number;
  branch_id: number;
  branch_name: string;
  plan_name: string | null;
  checked_in_at: string;
  checked_out_at: string | null;
  method: 'QR' | 'MANUAL';
  result: 'ALLOWED' | 'DENIED';
  deny_reason: string;
  deny_message: string;
}

export interface ClassBooking {
  id: number;
  schedule_id: number;
  status: BookingStatus;
  waitlist_position: number | null;
  source: 'COUNTER' | 'PORTAL';
  booked_at: string;
  cancelled_at: string | null;
  attended_at: string | null;
  class_type_name: string;
  class_type_color: string;
  branch_id: number;
  branch_name: string;
  room_name: string;
  trainer_name: string;
  start_at: string;
  end_at: string;
  schedule_status: 'SCHEDULED' | 'CANCELLED' | 'COMPLETED';
}

export interface Booking extends ClassBooking {
  cancel_deadline: string;
  can_cancel: boolean;
}

export interface ScheduleItem {
  id: number;
  class_type_name: string;
  class_type_color: string;
  branch_id: number;
  branch_name: string;
  room_name: string;
  trainer_name: string;
  start_at: string;
  end_at: string;
  duration_min: number;
  capacity: number;
  booked_count: number;
  waitlist_count: number;
  available_slots: number;
  note: string;
  my_booking: ClassBooking | null;
  cancel_deadline: string;
  bookable: boolean;
}

export interface ClassesResponse {
  branch_id: number;
  from: string;
  to: string;
  advance_days: number;
  cancel_before_minutes: number;
  items: ScheduleItem[];
}

export interface PTContract {
  id: number;
  trainer_id: number;
  trainer_name: string;
  package_name: string;
  branch_id: number;
  branch_name: string;
  total_sessions: number;
  used_sessions: number;
  scheduled_sessions: number;
  remaining_sessions: number;
  bookable_sessions: number;
  session_minutes: number;
  start_date: string;
  end_date: string;
  status: PTContractStatus;
  days_left: number | null;
}

export interface PTSession {
  id: number;
  contract_id: number;
  trainer_name: string;
  branch_name: string;
  package_name: string;
  start_at: string;
  end_at: string;
  status: PTSessionStatus;
  counted: boolean;
  workout_note: string;
  trainer_note: string;
  cancel_reason: string;
  cancel_deadline: string;
  can_cancel: boolean;
}

export interface PTSessionsResponse {
  cancel_before_minutes: number;
  items: PTSession[];
}

export interface Slot {
  start_at: string;
  end_at: string;
  available: boolean;
  reason: SlotReason;
}

export interface Availability {
  date: string;
  contract_id: number;
  trainer_name: string;
  branch_name: string;
  open_time: string;
  close_time: string;
  session_minutes: number;
  slots: Slot[];
}

export interface Invoice {
  id: number;
  invoice_no: string;
  branch_name: string;
  subtotal: number;
  discount: number;
  total: number;
  paid_amount: number;
  refunded_amount: number;
  due: number;
  status: InvoiceStatus;
  issued_at: string;
  promotion_code: string;
}

export interface InvoiceDetail extends Invoice {
  items: {
    id: number;
    item_type: string;
    description: string;
    quantity: number;
    unit_price: number;
    discount: number;
    amount: number;
  }[];
  payments: { id: number; amount: number; method: string; paid_at: string }[];
  refunds: { id: number; amount: number; method: string; reason: string; created_at: string }[];
}

export interface Notification {
  id: number;
  type: string;
  title: string;
  body: string;
  link: string;
  is_read: boolean;
  created_at: string;
}

export interface FreezePayload {
  start_date: string;
  end_date: string;
  reason: string;
}

// ---- Sprint 13: Thanh toán VNPay Sandbox & Tự mua gói tập -----------------------

export interface PaymentURLRes {
  txn_ref: string;
  payment_url: string;
  amount: number;
  expires_at: string;
}

export interface CheckPromotionParams {
  code: string;
  plan_id?: number;
  amount?: number;
}

export interface CheckPromotionRes {
  code: string;
  name: string;
  discount: number;
  final_amount: number;
}

export interface CheckoutSubscriptionPayload {
  plan_id: number;
  branch_id?: number;
  promotion_code?: string;
}

export interface CheckoutSubscriptionRes {
  subscription_id: number;
  invoice_id: number;
  invoice_no: string;
  amount: number;
  txn_ref: string;
  payment_url: string;
}

export interface VNPayReturnRes {
  success: boolean;
  txn_ref: string;
  invoice_id: number;
  invoice_no: string;
  amount: number;
  response_code: string;
  message: string;
}

// ---- Sprint 14: Giáo án & Nhật ký tập luyện -------------------------------------

export type MuscleGroup =
  | 'CHEST'
  | 'BACK'
  | 'SHOULDERS'
  | 'LEGS'
  | 'ARMS'
  | 'CORE'
  | 'CARDIO'
  | 'FULL_BODY';

export interface Exercise {
  id: number;
  name: string;
  muscle_group: MuscleGroup | string;
  equipment: string;
  video_url: string;
  description: string;
}

export type ExerciseItem = Exercise;

export interface WorkoutPlanItem {
  id: number;
  day_id?: number;
  exercise_id: number;
  exercise_name?: string;
  muscle_group?: string;
  equipment?: string;
  video_url?: string;
  sets: number;
  reps: number | string;
  weight?: number;
  weight_kg?: number | null;
  rest_sec: number | null;
  note: string;
  sort?: number;
  sort_order?: number;
}

export interface WorkoutPlanDay {
  id: number;
  plan_id: number;
  day_index: number;
  title: string;
  items: WorkoutPlanItem[];
}

export interface WorkoutPlan {
  id: number;
  member_id: number | null;
  trainer_id: number;
  trainer_name?: string;
  pt_contract_id: number | null;
  name?: string;
  title?: string;
  description?: string;
  goal: string;
  start_date: string | null;
  end_date: string | null;
  status: 'DRAFT' | 'ACTIVE' | 'COMPLETED' | 'ARCHIVED';
  days: WorkoutPlanDay[];
  created_at?: string;
}

export interface WorkoutLog {
  id: number;
  member_id: number;
  plan_item_id: number | null;
  exercise_id: number;
  exercise_name?: string;
  muscle_group?: string;
  performed_at?: string;
  logged_at?: string;
  sets: number;
  reps: number;
  weight?: number;
  weight_kg?: number;
  volume?: number;
  note: string;
  created_at?: string;
}

export interface WorkoutProgressPoint {
  date: string;
  max_weight: number;
  total_volume: number;
  MaxWeight?: number;
  TotalVolume?: number;
}

export type WorkoutLogsResponse =
  | WorkoutLog[]
  | {
      items?: WorkoutLog[];
      logs?: WorkoutLog[];
      progress?: WorkoutProgressPoint[];
      series?: WorkoutProgressPoint[];
    };

export interface CreateWorkoutLogPayload {
  plan_item_id?: number | null;
  exercise_id: number;
  sets: number;
  reps: number;
  weight?: number;
  weight_kg?: number;
  note?: string;
  performed_at?: string;
}

// ---- Sprint 15: Đánh giá HLV & Lớp học ------------------------------------------

export type ReviewTargetType = 'PT_SESSION' | 'CLASS_SCHEDULE';

export interface MemberReview {
  id: number;
  member_id: number;
  target_type: ReviewTargetType;
  target_id: number;
  trainer_id: number;
  trainer_name?: string;
  branch_id: number | null;
  rating: number;
  comment: string;
  created_at: string;
}

export type MemberReviewItem = MemberReview;

export interface CreateMemberReviewPayload {
  target_type: ReviewTargetType;
  target_id: number;
  rating: number;
  comment?: string;
}

// ---- Sprint 18: Thiết bị kiểm soát ra vào & Kiosk --------------------------------

export interface DeviceCheckinPayload {
  qr_token: string;
  device_key?: string;
}

export interface DeviceCheckinRes {
  allow?: boolean;
  allowed?: boolean;
  result?: 'ALLOWED' | 'DENIED' | string;
  deny_reason?: string;
  message?: string;
  deny_message?: string;
  member_name?: string;
  full_name?: string;
  member_code?: string;
  avatar_url?: string;
  plan_name?: string;
  device_name?: string;
  branch_name?: string;
  locker_code?: string;
}

export interface DeviceHeartbeatRes {
  device?: {
    id: number;
    branch_id: number;
    branch_name?: string;
    code: string;
    name: string;
    type: string;
    status: string;
    last_seen_at?: string | null;
  } | null;
  server_time?: string;
}

// ---- Sprint 19 & 26: Tủ đồ & Mật độ phòng tập -----------------------------------

export type LockerRentalType = 'DAILY' | 'MONTHLY';

export interface MemberLocker {
  id?: number;
  rental_id?: number;
  locker_id?: number;
  code?: string;
  locker_code?: string;
  zone?: string;
  size?: string;
  type?: LockerRentalType | string;
  rental_type?: LockerRentalType | string;
  start_at?: string;
  end_at?: string | null;
  end_date?: string | null;
  assigned_until?: string | null;
  status?: string;
  branch_id?: number;
  branch_name?: string;
  note?: string;
}

export type OccupancyLevel = 'QUIET' | 'NORMAL' | 'BUSY' | 'FULL';
export type OccupancyStatus = OccupancyLevel;

export interface BranchOccupancy {
  branch_id: number;
  branch_name?: string;
  branch_code?: string;
  present?: number;
  current_count?: number;
  active_count?: number;
  capacity: number;
  occupancy_pct?: number;
  percentage?: number;
  percent?: number;
  level?: OccupancyLevel;
  status?: OccupancyLevel;
}

// ---- Sprint 21: Kế hoạch trả góp ------------------------------------------------

export type InstallmentStatus = 'PENDING' | 'PAID' | 'OVERDUE';
export type InstallmentScheduleStatus = InstallmentStatus;

export interface InstallmentScheduleItem {
  id: number;
  plan_id: number;
  seq: number;
  due_date: string;
  amount: number;
  paid_amount: number;
  status: InstallmentStatus;
  paid_at?: string | null;
}

export interface InstallmentPlanItem {
  id: number;
  invoice_id: number;
  invoice_no?: string;
  member_id?: number;
  total: number;
  total_amount?: number;
  installments: number;
  count?: number;
  schedules?: InstallmentScheduleItem[];
  items?: InstallmentScheduleItem[];
  created_at?: string;
}

// ---- Sprint 22: Điểm thưởng, Đổi quà, Huy hiệu, Giới thiệu & Thử thách ----------

export type LoyaltyReason =
  | 'CHECKIN'
  | 'RENEW'
  | 'REFERRAL'
  | 'REVIEW'
  | 'REDEEM'
  | 'CHALLENGE'
  | 'ADJUST';

export interface LoyaltyLedgerItem {
  id: number;
  member_id: number;
  points: number;
  reason: LoyaltyReason | string;
  ref_type?: string;
  ref_id?: number | null;
  note: string;
  created_at: string;
}

export interface LoyaltyRewardItem {
  id: number;
  name: string;
  description: string;
  cost_points: number;
  points_cost?: number;
  reward_type: 'VOUCHER' | 'PRODUCT' | string;
  ref_id?: number | null;
  discount_pct?: number;
  stock: number | null;
  status: string;
}

export interface MemberLoyaltyRes {
  balance?: number;
  points?: number;
  loyalty_points?: number;
  referral_code?: string;
  ledgers?: LoyaltyLedgerItem[];
  loyalty_ledgers?: LoyaltyLedgerItem[];
  rewards?: LoyaltyRewardItem[];
  loyalty_rewards?: LoyaltyRewardItem[];
}

export interface RedeemRewardRes {
  points?: number;
  loyalty_points?: number;
  voucher_code?: string;
  promotion_code?: string;
  code?: string;
  reward?: LoyaltyRewardItem;
  ledger?: LoyaltyLedgerItem;
  message?: string;
}

export interface BadgeItem {
  id: number;
  code: string;
  name: string;
  description: string;
  icon: string;
  rule?: Record<string, unknown>;
  earned: boolean;
  earned_at?: string | null;
  created_at?: string;
}

export type ChallengeMetric = 'CHECKINS' | 'CLASSES' | 'PT_SESSIONS';

export interface ChallengeItem {
  id: number;
  branch_id?: number | null;
  branch_name?: string;
  name: string;
  description: string;
  metric: ChallengeMetric | string;
  metric_type?: ChallengeMetric | string;
  target: number;
  target_value?: number;
  start_date: string;
  end_date: string;
  reward_points: number;
  status: string;
  participant_count?: number;
  joined: boolean;
  progress: number;
  completed?: boolean;
  completed_at?: string | null;
  joined_at?: string | null;
}

export interface LeaderboardEntry {
  rank: number;
  member_id: number;
  member_code: string;
  full_name: string;
  avatar_url?: string;
  loyalty_points: number;
  points?: number;
  checkin_count: number;
  checkins?: number;
}

export type MemberLeaderboardRes =
  | LeaderboardEntry[]
  | {
      by_points?: LeaderboardEntry[];
      by_checkins?: LeaderboardEntry[];
      items?: LeaderboardEntry[];
    };

// ---- Sprint 23: Dinh dưỡng, Bảng tin & Khảo sát NPS -----------------------------

export type MealType = 'BREAKFAST' | 'LUNCH' | 'DINNER' | 'SNACK';

export interface FoodItem {
  id: number;
  name: string;
  unit: string;
  calories: number;
  protein: number;
  protein_g?: number;
  carbs: number;
  carbs_g?: number;
  fat: number;
  fat_g?: number;
}

export interface MealPlanEntry {
  id: number;
  plan_id: number;
  day_index: number;
  meal: MealType | string;
  meal_type?: MealType | string;
  food_id: number;
  food_name?: string;
  unit?: string;
  calories?: number;
  protein?: number;
  carbs?: number;
  fat?: number;
  qty: number;
  note: string;
}

export interface MealPlanItem {
  id: number;
  member_id: number;
  trainer_id?: number | null;
  trainer_name?: string;
  name: string;
  goal: string;
  description?: string;
  target_calories: number;
  start_date?: string | null;
  end_date?: string | null;
  status: string;
  items?: MealPlanEntry[];
}

export interface FoodLogItem {
  id: number;
  member_id: number;
  food_id: number;
  food_name?: string;
  unit?: string;
  qty: number;
  meal: MealType | string;
  meal_type?: MealType | string;
  eaten_at: string;
  calories?: number;
  protein?: number;
  protein_g?: number;
  carbs?: number;
  carbs_g?: number;
  fat?: number;
  fat_g?: number;
  created_at?: string;
}

export interface MemberNutritionOverviewRes {
  recommended_calories?: number;
  plans?: MealPlanItem[];
  meal_plans?: MealPlanItem[];
  foods?: FoodItem[];
}

export interface FoodLogsResponse {
  date?: string;
  recommended_calories?: number;
  target_calories?: number;
  total_calories?: number;
  total_protein?: number;
  total_protein_g?: number;
  total_carbs?: number;
  total_carbs_g?: number;
  total_fat?: number;
  total_fat_g?: number;
  totals?: {
    calories?: number;
    protein?: number;
    carbs?: number;
    fat?: number;
  };
  items?: FoodLogItem[];
  logs?: FoodLogItem[];
  foods?: FoodItem[];
  meal_plans?: MealPlanItem[];
  plans?: MealPlanItem[];
}

export interface CreateFoodLogPayload {
  food_id: number;
  qty: number;
  meal?: MealType | string;
  meal_type?: MealType | string;
  date?: string;
  eaten_at?: string;
}

export interface AnnouncementItem {
  id: number;
  branch_id?: number | null;
  branch_name?: string;
  title: string;
  body: string;
  content?: string;
  image_url?: string;
  publish_at: string;
  published_at?: string;
  expire_at?: string | null;
  pinned: boolean;
  is_pinned?: boolean;
  created_at?: string;
}

export interface SurveyItem {
  id: number;
  title: string;
  description?: string;
  type: 'NPS' | 'CUSTOM' | string;
  start_at: string;
  end_at?: string | null;
  status: string;
  responded?: boolean;
  my_score?: number | null;
  my_comment?: string;
  my_response?: {
    id: number;
    score: number;
    comment: string;
    created_at: string;
  } | null;
}

export interface SubmitSurveyPayload {
  score: number;
  comment?: string;
  answers?: Record<string, unknown>;
}

// ---- Sprint 24: Hợp đồng điện tử & Vé khách mời ---------------------------------

export type ContractStatus = 'PENDING' | 'SIGNED';

export interface MemberContractItem {
  id: number;
  member_id: number;
  template_id: number;
  template_name?: string;
  name?: string;
  title?: string;
  template_version: number;
  version?: number;
  subscription_id?: number | null;
  plan_name?: string;
  rendered_html?: string;
  body_html?: string;
  content?: string;
  status: ContractStatus | string;
  signed_at?: string | null;
  signature_url?: string;
  signature_data?: string;
  pdf_url?: string;
  created_at?: string;
}

export interface SignContractPayload {
  signature_url?: string;
  signature_data?: string;
  signature?: string;
}

export interface GuestPassItem {
  id: number;
  branch_id: number;
  branch_name?: string;
  buyer_name: string;
  guest_name?: string;
  phone: string;
  guest_phone?: string;
  type: 'DAY' | 'GUEST' | string;
  valid_date: string;
  qr_token: string;
  token?: string;
  price?: number;
  used_at?: string | null;
  created_at?: string;
}

export interface CreateGuestPassPayload {
  buyer_name?: string;
  guest_name?: string;
  phone?: string;
  guest_phone?: string;
  branch_id?: number;
  valid_date?: string;
}

// ---- Sprint 27: Bảo mật 2FA, Phiên đăng nhập & Quyền riêng tư -------------------

export interface MFASetupRes {
  secret?: string;
  totp_secret?: string;
  otpauth_url?: string;
  qr_url?: string;
  enabled?: boolean;
  recovery_codes?: string[];
}

export interface MFAEnablePayload {
  code: string;
  otp?: string;
  secret?: string;
}

export interface MFAEnableRes {
  enabled?: boolean;
  recovery_codes?: string[];
  message?: string;
}

export interface SessionItem {
  id: number;
  device_name?: string;
  device?: string;
  user_agent?: string;
  ip?: string;
  ip_address?: string;
  last_used_at?: string | null;
  created_at?: string;
  expires_at?: string;
  is_current?: boolean;
  current?: boolean;
}

export type ConsentType = 'TERMS' | 'MARKETING' | 'PHOTO';

export interface ConsentItem {
  id?: number;
  type: ConsentType;
  consent_type?: ConsentType;
  granted: boolean;
  granted_at?: string;
}

export interface UpdateConsentPayload {
  type?: ConsentType;
  consent_type?: ConsentType;
  granted?: boolean;
  consents?: { type: ConsentType; granted: boolean }[];
}

export type DataRequestType = 'EXPORT' | 'DELETE';
export type DataRequestStatus = 'PENDING' | 'PROCESSING' | 'DONE' | 'REJECTED';

export interface DataRequestItem {
  id: number;
  member_id?: number;
  type: DataRequestType;
  status: DataRequestStatus;
  reason?: string;
  file_url?: string;
  export_url?: string;
  note?: string;
  created_at: string;
  done_at?: string | null;
}

export interface CreateDataRequestPayload {
  type: DataRequestType;
  reason?: string;
}

