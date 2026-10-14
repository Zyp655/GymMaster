import { http } from '@/utils/api';
import type { ListParams, Page } from './types';

/** Thiết bị phòng tập & Nhật ký bảo trì (Sprint 15 – permission `equipment.view` / `equipment.manage`). */

export type EquipmentCategory =
  | 'CARDIO'
  | 'STRENGTH'
  | 'FREE_WEIGHT'
  | 'FUNCTIONAL'
  | 'ACCESSORY'
  | 'OTHER';

export type EquipmentStatus = 'ACTIVE' | 'BROKEN' | 'MAINTENANCE' | 'RETIRED';
export type MaintenanceType = 'PERIODIC' | 'REPAIR';
export type MaintenanceStatus = 'OPEN' | 'IN_PROGRESS' | 'DONE';

export const EQUIPMENT_CATEGORY_LABELS: Record<EquipmentCategory, string> = {
  CARDIO: 'Máy Cardio (Chạy bộ, đạp xe...)',
  STRENGTH: 'Máy tập cơ (Strength)',
  FREE_WEIGHT: 'Tạ rời & Ghế tập (Free Weight)',
  FUNCTIONAL: 'Dụng cụ Functional',
  ACCESSORY: 'Phụ kiện tập luyện',
  OTHER: 'Khác',
};

export const EQUIPMENT_CATEGORY_OPTIONS = (
  Object.keys(EQUIPMENT_CATEGORY_LABELS) as EquipmentCategory[]
).map((value) => ({ value, label: EQUIPMENT_CATEGORY_LABELS[value] }));

export const MAINTENANCE_TYPE_LABELS: Record<MaintenanceType, string> = {
  PERIODIC: 'Bảo trì định kỳ',
  REPAIR: 'Sửa chữa sự cố',
};

export const MAINTENANCE_TYPE_COLORS: Record<MaintenanceType, string> = {
  PERIODIC: 'blue',
  REPAIR: 'volcano',
};

export const MAINTENANCE_TYPE_OPTIONS = (
  Object.keys(MAINTENANCE_TYPE_LABELS) as MaintenanceType[]
).map((value) => ({ value, label: MAINTENANCE_TYPE_LABELS[value] }));

export interface Equipment {
  id: number;
  branch_id: number;
  branch_name?: string;
  room_id: number | null;
  room_name?: string;
  code: string;
  name: string;
  category: EquipmentCategory;
  purchase_date: string | null;
  price: number;
  warranty_until: string | null;
  status: EquipmentStatus;
  maintenance_interval_days: number;
  next_maintenance_date: string | null;
  image_url: string;
  note: string;
  created_at?: string;
  updated_at?: string;
}

export interface EquipmentInput {
  branch_id: number;
  room_id?: number | null;
  code: string;
  name: string;
  category: EquipmentCategory;
  purchase_date?: string | null;
  price?: number;
  warranty_until?: string | null;
  status?: EquipmentStatus;
  maintenance_interval_days?: number;
  next_maintenance_date?: string | null;
  image_url?: string;
  note?: string;
}

export interface EquipmentListParams extends ListParams {
  branch_id?: number;
  room_id?: number;
  category?: EquipmentCategory;
  status?: EquipmentStatus;
}

export interface MaintenanceLog {
  id: number;
  equipment_id: number;
  equipment_code?: string;
  equipment_name?: string;
  branch_id?: number;
  branch_name?: string;
  type: MaintenanceType;
  reported_by?: number | null;
  reported_by_name?: string;
  description: string;
  cost: number;
  status: MaintenanceStatus;
  started_at?: string | null;
  finished_at?: string | null;
  created_at: string;
  updated_at?: string;
}

export interface MaintenanceLogInput {
  equipment_id: number;
  type: MaintenanceType;
  description: string;
  cost?: number;
  status?: MaintenanceStatus;
}

export interface MaintenanceLogUpdateInput {
  type?: MaintenanceType;
  description?: string;
  cost?: number;
  status?: MaintenanceStatus;
}

export interface MaintenanceListParams extends ListParams {
  branch_id?: number;
  equipment_id?: number;
  type?: MaintenanceType;
  status?: MaintenanceStatus;
  from?: string;
  to?: string;
}

export interface MaintenanceSummary {
  total_cost: number;
  total_count: number;
  open_count?: number;
  in_progress_count?: number;
  done_count?: number;
  by_branch?: { branch_id: number; branch_name: string; cost: number; count: number }[];
}

function normalizePage<T>(res: Page<T> | T[]): Page<T> {
  if (Array.isArray(res)) {
    return {
      items: res,
      total: res.length,
      total_page: 1,
      current_page: 1,
      limit: res.length || 20,
    };
  }
  return res;
}

class EquipmentApi {
  readonly key = 'equipments';
  readonly logKey = 'maintenance-logs';

  async list(params: EquipmentListParams = {}): Promise<Page<Equipment>> {
    const res = await http.get<Page<Equipment> | Equipment[]>('/admin/equipments', { params });
    return normalizePage(res);
  }

  get(id: number) {
    return http.get<Equipment>(`/admin/equipments/${id}`);
  }

  create(data: EquipmentInput) {
    return http.post<Equipment>('/admin/equipments', data);
  }

  update(id: number, data: EquipmentInput) {
    return http.put<Equipment>(`/admin/equipments/${id}`, data);
  }

  remove(id: number) {
    return http.delete(`/admin/equipments/${id}`);
  }

  reportIssue(id: number, description: string, cost?: number) {
    return http.post<MaintenanceLog>(`/admin/equipments/${id}/report-issue`, {
      description,
      cost: cost ?? 0,
    });
  }

  async listLogs(params: MaintenanceListParams = {}): Promise<Page<MaintenanceLog>> {
    const res = await http.get<Page<MaintenanceLog> | MaintenanceLog[]>(
      '/admin/maintenance-logs',
      { params },
    );
    return normalizePage(res);
  }

  createLog(data: MaintenanceLogInput) {
    return http.post<MaintenanceLog>('/admin/maintenance-logs', data);
  }

  updateLog(id: number, data: MaintenanceLogUpdateInput) {
    return http.put<MaintenanceLog>(`/admin/maintenance-logs/${id}`, data);
  }

  removeLog(id: number) {
    return http.delete(`/admin/maintenance-logs/${id}`);
  }

  summary(params: { branch_id?: number; from?: string; to?: string } = {}) {
    return http.get<MaintenanceSummary>('/admin/maintenance-logs/summary', { params });
  }
}

export const equipmentApi = new EquipmentApi();
export default equipmentApi;
