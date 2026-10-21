import type { CSSProperties, ReactNode } from 'react';
import { Card, Typography } from 'antd';
import { formatMoney, formatNumber } from '@/utils/format';
import { CARD_STYLE, INK, TONES } from './tokens';

const { Text } = Typography;

const STAT_BODY_STYLE: CSSProperties = {
  padding: '14px 16px',
  // Cho phép cỡ chữ số liệu co theo bề rộng thẻ (cqi) → số tiền dài không tràn/xuống dòng.
  containerType: 'inline-size',
  height: '100%',
  display: 'flex',
  flexDirection: 'column',
};

const VALUE_STYLE: CSSProperties = {
  display: 'flex',
  alignItems: 'baseline',
  flexWrap: 'wrap',
  gap: 5,
  flex: '1 1 auto',
  minWidth: 0,
  fontSize: 'clamp(15px, 9cqi, 22px)',
  fontWeight: 700,
  lineHeight: '28px',
  color: INK.value,
  fontVariantNumeric: 'tabular-nums',
};

const LABEL_STYLE: CSSProperties = {
  display: 'block',
  fontSize: 12,
  fontWeight: 500,
  color: INK.label,
  lineHeight: '16px',
};

const SECONDARY_STYLE: CSSProperties = {
  display: 'block',
  marginTop: 5,
  fontSize: 12,
  lineHeight: '16px',
  color: INK.label,
  fontVariantNumeric: 'tabular-nums',
};

const fmt = (value: number, currency?: boolean) =>
  currency ? formatMoney(value) : formatNumber(value);

export interface StatCardProps {
  label: ReactNode;
  /** Số (được định dạng) hoặc nội dung bất kỳ; `null`/`undefined` hiển thị "—". */
  value: ReactNode;
  currency?: boolean;
  valueSuffix?: ReactNode;
  /** Tổng toàn thời gian (dòng phụ). */
  total?: number | null;
  totalCurrency?: boolean;
  /** Chênh lệch so với kỳ trước (dương = xanh). */
  delta?: number | null;
  deltaCurrency?: boolean;
  deltaLabel?: ReactNode;
  caption?: ReactNode;
  /** 0–100 → thanh tỉ lệ. */
  progress?: number | null;
  icon?: ReactNode;
  loading?: boolean;
  onClick?: () => void;
}

/** Ô KPI: số trước, nhãn dưới, sau đó là delta/tỉ lệ tùy chọn. */
export function StatCard({
  label,
  value,
  currency = false,
  valueSuffix,
  total,
  totalCurrency = false,
  delta,
  deltaCurrency = false,
  deltaLabel,
  caption,
  progress,
  icon,
  loading = false,
  onClick,
}: StatCardProps) {
  const shown =
    value === null || value === undefined
      ? '—'
      : typeof value === 'number'
        ? fmt(value, currency)
        : value;
  const deltaNum = delta === null || delta === undefined ? null : Number(delta) || 0;

  return (
    <Card
      className="gm-stat-card"
      loading={loading}
      style={{ ...CARD_STYLE, cursor: onClick ? 'pointer' : undefined }}
      styles={{ body: STAT_BODY_STYLE }}
      hoverable={!!onClick}
      onClick={onClick}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8 }}>
        <div style={VALUE_STYLE}>
          {/* Giữ số và "₫" trên cùng một dòng. */}
          <span style={{ whiteSpace: 'nowrap' }}>{shown}</span>
          {valueSuffix ? (
            <span style={{ fontSize: 13, fontWeight: 500, color: INK.label }}>{valueSuffix}</span>
          ) : null}
        </div>
        {icon ? (
          <span
            className="gm-stat-card__icon"
            style={{
              width: 34,
              height: 34,
              borderRadius: 8,
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: TONES.navy.bg,
              color: TONES.navy.fg,
              fontSize: 17,
              flex: '0 0 auto',
            }}
          >
            {icon}
          </span>
        ) : null}
      </div>

      <Text style={LABEL_STYLE}>{label}</Text>

      {progress !== null && progress !== undefined ? (
        <div
          style={{
            marginTop: 10,
            height: 4,
            borderRadius: 999,
            background: TONES.slate.bg,
            overflow: 'hidden',
          }}
        >
          <div
            style={{
              width: `${Math.min(Math.max(Number(progress) || 0, 0), 100)}%`,
              height: '100%',
              borderRadius: 999,
              background: TONES.navy.fg,
            }}
          />
        </div>
      ) : null}

      {total !== null && total !== undefined ? (
        <Text style={SECONDARY_STYLE}>
          Tổng <span style={{ fontWeight: 600 }}>{fmt(total, totalCurrency)}</span>
        </Text>
      ) : null}

      {deltaNum !== null ? (
        <Text
          style={{
            ...SECONDARY_STYLE,
            fontWeight: 600,
            color: deltaNum > 0 ? TONES.green.fg : INK.label,
          }}
        >
          {deltaNum > 0 ? '+' : deltaNum < 0 ? '-' : ''}
          {fmt(Math.abs(deltaNum), deltaCurrency)}
          {deltaLabel ? <span style={{ fontWeight: 500 }}> {deltaLabel}</span> : null}
        </Text>
      ) : null}

      {caption ? <Text style={SECONDARY_STYLE}>{caption}</Text> : null}
    </Card>
  );
}

/** Tiêu đề nhóm trên dashboard. */
export function SectionHeader({ title, extra }: { title: ReactNode; extra?: ReactNode }) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 12,
        marginBottom: 12,
      }}
    >
      <Text style={{ fontSize: 15, fontWeight: 600, color: INK.value }}>{title}</Text>
      {extra}
    </div>
  );
}

export default StatCard;
