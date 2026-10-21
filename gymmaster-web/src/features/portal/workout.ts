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
