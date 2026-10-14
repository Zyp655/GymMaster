import { useState } from 'react';
import { Button, Grid, Layout, Space } from 'antd';
import { MenuFoldOutlined, MenuOutlined, MenuUnfoldOutlined } from '@ant-design/icons';
import AppDrawer from './AppDrawer';
import BranchSwitcher from './BranchSwitcher';
import NotificationBell from './NotificationBell';
import UserMenu from './UserMenu';
import QuickSellButton from '@/routes/subscriptions/QuickSellButton';

export interface AppHeaderProps {
  collapsed: boolean;
  onToggle: () => void;
}

/** Header trắng: thu gọn menu / hamburger, chi nhánh làm việc, thông báo, tài khoản. */
export function AppHeader({ collapsed, onToggle }: AppHeaderProps) {
  const screens = Grid.useBreakpoint();
  const [drawer, setDrawer] = useState(false);
  const desktop = !!screens.lg;

  return (
    <Layout.Header className="app-header">
      <Space size={8} align="center" style={{ minWidth: 0 }}>
        {desktop ? (
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={onToggle}
            aria-label={collapsed ? 'Mở rộng menu' : 'Thu gọn menu'}
          />
        ) : (
          <Button
            type="text"
            icon={<MenuOutlined />}
            onClick={() => setDrawer(true)}
            aria-label="Mở menu"
          />
        )}
        {!desktop && !screens.sm ? null : <BranchSwitcher compact={!desktop} />}
      </Space>

      <Space size={screens.md ? 12 : 4} align="center">
        <QuickSellButton compact={!screens.md} />
        <NotificationBell />
        <UserMenu showName={!!screens.md} />
      </Space>

      {!desktop && <AppDrawer open={drawer} onClose={() => setDrawer(false)} />}
    </Layout.Header>
  );
}

export default AppHeader;
