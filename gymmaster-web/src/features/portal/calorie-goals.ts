/**
 * Chuẩn hóa gợi ý calo theo mục tiêu (giảm mỡ / tăng cơ / duy trì) từ response tổng quan dinh dưỡng.
 * Thuần (không phụ thuộc React) – chịu được khi API chưa có trường, trả `null` thay vì lỗi.
 * Hợp đồng API (NU-01, GET /v1/member/meal-plans): `bmr`, `tdee`, `has_body_metric`, `goal`, `goal_source`,
 * `goal_targets: [{ goal: CUT|MAINTAIN|BULK, calories, protein_g, carbs_g, fat_g }]`.
 * Vẫn chấp nhận dạng map `{ cut: {...} | number }` / tên trường khác để không vỡ khi hợp đồng đổi nhẹ.
 */

export type CalorieGoalKey = 'LOSE_FAT' | 'MAINTAIN' | 'GAIN_MUSCLE';

export interface CalorieGoalSuggestion {
  goal: CalorieGoalKey;
  calories: number;
  proteinG?: number;
  carbsG?: number;
  fatG?: number;
}

export interface CalorieGoalSummary {
  bmr?: number;
  tdee?: number;
  /** Mục tiêu hiện tại của hội viên (nếu API trả) – để làm nổi bật ô tương ứng. */
  currentGoal?: CalorieGoalKey;
  /** false → API chưa có chỉ số cơ thể, TDEE là giá trị mặc định. */
  hasBodyMetric?: boolean;
  goals: CalorieGoalSuggestion[];
}

const GOAL_ORDER: CalorieGoalKey[] = ['LOSE_FAT', 'MAINTAIN', 'GAIN_MUSCLE'];

const GOAL_ALIASES: Record<string, CalorieGoalKey> = {
  LOSE_FAT: 'LOSE_FAT',
  FAT_LOSS: 'LOSE_FAT',
  LOSE_WEIGHT: 'LOSE_FAT',
  WEIGHT_LOSS: 'LOSE_FAT',
  CUT: 'LOSE_FAT',
  CUTTING: 'LOSE_FAT',
  LOSE: 'LOSE_FAT',
  MAINTAIN: 'MAINTAIN',
  MAINTENANCE: 'MAINTAIN',
  GAIN_MUSCLE: 'GAIN_MUSCLE',
  MUSCLE_GAIN: 'GAIN_MUSCLE',
  BUILD_MUSCLE: 'GAIN_MUSCLE',
  BULK: 'GAIN_MUSCLE',
  BULKING: 'GAIN_MUSCLE',
  GAIN: 'GAIN_MUSCLE',
};

/** Các tên trường có thể chứa danh sách/map gợi ý theo mục tiêu. */
const GOAL_FIELDS = [
  'goal_targets',
  'goal_suggestions',
  'calorie_goals',
  'goal_calories',
  'calorie_suggestions',
  'calorie_targets',
  'goals',
  'suggestions',
] as const;

function num(v: unknown): number | undefined {
  const n = typeof v === 'string' ? Number(v) : v;
  return typeof n === 'number' && Number.isFinite(n) && n > 0 ? n : undefined;
}

function pick(obj: Record<string, unknown>, keys: string[]): number | undefined {
  for (const k of keys) {
    const v = num(obj[k]);
    if (v !== undefined) return v;
  }
  return undefined;
}

function goalKey(raw: unknown): CalorieGoalKey | undefined {
  if (typeof raw !== 'string') return undefined;
  return GOAL_ALIASES[
    raw
      .trim()
      .toUpperCase()
      .replace(/[\s-]+/g, '_')
  ];
}

function toSuggestion(goal: CalorieGoalKey, raw: unknown): CalorieGoalSuggestion | null {
  if (typeof raw === 'number' || typeof raw === 'string') {
    const calories = num(raw);
    return calories ? { goal, calories: Math.round(calories) } : null;
  }
  if (!raw || typeof raw !== 'object') return null;
  const o = raw as Record<string, unknown>;
  const calories = pick(o, ['calories', 'target_calories', 'kcal', 'calories_kcal']);
  if (!calories) return null;
  const macros = (o.macros && typeof o.macros === 'object' ? o.macros : o) as Record<
    string,
    unknown
  >;
  const round1 = (v: number | undefined) => (v === undefined ? undefined : Math.round(v));
  return {
    goal,
    calories: Math.round(calories),
    proteinG: round1(pick(macros, ['protein_g', 'protein'])),
    carbsG: round1(pick(macros, ['carbs_g', 'carbs', 'carb_g'])),
    fatG: round1(pick(macros, ['fat_g', 'fat'])),
  };
}

export function normalizeCalorieGoals(data: unknown): CalorieGoalSummary | null {
  if (!data || typeof data !== 'object') return null;
  const root = data as Record<string, unknown>;
  // Trường có thể nằm trực tiếp hoặc lồng trong một object (VD `calorie_recommendation`).
  const nested = ['calorie_recommendation', 'calorie_plan', 'energy', 'tdee_info']
    .map(k => root[k])
    .find(v => v && typeof v === 'object' && !Array.isArray(v)) as
    Record<string, unknown> | undefined;
  const sources = nested ? [nested, root] : [root];

  const found = new Map<CalorieGoalKey, CalorieGoalSuggestion>();
  for (const src of sources) {
    for (const field of GOAL_FIELDS) {
      const raw = src[field];
      if (Array.isArray(raw)) {
        for (const item of raw) {
          if (!item || typeof item !== 'object') continue;
          const it = item as Record<string, unknown>;
          const key = goalKey(it.goal ?? it.key ?? it.type ?? it.code);
          const s = key ? toSuggestion(key, it) : null;
          if (s && !found.has(s.goal)) found.set(s.goal, s);
        }
      } else if (raw && typeof raw === 'object') {
        for (const [k, v] of Object.entries(raw as Record<string, unknown>)) {
          const key = goalKey(k);
          const s = key ? toSuggestion(key, v) : null;
          if (s && !found.has(s.goal)) found.set(s.goal, s);
        }
      }
    }
  }

  const bmr = sources.map(s => pick(s, ['bmr', 'bmr_kcal'])).find(Boolean);
  const tdee = sources.map(s => pick(s, ['tdee', 'tdee_kcal'])).find(Boolean);
  const goals = GOAL_ORDER.map(g => found.get(g)).filter((g): g is CalorieGoalSuggestion =>
    Boolean(g),
  );
  if (!goals.length && !bmr && !tdee) return null;
  // goal_source = DEFAULT nghĩa là API tự chọn mặc định (không phải mục tiêu hội viên) → không làm nổi bật.
  const currentGoal = sources
    .filter(s => String(s.goal_source ?? '').toUpperCase() !== 'DEFAULT')
    .map(s => goalKey(s.current_goal ?? s.goal ?? s.member_goal ?? s.fitness_goal))
    .find(Boolean);
  const hasBodyMetric = sources
    .map(s => s.has_body_metric)
    .find((v): v is boolean => typeof v === 'boolean');
  return {
    bmr: bmr ? Math.round(bmr) : undefined,
    tdee: tdee ? Math.round(tdee) : undefined,
    currentGoal,
    hasBodyMetric,
    goals,
  };
}
