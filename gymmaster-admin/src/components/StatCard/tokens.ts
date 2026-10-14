import type { CSSProperties } from 'react';

/**
 * Hai tông màu mang nghĩa (theo mẫu strongbody_admin):
 *   navy  = màu nhấn duy nhất cho dữ liệu (thanh tỉ lệ)
 *   green = tích cực (tăng trưởng)
 */
export const TONES = {
  navy: { fg: '#0A3150', bg: '#E9EFF5' },
  slate: { fg: '#4B5563', bg: '#F1F3F6' },
  green: { fg: '#12805C', bg: '#E5F6F0' },
  orange: { fg: '#C2410C', bg: '#FFF1E6' },
} as const;

export const CARD_STYLE: CSSProperties = {
  borderRadius: 10,
  border: '1px solid #ECEFF3',
  height: '100%',
};
