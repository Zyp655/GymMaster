import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

/** Thư viện bài tập & Giáo án tập luyện (Sprint 14 – permission `workout.view` / `workout.manage`). */

export type MuscleGroup =
  | 'CHEST'
  | 'BACK'
  | 'SHOULDERS'
  | 'LEGS'
  | 'ARMS'
  | 'CORE'
  | 'CARDIO'
  | 'FULL_BODY';

export type WorkoutPlanStatus = 'DRAFT' | 'ACTIVE' | 'COMPLETED' | 'ARCHIVED';

export const MUSCLE_GROUP_LABELS: Record<MuscleGroup, string> = {
  CHEST: 'Ngực (Chest)',
  BACK: 'Lưng – Xô (Back)',
  SHOULDERS: 'Vai (Shoulders)',
  LEGS: 'Chân – Mông (Legs)',
  ARMS: 'Tay (Arms)',
  CORE: 'Bụng – Core',
  CARDIO: 'Cardio / Tim mạch',
  FULL_BODY: 'Toàn thân (Full Body)',
};

export const MUSCLE_GROUP_COLORS: Record<MuscleGroup, string> = {
  CHEST: 'red',
  BACK: 'blue',
  SHOULDERS: 'orange',
  LEGS: 'green',
  ARMS: 'purple',
  CORE: 'gold',
  CARDIO: 'cyan',
  FULL_BODY: 'geekblue',
};

export const MUSCLE_GROUP_OPTIONS = (Object.keys(MUSCLE_GROUP_LABELS) as MuscleGroup[]).map(
  (value) => ({ value, label: MUSCLE_GROUP_LABELS[value] }),
);

export interface Exercise {
  id: number;
  name: string;
  muscle_group: MuscleGroup;
  equipment: string;
  video_url: string;
  description: string;
  created_at?: string;
  updated_at?: string;
}

export interface ExerciseInput {
  name: string;
  muscle_group: MuscleGroup;
  equipment?: string;
  video_url?: string;
  description?: string;
}

export interface ExerciseListParams extends ListParams {
  muscle_group?: MuscleGroup;
}

export interface WorkoutPlanItem {
  id?: number;
  day_id?: number;
  exercise_id: number;
  exercise_name?: string;
  muscle_group?: MuscleGroup;
  sets: number;
  reps: number;
  weight: number;
  rest_sec: number;
  note?: string;
  sort?: number;
}

export interface WorkoutPlanDay {
  id?: number;
  plan_id?: number;
  day_index: number;
  title: string;
  items: WorkoutPlanItem[];
}

export interface WorkoutPlan {
  id: number;
  member_id: number | null;
  member_name?: string;
  member_code?: string;
  trainer_id: number;
  trainer_name?: string;
  pt_contract_id?: number | null;
  name: string;
  goal: string;
  start_date: string | null;
  end_date: string | null;
  status: WorkoutPlanStatus;
  day_count?: number;
  days?: WorkoutPlanDay[];
  created_at?: string;
  updated_at?: string;
}

export interface WorkoutPlanInput {
  member_id?: number | null;
  trainer_id?: number;
  pt_contract_id?: number | null;
  name: string;
  goal?: string;
  start_date?: string | null;
  end_date?: string | null;
  status?: WorkoutPlanStatus;
  days: WorkoutPlanDay[];
}

export interface WorkoutPlanCloneInput {
  member_id: number;
  trainer_id?: number;
  name?: string;
  start_date?: string | null;
  end_date?: string | null;
}

export interface WorkoutPlanListParams extends ListParams {
  member_id?: number;
  trainer_id?: number;
  status?: WorkoutPlanStatus;
  is_template?: boolean;
}

function normalizePage<T>(res: Page<T> | T[]): Page<T> {
  if (Array.isArray(res)) {
    return {
      items: res,
      total: res.length,
      total_page: 1,
      current_page: 1,
      limit: res.length || 20,
    };
  }
  return res;
}

class WorkoutApi {
  readonly exerciseKey = 'exercises';
  readonly planKey = 'workout-plans';

  async listExercises(params: ExerciseListParams = {}): Promise<Page<Exercise>> {
    const res = await http.get<Page<Exercise> | Exercise[]>('/admin/exercises', { params });
    return normalizePage(res);
  }

  createExercise(data: ExerciseInput) {
    return http.post<Exercise>('/admin/exercises', data);
  }

  updateExercise(id: number, data: ExerciseInput) {
    return http.put<Exercise>(`/admin/exercises/${id}`, data);
  }

  deleteExercise(id: number) {
    return http.delete(`/admin/exercises/${id}`);
  }

  async listPlans(params: WorkoutPlanListParams = {}): Promise<Page<WorkoutPlan>> {
    const res = await http.get<Page<WorkoutPlan> | WorkoutPlan[]>('/admin/workout-plans', {
      params,
    });
    return normalizePage(res);
  }

  getPlan(id: number) {
    return http.get<WorkoutPlan>(`/admin/workout-plans/${id}`);
  }

  createPlan(data: WorkoutPlanInput) {
    return http.post<WorkoutPlan>('/admin/workout-plans', data);
  }

  updatePlan(id: number, data: WorkoutPlanInput) {
    return http.put<WorkoutPlan>(`/admin/workout-plans/${id}`, data);
  }

  deletePlan(id: number) {
    return http.delete(`/admin/workout-plans/${id}`);
  }

  clonePlan(id: number, data: WorkoutPlanCloneInput) {
    return http.post<WorkoutPlan>(`/admin/workout-plans/${id}/clone`, data);
  }
}

export const workoutApi = new WorkoutApi();
export default workoutApi;
