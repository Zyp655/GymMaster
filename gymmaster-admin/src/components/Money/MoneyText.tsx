import type { CSSProperties } from 'react';
import { formatMoney } from '@/utils/format';

export interface MoneyTextProps {
  value: number | string | null | undefined;
  /** Hậu tố, mặc định " ₫". */
  suffix?: string;
  /** Tô màu đỏ khi âm, xanh khi dương. */
  colored?: boolean;
  strong?: boolean;
  style?: CSSProperties;
}

/** Hiển thị tiền VND: `<MoneyText value={1500000} />` → "1.500.000 ₫". */
export function MoneyText({ value, suffix = ' ₫', colored, strong, style }: MoneyTextProps) {
  const n = Number(value);
  const color =
    colored && Number.isFinite(n) ? (n < 0 ? '#cf1322' : n > 0 ? '#12805C' : undefined) : undefined;
  return (
    <span
      className="money-text"
      style={{ color, fontWeight: strong ? 600 : undefined, whiteSpace: 'nowrap', ...style }}
    >
      {formatMoney(value, suffix)}
    </span>
  );
}

export default MoneyText;
