/** Envelope chuẩn của gymmaster-api: mọi response đều có dạng này. */
export interface ApiEnvelope<T> {
  code: number;
  message: string;
  data: T;
}

/** Dữ liệu danh sách có phân trang (`data` của các API list). */
export interface Paginated<T> {
  items: T[];
  total: number;
  total_page: number;
  current_page: number;
  limit: number;
}

export interface ListParams {
  page?: number;
  limit?: number;
  keyword?: string;
  order_by?: string;
  order_dir?: 'ASC' | 'DESC';
}

/** Mã lỗi nghiệp vụ – khớp docs/api/sprint-01-auth-rbac-branch.md */
export const ApiCode = {
  OK: 0,
  VALIDATION: 51,
  BAD_REQUEST: 40000,
  UNAUTHORIZED: 40100,
  TOKEN_EXPIRED: 40101,
  INVALID_CREDENTIALS: 40102,
  REFRESH_TOKEN_INVALID: 40103,
  RESET_TOKEN_INVALID: 40104,
  TOO_MANY_ATTEMPTS: 40105,
  /** Tài khoản bật 2FA – cần gửi kèm mfa_code. */
  MFA_REQUIRED: 40107,
  /** Mã 2FA (TOTP/mã khôi phục) không đúng. */
  MFA_INVALID_CODE: 40108,
  PERMISSION_DENIED: 40300,
  ACCOUNT_LOCKED: 40301,
  NOT_FOUND: 40400,
  CONFLICT: 40900,
  TOO_MANY_REQUESTS: 42900,
  PAYMENT_INVALID_SIGNATURE: 44001,
  PAYMENT_AMOUNT_MISMATCH: 44002,
  PAYMENT_TXN_EXPIRED: 44003,
  INTERNAL: 50000,
  /** Không gọi được backend (mất mạng, server tắt). */
  NETWORK: -1,
} as const;

export class ApiError extends Error {
  readonly code: number;
  readonly status: number;

  constructor(code: number, status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}
