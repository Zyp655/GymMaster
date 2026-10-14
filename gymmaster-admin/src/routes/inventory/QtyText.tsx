import { formatNumber } from '@/utils/format';

/** Số lượng có dấu: dương xanh (+), âm đỏ (−). */
export default function QtyText({ value, unit }: { value: number; unit?: string }) {
  const color = value > 0 ? '#12805C' : value < 0 ? '#cf1322' : undefined;
  const sign = value > 0 ? '+' : value < 0 ? '−' : '';
  return (
    <span style={{ color, fontWeight: 600, whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums' }}>
      {sign}
      {formatNumber(Math.abs(value))}
      {unit ? ` ${unit}` : ''}
    </span>
  );
}
