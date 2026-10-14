import { Card, Segmented, Tabs } from 'antd';
import PageTitle from '@/components/PageTitle';
import { useIsMobile } from '@/hooks/useIsMobile';
import { useUrlParams } from '@/hooks/useUrlParams';
import { MembershipPlansDesktop, MembershipPlansMobile } from './MembershipPlansTab';
import { PtPackagesDesktop, PtPackagesMobile } from './PtPackagesTab';

type TabKey = 'membership' | 'pt';

/** Gói tập & gói PT – hai tab, tab đang chọn lưu trên URL (`?tab=pt`). */
export default function PlansPage() {
  const isMobile = useIsMobile();
  const url = useUrlParams();
  const tab: TabKey = url.get('tab') === 'pt' ? 'pt' : 'membership';

  // Đổi tab → bỏ hết lọc/phân trang của tab cũ (hai bảng dùng chung tham số URL).
  const switchTab = (key: TabKey) => {
    const reset = Object.fromEntries(Object.keys(url.values).map((k) => [k, undefined]));
    url.set({ ...reset, tab: key === 'pt' ? 'pt' : undefined });
  };

  if (isMobile) {
    return (
      <>
        <div style={{ padding: '0 0 8px' }}>
          <Segmented<TabKey>
            block
            value={tab}
            onChange={switchTab}
            options={[
              { label: 'Gói tập', value: 'membership' },
              { label: 'Gói PT', value: 'pt' },
            ]}
          />
        </div>
        {tab === 'pt' ? <PtPackagesMobile /> : <MembershipPlansMobile />}
      </>
    );
  }

  return (
    <Card variant="borderless" className="page-card">
      <PageTitle title="Gói tập & gói PT" subtitle="Danh mục gói bán cho hội viên" />
      <Tabs
        activeKey={tab}
        onChange={(k) => switchTab(k as TabKey)}
        destroyOnHidden
        items={[
          { key: 'membership', label: 'Gói tập', children: <MembershipPlansDesktop /> },
          { key: 'pt', label: 'Gói PT', children: <PtPackagesDesktop /> },
        ]}
      />
    </Card>
  );
}
