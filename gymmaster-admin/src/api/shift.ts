import { http } from '@/utils/api';
import type { StaffPosition } from './staff';

/** Ca làm việc – permission `shift.view` / `shift.manage`. */

export interface Shift {
  id: number;
  branch_id: number;
  branch_name: string;
  staff_id: number;
  staff_name: string;
  staff_code: string;
  position: StaffPosition;
  /** RFC3339. */
  start_at: string;
  end_at: string;
  note: string;
}

export interface ShiftInput {
  branch_id: number;
  staff_id: number;
  /** RFC3339 hoặc `YYYY-MM-DDTHH:mm` (giờ VN). */
  start_at: string;
  end_at: string;
  note?: string;
}

export interface ShiftListParams {
  /** `YYYY-MM-DD`, bao gồm hai đầu, tối đa 62 ngày. */
  from: string;
  to: string;
  branch_id?: number;
  staff_id?: number;
}

export interface CopyWeekInput {
  branch_id: number;
  /** Thứ Hai tuần nguồn `YYYY-MM-DD`. */
  from_week_start: string;
  to_week_start: string;
}

export interface CopyWeekResult {
  created: number;
  skipped: number;
}

class ShiftApi {
  readonly key = 'shifts';

  /** Mảng (không phân trang), sắp theo `start_at`. */
  list(params: ShiftListParams) {
    return http.get<Shift[]>('/admin/shifts', { params });
  }

  create(data: ShiftInput) {
    return http.post<Shift>('/admin/shifts', data);
  }

  update(id: number, data: ShiftInput) {
    return http.put<Shift>(`/admin/shifts/${id}`, data);
  }

  remove(id: number) {
    return http.delete(`/admin/shifts/${id}`);
  }

  copyWeek(data: CopyWeekInput) {
    return http.post<CopyWeekResult>('/admin/shifts/copy-week', data);
  }
}

export const shiftApi = new ShiftApi();
export default shiftApi;
