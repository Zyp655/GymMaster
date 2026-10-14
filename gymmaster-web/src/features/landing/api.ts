import 'server-only';
import { serverApi } from '@/lib/http.server';
import type { LoadResult, PublicBranch, PublicPlansRes } from './types';

async function safe<T>(fn: () => Promise<T>): Promise<LoadResult<T>> {
  try {
    return { ok: true, data: await fn() };
  } catch {
    return { ok: false };
  }
}

/** Endpoint công khai – không gắn token; http.core luôn dùng cache: 'no-store'. */
export const landingApi = {
  plans: () => safe(() => serverApi.get<PublicPlansRes>('/public/plans', { auth: false })),
  branches: () => safe(() => serverApi.get<PublicBranch[]>('/public/branches', { auth: false })),
};
