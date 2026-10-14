import { http } from '@/utils/api';
import type { ActiveStatus, ListParams, Page } from './types';

/** Phòng tập – permission `room.view` / `room.manage`. */

export interface Room {
  id: number;
  branch_id: number;
  branch_name: string;
  name: string;
  capacity: number;
  description: string;
  status: ActiveStatus;
  created_at: string;
  updated_at: string;
}

export interface RoomInput {
  branch_id: number;
  name: string;
  capacity: number;
  description?: string;
  status: ActiveStatus;
}

export interface RoomListParams extends ListParams {
  branch_id?: number;
  status?: ActiveStatus;
}

export interface RoomOption {
  id: number;
  branch_id: number;
  name: string;
  capacity: number;
}

class RoomApi {
  readonly key = 'rooms';

  list(params: RoomListParams = {}) {
    return http.get<Page<Room>>('/admin/rooms', { params });
  }

  get(id: number) {
    return http.get<Room>(`/admin/rooms/${id}`);
  }

  create(data: RoomInput) {
    return http.post<Room>('/admin/rooms', data);
  }

  update(id: number, data: RoomInput) {
    return http.put<Room>(`/admin/rooms/${id}`, data);
  }

  remove(id: number) {
    return http.delete(`/admin/rooms/${id}`);
  }

  /** Phòng ACTIVE (sắp theo chi nhánh rồi tên). */
  options(branchId?: number | null) {
    return http.get<RoomOption[]>('/admin/rooms/options', {
      params: { branch_id: branchId ?? undefined },
    });
  }
}

export const roomApi = new RoomApi();
export default roomApi;
