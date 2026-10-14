import dayjs from 'dayjs';
import { invoicePrintUrl } from '@/api/invoice';
import type { Subscription, SubscriptionDetail } from '@/api/subscription';

/** Mở trang in hóa đơn ở tab mới (`?auto=1` → trang in tự mở hộp thoại in). */
export function openInvoicePrint(invoiceId: number) {
  window.open(`${invoicePrintUrl(invoiceId)}?auto=1`, '_blank', 'noopener');
}

/** "Còn lại" của gói: số ngày (và số buổi với gói theo buổi); gói đã kết thúc → "—". */
export function remainingText(
  s: Pick<
    Subscription,
    'status' | 'days_left' | 'remaining_sessions' | 'plan_type' | 'total_sessions'
  >,
): string {
  if (!isLiveSubscription(s)) return '—';
  const parts: string[] = [];
  if (s.days_left !== null && s.days_left !== undefined) parts.push(`${s.days_left} ngày`);
  if (s.plan_type === 'SESSION' && s.remaining_sessions !== null)
    parts.push(`${s.remaining_sessions}/${s.total_sessions ?? '?'} buổi`);
  return parts.join(' · ') || '—';
}

/** Gói còn hiệu lực (chưa kết thúc). */
export function isLiveSubscription(s: Pick<Subscription, 'status'>) {
  return s.status === 'PENDING' || s.status === 'ACTIVE' || s.status === 'FROZEN';
}

/**
 * Hạn mức bảo lưu còn lại = tối đa − (đã áp dụng + yêu cầu đang chờ/đã duyệt chưa áp).
 * Bảo lưu đã áp đã được cộng vào `freeze_days_used`; APPROVED có `start_date` > hôm nay là chưa áp.
 */
export function freezeQuota(sub: SubscriptionDetail) {
  const today = dayjs().format('YYYY-MM-DD');
  const pending = sub.freezes.filter(
    (f) => f.status === 'PENDING' || (f.status === 'APPROVED' && f.start_date > today),
  );
  const usedDays = sub.freeze_days_used + pending.reduce((n, f) => n + f.days, 0);
  const usedTimes = sub.freeze_times_used + pending.length;
  return {
    maxDays: sub.plan.max_freeze_days,
    maxTimes: sub.plan.max_freeze_times,
    daysLeft: Math.max(0, sub.plan.max_freeze_days - usedDays),
    timesLeft: Math.max(0, sub.plan.max_freeze_times - usedTimes),
  };
}

/** Cột Descriptions: 1 cột trên điện thoại, 2 cột từ `sm` trở lên (antd mặc định 3 ở md+). */
export const DESC_COLUMNS = { xs: 1, sm: 2, md: 2, lg: 2, xl: 2, xxl: 2 };
