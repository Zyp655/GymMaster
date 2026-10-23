import type { AccessHoursSlot, PublicPlan } from './types';

export type DurationParts = { unit: 'days' | 'months' | 'years'; count: number };

/** 365 → 1 năm, 90 → 3 tháng, 45 → 45 ngày. */
export function splitDuration(days: number): DurationParts {
  if (days >= 365 && days % 365 === 0) return { unit: 'years', count: days / 365 };
  if (days >= 30 && days % 30 === 0) return { unit: 'months', count: days / 30 };
  return { unit: 'days', count: days };
}

/** Gói TIME có giá/ngày thấp nhất (chỉ đánh dấu khi có từ 2 gói TIME trở lên). */
export function bestValuePlanId(plans: PublicPlan[]): number | null {
  const timePlans = plans.filter(p => p.type === 'TIME' && p.price > 0);
  if (timePlans.length < 2) return null;
  return timePlans.reduce((best, p) =>
    p.price / p.duration_days < best.price / best.duration_days ? p : best,
  ).id;
}

/** Khung giờ đã chuẩn hóa: ngày 1=T2 … 7=CN (tăng dần, không trùng), giờ dạng HH:MM. */
export interface NormalizedAccessSlot {
  days: number[];
  from: string;
  to: string;
}

const ALL_DAYS = [1, 2, 3, 4, 5, 6, 7];

const hhmm = (v: string | undefined) => {
  const m = /^(\d{1,2}):(\d{2})/.exec((v ?? '').trim());
  return m ? `${m[1].padStart(2, '0')}:${m[2]}` : '';
};

/** Chuẩn hóa `access_hours` (chịu được thiếu trường / dữ liệu lạ); bỏ khung giờ không hợp lệ. */
export function normalizeAccessHours(
  raw: AccessHoursSlot[] | null | undefined,
): NormalizedAccessSlot[] {
  if (!Array.isArray(raw)) return [];
  const out: NormalizedAccessSlot[] = [];
  for (const slot of raw) {
    if (!slot || typeof slot !== 'object') continue;
    const from = hhmm(slot.from ?? slot.start);
    const to = hhmm(slot.to ?? slot.end);
    if (!from || !to) continue;
    const list = slot.dow == null ? [] : Array.isArray(slot.dow) ? slot.dow : [slot.dow];
    const days = [
      ...new Set(
        list.filter(d => Number.isInteger(d) && d >= 0 && d <= 7).map(d => (d === 0 ? 7 : d)),
      ),
    ].sort((a, b) => a - b);
    out.push({ days: days.length > 0 ? days : ALL_DAYS, from, to });
  }
  return out;
}

/**
 * Ghép ngày thành chuỗi ngắn: 3+ ngày liên tiếp → "T2–T6", còn lại liệt kê "T7, CN".
 * Đủ 7 ngày → `everyDay`. `dayLabel(n)` trả nhãn ngày (1=T2 … 7=CN).
 */
export function formatAccessDays(
  days: number[],
  dayLabel: (day: number) => string,
  everyDay: string,
): string {
  if (days.length === 7) return everyDay;
  const runs: number[][] = [];
  for (const d of days) {
    const last = runs[runs.length - 1];
    if (last && d === last[last.length - 1] + 1) last.push(d);
    else runs.push([d]);
  }
  return runs
    .flatMap(run =>
      run.length >= 3
        ? [`${dayLabel(run[0])}–${dayLabel(run[run.length - 1])}`]
        : run.map(dayLabel),
    )
    .join(', ');
}

/** Các dòng mô tả khung giờ, VD "T2–T6 · 06:00–16:00". Rỗng = tập mọi giờ (không cần hiển thị). */
export function formatAccessHours(
  raw: AccessHoursSlot[] | null | undefined,
  dayLabel: (day: number) => string,
  everyDay: string,
): string[] {
  return normalizeAccessHours(raw).map(
    s => `${formatAccessDays(s.days, dayLabel, everyDay)} · ${s.from}–${s.to}`,
  );
}

/** Số vé khách mời/tháng hợp lệ (> 0), ngược lại 0. */
export function guestPassesPerMonth(plan: Pick<PublicPlan, 'guest_passes_per_month'>): number {
  const n = Number(plan.guest_passes_per_month ?? 0);
  return Number.isFinite(n) && n > 0 ? Math.floor(n) : 0;
}
