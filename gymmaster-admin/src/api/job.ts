import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

/** Job định kỳ – `docs/api/reports-system.md` mục 4. */

export type JobRunStatus = 'RUNNING' | 'SUCCESS' | 'FAILED';

export interface JobRun {
  id: number;
  job: string;
  started_at: string;
  finished_at: string | null;
  /** `null` khi đang chạy. */
  duration_ms: number | null;
  status: JobRunStatus;
  affected: number;
  message: string;
}

export interface JobInfo {
  name: string;
  spec: string;
  description: string;
  /** `false` khi `CRON_ENABLED=false` (chỉ chạy thủ công). */
  scheduled: boolean;
  running: boolean;
  next_run_at: string | null;
  last_run: JobRun | null;
}

/** Mã trạng thái thô trong mô tả job (do backend sinh) → nhãn tiếng Việt khi hiển thị. */
const JOB_TEXT_CODES: Record<string, string> = {
  ACTIVE: 'Đang hiệu lực',
  EXPIRED: 'Hết hạn',
  OFFLINE: 'Mất kết nối',
  ONLINE: 'Đang kết nối',
  NO_SHOW: 'Vắng mặt',
  PENDING: 'Chờ xử lý',
  CANCELLED: 'Đã hủy',
};

export function humanizeJobText(text: string | null | undefined): string {
  if (!text) return '';
  return text.replace(/\b[A-Z][A-Z_]{2,}\b/g, (code) => JOB_TEXT_CODES[code] ?? code);
}

export interface JobRunListParams extends ListParams {
  status?: JobRunStatus;
}

class JobApi {
  readonly key = 'jobs';

  list() {
    return http.get<JobInfo[]>('/admin/jobs');
  }

  runs(name: string, params: JobRunListParams = {}) {
    return http.get<Page<JobRun>>(`/admin/jobs/${encodeURIComponent(name)}/runs`, { params });
  }

  /** Chạy đồng bộ – chờ job xong (timeout dài). */
  run(name: string) {
    return http.post<JobRun>(`/admin/jobs/${encodeURIComponent(name)}/run`, undefined, {
      timeout: 11 * 60 * 1000,
    });
  }

  testEmail(to: string) {
    return http.post<{ sent: boolean; to: string }>('/admin/jobs/test-email', { to });
  }
}

export const jobApi = new JobApi();
export default jobApi;
