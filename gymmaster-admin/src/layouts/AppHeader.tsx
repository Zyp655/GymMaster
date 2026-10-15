import { useEffect, useState } from 'react';
import {
  Button,
  Drawer,
  Grid,
  Input,
  Layout,
  List,
  Modal,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import {
  BulbOutlined,
  MenuFoldOutlined,
  MenuOutlined,
  MenuUnfoldOutlined,
  QuestionCircleOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { useLocation, useNavigate } from 'react-router-dom';
import { useThemeMode } from '@/App';
import { cmsSystemApi } from '@/api/cms-system';
import AppDrawer from './AppDrawer';
import BranchSwitcher from './BranchSwitcher';
import NotificationBell from './NotificationBell';
import UserMenu from './UserMenu';
import QuickSellButton from '@/routes/subscriptions/QuickSellButton';

export interface AppHeaderProps {
  collapsed: boolean;
  onToggle: () => void;
}

/** Header: thu gọn menu, chi nhánh, tìm kiếm nhanh Ctrl+K, trợ giúp ngữ cảnh, chế độ sáng/tối, thông báo, tài khoản. */
export function AppHeader({ collapsed, onToggle }: AppHeaderProps) {
  const screens = Grid.useBreakpoint();
  const navigate = useNavigate();
  const location = useLocation();
  const { darkMode, toggleDarkMode } = useThemeMode();

  const [drawer, setDrawer] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [helpOpen, setHelpOpen] = useState(false);
  const desktop = !!screens.lg;

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setSearchOpen((v) => !v);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  const searchQuery = useQuery({
    queryKey: ['admin-global-search', searchText],
    queryFn: () => cmsSystemApi.globalSearch(searchText.trim()),
    enabled: searchOpen && searchText.trim().length >= 1,
  });

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

      <Space size={screens.md ? 10 : 4} align="center">
        <Tooltip title="Tìm kiếm nhanh Hội viên, Hóa đơn, Sản phẩm (Ctrl + K)">
          <Button icon={<SearchOutlined />} onClick={() => setSearchOpen(true)}>
            {screens.md ? 'Tìm nhanh (Ctrl+K)' : null}
          </Button>
        </Tooltip>
        <QuickSellButton compact={!screens.md} />
        <Tooltip title={darkMode ? 'Chuyển sang Giao diện Sáng' : 'Chuyển sang Giao diện Tối'}>
          <Button
            type={darkMode ? 'primary' : 'text'}
            icon={<BulbOutlined />}
            onClick={toggleDarkMode}
            aria-label="Đổi chế độ Sáng/Tối"
          />
        </Tooltip>
        <Tooltip title="Hướng dẫn sử dụng màn hình này">
          <Button
            type="text"
            icon={<QuestionCircleOutlined />}
            onClick={() => setHelpOpen(true)}
            aria-label="Hướng dẫn sử dụng"
          />
        </Tooltip>
        <NotificationBell />
        <UserMenu showName={!!screens.md} />
      </Space>

      {!desktop && <AppDrawer open={drawer} onClose={() => setDrawer(false)} />}

      <Modal
        title="Tìm kiếm nhanh toàn hệ thống (Ctrl + K)"
        open={searchOpen}
        onCancel={() => setSearchOpen(false)}
        footer={null}
      >
        <Input
          autoFocus
          prefix={<SearchOutlined />}
          placeholder="Nhập tên hội viên, SĐT, mã GMxxxxxx, số hóa đơn HD..., hoặc tên sản phẩm..."
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          allowClear
        />
        <List
          style={{ marginTop: 12, maxHeight: 360, overflowY: 'auto' }}
          loading={searchQuery.isFetching}
          dataSource={searchQuery.data?.items ?? []}
          locale={{ emptyText: searchText ? 'Không tìm thấy kết quả phù hợp' : 'Nhập từ khóa để tra cứu' }}
          renderItem={(hit) => (
            <List.Item
              style={{ cursor: 'pointer' }}
              onClick={() => {
                setSearchOpen(false);
                navigate(hit.link);
              }}
            >
              <List.Item.Meta
                title={
                  <Space>
                    <Tag color="blue">{hit.category}</Tag>
                    <Typography.Text strong>{hit.title}</Typography.Text>
                  </Space>
                }
                description={hit.subtitle}
              />
            </List.Item>
          )}
        />
      </Modal>

      <Drawer
        title="Hướng dẫn Vận hành Nhanh"
        open={helpOpen}
        onClose={() => setHelpOpen(false)}
        width={420}
      >
        <Space direction="vertical" size={12}>
          <Typography.Paragraph>
            Đường dẫn hiện tại: <Tag color="orange">{location.pathname}</Tag>
          </Typography.Paragraph>
          <Typography.Title level={5}>Mẹo thao tác nhanh tại quầy</Typography.Title>
          <ul>
            <li>
              Nhấn <strong>Ctrl + K</strong> ở bất kỳ đâu để tìm nhanh hồ sơ hội viên theo SĐT, mã
              hội viên <code>GMxxxxxx</code> hoặc mã hóa đơn.
            </li>
            <li>
              Tại màn hình <strong>Check-in (/app/checkin)</strong> hoặc <strong>Kiosk (/kiosk)</strong>:
              hỗ trợ quét QR động 60s, vé tập thử <code>TP-...</code> và vé ngày/vé khách{' '}
              <code>DP-...</code>.
            </li>
            <li>
              Để chuyển đổi dữ liệu từ phần mềm cũ hoặc chạy <strong>Setup Wizard</strong>, truy
              cập mục <strong>Hệ thống → Thiết lập &amp; Chuyển đổi (/app/system-tools)</strong>.
            </li>
          </ul>
          <Button
            type="primary"
            block
            onClick={() => {
              setHelpOpen(false);
              navigate('/app/system-tools');
            }}
          >
            Mở Trung tâm Thiết lập &amp; Hướng dẫn Chi tiết
          </Button>
        </Space>
      </Drawer>
    </Layout.Header>
  );
}

export default AppHeader;
