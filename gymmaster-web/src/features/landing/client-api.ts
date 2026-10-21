import { apiClient } from '@/lib/http';
import type {
  CreatePublicLeadPayload,
  CreatePublicLeadRes,
  PublicClassSchedule,
  PublicClassSchedulesRaw,
} from './types';
import { normalizePublicSchedules } from './schedule';

export async function getPublicClassSchedules(params: {
  branch_id?: number;
  from?: string;
  to?: string;
}): Promise<PublicClassSchedule[]> {
  const raw = await apiClient.get<PublicClassSchedulesRaw>('/public/class-schedules', {
    auth: false,
    query: params,
  });
  return normalizePublicSchedules(raw);
}

export async function createPublicLead(
  payload: CreatePublicLeadPayload,
): Promise<CreatePublicLeadRes> {
  return apiClient.post<CreatePublicLeadRes>('/public/leads', payload, { auth: false });
}
