import type { ReactNode } from 'react';
import { App, Button, Space } from 'antd';
import {
  DeleteOutlined,
  ExclamationCircleFilled,
  PlusOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { usePermission } from '@/hooks/usePermission';
import type { PermInput } from '@/hooks/usePermission';

export interface ActionBarProps {
  /** Bấm "Thêm mới". */
  onAdd?: () => void;
  /** Xóa các mục đã chọn – được gọi SAU khi người dùng xác nhận. Có thể trả Promise. */
  onDelete?: () => unknown;
  onRefresh?: () => void;

  showAdd?: boolean;
  showDelete?: boolean;
  showRefresh?: boolean;

  /** Quyền cần có để thấy nút Thêm mới / Xóa (anyOf). */
  addPerm?: PermInput;
  deletePerm?: PermInput;

  addText?: ReactNode;
  /** Số mục đang chọn – nút Xóa bị khóa khi = 0. */
  selectedCount?: number;
  /** Nội dung xác nhận xóa (mặc định "Xóa N mục đã chọn?"). */
  deleteConfirmTitle?: ReactNode;
  deleteConfirmContent?: ReactNode;
  refreshing?: boolean;
  /** Nút bổ sung (xuất Excel, nhập file...). */
  children?: ReactNode;
}

/** Hàng nút hành động góc phải trang danh sách (theo mẫu strongbody_admin). */
export function ActionBar({
  onAdd,
  onDelete,
  onRefresh,
  showAdd = true,
  showDelete = true,
  showRefresh = true,
  addPerm,
  deletePerm,
  addText = 'Thêm mới',
  selectedCount = 0,
  deleteConfirmTitle,
  deleteConfirmContent = 'Thao tác này không thể hoàn tác.',
  refreshing,
  children,
}: ActionBarProps) {
  const { modal } = App.useApp();
  const { can } = usePermission();

  const confirmDelete = () => {
    modal.confirm({
      title: deleteConfirmTitle ?? `Xóa ${selectedCount} mục đã chọn?`,
      icon: <ExclamationCircleFilled />,
      content: deleteConfirmContent,
      okText: 'Xóa',
      okButtonProps: { danger: true },
      cancelText: 'Hủy',
      onOk: async () => {
        try {
          await onDelete?.();
        } catch {
          /* lỗi đã được trang hiển thị */
        }
      },
    });
  };

  return (
    <Space size={8} wrap className="action-bar">
      {showAdd && onAdd && can(addPerm) && (
        <Button color="orange" variant="solid" icon={<PlusOutlined />} onClick={onAdd}>
          {addText}
        </Button>
      )}
      {children}
      {showDelete && onDelete && can(deletePerm) && (
        <Button
          danger
          icon={<DeleteOutlined />}
          disabled={selectedCount === 0}
          onClick={confirmDelete}
        >
          {selectedCount > 0 ? `Xóa (${selectedCount})` : 'Xóa'}
        </Button>
      )}
      {showRefresh && onRefresh && (
        <Button icon={<ReloadOutlined spin={refreshing} />} onClick={onRefresh}>
          Làm mới
        </Button>
      )}
    </Space>
  );
}

export default ActionBar;
