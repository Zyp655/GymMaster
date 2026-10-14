import { Tooltip } from 'antd';
import dayjs from 'dayjs';
import { DATETIME_FORMAT, formatDate } from '@/utils/format';

export interface DateTextProps {
  value: string | number | Date | null | undefined;
  /** Hiển thị cả giờ (DD/MM/YYYY HH:mm). */
  withTime?: boolean;
  /** Định dạng tùy chỉnh (dayjs). */
  format?: string;
  /** Hiện thời gian tương đối khi rê chuột ("3 giờ trước"). */
  relative?: boolean;
}

/** `<DateText value={row.created_at} withTime />` → "14/10/2026 08:30". */
export function DateText({ value, withTime, format, relative = true }: DateTextProps) {
  const text = formatDate(value, format ?? (withTime ? DATETIME_FORMAT : undefined));
  if (!value || !relative || !dayjs(value).isValid()) return <span>{text}</span>;
  return (
    <Tooltip title={dayjs(value).fromNow()}>
      <span style={{ whiteSpace: 'nowrap' }}>{text}</span>
    </Tooltip>
  );
}

export default DateText;
