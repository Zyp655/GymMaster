import type { StatusMeta } from '@/constants/status';

/** Chức vụ nhân viên (staff_profile.position). */
export const POSITION_META: Record<string, StatusMeta> = {
  MANAGER: { label: 'Quản lý', color: 'geekblue' },
  RECEPTIONIST: { label: 'Lễ tân', color: 'cyan' },
  TRAINER: { label: 'Huấn luyện viên', color: 'orange' },
  OTHER: { label: 'Khác', color: 'default' },
};

export const POSITION_OPTIONS = Object.entries(POSITION_META).map(([value, m]) => ({
  value,
  label: m.label,
}));

export const GENDER_OPTIONS = [
  { label: 'Nam', value: 'MALE' },
  { label: 'Nữ', value: 'FEMALE' },
  { label: 'Khác', value: 'OTHER' },
];

export const TRAINER_LEVEL_META: Record<string, StatusMeta> = {
  JUNIOR: { label: 'Junior', color: 'default' },
  SENIOR: { label: 'Senior', color: 'blue' },
  MASTER: { label: 'Master', color: 'gold' },
};

export const TRAINER_LEVEL_OPTIONS = Object.entries(TRAINER_LEVEL_META).map(([value, m]) => ({
  value,
  label: m.label,
}));

/** Trạng thái tài khoản nhân viên. */
export const STAFF_STATUS_MAP: Record<string, StatusMeta> = {
  ACTIVE: { label: 'Đang làm việc', color: 'green' },
  LOCKED: { label: 'Đã khóa', color: 'red' },
};

export const STAFF_STATUS_OPTIONS = Object.entries(STAFF_STATUS_MAP).map(([value, m]) => ({
  value,
  label: m.label,
}));

export const SPECIALTY_SUGGESTIONS = [
  'Gym',
  'Yoga',
  'Pilates',
  'Giảm cân',
  'Tăng cơ',
  'Boxing',
  'Cardio',
  'Zumba',
  'Phục hồi chức năng',
  'Dinh dưỡng',
];
