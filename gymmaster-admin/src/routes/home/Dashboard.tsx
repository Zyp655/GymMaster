import type { ReactNode } from 'react';
import { Card, Col, Empty, Row, Typography } from 'antd';
import {
  BankOutlined,
  CalendarOutlined,
  DollarOutlined,
  IdcardOutlined,
  LoginOutlined,
  SafetyCertificateOutlined,
  ShoppingCartOutlined,
  TeamOutlined,
  UserAddOutlined,
} from '@ant-design/icons';
import { Pie } from '@ant-design/charts';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import { useNavigate } from 'react-router-dom';
import branchApi from '@/api/branch';
import StatCard, { SectionHeader } from '@/components/StatCard';
import { CARD_STYLE, TONES } from '@/components/StatCard/tokens';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { useBranchOptions } from '@/hooks/useBranchOptions';

const { Title, Text } = Typography;

// 4 ô / hàng trên desktop, 2 trên tablet, 1 trên điện thoại (giống mẫu).
const QUARTER_COL = { xs: 24, sm: 12, lg: 6 };
const PENDING_CAPTION = 'Chờ module báo cáo';

interface Shortcut {
  label: string;
  path: string;
  icon: ReactNode;
  perm: string[];
}

const SHORTCUTS: Shortcut[] = [
  {
    label: 'Thêm hội viên',
    path: '/app/members',
    icon: <UserAddOutlined />,
    perm: [PERM.MEMBER_CREATE],
  },
  {
    label: 'Bán gói tập',
    path: '/app/subscriptions',
    icon: <IdcardOutlined />,
    perm: [PERM.SUBSCRIPTION_SELL],
  },
  { label: 'Check-in', path: '/app/checkins', icon: <LoginOutlined />, perm: [PERM.CHECKIN_SCAN] },
  {
    label: 'Bán hàng (POS)',
    path: '/app/pos',
    icon: <ShoppingCartOutlined />,
    perm: [PERM.POS_SELL],
  },
  {
    label: 'Lịch lớp',
    path: '/app/classes',
    icon: <CalendarOutlined />,
    perm: [PERM.CLASS_VIEW, PERM.BOOKING_MANAGE],
  },
  { label: 'Chi nhánh', path: '/app/branches', icon: <BankOutlined />, perm: [PERM.BRANCH_VIEW] },
  {
    label: 'Vai trò & quyền',
    path: '/app/roles',
    icon: <SafetyCertificateOutlined />,
    perm: [PERM.ROLE_VIEW],
  },
];

function greeting(): string {
  const h = dayjs().hour();
  if (h < 11) return 'Chào buổi sáng';
  if (h < 14) return 'Chào buổi trưa';
  if (h < 18) return 'Chào buổi chiều';
  return 'Chào buổi tối';
}

export default function Dashboard() {
  const { user, can } = usePermission();
  const navigate = useNavigate();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const canBranch = can(PERM.BRANCH_VIEW);

  // Đếm chi nhánh theo trạng thái (limit=1, chỉ lấy total).
  const branchStats = useQuery({
    queryKey: [branchApi.key, 'dashboard-stats'],
    enabled: canBranch,
    queryFn: async () => {
      const [all, active] = await Promise.all([
        branchApi.list({ limit: 1 }),
        branchApi.list({ limit: 1, status: 'ACTIVE' }),
      ]);
      return { total: all.total, active: active.total, inactive: all.total - active.total };
    },
  });

  const workingName = branchId
    ? (branches.find((b) => b.id === branchId)?.name ?? 'Chi nhánh đang chọn')
    : 'Toàn hệ thống';

  const shortcuts = SHORTCUTS.filter((s) => can(s.perm));
  const pieData = branchStats.data
    ? [
        { type: 'Đang hoạt động', value: branchStats.data.active },
        { type: 'Ngừng hoạt động', value: branchStats.data.inactive },
      ].filter((d) => d.value > 0)
    : [];

  return (
    <div style={{ padding: 20 }}>
      <div style={{ marginBottom: 24 }}>
        <Title level={3} style={{ margin: 0 }}>
          {greeting()}, {user?.full_name || user?.email} 👋
        </Title>
        <Text type="secondary">
          {dayjs().format('dddd, DD/MM/YYYY')} · Đang làm việc tại <b>{workingName}</b>
        </Text>
      </div>

      <SectionHeader title="Số liệu nhanh" />
      <Row gutter={[16, 16]} style={{ marginBottom: 28 }}>
        <Col {...QUARTER_COL}>
          <StatCard
            label="Chi nhánh"
            icon={<BankOutlined />}
            value={canBranch ? branchStats.data?.total : '—'}
            loading={canBranch && branchStats.isLoading}
            caption={
              canBranch && branchStats.data
                ? `${branchStats.data.active} đang hoạt động`
                : 'Không có quyền xem'
            }
            progress={
              branchStats.data && branchStats.data.total > 0
                ? (branchStats.data.active / branchStats.data.total) * 100
                : null
            }
            onClick={canBranch ? () => navigate('/app/branches') : undefined}
          />
        </Col>
        <Col {...QUARTER_COL}>
          <StatCard
            label="Hội viên đang hoạt động"
            icon={<TeamOutlined />}
            value={null}
            caption={PENDING_CAPTION}
          />
        </Col>
        <Col {...QUARTER_COL}>
          <StatCard
            label="Lượt check-in hôm nay"
            icon={<LoginOutlined />}
            value={null}
            caption={PENDING_CAPTION}
          />
        </Col>
        <Col {...QUARTER_COL}>
          <StatCard
            label="Doanh thu tháng này"
            icon={<DollarOutlined />}
            value={null}
            currency
            caption={PENDING_CAPTION}
          />
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={14}>
          <SectionHeader title="Lối tắt" />
          {shortcuts.length === 0 ? (
            <Card style={CARD_STYLE}>
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="Chưa có lối tắt phù hợp với quyền của bạn"
              />
            </Card>
          ) : (
            <Row gutter={[12, 12]}>
              {shortcuts.map((s) => (
                <Col key={s.path} xs={12} sm={8}>
                  <Card
                    hoverable
                    style={CARD_STYLE}
                    styles={{ body: { padding: 16 } }}
                    onClick={() => navigate(s.path)}
                  >
                    <div
                      style={{
                        width: 38,
                        height: 38,
                        borderRadius: 10,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        background: TONES.orange.bg,
                        color: TONES.orange.fg,
                        fontSize: 18,
                        marginBottom: 10,
                      }}
                    >
                      {s.icon}
                    </div>
                    <Text strong>{s.label}</Text>
                  </Card>
                </Col>
              ))}
            </Row>
          )}
        </Col>

        <Col xs={24} lg={10}>
          <SectionHeader title="Chi nhánh theo trạng thái" />
          <Card
            style={{ ...CARD_STYLE, height: 'auto' }}
            loading={canBranch && branchStats.isLoading}
          >
            {pieData.length === 0 ? (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={canBranch ? 'Chưa có chi nhánh' : 'Không có quyền xem chi nhánh'}
              />
            ) : (
              <Pie
                height={240}
                data={pieData}
                angleField="value"
                colorField="type"
                innerRadius={0.62}
                scale={{ color: { range: [TONES.navy.fg, '#CBD5E1'] } }}
                label={{ text: 'value', style: { fontWeight: 600 } }}
                legend={{ color: { position: 'bottom', layout: { justifyContent: 'center' } } }}
                tooltip={{ title: 'type' }}
              />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
}
