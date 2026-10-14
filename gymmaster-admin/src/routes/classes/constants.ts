import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { ClassLevel, ClassSchedule } from '@/api/class';
import type { StatusMeta } from '@/constants/status';

/** Cấp độ loại lớp. */
export const LEVEL_META: Record<ClassLevel, StatusMeta> = {
  ALL: { label: 'Mọi cấp độ', color: 'default' },
  BEGINNER: { label: 'Cơ bản', color: 'green' },
  INTERMEDIATE: { label: 'Trung cấp', color: 'blue' },
  ADVANCED: { label: 'Nâng cao', color: 'volcano' },
};

export const LEVEL_OPTIONS = (Object.keys(LEVEL_META) as ClassLevel[]).map((value) => ({
  value,
  label: LEVEL_META[value].label,
}));

/** Trạng thái buổi lớp. */
export const SCHEDULE_STATUS_MAP: Record<string, StatusMeta> = {
  SCHEDULED: { label: 'Sắp diễn ra', color: 'blue' },
  CANCELLED: { label: 'Đã hủy', color: 'red' },
  COMPLETED: { label: 'Đã kết thúc', color: 'default' },
};

/** Trạng thái lượt đặt chỗ. */
export const BOOKING_STATUS_MAP: Record<string, StatusMeta> = {
  BOOKED: { label: 'Đã đặt', color: 'blue' },
  WAITLIST: { label: 'Danh sách chờ', color: 'gold' },
  CANCELLED: { label: 'Đã hủy', color: 'default' },
  ATTENDED: { label: 'Có mặt', color: 'green' },
  NO_SHOW: { label: 'Vắng', color: 'red' },
};

export const BOOKING_SOURCE_LABEL: Record<string, string> = {
  COUNTER: 'Tại quầy',
  PORTAL: 'Cổng hội viên',
};

/** 1 = Thứ Hai … 7 = Chủ nhật (giống backend). */
export const WEEKDAY_OPTIONS = [
  { value: 1, label: 'T2' },
  { value: 2, label: 'T3' },
  { value: 3, label: 'T4' },
  { value: 4, label: 'T5' },
  { value: 5, label: 'T6' },
  { value: 6, label: 'T7' },
  { value: 7, label: 'CN' },
];

const WEEKDAY_LONG = ['Chủ nhật', 'Thứ Hai', 'Thứ Ba', 'Thứ Tư', 'Thứ Năm', 'Thứ Sáu', 'Thứ Bảy'];

/** "Thứ Hai" … "Chủ nhật". */
export const weekdayName = (d: Dayjs) => WEEKDAY_LONG[d.day()];

/** Thứ Hai của tuần chứa `d` (không phụ thuộc locale). */
export const startOfWeek = (d: Dayjs) => {
  const day = d.startOf('day');
  return day.subtract((day.day() + 6) % 7, 'day');
};

/** Gửi thời điểm lên API theo giờ VN: `YYYY-MM-DDTHH:mm:00+07:00`. */
export const toApiDateTime = (d: Dayjs) => `${d.format('YYYY-MM-DDTHH:mm')}:00+07:00`;

/** Khung điểm danh: từ 30 phút trước giờ bắt đầu tới 24 giờ sau khi kết thúc. */
export function attendanceWindow(s: Pick<ClassSchedule, 'start_at' | 'end_at' | 'status'>) {
  const now = dayjs();
  if (s.status === 'CANCELLED') return { open: false, reason: 'Buổi lớp đã hủy' };
  const opensAt = dayjs(s.start_at).subtract(30, 'minute');
  const closesAt = dayjs(s.end_at).add(24, 'hour');
  if (now.isBefore(opensAt))
    return {
      open: false,
      reason: `Chưa đến giờ điểm danh (mở từ ${opensAt.format('HH:mm DD/MM/YYYY')})`,
    };
  if (now.isAfter(closesAt))
    return { open: false, reason: 'Đã quá thời hạn điểm danh (24 giờ sau khi lớp kết thúc)' };
  return { open: true, reason: '' };
}

/** Buổi lớp còn sửa/hủy được: SCHEDULED và chưa bắt đầu. */
export const isUpcoming = (s: Pick<ClassSchedule, 'start_at' | 'status'>) =>
  s.status === 'SCHEDULED' && dayjs(s.start_at).isAfter(dayjs());

/** Chữ trắng/đen tương phản trên nền màu `#RRGGBB`. */
export function readableTextColor(hex: string): string {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex || '');
  if (!m) return '#fff';
  const n = parseInt(m[1], 16);
  const r = (n >> 16) & 255;
  const g = (n >> 8) & 255;
  const b = n & 255;
  return (r * 299 + g * 587 + b * 114) / 1000 > 160 ? '#1f2937' : '#fff';
}

/** `#RRGGBB` → rgba với độ trong suốt. */
export function withAlpha(hex: string, alpha: number): string {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex || '');
  if (!m) return `rgba(249, 115, 22, ${alpha})`;
  const n = parseInt(m[1], 16);
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`;
}
