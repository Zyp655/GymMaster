import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

/** Thông báo in-app của nhân viên – `docs/api/reports-system.md` mục 2. */

export type NotificationType =
  | 'SUBSCRIPTION_EXPIRING'
  | 'SUBSCRIPTION_EXPIRED'
  | 'FREEZE_REQUEST'
  | 'FREEZE_RESULT'
  | 'BOOKING_PROMOTED'
  | 'CLASS_CANCELLED'
  | 'PT_SCHEDULED'
  | 'PT_CANCELLED'
  | 'PT_NO_SHOW'
  | 'COMMISSION_PAID'
  | 'LOW_STOCK'
  | 'PAYMENT_SUCCESS'
  | 'REVIEW_PROMPT'
  | 'MAINTENANCE_DUE'
  | 'BIRTHDAY'
  | 'LEAD_NEW'
  | 'CHECKIN'
  | 'SYSTEM';

export interface Notification {
  id: number;
  type: NotificationType | string;
  title: string;
  /** Có thể nhiều dòng (`\n`). */
  body: string;
  /** Đường dẫn backend sinh (có thể dạng `/admin/...`); dùng `toAppLink` trước khi điều hướng. */
  link: string;
  is_read: boolean;
  read_at: string | null;
  created_at: string;
}

export interface NotificationListParams extends ListParams {
  unread?: boolean;
  type?: string;
}

class NotificationApi {
  readonly key = 'notifications';

  list(params: NotificationListParams = {}) {
    return http.get<Page<Notification>>('/admin/notifications', { params });
  }

  unreadCount() {
    return http.get<{ count: number }>('/admin/notifications/unread-count', { silent: true });
  }

  markRead(id: number) {
    return http.post<Notification>(`/admin/notifications/${id}/read`);
  }

  markAllRead() {
    return http.post<{ updated: number }>('/admin/notifications/read-all');
  }
}

export const NOTIFICATION_TYPE_LABEL: Record<string, string> = {
  SUBSCRIPTION_EXPIRING: 'Gói sắp hết hạn',
  SUBSCRIPTION_EXPIRED: 'Gói đã hết hạn',
  FREEZE_REQUEST: 'Yêu cầu bảo lưu',
  FREEZE_RESULT: 'Kết quả bảo lưu',
  BOOKING_PROMOTED: 'Được lên lớp',
  CLASS_CANCELLED: 'Hủy lớp',
  PT_SCHEDULED: 'Lịch PT',
  PT_CANCELLED: 'Hủy buổi PT',
  PT_NO_SHOW: 'Vắng buổi PT',
  COMMISSION_PAID: 'Chi hoa hồng',
  LOW_STOCK: 'Tồn kho thấp',
  PAYMENT_SUCCESS: 'Thanh toán thành công',
  REVIEW_PROMPT: 'Đánh giá buổi tập',
  MAINTENANCE_DUE: 'Đến hạn bảo trì',
  BIRTHDAY: 'Sinh nhật hội viên',
  LEAD_NEW: 'Khách tiềm năng mới',
  CHECKIN: 'Check-in mới',
  SYSTEM: 'Hệ thống',
  DEVICE_OFFLINE: 'Thiết bị mất kết nối',
  PT_MESSAGE: 'Tin nhắn PT',
  CLASS_REMINDER: 'Nhắc lịch lớp',
  PT_REMINDER: 'Nhắc buổi PT',
};

/** Nhãn loại thông báo – loại chưa biết không hiện mã thô. */
export function notificationTypeLabel(type: string | null | undefined): string {
  return (type && NOTIFICATION_TYPE_LABEL[type]) || 'Thông báo';
}

/**
 * Backend sinh link dạng `/admin/...` (hoặc `/inventory/...`) → đổi sang route của SPA `/app/...`.
 * Link của cổng hội viên (`/member/...`) không mở được ở đây → `null`.
 */
export function toAppLink(link: string | null | undefined): string | null {
  if (!link) return null;
  const raw = link.trim();
  if (/^https?:\/\//i.test(raw)) return null;
  const [pathPart, query = ''] = raw.split('?');
  const qs = new URLSearchParams(query);
  let path = pathPart.replace(/\/+$/, '');

  if (path.startsWith('/member')) return null;
  if (path.startsWith('/app/')) return raw;
  if (path.startsWith('/admin/')) path = path.slice('/admin'.length);
  // Liên kết tồn kho kiểu cũ `/inventory/stock`.
  if (path.startsWith('/inventory')) path = '/inventory';

  // Gói sắp hết hạn: `status=expiring` → bộ lọc `expiring_in_days=7` của trang Đăng ký.
  if (path === '/subscriptions' && qs.get('status') === 'expiring') {
    qs.delete('status');
    qs.set('expiring_in_days', '7');
  }
  // Chi tiết hợp đồng PT: `/pt-contracts/{id}` → trang danh sách mở drawer `cid`.
  const contract = /^\/pt-contracts\/(\d+)$/.exec(path);
  if (contract) {
    path = '/pt-contracts';
    qs.set('cid', contract[1]);
  }
  // Buổi PT: trang lịch chưa hỗ trợ mở theo id → về lịch PT.
  if (/^\/pt-sessions\/\d+$/.test(path)) path = '/pt-sessions';

  const search = qs.toString();
  return `/app${path}${search ? `?${search}` : ''}`;
}

export const notificationApi = new NotificationApi();
export default notificationApi;
