import { Select } from 'antd';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { normalizeText } from '@/utils/format';

/** Giá trị đại diện "Tất cả chi nhánh" – antd Select cảnh báo khi option/value là `null`. */
const ALL = 0;

interface Props {
  /** null = tất cả chi nhánh; undefined = chưa chọn. */
  value: number | null | undefined;
  onChange: (v: number | null) => void;
}

/**
 * Ô lọc chi nhánh có lựa chọn "Tất cả chi nhánh" (cho user toàn hệ thống) – dùng cho lịch lớp, buổi PT, hoa hồng.
 * Dùng giá trị 0 thay cho `null` để tránh cảnh báo `value null` của antd Select.
 */
export default function BranchFilterSelect({ value, onChange }: Props) {
  const { branches, isGlobal, isLoading } = useBranchOptions();
  const options = [
    ...(isGlobal ? [{ label: 'Tất cả chi nhánh', value: ALL }] : []),
    ...branches.map((b) => ({
      label: b.status === 'INACTIVE' ? `${b.name} (ngừng hoạt động)` : b.name,
      value: b.id,
    })),
  ];
  return (
    <Select<number>
      showSearch
      loading={isLoading}
      placeholder="Chọn chi nhánh"
      options={options}
      value={value === null ? (isGlobal ? ALL : undefined) : (value ?? undefined)}
      onChange={(v) => onChange(v === ALL ? null : v)}
      filterOption={(input, option) =>
        normalizeText(String(option?.label ?? '')).includes(normalizeText(input))
      }
      style={{ width: '100%' }}
    />
  );
}
