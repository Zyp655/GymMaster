import type { StatusMeta } from '@/constants/status';

/** Trạng thái bán của gói tập / gói PT. */
export const SALE_STATUS_MAP: Record<string, StatusMeta> = {
  ACTIVE: { label: 'Đang bán', color: 'green' },
  INACTIVE: { label: 'Ngừng bán', color: 'default' },
};

export const SALE_STATUS_OPTIONS = Object.entries(SALE_STATUS_MAP).map(([value, m]) => ({
  value,
  label: m.label,
}));

export const PLAN_TYPE_META: Record<string, StatusMeta> = {
  TIME: { label: 'Theo thời gian', color: 'blue' },
  SESSION: { label: 'Theo buổi', color: 'purple' },
};

export const PLAN_TYPE_OPTIONS = Object.entries(PLAN_TYPE_META).map(([value, m]) => ({
  value,
  label: m.label,
}));

export const ACCESS_SCOPE_LABELS: Record<string, string> = {
  SINGLE_BRANCH: 'Một chi nhánh',
  ALL_BRANCHES: 'Mọi chi nhánh',
};
