import { Select } from 'antd';
import { EnvironmentOutlined } from '@ant-design/icons';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { normalizeText } from '@/utils/format';

const ALL = 0;

/** Ô chọn "chi nhánh làm việc" ở header – các trang danh sách lọc theo giá trị này. */
export function BranchSwitcher({ compact }: { compact?: boolean }) {
  const { branches, isGlobal, isLoading } = useBranchOptions();
  const { branchId, setBranchId } = useWorkingBranch();

  const options = [
    ...(isGlobal ? [{ label: 'Tất cả chi nhánh', value: ALL }] : []),
    ...branches.map((b) => ({ label: b.name, value: b.id })),
  ];

  return (
    <Select<number>
      className="branch-switcher"
      variant="filled"
      prefix={<EnvironmentOutlined style={{ color: '#F97316' }} />}
      value={branchId ?? (isGlobal ? ALL : undefined)}
      options={options}
      loading={isLoading}
      placeholder="Chọn chi nhánh"
      showSearch
      popupMatchSelectWidth={false}
      style={{ width: compact ? 170 : 240 }}
      disabled={!isGlobal && branches.length <= 1}
      filterOption={(input, option) =>
        normalizeText(String(option?.label ?? '')).includes(normalizeText(input))
      }
      onChange={(value) => setBranchId(value === ALL ? null : value)}
      aria-label="Chi nhánh làm việc"
    />
  );
}

export default BranchSwitcher;
