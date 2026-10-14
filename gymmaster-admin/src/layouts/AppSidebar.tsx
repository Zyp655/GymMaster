import { useEffect, useMemo, useState } from 'react';
import { Menu } from 'antd';
import { useLocation, useNavigate } from 'react-router-dom';
import { buildMenuItems, findMenuTrail } from '@/menu';
import { usePermission } from '@/hooks/usePermission';

export interface AppSidebarProps {
  collapsed?: boolean;
  /** Gọi sau khi chọn một mục (Drawer dùng để tự đóng). */
  onNavigate?: () => void;
}

/** Menu điều hướng đã lọc theo quyền (dùng cho cả Sider và Drawer). */
export function AppSidebar({ collapsed, onNavigate }: AppSidebarProps) {
  const { can } = usePermission();
  const navigate = useNavigate();
  const { pathname } = useLocation();

  const items = useMemo(() => buildMenuItems((perm) => can(perm)), [can]);
  const trail = findMenuTrail(pathname);
  const selectedKey = trail.leaf?.key ?? pathname;

  const [openKeys, setOpenKeys] = useState<string[]>(trail.group ? [trail.group.key] : []);

  // Mở nhóm chứa trang hiện tại khi điều hướng.
  useEffect(() => {
    const groupKey = trail.group?.key;
    if (groupKey) setOpenKeys((keys) => (keys.includes(groupKey) ? keys : [...keys, groupKey]));
  }, [trail.group?.key]);

  return (
    <Menu
      mode="inline"
      className="app-menu"
      items={items}
      selectedKeys={[selectedKey]}
      {...(collapsed ? {} : { openKeys, onOpenChange: setOpenKeys })}
      onClick={({ key }) => {
        if (key.startsWith('/')) navigate(key);
        onNavigate?.();
      }}
    />
  );
}

export default AppSidebar;
