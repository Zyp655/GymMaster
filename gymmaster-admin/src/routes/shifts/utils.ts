import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { Shift } from '@/api/shift';

/** Thứ Hai của tuần chứa `d`. */
export const weekStart = (d: Dayjs) => d.startOf('day').subtract((d.day() + 6) % 7, 'day');

export const WEEKDAY_LABELS = ['T2', 'T3', 'T4', 'T5', 'T6', 'T7', 'CN'];

/**
 * Backend trả thời điểm RFC3339 theo giờ VN (+07:00) → cắt chuỗi để không phụ thuộc múi giờ trình duyệt.
 */
export const shiftDate = (iso: string) => iso.slice(0, 10);
export const shiftTime = (iso: string) => iso.slice(11, 16);

export const shiftLabel = (s: Pick<Shift, 'start_at' | 'end_at'>) => {
  const overnight = shiftDate(s.end_at) !== shiftDate(s.start_at);
  return `${shiftTime(s.start_at)}–${shiftTime(s.end_at)}${overnight ? ' (+1)' : ''}`;
};

/** Số giờ của ca (theo chuỗi giờ VN). */
export const shiftHours = (s: Pick<Shift, 'start_at' | 'end_at'>) =>
  dayjs(s.end_at).diff(dayjs(s.start_at), 'minute') / 60;
