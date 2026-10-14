import { useCallback } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import notificationApi from '@/api/notification';
import type { Notification } from '@/api/notification';
import { useAppSelector } from '@/store';

/**
 * Thông báo in-app cho chuông ở header (`/admin/notifications`).
 * Badge: `unread-count` poll 60 giây + tải lại khi cửa sổ được focus.
 * Danh sách 10 thông báo mới nhất chỉ tải khi mở popover (`enabled`).
 */
export interface AppNotification {
  id: number;
  title: string;
  body?: string;
  link?: string;
  read: boolean;
  created_at: string;
  /** Loại thông báo (`SUBSCRIPTION_EXPIRING`, `LOW_STOCK`...). */
  type?: string;
}

export interface UseNotificationsResult {
  items: AppNotification[];
  unread: number;
  isLoading: boolean;
  markRead: (id: number) => void;
  markAllRead: () => void;
  /** Có lỗi khi tải danh sách. */
  isError?: boolean;
  refetch?: () => void;
}

const POLL_MS = 60_000;
const LATEST_LIMIT = 10;

export const toAppNotification = (n: Notification): AppNotification => ({
  id: n.id,
  title: n.title,
  body: n.body,
  link: n.link,
  read: n.is_read,
  created_at: n.created_at,
  type: n.type,
});

export function useNotifications(options: { listEnabled?: boolean } = {}): UseNotificationsResult {
  const { listEnabled = true } = options;
  const loggedIn = useAppSelector((s) => !!s.auth.user);
  const qc = useQueryClient();

  const count = useQuery({
    queryKey: [notificationApi.key, 'unread-count'],
    queryFn: () => notificationApi.unreadCount(),
    enabled: loggedIn,
    refetchInterval: POLL_MS,
    refetchOnWindowFocus: true,
    retry: false,
  });

  const latest = useQuery({
    queryKey: [notificationApi.key, 'latest'],
    queryFn: () => notificationApi.list({ limit: LATEST_LIMIT, page: 1 }),
    enabled: loggedIn && listEnabled,
    refetchOnWindowFocus: true,
    staleTime: 15_000,
  });

  const invalidate = useCallback(
    () => qc.invalidateQueries({ queryKey: [notificationApi.key] }),
    [qc],
  );

  const readOne = useMutation({
    mutationFn: (id: number) => notificationApi.markRead(id),
    onSettled: invalidate,
  });
  const readAll = useMutation({
    mutationFn: () => notificationApi.markAllRead(),
    onSettled: invalidate,
  });

  const items = (latest.data?.items ?? []).map(toAppNotification);
  const { mutate: markOne } = readOne;
  const { mutate: markEvery } = readAll;

  return {
    items,
    unread: count.data?.count ?? 0,
    isLoading: latest.isLoading,
    isError: latest.isError,
    refetch: () => void latest.refetch(),
    markRead: useCallback((id: number) => markOne(id), [markOne]),
    markAllRead: useCallback(() => markEvery(), [markEvery]),
  };
}

export default useNotifications;
