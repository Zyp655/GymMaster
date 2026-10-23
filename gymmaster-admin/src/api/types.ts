/** Kiểu dữ liệu dùng chung giữa các API module. */

export type SortDir = 'ASC' | 'DESC';

/** Tham số danh sách có phân trang (khớp `dto.PageReq` backend). */
export interface ListParams {
  page?: number;
  limit?: number;
  keyword?: string;
  order_by?: string;
  order_dir?: SortDir;
}

/** `data` của API danh sách có phân trang. */
export interface Page<T> {
  items: T[];
  total: number;
  total_page: number;
  current_page: number;
  limit: number;
}

/** Người dùng đăng nhập (`/auth/me`, `user` trong login/refresh). */
export interface AuthUser {
  id: number;
  email: string;
  full_name: string;
  phone: string;
  avatar_url: string;
  roles: string[];
  permissions: string[];
  /** Rỗng = làm việc trên toàn hệ thống. */
  branch_ids: number[];
}

export interface LoginResult {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: AuthUser;
}

export type ActiveStatus = 'ACTIVE' | 'INACTIVE';

export interface Branch {
  id: number;
  code: string;
  name: string;
  address: string;
  phone: string;
  email: string;
  open_time: string;
  close_time: string;
  status: ActiveStatus;
  created_at: string;
  updated_at: string;
}

export type BranchInput = Omit<Branch, 'id' | 'created_at' | 'updated_at'> & {
  /** Vẫn lưu giờ mở cửa mới dù còn lớp/PT tương lai ngoài khung giờ (sau khi người dùng xác nhận 409). */
  force?: boolean;
};

export interface BranchListParams extends ListParams {
  status?: ActiveStatus;
}

/** Chi nhánh rút gọn (dropdown, `/public/branches`). */
export interface BranchOption {
  id: number;
  name: string;
  code?: string;
  status?: ActiveStatus;
}

export interface Role {
  id: number;
  code: string;
  name: string;
  description: string;
  is_system: boolean;
  permissions: string[];
}

export interface RoleCreateInput {
  code: string;
  name: string;
  description?: string;
  permissions: string[];
}

export type RoleUpdateInput = Omit<RoleCreateInput, 'code'>;

export interface Permission {
  id: number;
  code: string;
  module: string;
  name: string;
}
