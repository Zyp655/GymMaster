import 'server-only';
import dayjs from 'dayjs';
import { serverApi } from '@/lib/http.server';
import { ApiCode, isApiError } from '@/types/api';
import { isValidPostSlug } from './post-slug';
import { normalizePublicSchedules } from './schedule';
import type {
  LoadResult,
  PublicBranch,
  PublicClassSchedulesRaw,
  PublicPlansRes,
  PublicPost,
  PublicTestimonial,
  PublicTrainer,
} from './types';

async function safe<T>(fn: () => Promise<T>): Promise<LoadResult<T>> {
  try {
    return { ok: true, data: await fn() };
  } catch {
    return { ok: false };
  }
}

export { normalizePublicSchedules } from './schedule';

/** Endpoint công khai – không gắn token; http.core luôn dùng cache: 'no-store'. */
export const landingApi = {
  plans: () => safe(() => serverApi.get<PublicPlansRes>('/public/plans', { auth: false })),
  branches: () => safe(() => serverApi.get<PublicBranch[]>('/public/branches', { auth: false })),
  classSchedules: (params?: { branch_id?: number; from?: string; to?: string }) =>
    safe(async () => {
      const from = params?.from ?? dayjs().format('YYYY-MM-DD');
      const to = params?.to ?? dayjs().add(7, 'day').format('YYYY-MM-DD');
      const raw = await serverApi.get<PublicClassSchedulesRaw>('/public/class-schedules', {
        auth: false,
        query: { branch_id: params?.branch_id, from, to },
      });
      return normalizePublicSchedules(raw);
    }),
  trainers: () => safe(() => serverApi.get<PublicTrainer[]>('/public/trainers', { auth: false })),
  testimonials: () =>
    safe(() => serverApi.get<PublicTestimonial[]>('/public/testimonials', { auth: false })),
  /**
   * Chi tiết bài viết theo slug (API chỉ trả bài đã xuất bản). `notFound: true` khi API trả 404
   * hoặc slug không hợp lệ – để trang gọi notFound() thay vì hiển thị lỗi tải.
   */
  postDetail: async (
    slug: string,
  ): Promise<{ ok: true; data: PublicPost } | { ok: false; notFound: boolean }> => {
    if (!isValidPostSlug(slug)) return { ok: false, notFound: true };
    try {
      const data = await serverApi.get<PublicPost>(`/public/posts/${encodeURIComponent(slug)}`, {
        auth: false,
      });
      return { ok: true, data };
    } catch (error) {
      const notFound =
        isApiError(error) && (error.code === ApiCode.NOT_FOUND || error.status === 404);
      return { ok: false, notFound };
    }
  },
  posts: (category?: string) =>
    safe(() =>
      serverApi.get<PublicPost[]>('/public/posts', {
        auth: false,
        query: category ? { category } : undefined,
      }),
    ),
};
