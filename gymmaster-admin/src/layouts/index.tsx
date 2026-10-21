import { Suspense, useState } from 'react';
import { Grid, Layout } from 'antd';
import { Link, Outlet, useLocation } from 'react-router-dom';
import ErrorBoundary from '@/components/ErrorBoundary';
import { RouteFallback } from '@/components/LoadingScreen';
import AppFooter from './AppFooter';
import AppHeader from './AppHeader';
import AppSidebar from './AppSidebar';
import BranchSwitcher from './BranchSwitcher';

const COLLAPSE_KEY = 'gm_sider_collapsed';

function readCollapsed(): boolean {
  try {
    return localStorage.getItem(COLLAPSE_KEY) === '1';
  } catch {
    return false;
  }
}

/**
 * Bố cục khu vực quản trị (theo mẫu strongbody_admin): Sider trắng thu gọn được (≥ lg),
 * Header, Content margin 16 nền trắng, Footer. Dưới lg: ẩn Sider, dùng AppDrawer.
 */
export function AppLayout() {
  const screens = Grid.useBreakpoint();
  const { pathname } = useLocation();
  const [collapsed, setCollapsed] = useState(readCollapsed);
  const desktop = !!screens.lg;

  const toggle = () =>
    setCollapsed((c) => {
      try {
        localStorage.setItem(COLLAPSE_KEY, c ? '0' : '1');
      } catch {
        /* bỏ qua */
      }
      return !c;
    });

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {desktop && (
        <Layout.Sider
          className="app-sider"
          theme="light"
          width={256}
          collapsedWidth={72}
          collapsed={collapsed}
          trigger={null}
          collapsible
        >
          <Link to="/app/dashboard" className="app-logo">
            {collapsed ? (
              <img src="/logo-icon.svg" alt="GymMaster" style={{ height: 34 }} />
            ) : (
              <>
                <img
                  className="app-logo__img--light"
                  src="/logo.svg"
                  alt="GymMaster"
                  style={{ height: 36 }}
                />
                <img
                  className="app-logo__img--dark"
                  src="/logo-dark.svg"
                  alt="GymMaster"
                  style={{ height: 36 }}
                />
              </>
            )}
          </Link>
          <AppSidebar collapsed={collapsed} />
        </Layout.Sider>
      )}

      <Layout>
        <AppHeader collapsed={collapsed} onToggle={toggle} />
        {!screens.sm && Object.keys(screens).length > 0 && (
          <div className="mobile-branch-bar">
            <BranchSwitcher compact />
          </div>
        )}
        <Layout.Content className="app-content">
          <div className="site-layout-background">
            <ErrorBoundary resetKey={pathname}>
              <Suspense fallback={<RouteFallback />}>
                <Outlet />
              </Suspense>
            </ErrorBoundary>
          </div>
        </Layout.Content>
        <AppFooter />
      </Layout>
    </Layout>
  );
}

export default AppLayout;
