import { Tag } from 'antd';
import { STATUS_PRESETS } from '@/constants/status';
import type { StatusMeta } from '@/constants/status';

export interface StatusTagProps {
  status: string | null | undefined;
  /** Ghi đè/bổ sung nhãn + màu cho module cụ thể. */
  map?: Record<string, StatusMeta>;
}

/** `<StatusTag status="ACTIVE" />` → tag xanh "Đang hoạt động". */
export function StatusTag({ status, map }: StatusTagProps) {
  if (!status) return <>—</>;
  const meta = map?.[status] ?? STATUS_PRESETS[status];
  return (
    <Tag color={meta?.color ?? 'default'} bordered={false}>
      {meta?.label ?? status}
    </Tag>
  );
}

export default StatusTag;
