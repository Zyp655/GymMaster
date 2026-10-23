import type { ReactNode } from 'react';
import { Button, Card, Col, Empty, Row, Table, Tag, Typography } from 'antd';
import {
  BankOutlined,
  CalendarOutlined,
  DollarOutlined,
  IdcardOutlined,
  InboxOutlined,
  LoginOutlined,
  PauseCircleOutlined,
  SafetyCertificateOutlined,
  ScheduleOutlined,
  ShoppingCartOutlined,
  TeamOutlined,
  UserAddOutlined,
} from '@ant-design/icons';
import { Pie } from '@/components/Charts';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import { useNavigate } from 'react-router-dom';
import branchApi from '@/api/branch';
import checkinApi from '@/api/checkin';
import memberApi from '@/api/member';
import reportApi from '@/api/report';
import type { LowStockItem } from '@/api/report';
import HelpLabel from '@/components/HelpLabel';
import StatCard, { SectionHeader } from '@/components/StatCard';
import { CARD_STYLE, TONES } from '@/components/StatCard/tokens';
import { PERM } from '@/constants/permissions';
import { REVENUE_HELP } from '@/constants/revenueBasis';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { useThemeMode } from '@/theme';
import { formatMoney, formatNumber } from '@/utils/format';
import { summarizeBranches } from './dashboardStats';

/** Cùng định nghĩa với trang Mật độ phòng tập (API `checkins.present`). */
const PRESENT_HELP =
  'Số hội viên đang tập: lượt check-in hợp lệ trong 3 giờ gần nhất và chưa check-out ' +
  '(cùng cách tính với trang Mật độ phòng tập).';

const { Title, Text } = Typography;

// 4 ô / hàng trên desktop, 2 trên tablet, 1 trên điện thoại (giống mẫu).
const QUARTER_COL = { xs: 24, sm: 12, lg: 6 };
const NO_PERM_CAPTION = 'Không có quyền xem';

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
  { label: 'Check-in', path: '/app/checkin', icon: <LoginOutlined />, perm: [PERM.CHECKIN_SCAN] },
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
  const { darkMode } = useThemeMode();
  const navigate = useNavigate();
  const { branchId } = useWorkingBranch();
  const branchOptions = useBranchOptions();
  const { branches } = branchOptions;
  const canBranch = can(PERM.BRANCH_VIEW);
  const canReport = can(PERM.REPORT_VIEW);
  const allowedBranchIds = user?.branch_ids ?? [];
  const isGlobalUser = allowedBranchIds.length === 0;

  // Đếm chi nhánh theo trạng thái (limit=1, chỉ lấy total) – chỉ cho user toàn hệ thống.
  const globalBranchStats = useQuery({
    queryKey: [branchApi.key, 'dashboard-stats'],
    enabled: canBranch && isGlobalUser,
    queryFn: async () => {
      const [all, active] = await Promise.all([
        branchApi.list({ limit: 1 }),
        branchApi.list({ limit: 1, status: 'ACTIVE' }),
      ]);
      return { total: all.total, active: active.total, inactive: all.total - active.total };
    },
  });
  // User bị giới hạn chi nhánh → chỉ đếm các chi nhánh được phân công.
  // Không có branch.view → không thống kê (danh sách public không đủ dữ liệu trạng thái).
  const branchStats = !canBranch
    ? { data: undefined, isLoading: false }
    : isGlobalUser
      ? globalBranchStats
      : {
          data: branchOptions.data
            ? summarizeBranches(branchOptions.data, allowedBranchIds)
            : undefined,
          isLoading: branchOptions.isLoading,
        };

  const overview = useQuery({
    queryKey: [reportApi.key, 'overview', branchId ?? null],
    enabled: canReport,
    queryFn: () => reportApi.overview({ branch_id: branchId ?? undefined }),
  });

  const ov = overview.data;

  // Không có report.view → vẫn hiện số liệu theo quyền riêng (member.view / checkin.view).
  const canMember = can(PERM.MEMBER_VIEW);
  const canCheckin = can(PERM.CHECKIN_VIEW);
  const memberFallback = useQuery({
    queryKey: [memberApi.key, 'dashboard-active', branchId ?? null],
    enabled: !canReport && canMember,
    queryFn: () =>
      memberApi.list({ limit: 1, subscription_status: 'active', branch_id: branchId ?? undefined }),
  });
  const checkinFallback = useQuery({
    queryKey: [checkinApi.key, 'dashboard-today', branchId ?? null],
    enabled: !canReport && canCheckin,
    queryFn: () => {
      const today = dayjs().format('YYYY-MM-DD');
      // Chỉ đếm lượt được vào (giống báo cáo tổng quan), bỏ lượt bị từ chối.
      return checkinApi.list({
        limit: 1,
        result: 'ALLOWED',
        from: today,
        to: today,
        branch_id: branchId ?? undefined,
      });
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
                : NO_PERM_CAPTION
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
            value={canReport ? ov?.members.active : canMember ? memberFallback.data?.total : '—'}
            loading={canReport ? overview.isLoading : canMember && memberFallback.isLoading}
            caption={
              canReport && ov
                ? `+${formatNumber(ov.members.new_this_month)} mới tháng này · ${formatNumber(ov.members.expiring_7d)} sắp hết hạn`
                : canReport
                  ? undefined
                  : canMember
                    ? 'Có gói tập đang hiệu lực'
                    : NO_PERM_CAPTION
            }
            onClick={canMember ? () => navigate('/app/members') : undefined}
          />
        </Col>
        <Col {...QUARTER_COL}>
          <StatCard
            label="Lượt check-in hôm nay"
            icon={<LoginOutlined />}
            value={canReport ? ov?.checkins.today : canCheckin ? checkinFallback.data?.total : '—'}
            loading={canReport ? overview.isLoading : canCheckin && checkinFallback.isLoading}
            caption={
              canReport && ov ? (
                <HelpLabel help={PRESENT_HELP}>
                  {`${formatNumber(ov.checkins.present)} đang tập (3 giờ gần nhất)`}
                </HelpLabel>
              ) : canReport || canCheckin ? undefined : (
                NO_PERM_CAPTION
              )
            }
            onClick={canCheckin ? () => navigate('/app/checkins') : undefined}
          />
        </Col>
        <Col {...QUARTER_COL}>
          <StatCard
            label={
              <HelpLabel
                help={
                  <>
                    {REVENUE_HELP.cash}
                    <br />
                    {REVENUE_HELP.dashboardCompare}
                  </>
                }
              >
                Doanh thu tháng này
              </HelpLabel>
            }
            icon={<DollarOutlined />}
            value={canReport ? ov?.revenue.month : '—'}
            currency
            loading={canReport && overview.isLoading}
            delta={canReport ? ov?.revenue.change_pct : null}
            deltaLabel="% so với cùng kỳ tháng trước"
            caption={
              canReport && ov
                ? `Hôm nay: ${formatMoney(ov.revenue.today)}`
                : canReport
                  ? undefined
                  : NO_PERM_CAPTION
            }
            onClick={canReport ? () => navigate('/app/reports') : undefined}
          />
        </Col>
      </Row>

      {canReport && (
        <>
          <SectionHeader title="Vận hành hôm nay" />
          <Row gutter={[16, 16]} style={{ marginBottom: 28 }}>
            <Col xs={24} sm={8}>
              <StatCard
                label="Lớp học hôm nay"
                icon={<CalendarOutlined />}
                value={ov ? `${formatNumber(ov.classes.today)} buổi` : null}
                loading={overview.isLoading}
                progress={ov?.classes.fill_rate ?? null}
                caption={
                  ov
                    ? `Lấp đầy ${ov.classes.fill_rate}% (${formatNumber(ov.classes.booked)}/${formatNumber(ov.classes.capacity)} chỗ)`
                    : undefined
                }
                onClick={
                  can([PERM.CLASS_VIEW, PERM.BOOKING_MANAGE])
                    ? () => navigate('/app/classes')
                    : undefined
                }
              />
            </Col>
            <Col xs={24} sm={8}>
              <StatCard
                label="Buổi PT hôm nay"
                icon={<ScheduleOutlined />}
                value={ov ? `${formatNumber(ov.pt.completed)}/${formatNumber(ov.pt.today)}` : null}
                valueSuffix="hoàn thành"
                loading={overview.isLoading}
                progress={
                  ov && ov.pt.today > 0 ? Math.round((ov.pt.completed / ov.pt.today) * 100) : null
                }
                caption={ov ? `Tổng ${formatNumber(ov.pt.today)} buổi lên lịch hôm nay` : undefined}
                onClick={
                  can([PERM.PT_VIEW, PERM.PT_SCHEDULE])
                    ? () => navigate('/app/pt-sessions')
                    : undefined
                }
              />
            </Col>
            <Col xs={24} sm={8}>
              <StatCard
                label="Bảo lưu chờ duyệt"
                icon={<PauseCircleOutlined />}
                value={ov?.pending_freezes ?? 0}
                loading={overview.isLoading}
                caption="Bấm để xử lý yêu cầu bảo lưu"
                onClick={
                  can([PERM.SUBSCRIPTION_VIEW, PERM.SUBSCRIPTION_APPROVE_FREEZE])
                    ? () => navigate('/app/subscription-freezes')
                    : undefined
                }
              />
            </Col>
          </Row>
        </>
      )}

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

          {canReport && (
            <div style={{ marginTop: 24 }}>
              <SectionHeader
                title={
                  <span>
                    <InboxOutlined style={{ marginRight: 6 }} />
                    <HelpLabel help="Dòng tồn (sản phẩm đã nhập về chi nhánh) có số lượng NHỎ HƠN mức tồn tối thiểu; chỉ tính sản phẩm & chi nhánh đang hoạt động – cùng định nghĩa với màn Kho hàng.">
                      Cảnh báo tồn kho thấp ({ov?.low_stock_count ?? 0})
                    </HelpLabel>
                  </span>
                }
                extra={
                  can(PERM.INVENTORY_VIEW) ? (
                    <Button type="link" size="small" onClick={() => navigate('/app/inventory')}>
                      Xem kho hàng
                    </Button>
                  ) : null
                }
              />
              <Card style={CARD_STYLE} loading={overview.isLoading}>
                {!ov?.low_stock_items?.length ? (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description="Không có sản phẩm nào dưới mức tồn tối thiểu"
                  />
                ) : (
                  <Table<LowStockItem>
                    size="small"
                    rowKey={(r) => `${r.product_id}-${r.branch_id}`}
                    pagination={false}
                    dataSource={ov.low_stock_items}
                    columns={[
                      {
                        title: 'Sản phẩm',
                        key: 'name',
                        render: (_, r) => (
                          <span>
                            <b>{r.name}</b>{' '}
                            <Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
                              ({r.sku})
                            </Text>
                          </span>
                        ),
                      },
                      { title: 'Chi nhánh', dataIndex: 'branch_name' },
                      {
                        title: 'Tồn / Tối thiểu',
                        key: 'stock',
                        align: 'right',
                        render: (_, r) => (
                          <Tag color="orange">
                            {formatNumber(r.quantity)} / {formatNumber(r.min_stock)}
                          </Tag>
                        ),
                      },
                    ]}
                  />
                )}
              </Card>
            </div>
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
                theme={darkMode ? 'classicDark' : 'classic'}
                height={240}
                data={pieData}
                angleField="value"
                colorField="type"
                innerRadius={0.62}
                scale={{ color: { range: ['#1D6FB8', '#94A3B8'] } }}
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
