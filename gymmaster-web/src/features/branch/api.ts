import { apiClient } from '@/lib/http';
import type { Paginated } from '@/types/api';
import type { Branch, BranchListParams, BranchPayload } from './types';

const BASE = '/admin/branches';

export const branchApi = {
  list: (params: BranchListParams, signal?: AbortSignal) =>
    apiClient.get<Paginated<Branch>>(BASE, { query: { ...params }, signal }),
  get: (id: number) => apiClient.get<Branch>(`${BASE}/${id}`),
  create: (payload: BranchPayload) => apiClient.post<Branch>(BASE, payload),
  update: (id: number, payload: BranchPayload) => apiClient.put<Branch>(`${BASE}/${id}`, payload),
  remove: (id: number) => apiClient.del<null>(`${BASE}/${id}`),
};
