import { apiClient } from '@/lib/http';
import type {
  CreatePublicLeadPayload,
  CreatePublicLeadRes,
  PublicClassSchedule,
  PublicClassSchedulesRaw,
} from './types';

export function extractPublicSchedules(
  raw: PublicClassSchedulesRaw | null | undefined,
): PublicClassSchedule[] {
  if (!raw) return [];
  if (Array.isArray(raw)) return raw;
  return Array.isArray(raw.items) ? raw.items : [];
}

export async function getPublicClassSchedules(params: {
  branch_id?: number;
  from?: string;
  to?: string;
}): Promise<PublicClassSchedule[]> {
  const raw = await apiClient.get<PublicClassSchedulesRaw>('/public/class-schedules', {
    auth: false,
    query: params,
  });
  return extractPublicSchedules(raw);
}

export async function createPublicLead(
  payload: CreatePublicLeadPayload,
): Promise<CreatePublicLeadRes> {
  return apiClient.post<CreatePublicLeadRes>('/public/leads', payload, { auth: false });
}
