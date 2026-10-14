/** Dữ liệu công khai cho landing page – khớp `GET /v1/public/plans` và `/v1/public/branches`. */
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

/** Kết quả tải dữ liệu phía server: null = lỗi (hiển thị thông báo thay vì làm hỏng cả trang). */
export type LoadResult<T> = { ok: true; data: T } | { ok: false };
