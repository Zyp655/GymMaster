/** Dữ liệu công khai cho landing page – khớp `GET /v1/public/plans`, `/v1/public/branches`, `/v1/public/class-schedules`, `POST /v1/public/leads`. */
export type PlanType = 'TIME' | 'SESSION';
export type AccessScope = 'SINGLE_BRANCH' | 'ALL_BRANCHES';

export interface PublicPlan {
  id: number;
  code: string;
  name: string;
  description: string;
  type: PlanType;
  duration_days: number;
  session_count: number | null;
  price: number;
  access_scope: AccessScope;
  max_freeze_days: number;
  includes_classes: boolean;
}

export interface PublicPTPackage {
  id: number;
  code: string;
  name: string;
  description: string;
  session_count: number;
  session_minutes: number;
  validity_days: number;
  price: number;
}

export interface PublicPlansRes {
  membership_plans: PublicPlan[];
  pt_packages: PublicPTPackage[];
}

export interface PublicBranch {
  id: number;
  name: string;
  address: string;
  phone: string;
  open_time: string;
  close_time: string;
}

export interface PublicClassSchedule {
  id: number;
  branch_id: number;
  branch_name: string;
  class_type_id?: number;
  class_type_name: string;
  room_id?: number;
  room_name: string;
  trainer_id?: number;
  trainer_name: string;
  start_at: string;
  end_at: string;
  duration_min?: number;
  capacity: number;
  booked_count?: number;
  available_slots: number;
  note?: string;
}

export type PublicClassSchedulesRaw =
  | PublicClassSchedule[]
  | { items?: PublicClassSchedule[] };

export interface CreatePublicLeadPayload {
  full_name: string;
  phone: string;
  email?: string;
  branch_id?: number;
  interest_plan_id?: number;
  note?: string;
}

export interface CreatePublicLeadRes {
  id?: number;
  message?: string;
}

export interface PublicTrainer {
  id: number;
  user_id: number;
  full_name: string;
  slug: string;
  headline: string;
  bio: string;
  specialties: string;
  certifications: string;
  experience_years: number;
  avatar_url: string;
  featured: boolean;
  rating_avg: number;
  rating_count: number;
}

export interface PublicTestimonial {
  id: number;
  author_name: string;
  author_role: string;
  avatar_url?: string;
  rating: number;
  content: string;
  before_after_note?: string;
}

export interface PublicPost {
  id: number;
  slug: string;
  title: string;
  excerpt: string;
  cover_url?: string;
  category: string;
  content: string;
  published_at?: string;
  created_at: string;
}

/** Kết quả tải dữ liệu phía server: null = lỗi (hiển thị thông báo thay vì làm hỏng cả trang). */
export type LoadResult<T> = { ok: true; data: T } | { ok: false };
