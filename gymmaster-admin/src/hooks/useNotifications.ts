/**
 * Thông báo in-app cho chuông ở header.
 *
 * TẠM THỜI rỗng – module thông báo sẽ thay phần thân hook này bằng TanStack Query
 * (ví dụ `notificationApi.list()` + `refetchInterval`) mà KHÔNG cần sửa `NotificationBell`.
 */
export interface AppNotification {
  id: number;
  title: string;
  body?: string;
  link?: string;
  read: boolean;
  created_at: string;
}

export interface UseNotificationsResult {
  items: AppNotification[];
  unread: number;
  isLoading: boolean;
  markRead: (id: number) => void;
  markAllRead: () => void;
}

export function useNotifications(): UseNotificationsResult {
  return {
    items: [],
    unread: 0,
    isLoading: false,
    markRead: () => undefined,
    markAllRead: () => undefined,
  };
}

export default useNotifications;
