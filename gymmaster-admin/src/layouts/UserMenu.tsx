import { useState } from 'react';
import { App, Avatar, Button, Divider, Popover, Space, Tag, Typography } from 'antd';
import {
  ExclamationCircleFilled,
  KeyOutlined,
  LogoutOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import ChangePasswordModal from '@/components/ChangePasswordModal';
import { ROLE_LABELS } from '@/constants/permissions';
import { logout } from '@/store/authSlice';
import { useAppDispatch, useAppSelector } from '@/store';
import { queryClient } from '@/queryClient';

/** Avatar + popover: hồ sơ, đổi mật khẩu, đăng xuất. */
export function UserMenu({ showName }: { showName?: boolean }) {
  const user = useAppSelector((s) => s.auth.user);
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const { modal } = App.useApp();
  const [open, setOpen] = useState(false);
  const [pwdOpen, setPwdOpen] = useState(false);

  if (!user) return null;
  const name = user.full_name || user.email;
  const initial = name.charAt(0).toUpperCase();

  const onLogout = () => {
    setOpen(false);
    modal.confirm({
      title: 'Đăng xuất khỏi hệ thống?',
      icon: <ExclamationCircleFilled />,
      okText: 'Đăng xuất',
      cancelText: 'Hủy',
      onOk: async () => {
        await dispatch(logout());
        queryClient.clear();
        navigate('/login', { replace: true });
      },
    });
  };

  const content = (
    <div style={{ width: 260 }}>
      <Space align="start" size={12}>
        <Avatar size={44} src={user.avatar_url || undefined} className="user-avatar">
          {initial}
        </Avatar>
        <div style={{ minWidth: 0 }}>
          <Typography.Text strong ellipsis style={{ display: 'block', maxWidth: 190 }}>
            {name}
          </Typography.Text>
          <Typography.Text
            type="secondary"
            ellipsis
            style={{ fontSize: 12, display: 'block', maxWidth: 190 }}
          >
            {user.email}
          </Typography.Text>
          <div style={{ marginTop: 6 }}>
            {user.roles.map((r) => (
              <Tag key={r} color="blue" bordered={false} style={{ marginBottom: 4 }}>
                {ROLE_LABELS[r] ?? r}
              </Tag>
            ))}
          </div>
        </div>
      </Space>
      <Divider style={{ margin: '10px 0' }} />
      <div className="user-menu-actions">
        <Button
          type="text"
          block
          icon={<UserOutlined />}
          onClick={() => {
            setOpen(false);
            navigate('/app/profile');
          }}
        >
          Hồ sơ cá nhân
        </Button>
        <Button
          type="text"
          block
          icon={<KeyOutlined />}
          onClick={() => {
            setOpen(false);
            setPwdOpen(true);
          }}
        >
          Đổi mật khẩu
        </Button>
        <Button type="text" danger block icon={<LogoutOutlined />} onClick={onLogout}>
          Đăng xuất
        </Button>
      </div>
    </div>
  );

  return (
    <>
      <Popover
        content={content}
        trigger="click"
        placement="bottomRight"
        open={open}
        onOpenChange={setOpen}
        arrow={false}
      >
        <button type="button" className="user-trigger" aria-label="Tài khoản">
          <Avatar src={user.avatar_url || undefined} className="user-avatar">
            {initial}
          </Avatar>
          {showName && <span className="user-trigger__name">{name}</span>}
        </button>
      </Popover>
      <ChangePasswordModal open={pwdOpen} onClose={() => setPwdOpen(false)} />
    </>
  );
}

export default UserMenu;
