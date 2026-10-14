import { Select } from 'antd';
import type { SelectProps } from 'antd';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { normalizeText } from '@/utils/format';

export interface BranchSelectProps extends Omit<
  SelectProps<number | number[] | null>,
  'options' | 'loading'
> {
  /** Thêm lựa chọn "Tất cả chi nhánh" (value = null) khi user làm việc toàn hệ thống. */
  allowAll?: boolean;
  /** Ẩn chi nhánh ngừng hoạt động. */
  activeOnly?: boolean;
}

/**
 * Ô chọn chi nhánh trong phạm vi người dùng được phép (theo `branch_ids`).
 *   <Form.Item name="branch_id" label="Chi nhánh"><BranchSelect /></Form.Item>
 */
export function BranchSelect({ allowAll, activeOnly, placeholder, ...rest }: BranchSelectProps) {
  const { branches, isGlobal, isLoading } = useBranchOptions();
  const list = activeOnly ? branches.filter((b) => b.status !== 'INACTIVE') : branches;

  const options: SelectProps['options'] = [
    ...(allowAll && isGlobal ? [{ label: 'Tất cả chi nhánh', value: null }] : []),
    ...list.map((b) => ({
      label: b.status === 'INACTIVE' ? `${b.name} (ngừng hoạt động)` : b.name,
      value: b.id,
    })),
  ];

  return (
    <Select<number | number[] | null>
      showSearch
      loading={isLoading}
      placeholder={placeholder ?? 'Chọn chi nhánh'}
      options={options}
      filterOption={(input, option) =>
        normalizeText(String(option?.label ?? '')).includes(normalizeText(input))
      }
      style={{ width: '100%' }}
      {...rest}
    />
  );
}

export default BranchSelect;
