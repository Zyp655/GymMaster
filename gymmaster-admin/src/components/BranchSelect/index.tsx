import { Select } from 'antd';
import type { SelectProps } from 'antd';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { normalizeText } from '@/utils/format';
import { ALL_BRANCHES, fromSelectValue, toSelectValue } from './value';

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
export function BranchSelect({
  allowAll,
  activeOnly,
  placeholder,
  value,
  onChange,
  ...rest
}: BranchSelectProps) {
  const { branches, isGlobal, isLoading } = useBranchOptions();
  const list = activeOnly ? branches.filter((b) => b.status !== 'INACTIVE') : branches;
  // "Tất cả chi nhánh" (value null) chỉ có khi allowAll và user làm việc toàn hệ thống.
  const showAll = !!allowAll && isGlobal;

  const options: SelectProps['options'] = [
    ...(showAll ? [{ label: 'Tất cả chi nhánh', value: ALL_BRANCHES }] : []),
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
      value={toSelectValue(value, showAll)}
      onChange={(v, option) =>
        onChange?.(fromSelectValue(v, showAll) as number | number[] | null, option)
      }
    />
  );
}

export default BranchSelect;
