import { http } from '@/utils/api';
import type { ActiveStatus, ListParams, Page } from './types';

/* ------------------------------------------------------------------ Loại lớp */

export type ClassLevel = 'ALL' | 'BEGINNER' | 'INTERMEDIATE' | 'ADVANCED';

export interface ClassType {
  id: number;
  code: string;
  name: string;
  description: string;
  /** `#RRGGBB` (IN HOA) – màu sự kiện trên lịch tuần. */
  color: string;
  default_duration_min: number;
  default_capacity: number;
  level: ClassLevel;
  status: ActiveStatus;
  created_at: string;
  updated_at: string;
}

export type ClassTypeInput = Omit<ClassType, 'id' | 'created_at' | 'updated_at'>;

export interface ClassTypeListParams extends ListParams {
  status?: ActiveStatus;
  level?: ClassLevel;
}

/** `GET /admin/class-types/options` – loại lớp ACTIVE. */
export interface ClassTypeOption {
  id: number;
  code: string;
  name: string;
  color: string;
  level: ClassLevel;
  default_duration_min: number;
  default_capacity: number;
}

/* ------------------------------------------------------------------ Lịch lớp */

export type ClassScheduleStatus = 'SCHEDULED' | 'CANCELLED' | 'COMPLETED';

export interface ClassSchedule {
  id: number;
  class_type_id: number;
  class_type_code: string;
  class_type_name: string;
  class_type_color: string;
  branch_id: number;
  branch_name: string;
  room_id: number;
  room_name: string;
  trainer_id: number;
  trainer_name: string;
  start_at: string;
  end_at: string;
  duration_min: number;
  capacity: number;
  /** Số chỗ đã chiếm = BOOKED + ATTENDED + NO_SHOW. */
  booked_count: number;
  waitlist_count: number;
  attended_count: number;
  available_slots: number;
  status: ClassScheduleStatus;
  /** `""` nếu là buổi lẻ. */
  series_id: string;
  cancel_reason: string;
  note: string;
  created_at: string;
  updated_at: string;
}

export interface ClassScheduleListParams {
  /** `YYYY-MM-DD`, bắt buộc. */
  from: string;
  /** `YYYY-MM-DD`, bắt buộc (bao gồm). Tối đa 62 ngày. */
  to: string;
  branch_id?: number;
  trainer_id?: number;
  class_type_id?: number;
  room_id?: number;
  status?: ClassScheduleStatus;
}

export interface ClassScheduleCreateInput {
  class_type_id: number;
  branch_id: number;
  room_id: number;
  trainer_id: number;
  /** RFC3339 hoặc `YYYY-MM-DDTHH:mm` (giờ VN). */
  start_at: string;
  duration_min?: number;
  capacity?: number;
  note?: string;
}

export interface ClassScheduleRecurringInput {
  class_type_id: number;
  branch_id: number;
  room_id: number;
  trainer_id: number;
  start_date: string;
  end_date: string;
  /** 1 = Thứ Hai … 7 = Chủ nhật. */
  weekdays: number[];
  /** `HH:mm`. */
  start_time: string;
  duration_min?: number;
  capacity?: number;
  note?: string;
}

export interface RecurringSkipped {
  date: string;
  reason: string;
}

export interface ClassScheduleRecurringResult {
  series_id: string;
  created: number;
  schedule_ids: number[];
  skipped: RecurringSkipped[];
}

export interface ClassScheduleUpdateInput {
  room_id: number;
  trainer_id: number;
  start_at: string;
  duration_min: number;
  capacity: number;
  note?: string;
}

export type CancelScope = 'one' | 'following';

export interface ClassScheduleCancelInput {
  reason: string;
  scope?: CancelScope;
}

export interface ClassScheduleCancelResult {
  cancelled_schedules: number;
  cancelled_bookings: number;
  schedule_ids: number[];
}

/* ------------------------------------------------------------------ Đặt chỗ */

export type BookingStatus = 'BOOKED' | 'WAITLIST' | 'CANCELLED' | 'ATTENDED' | 'NO_SHOW';
export type BookingSource = 'COUNTER' | 'PORTAL';

export interface ClassBooking {
  id: number;
  schedule_id: number;
  member_id: number;
  member_code: string;
  member_name: string;
  member_phone: string;
  subscription_id: number;
  status: BookingStatus;
  waitlist_position: number | null;
  source: BookingSource;
  booked_at: string;
  cancelled_at: string | null;
  attended_at: string | null;
  class_type_id: number;
  class_type_name: string;
  class_type_color: string;
  branch_id: number;
  branch_name: string;
  room_name: string;
  trainer_id: number;
  trainer_name: string;
  start_at: string;
  end_at: string;
  schedule_status: ClassScheduleStatus;
}

export interface AttendanceItem {
  booking_id: number;
  status: 'ATTENDED' | 'NO_SHOW';
}

export interface MemberBookingParams {
  member_id: number;
  from?: string;
  to?: string;
}

/* ------------------------------------------------------------------ Dropdown từ module khác */

export interface TrainerOption {
  id: number;
  full_name: string;
  level?: string;
  specialties?: string[];
}

export interface RoomOption {
  id: number;
  branch_id: number;
  name: string;
  capacity: number;
}

/** Giờ mở cửa chi nhánh (`/public/branches`) – trục giờ của lịch tuần. */
export interface BranchHours {
  id: number;
  name: string;
  open_time: string;
  close_time: string;
}

/** Lớp nhóm – permission `class.view`, `class.manage`, `booking.manage`. */
class ClassApi {
  /** Khóa gốc cho loại lớp. */
  readonly typeKey = 'class-types';
  /** Khóa gốc cho lịch lớp + đặt chỗ (đổi lịch/đặt chỗ → invalidate cả hai). */
  readonly key = 'class-schedules';

  // Loại lớp
  listTypes(params: ClassTypeListParams = {}) {
    return http.get<Page<ClassType>>('/admin/class-types', { params });
  }
  typeOptions() {
    return http.get<ClassTypeOption[]>('/admin/class-types/options');
  }
  getType(id: number) {
    return http.get<ClassType>(`/admin/class-types/${id}`);
  }
  createType(data: ClassTypeInput) {
    return http.post<ClassType>('/admin/class-types', data);
  }
  updateType(id: number, data: ClassTypeInput) {
    return http.put<ClassType>(`/admin/class-types/${id}`, data);
  }
  removeType(id: number) {
    return http.delete(`/admin/class-types/${id}`);
  }

  // Lịch lớp
  listSchedules(params: ClassScheduleListParams) {
    return http.get<ClassSchedule[]>('/admin/class-schedules', { params });
  }
  getSchedule(id: number) {
    return http.get<ClassSchedule>(`/admin/class-schedules/${id}`);
  }
  createSchedule(data: ClassScheduleCreateInput) {
    return http.post<ClassSchedule>('/admin/class-schedules', data);
  }
  createRecurring(data: ClassScheduleRecurringInput) {
    return http.post<ClassScheduleRecurringResult>('/admin/class-schedules/recurring', data);
  }
  updateSchedule(id: number, data: ClassScheduleUpdateInput) {
    return http.put<ClassSchedule>(`/admin/class-schedules/${id}`, data);
  }
  cancelSchedule(id: number, data: ClassScheduleCancelInput) {
    return http.post<ClassScheduleCancelResult>(`/admin/class-schedules/${id}/cancel`, data);
  }

  // Đặt chỗ
  listBookings(scheduleId: number, status?: BookingStatus) {
    return http.get<ClassBooking[]>(`/admin/class-schedules/${scheduleId}/bookings`, {
      params: { status },
    });
  }
  book(scheduleId: number, memberId: number) {
    return http.post<ClassBooking>(`/admin/class-schedules/${scheduleId}/bookings`, {
      member_id: memberId,
    });
  }
  cancelBooking(bookingId: number) {
    return http.post<ClassBooking>(`/admin/class-bookings/${bookingId}/cancel`);
  }
  markAttendance(scheduleId: number, items: AttendanceItem[]) {
    return http.put<{ updated: number }>(`/admin/class-schedules/${scheduleId}/attendance`, {
      items,
    });
  }
  memberBookings(params: MemberBookingParams) {
    return http.get<ClassBooking[]>('/admin/class-bookings', { params });
  }

  // Dropdown
  trainerOptions(branchId?: number) {
    return http.get<TrainerOption[]>('/admin/trainers/options', {
      params: { branch_id: branchId },
    });
  }
  roomOptions(branchId?: number) {
    return http.get<RoomOption[]>('/admin/rooms/options', { params: { branch_id: branchId } });
  }
  /** Chi nhánh đang hoạt động kèm giờ mở cửa (public, không cần quyền). */
  branchHours() {
    return http.get<BranchHours[]>('/public/branches', { silent: true });
  }
}

export const classApi = new ClassApi();
export default classApi;
