import type { ClassBooking, ClassesResponse, ScheduleItem } from './types';

const VN_TZ = 'Asia/Ho_Chi_Minh';

/** Ngày (YYYY-MM-DD) theo giờ Việt Nam – khớp cách API so sánh hạn đặt trước. */
function vnDate(value: Date): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: VN_TZ,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(value);
}

/** Thứ hạng chờ thực tế: ưu tiên `waitlist_rank` (API mới), thiếu thì `waitlist_position`. */
export function waitlistRank(
  booking: Pick<ClassBooking, 'waitlist_position' | 'waitlist_rank'> | null | undefined,
): number | null {
  if (!booking) return null;
  const rank = booking.waitlist_rank ?? booking.waitlist_position;
  return typeof rank === 'number' && rank > 0 ? rank : null;
}

export type NotBookableReason =
  'BLOCKED' | 'OUT_OF_RANGE' | 'TRAINER_LEAVE' | 'HOLIDAY' | 'FULL' | 'STARTED' | 'UNAVAILABLE';

const KNOWN_REASONS: readonly NotBookableReason[] = [
  'BLOCKED',
  'OUT_OF_RANGE',
  'TRAINER_LEAVE',
  'HOLIDAY',
  'FULL',
  'STARTED',
];

/**
 * Lý do buổi lớp không đặt được. Thứ tự: lý do API trả (nếu có) → khóa đặt lớp → đã bắt đầu →
 * quá hạn đặt trước (advance_days, tính theo ngày giờ VN như API) → còn lại (HLV nghỉ phép /
 * ngày lễ… API chưa trả lý do) là 'UNAVAILABLE' – nhãn trung tính, tránh báo sai nguyên nhân.
 */
export function notBookableReason(
  item: Pick<ScheduleItem, 'bookable' | 'start_at' | 'not_bookable_reason'>,
  res: Pick<ClassesResponse, 'advance_days' | 'booking_blocked_until'>,
  now: Date = new Date(),
): NotBookableReason | null {
  if (item.bookable) return null;
  const apiReason = item.not_bookable_reason?.toUpperCase() as NotBookableReason | undefined;
  if (apiReason && KNOWN_REASONS.includes(apiReason)) return apiReason;
  if (res.booking_blocked_until && new Date(res.booking_blocked_until) > now) return 'BLOCKED';
  const start = new Date(item.start_at);
  if (Number.isNaN(start.getTime())) return 'UNAVAILABLE';
  if (start <= now) return 'STARTED';
  if (typeof res.advance_days === 'number' && res.advance_days >= 0) {
    const lastDay = vnDate(new Date(now.getTime() + res.advance_days * 86_400_000));
    if (vnDate(start) > lastDay) return 'OUT_OF_RANGE';
  }
  return 'UNAVAILABLE';
}
