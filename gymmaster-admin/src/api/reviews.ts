import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

/** Đánh giá HLV & lớp học (Sprint 15 – permission `review.view`). */

export type ReviewTargetType = 'PT_SESSION' | 'CLASS_SCHEDULE';

export const REVIEW_TARGET_LABELS: Record<ReviewTargetType, string> = {
  PT_SESSION: 'Buổi tập PT',
  CLASS_SCHEDULE: 'Lớp học nhóm',
};

export const REVIEW_TARGET_COLORS: Record<ReviewTargetType, string> = {
  PT_SESSION: 'orange',
  CLASS_SCHEDULE: 'blue',
};

export const REVIEW_TARGET_OPTIONS = (Object.keys(REVIEW_TARGET_LABELS) as ReviewTargetType[]).map(
  (value) => ({ value, label: REVIEW_TARGET_LABELS[value] }),
);

export const RATING_OPTIONS = [5, 4, 3, 2, 1].map((n) => ({
  value: n,
  label: `${n} sao (${'★'.repeat(n)})`,
}));

export interface ReviewItem {
  id: number;
  member_id: number;
  member_name?: string;
  member_code?: string;
  target_type: ReviewTargetType;
  target_id: number;
  target_title?: string;
  trainer_id: number;
  trainer_name?: string;
  branch_id?: number | null;
  branch_name?: string;
  rating: number;
  comment: string;
  created_at: string;
}

export interface ReviewListParams extends ListParams {
  branch_id?: number;
  trainer_id?: number;
  member_id?: number;
  target_type?: ReviewTargetType;
  rating?: number;
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

class ReviewsApi {
  readonly key = 'reviews';

  async list(params: ReviewListParams = {}): Promise<Page<ReviewItem>> {
    const res = await http.get<Page<ReviewItem> | ReviewItem[]>('/admin/reviews', { params });
    return normalizePage(res);
  }
}

export const reviewsApi = new ReviewsApi();
export default reviewsApi;
