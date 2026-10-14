import 'axios';

declare module 'axios' {
  interface AxiosRequestConfig {
    /** Không tự refresh/đăng xuất khi gặp 401 (dùng cho /auth/login, /auth/refresh). */
    skipAuthRefresh?: boolean;
    /** Không hiện thông báo lỗi chung (5xx / mất kết nối). */
    silent?: boolean;
    /** Đánh dấu nội bộ: request đã được thử lại sau khi refresh token. */
    _retry?: boolean;
  }
}
