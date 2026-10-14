import { Drawer } from 'antd';
import { Link } from 'react-router-dom';
import AppSidebar from './AppSidebar';

/** Menu dạng ngăn kéo cho màn hình nhỏ (< lg). */
export function AppDrawer({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <Drawer
      placement="left"
      open={open}
      onClose={onClose}
      closable={false}
      width={272}
      styles={{ body: { padding: 0 } }}
    >
      <Link to="/app/dashboard" className="app-logo" onClick={onClose}>
        <img src="/logo.svg" alt="GymMaster" style={{ height: 36 }} />
      </Link>
      <AppSidebar onNavigate={onClose} />
    </Drawer>
  );
}

export default AppDrawer;
