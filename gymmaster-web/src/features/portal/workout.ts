import type { WorkoutLog } from './types';

/** Bài tập mặc định cho biểu đồ tiến bộ: bài có nhật ký mới nhất, nếu không thì bài đầu thư viện. */
export function pickDefaultExerciseId(
  logs: Pick<WorkoutLog, 'exercise_id' | 'performed_at'>[],
  exercises: { id: number }[],
): number | undefined {
  let latest: Pick<WorkoutLog, 'exercise_id' | 'performed_at'> | undefined;
  for (const log of logs) {
    if (!latest || (log.performed_at ?? '') > (latest.performed_at ?? '')) latest = log;
  }
  return latest?.exercise_id ?? exercises[0]?.id;
}

/** Mức tạ tối đa hợp lý cho một hiệp (kg) – chặn nhập nhầm kiểu 1000 kg (FD-05). */
export const MAX_WORKOUT_WEIGHT_KG = 500;

/** Ngày hiện tại theo giờ máy người dùng, dạng YYYY-MM-DD (khớp giá trị của <input type="date">). */
export function localDateISO(now: Date = new Date()): string {
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, '0');
  const d = String(now.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

/** true khi ngày tập (YYYY-MM-DD) hợp lệ và không nằm sau hôm nay. */
export function isNotFutureDate(value: string, now: Date = new Date()): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  return value <= localDateISO(now);
}
