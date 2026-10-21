import type { BranchHoliday } from '@/api/operations-ext';

/** "7:00" / "07:00" → "07:00"; null nếu sai định dạng HH:mm (00:00–23:59). Khớp quy tắc backend. */
export function normalizeHHMM(raw: string | null | undefined): string | null {
  const m = /^([01]?\d|2[0-3]):([0-5]\d)$/.exec((raw ?? '').trim());
  if (!m) return null;
  return `${m[1].padStart(2, '0')}:${m[2]}`;
}

/** Tóm tắt lịch đã xếp bị ảnh hưởng bởi ngày lễ vừa lưu; null nếu không có buổi nào. */
export function holidayImpactSummary(
  h: Pick<BranchHoliday, 'affected_class_schedules' | 'affected_pt_sessions'>,
): {
  classes: number;
  pt: number;
} | null {
  const classes = h.affected_class_schedules?.length ?? 0;
  const pt = h.affected_pt_sessions?.length ?? 0;
  return classes + pt > 0 ? { classes, pt } : null;
}
