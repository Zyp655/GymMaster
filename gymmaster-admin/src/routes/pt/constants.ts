import dayjs from 'dayjs';
import type { PtContract, PtSession } from '@/api/pt';
import type { StatusMeta } from '@/constants/status';
import { ROLE } from '@/constants/permissions';
import type { AuthUser } from '@/api/types';

/** Trạng thái hợp đồng PT. */
export const PT_CONTRACT_STATUS_MAP: Record<string, StatusMeta> = {
  PENDING: { label: 'Chờ thanh toán', color: 'gold' },
  ACTIVE: { label: 'Đang tập', color: 'green' },
  COMPLETED: { label: 'Đã dùng hết', color: 'blue' },
  EXPIRED: { label: 'Hết hạn', color: 'default' },
  CANCELLED: { label: 'Đã hủy', color: 'red' },
};

/** Trạng thái buổi PT (NO_SHOW chưa tính buổi dùng `NO_SHOW_PENDING`). */
export const PT_SESSION_STATUS_MAP: Record<string, StatusMeta> = {
  SCHEDULED: { label: 'Đã đặt', color: 'blue' },
  COMPLETED: { label: 'Hoàn thành', color: 'green' },
  CANCELLED: { label: 'Đã hủy', color: 'default' },
  NO_SHOW: { label: 'Vắng mặt', color: 'red' },
  NO_SHOW_PENDING: { label: 'Quá giờ chưa xác nhận', color: 'orange' },
};

/** Trạng thái hoa hồng. */
export const COMMISSION_STATUS_MAP: Record<string, StatusMeta> = {
  PENDING: { label: 'Chờ chi', color: 'gold' },
  PAID: { label: 'Đã chi', color: 'green' },
};

export const PT_SESSION_SOURCE_LABEL: Record<string, string> = {
  COUNTER: 'Quầy',
  PORTAL: 'Cổng hội viên',
  TRAINER: 'HLV tự đặt',
};

export const COMMISSION_SOURCE_LABEL: Record<string, string> = {
  PT_SESSION: 'Buổi PT',
  CLASS: 'Buổi dạy lớp',
};

/** Màu khối trên lịch theo trạng thái buổi. */
export const PT_SESSION_COLOR: Record<string, string> = {
  SCHEDULED: '#1677FF',
  COMPLETED: '#16A34A',
  CANCELLED: '#9CA3AF',
  NO_SHOW: '#DC2626',
  NO_SHOW_PENDING: '#F59E0B',
};

const toOptions = (map: Record<string, StatusMeta>) =>
  Object.entries(map).map(([value, meta]) => ({ value, label: meta.label }));

export const PT_CONTRACT_STATUS_OPTIONS = toOptions(PT_CONTRACT_STATUS_MAP);
export const PT_SESSION_STATUS_OPTIONS = toOptions(PT_SESSION_STATUS_MAP).filter(
  (o) => o.value !== 'NO_SHOW_PENDING',
);
export const COMMISSION_STATUS_OPTIONS = toOptions(COMMISSION_STATUS_MAP);

/**
 * Các trạng thái hiện trên chú thích lịch (luôn đủ, kể cả Vắng mặt).
 * Lịch HLV (`/trainers/{id}/schedule`) không có cờ `counted` → không có "Quá giờ chưa xác nhận".
 */
export const legendStatusKeys = (mode: 'pt' | 'trainer'): string[] =>
  Object.keys(PT_SESSION_STATUS_MAP).filter((k) => mode === 'pt' || k !== 'NO_SHOW_PENDING');

/** Khóa trạng thái hiển thị của buổi (tách NO_SHOW chưa chốt). */
export const sessionStatusKey = (s: Pick<PtSession, 'status' | 'counted'>) =>
  s.status === 'NO_SHOW' && !s.counted ? 'NO_SHOW_PENDING' : s.status;

/** Hạn hủy mặc định (phút) – cấu hình `pt.cancel_before_minutes`. */
export const DEFAULT_CANCEL_BEFORE_MINUTES = 720;

/** Buổi đổi giờ/hủy được: SCHEDULED (đổi giờ: chưa bắt đầu). */
export const canReschedule = (s: PtSession) =>
  s.status === 'SCHEDULED' && dayjs(s.start_at).isAfter(dayjs());
export const canCancelSession = (s: PtSession) => s.status === 'SCHEDULED';
/** Xác nhận hoàn thành / vắng mặt: sau giờ bắt đầu, SCHEDULED hoặc NO_SHOW chưa tính. */
export const canConfirmSession = (s: PtSession) =>
  (s.status === 'SCHEDULED' || (s.status === 'NO_SHOW' && !s.counted)) &&
  !dayjs(s.start_at).isAfter(dayjs());

export const isLiveContract = (c: Pick<PtContract, 'status'>) =>
  c.status === 'PENDING' || c.status === 'ACTIVE';

/** Hợp đồng đặt buổi được: ACTIVE, còn buổi chưa đặt. */
export const isBookable = (c: Pick<PtContract, 'status' | 'bookable_sessions'>) =>
  c.status === 'ACTIVE' && c.bookable_sessions > 0;

/** Người chỉ có vai trò TRAINER → mặc định chỉ xem dữ liệu của mình. */
export const isTrainerOnly = (user: AuthUser | null) =>
  !!user && user.roles.length > 0 && user.roles.every((r) => r === ROLE.TRAINER);

/** "Còn 9/10 buổi · 44 ngày". */
export function contractRemainText(c: PtContract) {
  const parts = [`${c.remaining_sessions}/${c.total_sessions} buổi`];
  if (c.days_left !== null && c.days_left !== undefined) parts.push(`${c.days_left} ngày`);
  return parts.join(' · ');
}
