import { isSafeRelativePath } from '@/lib/safe-return-url';

/** Trang mặc định khi link thông báo không trỏ tới trang nào tồn tại trong cổng hội viên. */
export const NOTIFICATION_FALLBACK_ROUTE = '/me/notifications';

/** Các trang có thật trong cổng hội viên (app/(member)/me/**). */
const PORTAL_ROUTES = new Set([
  '/me',
  '/me/account',
  '/me/bookings',
  '/me/challenges',
  '/me/classes',
  '/me/contracts',
  '/me/history',
  '/me/invoices',
  '/me/messages',
  '/me/notifications',
  '/me/nutrition',
  '/me/progress',
  '/me/pt',
  '/me/qr',
  '/me/rewards',
  '/me/subscriptions',
  '/me/subscriptions/buy',
  '/me/workout',
]);

/** Link backend kiểu /member/<x> → trang cổng hội viên tương ứng. */
const MEMBER_ALIASES: [prefix: string, route: string][] = [
  ['/member/bookings', '/me/bookings'],
  ['/member/classes', '/me/classes'],
  ['/member/class-schedules', '/me/classes'],
  ['/member/pt-sessions', '/me/pt'],
  ['/member/pt-contracts', '/me/pt'],
  ['/member/subscriptions', '/me/subscriptions'],
  ['/member/freezes', '/me/subscriptions'],
  ['/member/qr', '/me/qr'],
  ['/member/checkins', '/me/history'],
  ['/member/body-metrics', '/me/progress'],
  ['/member/workout', '/me/workout'],
  ['/member/profile', '/me/account'],
  ['/member/loyalty', '/me/rewards'],
  ['/member/challenges', '/me/challenges'],
  ['/member/contracts', '/me/contracts'],
  ['/member/conversations', '/me/messages'],
  ['/member/messages', '/me/messages'],
];

function matchesPrefix(path: string, prefix: string) {
  return path === prefix || path.startsWith(`${prefix}/`);
}

/**
 * Chuyển link trong thông báo backend sang đường dẫn hợp lệ của cổng hội viên.
 * - Chỉ nhận đường dẫn nội bộ (chặn "//host", "/\host", URL tuyệt đối).
 * - `/me|/member/invoices/{id}` → `/me/invoices?invoice={id}` (mở dialog chi tiết).
 * - Trang không tồn tại → `/me/notifications`. Link rỗng → `null` (không điều hướng).
 */
export function mapPortalNotificationLink(rawLink: string | null | undefined): string | null {
  const link = (rawLink ?? '').trim();
  if (!link) return null;
  if (!isSafeRelativePath(link)) return NOTIFICATION_FALLBACK_ROUTE;

  const url = new URL(link, 'http://portal.invalid');
  const path = url.pathname.replace(/\/+$/, '') || '/';

  const invoiceMatch = /^\/(?:me|member)\/invoices\/(\d+)$/.exec(path);
  if (invoiceMatch) return `/me/invoices?invoice=${invoiceMatch[1]}`;

  if (path.startsWith('/member/') || path === '/member') {
    const alias = MEMBER_ALIASES.find(([prefix]) => matchesPrefix(path, prefix));
    if (alias) return alias[1];
    const candidate = path.replace(/^\/member/, '/me');
    return PORTAL_ROUTES.has(candidate) ? candidate : NOTIFICATION_FALLBACK_ROUTE;
  }

  if (PORTAL_ROUTES.has(path)) return `${path}${url.search}`;
  return NOTIFICATION_FALLBACK_ROUTE;
}
