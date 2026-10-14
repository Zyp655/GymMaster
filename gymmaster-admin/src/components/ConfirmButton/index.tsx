import type { ReactNode } from 'react';
import { App, Button } from 'antd';
import type { ButtonProps } from 'antd';
import { ExclamationCircleFilled } from '@ant-design/icons';
import { usePermission } from '@/hooks/usePermission';
import type { PermInput } from '@/hooks/usePermission';

export interface ConfirmButtonProps extends Omit<ButtonProps, 'onClick'> {
  /** Tiêu đề hộp xác nhận, ví dụ "Xóa chi nhánh HN01?". */
  confirmTitle: ReactNode;
  confirmContent?: ReactNode;
  okText?: string;
  /** Được gọi khi người dùng bấm đồng ý. Trả Promise → nút OK hiển thị loading. */
  onConfirm: () => unknown;
  /** Ẩn nút nếu thiếu quyền. */
  perm?: PermInput;
}

/**
 * Nút có hộp xác nhận (mọi thao tác xóa/hủy/hoàn tiền PHẢI qua xác nhận).
 *   <ConfirmButton danger size="small" icon={<DeleteOutlined />} confirmTitle="Xóa vai trò?"
 *     onConfirm={() => deleteMutation.mutateAsync(id)} perm="role.delete">Xóa</ConfirmButton>
 */
export function ConfirmButton({
  confirmTitle,
  confirmContent,
  okText,
  onConfirm,
  perm,
  danger,
  ...rest
}: ConfirmButtonProps) {
  const { modal } = App.useApp();
  const { can } = usePermission();
  if (perm && !can(perm)) return null;

  return (
    <Button
      danger={danger}
      {...rest}
      onClick={(e) => {
        e.stopPropagation();
        modal.confirm({
          title: confirmTitle,
          icon: <ExclamationCircleFilled />,
          content: confirmContent,
          okText: okText ?? (danger ? 'Xóa' : 'Đồng ý'),
          okButtonProps: { danger },
          cancelText: 'Hủy',
          // Lỗi đã được nơi gọi hiển thị (showError) – nuốt để tránh unhandled rejection.
          onOk: async () => {
            try {
              await onConfirm();
            } catch {
              /* đã xử lý */
            }
          },
        });
      }}
    />
  );
}

export default ConfirmButton;
