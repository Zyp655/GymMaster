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
