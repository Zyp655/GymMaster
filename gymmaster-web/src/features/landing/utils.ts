import type { PublicPlan } from './types';

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
