import type { PublicClassSchedule, PublicClassSchedulesRaw } from './types';

/** Số chỗ còn trống: ưu tiên `available_slots` từ API, thiếu thì tính capacity − booked_count (≥ 0). */
export function scheduleAvailableSlots(
  item: Pick<PublicClassSchedule, 'capacity'> &
    Partial<Pick<PublicClassSchedule, 'available_slots' | 'booked_count'>>,
): number {
  if (typeof item.available_slots === 'number' && Number.isFinite(item.available_slots)) {
    return Math.max(0, item.available_slots);
  }
  const capacity = Number(item.capacity) || 0;
  const booked = Number(item.booked_count) || 0;
  return Math.max(0, capacity - booked);
}

export function normalizePublicSchedules(
  raw: PublicClassSchedulesRaw | null | undefined,
): PublicClassSchedule[] {
  if (!raw) return [];
  const list = Array.isArray(raw) ? raw : Array.isArray(raw.items) ? raw.items : [];
  return list.map(item => ({ ...item, available_slots: scheduleAvailableSlots(item) }));
}
