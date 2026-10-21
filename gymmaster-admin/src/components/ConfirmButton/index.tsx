import type { ReactNode } from 'react';
import { App, Button } from 'antd';
import type { ButtonProps } from 'antd';
import { ExclamationCircleFilled } from '@ant-design/icons';
import { usePermission } from '@/hooks/usePermission';
import type { PermInput } from '@/hooks/usePermission';
import { resolveOkText } from './okText';

export interface ConfirmButtonProps extends Omit<ButtonProps, 'onClick'> {
  /** Tiêu đề hộp xác nhận, ví dụ "Xóa chi nhánh HN01?". */
  confirmTitle: ReactNode;
  confirmContent?: ReactNode;
  /** Chữ nút đồng ý. Mặc định: chữ trên nút (nếu là chuỗi, vd "Từ chối", "Giải phóng") → nếu không có thì
   *  "Xóa" (danger) / "Đồng ý". Nên truyền rõ cho thao tác danger KHÔNG phải xóa. */
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
  children,
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
          okText: resolveOkText(okText, children, danger, rest['aria-label']),
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
    >
      {children}
    </Button>
  );
}

export default ConfirmButton;
