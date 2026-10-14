import { useState } from 'react';
import type { ReactNode } from 'react';
import { App } from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  KeyOutlined,
  LockOutlined,
  UnlockOutlined,
} from '@ant-design/icons';
import type { Staff } from '@/api/staff';
import { PERM, ROLE } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { getErrorMessage, showError } from '@/utils/errors';
import { useStaffAction } from './queries';

export interface StaffActionItem {
  key: string;
  label: ReactNode;
  icon: ReactNode;
  danger?: boolean;
  disabled?: boolean;
  onClick: (r: Staff) => void;
}

/**
 * Hành động trên một nhân viên (dùng chung desktop Dropdown + mobile action sheet):
 * sửa, khóa/mở khóa, đặt lại mật khẩu, xóa.
 */
export function useStaffActions(handlers: {
  onEdit: (r: Staff) => void;
  onResetPassword: (r: Staff) => void;
}) {
  const { can, isOwner, user } = usePermission();
  const { message, modal } = App.useApp();
  const action = useStaffAction();
  const [pendingId, setPendingId] = useState<number | null>(null);

  const run = async (r: Staff, kind: 'lock' | 'unlock' | 'delete') => {
    setPendingId(r.id);
    try {
      await action.mutateAsync({ id: r.id, action: kind });
      message.success(
        kind === 'lock'
          ? `Đã khóa tài khoản ${r.full_name}`
          : kind === 'unlock'
            ? `Đã mở khóa tài khoản ${r.full_name}`
            : `Đã xóa nhân viên ${r.full_name}`,
      );
    } catch (e) {
      if (kind === 'delete') {
        modal.error({ title: 'Không thể xóa nhân viên', content: getErrorMessage(e) });
      } else {
        showError(e, 'Không thể cập nhật trạng thái tài khoản');
      }
    } finally {
      setPendingId(null);
    }
  };

  const itemsFor = (r: Staff): StaffActionItem[] => {
    const isSelf = r.id === user?.id;
    const protectedOwner = !isOwner && r.roles.some((x) => x.role_code === ROLE.OWNER);
    const canUpdate = can(PERM.STAFF_UPDATE) && !protectedOwner;
    const items: StaffActionItem[] = [
      {
        key: 'edit',
        label: canUpdate ? 'Sửa thông tin' : 'Xem chi tiết',
        icon: canUpdate ? <EditOutlined /> : <EyeOutlined />,
        onClick: handlers.onEdit,
      },
    ];
    if (canUpdate) {
      items.push(
        r.status === 'LOCKED'
          ? {
              key: 'unlock',
              label: 'Mở khóa tài khoản',
              icon: <UnlockOutlined />,
              onClick: () => void run(r, 'unlock'),
            }
          : {
              key: 'lock',
              label: 'Khóa tài khoản',
              icon: <LockOutlined />,
              disabled: isSelf,
              onClick: () =>
                modal.confirm({
                  title: `Khóa tài khoản ${r.full_name}?`,
                  content:
                    'Nhân viên sẽ bị đăng xuất khỏi mọi thiết bị và không đăng nhập được cho tới khi mở khóa.',
                  okText: 'Khóa',
                  okButtonProps: { danger: true },
                  cancelText: 'Hủy',
                  onOk: () => run(r, 'lock'),
                }),
            },
        {
          key: 'reset',
          label: 'Đặt lại mật khẩu',
          icon: <KeyOutlined />,
          onClick: handlers.onResetPassword,
        },
      );
    }
    if (can(PERM.STAFF_DELETE) && !protectedOwner) {
      items.push({
        key: 'delete',
        label: 'Xóa nhân viên',
        icon: <DeleteOutlined />,
        danger: true,
        disabled: isSelf,
        onClick: () =>
          modal.confirm({
            title: `Xóa nhân viên ${r.full_name}?`,
            content:
              'Tài khoản bị vô hiệu, các ca làm chưa bắt đầu sẽ bị xóa. Không thể xóa nếu nhân viên còn lịch lớp hoặc buổi PT sắp diễn ra.',
            okText: 'Xóa',
            okButtonProps: { danger: true },
            cancelText: 'Hủy',
            onOk: () => run(r, 'delete'),
          }),
      });
    }
    return items;
  };

  return { itemsFor, pendingId };
}
