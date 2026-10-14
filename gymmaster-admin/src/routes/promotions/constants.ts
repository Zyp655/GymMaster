import type { Promotion } from '@/api/promotion';
import type { StatusMeta } from '@/constants/status';
import { formatMoney, formatNumber } from '@/utils/format';

export const VALIDITY_MAP: Record<string, StatusMeta> = {
  active: { label: 'Đang áp dụng', color: 'green' },
  upcoming: { label: 'Sắp diễn ra', color: 'blue' },
  expired: { label: 'Hết hạn', color: 'default' },
  exhausted: { label: 'Hết lượt', color: 'orange' },
  inactive: { label: 'Ngừng áp dụng', color: 'red' },
};

/** Bộ lọc hiệu lực (backend: `expired` gồm cả hết lượt). */
export const VALIDITY_OPTIONS = ['active', 'upcoming', 'expired', 'inactive'].map((value) => ({
  value,
  label: value === 'expired' ? 'Hết hạn / hết lượt' : VALIDITY_MAP[value].label,
}));

export const APPLIES_TO_LABELS: Record<string, string> = {
  ALL: 'Tất cả',
  MEMBERSHIP: 'Gói tập',
  PT: 'Gói PT',
  PRODUCT: 'Sản phẩm (POS)',
};

export const APPLIES_TO_OPTIONS = Object.entries(APPLIES_TO_LABELS).map(([value, label]) => ({
  value,
  label,
}));

/** Loại đơn khi kiểm tra mã. */
export const ORDER_TYPE_OPTIONS = APPLIES_TO_OPTIONS.filter((o) => o.value !== 'ALL');

export const discountText = (p: Pick<Promotion, 'type' | 'value' | 'max_discount'>) =>
  p.type === 'PERCENT'
    ? `Giảm ${formatNumber(p.value)}%${p.max_discount ? ` (tối đa ${formatMoney(p.max_discount)})` : ''}`
    : `Giảm ${formatMoney(p.value)}`;
