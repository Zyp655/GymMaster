import type { StatusMeta } from '@/constants/status';

/** Trạng thái bán của gói tập / gói PT. */
export const SALE_STATUS_MAP: Record<string, StatusMeta> = {
  ACTIVE: { label: 'Đang bán', color: 'green' },
  INACTIVE: { label: 'Ngừng bán', color: 'default' },
};

export const SALE_STATUS_OPTIONS = Object.entries(SALE_STATUS_MAP).map(([value, m]) => ({
  value,
  label: m.label,
}));

export const PLAN_TYPE_META: Record<string, StatusMeta> = {
  TIME: { label: 'Theo thời gian', color: 'blue' },
  SESSION: { label: 'Theo buổi', color: 'purple' },
};

export const PLAN_TYPE_OPTIONS = Object.entries(PLAN_TYPE_META).map(([value, m]) => ({
  value,
  label: m.label,
}));

export const ACCESS_SCOPE_LABELS: Record<string, string> = {
  SINGLE_BRANCH: 'Một chi nhánh',
  ALL_BRANCHES: 'Mọi chi nhánh',
};

/** Nhãn thứ trong tuần (API: 1=T2 … 6=T7, 0/7=CN). */
export const DOW_LABELS: Record<number, string> = {
  1: 'T2',
  2: 'T3',
  3: 'T4',
  4: 'T5',
  5: 'T6',
  6: 'T7',
  0: 'CN',
  7: 'CN',
};

/** "T2, T3 08:00–10:00; 18:00–20:00" – rỗng = "Mọi giờ". Khung không chọn thứ = mọi ngày. */
export function accessHoursText(
  slots: { dow?: number[] | null; from: string; to: string }[] | null | undefined,
): string {
  if (!slots?.length) return 'Mọi giờ';
  return slots
    .map((s) => {
      const days = [...new Set((s.dow ?? []).map((d) => (d === 7 ? 0 : d)))]
        .sort((a, b) => (a === 0 ? 7 : a) - (b === 0 ? 7 : b))
        .map((d) => DOW_LABELS[d] ?? String(d));
      return `${days.length ? `${days.join(', ')} ` : ''}${s.from}–${s.to}`;
    })
    .join('; ');
}
