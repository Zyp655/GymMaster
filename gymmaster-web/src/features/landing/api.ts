import 'server-only';
import dayjs from 'dayjs';
import { serverApi } from '@/lib/http.server';
import type {
  LoadResult,
  PublicBranch,
  PublicClassSchedule,
  PublicClassSchedulesRaw,
  PublicPlansRes,
} from './types';

async function safe<T>(fn: () => Promise<T>): Promise<LoadResult<T>> {
  try {
    return { ok: true, data: await fn() };
  } catch {
    return { ok: false };
  }
}

export function normalizePublicSchedules(
  raw: PublicClassSchedulesRaw | null | undefined,
): PublicClassSchedule[] {
  if (!raw) return [];
  if (Array.isArray(raw)) return raw;
  return Array.isArray(raw.items) ? raw.items : [];
}

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
};
