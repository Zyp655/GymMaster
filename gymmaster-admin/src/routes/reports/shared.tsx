import { useState } from 'react';
import type { CSSProperties, ReactNode } from 'react';
import { App, Button, Card, Dropdown, Empty, Result, Skeleton, Tooltip } from 'antd';
import { DownloadOutlined, DownOutlined } from '@ant-design/icons';
import reportApi, { saveFile } from '@/api/report';
import type { CheckinHeatmap, ReportExportType } from '@/api/report';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { getErrorMessage, showError } from '@/utils/errors';
import { formatNumber } from '@/utils/format';
import { BLOCK_STYLE, EXPORT_LABEL, heatColor } from './types';
import './reports.css';

export interface ChartCardProps {
  title: ReactNode;
  extra?: ReactNode;
  loading?: boolean;
  error?: unknown;
  onRetry?: () => void;
  empty?: boolean;
  emptyText?: string;
  height?: number;
  children?: ReactNode;
  bodyStyle?: CSSProperties;
}

/** Khối biểu đồ/bảng có đủ 3 trạng thái: Skeleton – lỗi + Thử lại – rỗng. */
export function ChartCard({
  title,
  extra,
  loading,
  error,
  onRetry,
  empty,
  emptyText = 'Chưa có dữ liệu trong kỳ',
  height = 280,
  children,
  bodyStyle,
}: ChartCardProps) {
  let content: ReactNode = children;
  if (loading) {
    content = (
      <div style={{ minHeight: height }}>
        <Skeleton active paragraph={{ rows: Math.max(3, Math.round(height / 60)) }} />
      </div>
    );
  } else if (error) {
    content = (
      <Result
        status="warning"
        title="Không tải được dữ liệu"
        subTitle={getErrorMessage(error)}
        extra={onRetry ? <Button onClick={onRetry}>Thử lại</Button> : undefined}
        style={{ padding: '24px 0' }}
      />
    );
  } else if (empty) {
    content = (
      <div style={{ minHeight: Math.min(height, 200), display: 'grid', placeItems: 'center' }}>
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={emptyText} />
      </div>
    );
  }
  return (
    <Card
      title={<span className="rp-card-title">{title}</span>}
      extra={extra}
      size="small"
      style={BLOCK_STYLE}
      styles={{ body: { padding: 16, ...bodyStyle } }}
    >
      {content}
    </Card>
  );
}

export interface ExportButtonProps {
  type: ReportExportType;
  branchId?: number;
  from: string;
  to: string;
}

/** Nút "Xuất Excel" (perm report.export): bấm = xuất loại của tab, mũi tên = chọn loại khác. */
export function ExportButton({ type, branchId, from, to }: ExportButtonProps) {
  const { can } = usePermission();
  const { message } = App.useApp();
  const [busy, setBusy] = useState<ReportExportType | null>(null);
  if (!can(PERM.REPORT_EXPORT)) return null;

  const run = async (t: ReportExportType) => {
    setBusy(t);
    try {
      const file = await reportApi.export({ type: t, branch_id: branchId, from, to });
      saveFile(file);
      message.success(`Đã tải ${file.filename}`);
    } catch (e) {
      showError(e, 'Không thể xuất báo cáo Excel');
    } finally {
      setBusy(null);
    }
  };

  return (
    <Dropdown.Button
      type="primary"
      icon={<DownOutlined />}
      loading={busy !== null}
      onClick={() => void run(type)}
      menu={{
        items: (Object.keys(EXPORT_LABEL) as ReportExportType[]).map((k) => ({
          key: k,
          label: `Báo cáo ${EXPORT_LABEL[k].toLowerCase()} (.xlsx)`,
        })),
        onClick: ({ key }) => void run(key as ReportExportType),
      }}
    >
      <DownloadOutlined /> Xuất Excel
    </Dropdown.Button>
  );
}

/** Heatmap 7 thứ × 24 giờ tự vẽ bằng CSS grid (tooltip từng ô, thang màu navy). */
export function HeatmapGrid({
  data,
  hourFrom = 0,
  hourTo = 23,
  compact = false,
}: {
  data: CheckinHeatmap;
  hourFrom?: number;
  hourTo?: number;
  compact?: boolean;
}) {
  const hours = Array.from({ length: hourTo - hourFrom + 1 }, (_, i) => hourFrom + i);
  const rows = [...data.matrix].sort((a, b) => a.dow - b.dow);
  return (
    <div className={compact ? 'rp-heat rp-heat--compact' : 'rp-heat'}>
      <div
        className="rp-heat__grid"
        style={{ gridTemplateColumns: `34px repeat(${hours.length}, minmax(0, 1fr))` }}
      >
        <span />
        {hours.map((h) => (
          <span key={h} className="rp-heat__hour">
            {h % (compact ? 3 : 2) === 0 ? h : ''}
          </span>
        ))}
        {rows.map((r) => (
          <HeatRow
            key={r.dow}
            label={r.label}
            values={hours.map((h) => r.hours[h] ?? 0)}
            hours={hours}
            max={data.max}
          />
        ))}
      </div>
      <div className="rp-heat__legend">
        <span>Ít</span>
        {[0, 0.15, 0.3, 0.45, 0.6, 0.75, 0.9, 1].map((f) => (
          <i key={f} style={{ background: heatColor(f * data.max, data.max) }} />
        ))}
        <span>Đông ({formatNumber(data.max)} lượt)</span>
      </div>
    </div>
  );
}

function HeatRow({
  label,
  values,
  hours,
  max,
}: {
  label: string;
  values: number[];
  hours: number[];
  max: number;
}) {
  return (
    <>
      <span className="rp-heat__dow">{label}</span>
      {values.map((v, i) => (
        <Tooltip
          key={hours[i]}
          title={`${label} ${String(hours[i]).padStart(2, '0')}:00 – ${formatNumber(v)} lượt`}
          mouseEnterDelay={0.05}
        >
          <span className="rp-heat__cell" style={{ background: heatColor(v, max) }} />
        </Tooltip>
      ))}
    </>
  );
}

export function SwatchLegend({
  items,
}: {
  items: { label: string; color: string; value?: ReactNode }[];
}) {
  return (
    <div className="rp-legend">
      {items.map((it) => (
        <div key={it.label} className="rp-legend__item">
          <i style={{ background: it.color }} />
          <span className="rp-legend__label">{it.label}</span>
          {it.value !== undefined && <span className="rp-legend__value">{it.value}</span>}
        </div>
      ))}
    </div>
  );
}

/** Mũi tên % thay đổi xanh/đỏ (null → "—"). */
export function DeltaBadge({
  pct,
  suffix = 'so với kỳ trước',
}: {
  pct: number | null | undefined;
  suffix?: string;
}) {
  if (pct === null || pct === undefined) {
    return <span className="rp-delta rp-delta--flat">— {suffix}</span>;
  }
  const up = pct > 0;
  const flat = pct === 0;
  return (
    <span className={`rp-delta ${flat ? 'rp-delta--flat' : up ? 'rp-delta--up' : 'rp-delta--down'}`}>
      {flat ? '→' : up ? '▲' : '▼'} {Math.abs(pct).toLocaleString('vi-VN')}%{' '}
      <span className="rp-delta__suffix">{suffix}</span>
    </span>
  );
}
