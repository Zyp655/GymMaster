import type { CSSProperties } from 'react';

/**
 * Hai tông màu mang nghĩa (theo mẫu strongbody_admin):
 *   navy  = màu nhấn duy nhất cho dữ liệu (thanh tỉ lệ)
 *   green = tích cực (tăng trưởng)
 */
// Giá trị là biến CSS (khai báo sáng/tối trong assets/css/custom.css) → tự đổi theo chế độ tối.
export const TONES = {
  navy: { fg: 'var(--gm-tone-navy-fg)', bg: 'var(--gm-tone-navy-bg)' },
  slate: { fg: 'var(--gm-tone-slate-fg)', bg: 'var(--gm-tone-slate-bg)' },
  green: { fg: 'var(--gm-tone-green-fg)', bg: 'var(--gm-tone-green-bg)' },
  orange: { fg: 'var(--gm-tone-orange-fg)', bg: 'var(--gm-tone-orange-bg)' },
} as const;

/** Màu chữ của ô KPI. */
export const INK = { value: 'var(--gm-ink)', label: 'var(--gm-text-secondary)' } as const;

export const CARD_STYLE: CSSProperties = {
  borderRadius: 10,
  border: '1px solid var(--gm-border)',
  height: '100%',
};
