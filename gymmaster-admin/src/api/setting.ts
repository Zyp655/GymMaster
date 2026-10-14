import { http } from '@/utils/api';

/** Cấu hình hệ thống – permission `setting.manage` (docs/api/catalog.md §4). */

export type SettingValue = number | number[];

export interface Setting {
  key: string;
  value: SettingValue;
  description: string;
  updated_at: string;
}

class SettingApi {
  readonly key = 'settings';

  list() {
    return http.get<Setting[]>('/admin/settings');
  }

  update(key: string, value: SettingValue) {
    return http.put<Setting>(`/admin/settings/${encodeURIComponent(key)}`, { value });
  }
}

export const settingApi = new SettingApi();
export default settingApi;
