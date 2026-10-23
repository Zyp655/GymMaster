import { useEffect, useId, useRef, useState } from 'react';
import type { KeyboardEvent as ReactKeyboardEvent } from 'react';
import {
  Button,
  Drawer,
  Grid,
  Input,
  Layout,
  Modal,
  Space,
  Spin,
  Tag,
  Tooltip,
  Typography,
  theme,
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
import type { InputRef } from 'antd';
import { useThemeMode } from '@/theme';
import { usePermission } from '@/hooks/usePermission';
import { routePerm } from '@/routes';
import AppDrawer from './AppDrawer';
import BranchSwitcher from './BranchSwitcher';
import NotificationBell from './NotificationBell';
import UserMenu from './UserMenu';
import {
  globalSearch,
  GLOBAL_SEARCH_DEBOUNCE_MS,
  nextActiveIndex,
  searchTerm,
} from './globalSearch';
import QuickSellButton from '@/routes/subscriptions/QuickSellButton';

/** Nhãn tiếng Việt cho loại kết quả tìm kiếm nhanh. */
const SEARCH_CATEGORY_LABEL: Record<string, string> = {
  MEMBER: 'Hội viên',
  INVOICE: 'Hóa đơn',
  PRODUCT: 'Sản phẩm',
  PLAN: 'Gói tập',
};

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
  const searchInputRef = useRef<InputRef>(null);
  const { can } = usePermission();
  // Liên kết "Thiết lập & Chuyển đổi" trong trợ giúp chỉ hiện khi có quyền vào route đó.
  const canSystemTools = can(routePerm('system-tools'));

  // Debounce từ khóa: chỉ gọi API khi ngừng gõ ~250ms.
  const [debouncedText, setDebouncedText] = useState('');
  useEffect(() => {
    const t = window.setTimeout(() => setDebouncedText(searchText), GLOBAL_SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(t);
  }, [searchText]);
  const term = searchTerm(debouncedText);
  const typingPending = searchTerm(searchText) !== term;

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setSearchOpen((v) => !v);
        return;
      }
      // "?" mở trợ giúp – bỏ qua khi đang gõ trong ô nhập/vùng soạn thảo.
      if (e.key === '?' && !e.ctrlKey && !e.metaKey && !e.altKey) {
        const el = e.target as HTMLElement | null;
        const typing =
          !!el &&
          (el.isContentEditable ||
            ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName) ||
            !!el.closest('[contenteditable="true"], [role="combobox"], [role="textbox"]'));
        if (typing) return;
        e.preventDefault();
        setHelpOpen((v) => !v);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  // Mỗi từ khóa một query key → kết quả cũ về muộn không đè kết quả mới; `signal` hủy request
  // của từ khóa cũ khi đổi từ khóa / đóng modal.
  const searchQuery = useQuery({
    queryKey: ['admin-global-search', term],
    queryFn: ({ signal }) => globalSearch(term!, signal),
    enabled: searchOpen && !!term,
    staleTime: 30_000,
    retry: false,
  });
  const searchHits = !term || searchQuery.isError ? [] : (searchQuery.data?.items ?? []);

  // Điều hướng bàn phím trong kết quả (combobox/listbox WAI-ARIA): ↑/↓ chọn (vòng tròn), Enter mở,
  // Esc đóng (Modal tự xử lý). Focus luôn ở ô nhập, mục đang chọn báo qua aria-activedescendant.
  const { token } = theme.useToken();
  const listboxId = useId();
  const optionId = (i: number) => `${listboxId}-opt-${i}`;
  const [activeIndex, setActiveIndex] = useState(-1);
  useEffect(() => {
    // Kết quả mới → chọn sẵn mục đầu để Enter mở ngay.
    setActiveIndex(searchHits.length ? 0 : -1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchQuery.data, term]);
  useEffect(() => {
    if (activeIndex < 0) return;
    document.getElementById(optionId(activeIndex))?.scrollIntoView({ block: 'nearest' });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeIndex]);
  const openHit = (hit: (typeof searchHits)[number]) => {
    setSearchOpen(false);
    navigate(hit.link);
  };
  const onSearchKeyDown = (e: ReactKeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      setActiveIndex((i) => nextActiveIndex(i, e.key, searchHits.length));
      return;
    }
    if (e.key === 'Enter' && !e.nativeEvent.isComposing) {
      const hit = searchHits[activeIndex];
      if (hit) {
        e.preventDefault();
        openHit(hit);
      }
    }
  };

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
        <Tooltip title="Hướng dẫn sử dụng màn hình này (phím ?)">
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
        afterOpenChange={(opened) => {
          // `autoFocus` chỉ có tác dụng lần mount đầu → focus lại mỗi lần mở modal.
          if (opened) searchInputRef.current?.focus({ cursor: 'all' });
        }}
        footer={null}
      >
        <Input
          ref={searchInputRef}
          autoFocus
          prefix={<SearchOutlined />}
          placeholder="Nhập tên hội viên, SĐT, mã GMxxxxxx, số hóa đơn HD..., hoặc tên sản phẩm..."
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          onKeyDown={onSearchKeyDown}
          allowClear
          role="combobox"
          aria-label="Từ khóa tìm kiếm nhanh"
          aria-autocomplete="list"
          aria-expanded={searchHits.length > 0}
          aria-controls={listboxId}
          aria-activedescendant={activeIndex >= 0 ? optionId(activeIndex) : undefined}
        />
        <Typography.Text type="secondary" style={{ display: 'block', marginTop: 6, fontSize: 12 }}>
          ↑/↓ chọn kết quả · Enter mở · Esc đóng
        </Typography.Text>
        <Spin spinning={!!term && (searchQuery.isFetching || typingPending)}>
          {searchHits.length ? (
            <div
              id={listboxId}
              role="listbox"
              aria-label="Kết quả tìm kiếm nhanh"
              style={{ marginTop: 8, maxHeight: 360, overflowY: 'auto' }}
            >
              {searchHits.map((hit, i) => (
                <div
                  key={`${hit.category}-${hit.id}`}
                  id={optionId(i)}
                  role="option"
                  aria-selected={i === activeIndex}
                  style={{
                    cursor: 'pointer',
                    padding: '10px 8px',
                    borderRadius: token.borderRadius,
                    borderBottom: `1px solid ${token.colorSplit}`,
                    background: i === activeIndex ? token.colorPrimaryBg : undefined,
                    boxShadow:
                      i === activeIndex ? `inset 3px 0 0 ${token.colorPrimary}` : undefined,
                  }}
                  onMouseEnter={() => setActiveIndex(i)}
                  onClick={() => openHit(hit)}
                >
                  <Space>
                    <Tag color="blue">{SEARCH_CATEGORY_LABEL[hit.category] ?? hit.category}</Tag>
                    <Typography.Text strong>{hit.title}</Typography.Text>
                  </Space>
                  {hit.subtitle ? (
                    <Typography.Text type="secondary" style={{ display: 'block', marginTop: 4 }}>
                      {hit.subtitle}
                    </Typography.Text>
                  ) : null}
                </div>
              ))}
            </div>
          ) : (
            <div
              id={listboxId}
              role="status"
              style={{ marginTop: 8, padding: '24px 0', textAlign: 'center' }}
            >
              {searchQuery.isError ? (
                <Space direction="vertical" size={4}>
                  <Typography.Text type="danger">
                    Không thể tìm kiếm lúc này. Vui lòng thử lại.
                  </Typography.Text>
                  <Button size="small" onClick={() => searchQuery.refetch()}>
                    Thử lại
                  </Button>
                </Space>
              ) : (
                <Typography.Text type="secondary">
                  {term ? 'Không tìm thấy kết quả phù hợp' : 'Nhập từ khóa để tra cứu'}
                </Typography.Text>
              )}
            </div>
          )}
        </Spin>
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
              Nhấn <strong>?</strong> (khi không gõ trong ô nhập) để mở/đóng bảng hướng dẫn này.
            </li>
            <li>
              Nhấn <strong>Ctrl + K</strong> ở bất kỳ đâu để tìm nhanh hồ sơ hội viên theo SĐT, mã
              hội viên <code>GMxxxxxx</code> hoặc mã hóa đơn.
            </li>
            <li>
              Tại màn hình <strong>Check-in (/app/checkin)</strong> hoặc{' '}
              <strong>Kiosk (/kiosk)</strong>: hỗ trợ quét QR động 60s, vé tập thử{' '}
              <code>TP-...</code> và vé ngày/vé khách <code>DP-...</code>.
            </li>
            {canSystemTools && (
              <li>
                Để chuyển đổi dữ liệu từ phần mềm cũ hoặc chạy <strong>Setup Wizard</strong>, truy
                cập mục <strong>Hệ thống → Thiết lập &amp; Chuyển đổi (/app/system-tools)</strong>.
              </li>
            )}
          </ul>
          {canSystemTools && (
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
          )}
        </Space>
      </Drawer>
    </Layout.Header>
  );
}

export default AppHeader;
