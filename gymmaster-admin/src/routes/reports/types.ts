import type { CSSProperties } from 'react';
import type { ReportExportType, ReportGroupBy, RevenueType } from '@/api/report';
import { formatNumber } from '@/utils/format';

/** Bộ lọc chung truyền xuống từng tab báo cáo. */
export interface ReportFilters {
  branchId?: number;
  from: string;
  to: string;
  groupBy: ReportGroupBy;
}

/** Tham số API từ bộ lọc. */
export const rangeParams = (f: ReportFilters) => ({
  branch_id: f.branchId,
  from: f.from,
  to: f.to,
});

/** Bảng màu phân loại cố định (thứ tự không đổi – màu đi theo thực thể, không theo hạng). */
export const CATEGORICAL = [
  '#2a78d6',
  '#eb6834',
  '#1baf7a',
  '#eda100',
  '#6250d6',
  '#e87ba4',
  '#008300',
  '#e34948',
] as const;

/** Màu theo loại doanh thu – dùng chung dashboard + báo cáo. */
export const REVENUE_TYPES: {
  key: RevenueType;
  field: 'membership' | 'pt' | 'product' | 'fee' | 'other';
  label: string;
  color: string;
}[] = [
  { key: 'MEMBERSHIP', field: 'membership', label: 'Gói tập', color: '#2a78d6' },
  { key: 'PT', field: 'pt', label: 'Huấn luyện cá nhân', color: '#eb6834' },
  { key: 'PRODUCT', field: 'product', label: 'Sản phẩm', color: '#1baf7a' },
  { key: 'FEE', field: 'fee', label: 'Phí', color: '#eda100' },
  { key: 'OTHER', field: 'other', label: 'Khác', color: '#94a3b8' },
];

export const POSITIVE = '#12805C';
export const NEGATIVE = '#C0392B';
export const NAVY = '#0A3150';

/** Ramp tuần tự 1 hue (navy) cho heatmap: 0 → nhạt, max → đậm. */
const HEAT_RAMP = [
  '#F1F5F9',
  '#DCE8F5',
  '#B9D2EC',
  '#8DB4DF',
  '#5B91CC',
  '#2F6DB0',
  '#1A4F8A',
  '#0A3150',
];

export function heatColor(value: number, max: number): string {
  if (!value || max <= 0) return HEAT_RAMP[0];
  const idx = Math.min(
    HEAT_RAMP.length - 1,
    1 + Math.floor((value / max) * (HEAT_RAMP.length - 1.01)),
  );
  return HEAT_RAMP[idx];
}

/** Rút gọn tiền cho trục biểu đồ: 1.250.000 → "1,3 tr". */
export function shortMoney(v: number): string {
  const a = Math.abs(v);
  if (a >= 1e9) return `${(v / 1e9).toLocaleString('vi-VN', { maximumFractionDigits: 1 })} tỷ`;
  if (a >= 1e6) return `${(v / 1e6).toLocaleString('vi-VN', { maximumFractionDigits: 1 })} tr`;
  if (a >= 1e3) return `${(v / 1e3).toLocaleString('vi-VN', { maximumFractionDigits: 0 })}k`;
  return formatNumber(v);
}

export const BLOCK_STYLE: CSSProperties = {
  borderRadius: 10,
  border: '1px solid #ECEFF3',
  height: '100%',
};

export const EXPORT_LABEL: Record<ReportExportType, string> = {
  revenue: 'Doanh thu',
  members: 'Hội viên',
  checkins: 'Check-in',
  subscriptions: 'Gói tập',
  invoices: 'Hóa đơn',
};
