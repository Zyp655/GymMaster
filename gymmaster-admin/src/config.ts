/** Cấu hình ứng dụng (đọc từ biến môi trường Vite lúc build). */
export const appConfig = {
  apiUrl: (import.meta.env?.VITE_API_URL as string | undefined) || 'http://localhost:8000/v1',
  appName: 'GymMaster',
  /** Số bản ghi mặc định mỗi trang của bảng danh sách. */
  defaultPageSize: 20,
  pageSizeOptions: [10, 20, 50, 100],
} as const;
