import { Button, Card, Col, DatePicker, Row, Segmented, Space, Tabs } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useIsFetching, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import reportApi from '@/api/report';
import type { ReportExportType, ReportGroupBy } from '@/api/report';
import BranchSelect from '@/components/BranchSelect';
import PageTitle from '@/components/PageTitle';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { API_DATE_FORMAT, DATE_FORMAT, formatDate } from '@/utils/format';
import CheckinsTab from './CheckinsTab';
import ClassesTab from './ClassesTab';
import MembersTab from './MembersTab';
import RetentionTab from './RetentionTab';
import RevenueTab from './RevenueTab';
import SalesTab from './SalesTab';
import { ExportButton } from './shared';
import SubscriptionsTab from './SubscriptionsTab';
import TrainersTab from './TrainersTab';
import type { ReportFilters } from './types';
import './reports.css';

const TABS = [
  { key: 'revenue', label: 'Doanh thu', series: true },
  { key: 'members', label: 'Hội viên', series: true },
  { key: 'retention', label: 'Giữ chân', series: false },
  { key: 'checkins', label: 'Check-in', series: false },
  { key: 'classes', label: 'Lớp học', series: false },
  { key: 'trainers', label: 'HLV', series: false },
  { key: 'sales', label: 'Bán hàng', series: false },
  { key: 'subscriptions', label: 'Gói tập', series: true },
] as const;

type TabKey = (typeof TABS)[number]['key'];

const TAB_EXPORT_MAP: Partial<Record<TabKey, ReportExportType>> = {
  revenue: 'revenue',
  members: 'members',
  checkins: 'checkins',
  subscriptions: 'subscriptions',
  sales: 'invoices',
};

const today = () => dayjs();
const quarterStart = (d: Dayjs) => d.month(Math.floor(d.month() / 3) * 3).startOf('month');

const PRESETS: { label: string; value: [Dayjs, Dayjs] }[] = [
  { label: '7 ngày qua', value: [today().subtract(6, 'day'), today()] },
  { label: '30 ngày qua', value: [today().subtract(29, 'day'), today()] },
  { label: 'Tháng này', value: [today().startOf('month'), today()] },
  {
    label: 'Tháng trước',
    value: [
      today().subtract(1, 'month').startOf('month'),
      today().subtract(1, 'month').endOf('month'),
    ],
  },
  { label: 'Quý này', value: [quarterStart(today()), today()] },
  { label: 'Năm nay', value: [today().startOf('year'), today()] },
];

/** Gợi ý group_by hợp lý theo độ dài kỳ. */
function autoGroup(days: number): ReportGroupBy {
  if (days > 120) return 'month';
  if (days > 45) return 'week';
  return 'day';
}

export default function ReportsPage() {
  const url = useUrlParams();
  const qc = useQueryClient();
  const fetching = useIsFetching({ queryKey: [reportApi.key] });
  const { branchId: workingBranch } = useWorkingBranch();

  const tab = (TABS.find((t) => t.key === url.get('tab'))?.key ?? 'revenue') as TabKey;
  const from = url.get('from') || today().startOf('month').format(API_DATE_FORMAT);
  const to = url.get('to') || today().format(API_DATE_FORMAT);
  const days = dayjs(to).diff(dayjs(from), 'day') + 1;
  const rawGroup = url.get('group_by') as ReportGroupBy | undefined;
  const groupBy: ReportGroupBy =
    rawGroup && !(rawGroup === 'day' && days > 400) ? rawGroup : autoGroup(days);
  // Chi nhánh: URL (`branch_id`, `all`) > chi nhánh làm việc ở header.
  const urlBranch = url.get('branch_id');
  const branchId =
    urlBranch === 'all' ? undefined : urlBranch ? Number(urlBranch) : (workingBranch ?? undefined);

  const filters: ReportFilters = { branchId, from, to, groupBy };
  const showGroup = TABS.find((t) => t.key === tab)?.series;
  const exportType = TAB_EXPORT_MAP[tab] ?? 'revenue';

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle
            title="Báo cáo"
            subtitle={`Kỳ ${formatDate(from)} – ${formatDate(to)} · ${days} ngày`}
          />
        </Col>
        <Col>
          <Space wrap>
            <Button
              icon={<ReloadOutlined />}
              loading={fetching > 0}
              onClick={() => void qc.invalidateQueries({ queryKey: [reportApi.key] })}
            >
              Làm mới
            </Button>
            <ExportButton type={exportType} branchId={branchId} from={from} to={to} />
          </Space>
        </Col>
      </Row>

      <div className="rp-filter">
        <Row gutter={[12, 12]} align="middle">
          <Col xs={24} sm={12} md={7} lg={6}>
            <BranchSelect
              allowAll
              value={branchId ?? null}
              onChange={(v) => url.set({ branch_id: v === null ? 'all' : (v as number) })}
              placeholder="Chi nhánh"
            />
          </Col>
          <Col xs={24} sm={12} md={9} lg={7}>
            <DatePicker.RangePicker
              style={{ width: '100%' }}
              allowClear={false}
              format={DATE_FORMAT}
              value={[dayjs(from), dayjs(to)]}
              presets={PRESETS}
              disabledDate={(d) => d.isAfter(today(), 'day')}
              onChange={(v) => {
                if (!v?.[0] || !v[1]) return;
                const nextDays = v[1].diff(v[0], 'day') + 1;
                url.set({
                  from: v[0].format(API_DATE_FORMAT),
                  to: v[1].format(API_DATE_FORMAT),
                  group_by: rawGroup === 'day' && nextDays > 400 ? 'month' : rawGroup,
                });
              }}
            />
          </Col>
          {showGroup && (
            <Col xs={24} md="auto">
              <Segmented<ReportGroupBy>
                block
                value={groupBy}
                onChange={(v) => url.set({ group_by: v })}
                options={[
                  { label: 'Ngày', value: 'day', disabled: days > 400 },
                  { label: 'Tuần', value: 'week' },
                  { label: 'Tháng', value: 'month' },
                ]}
              />
            </Col>
          )}
        </Row>
      </div>

      <Tabs
        className="rp-tabs"
        activeKey={tab}
        onChange={(k) => url.set({ tab: k })}
        destroyOnHidden
        items={TABS.map((t) => ({ key: t.key, label: t.label }))}
      />
      {tab === 'revenue' && <RevenueTab {...filters} />}
      {tab === 'members' && <MembersTab {...filters} />}
      {tab === 'retention' && <RetentionTab {...filters} />}
      {tab === 'checkins' && <CheckinsTab {...filters} />}
      {tab === 'classes' && <ClassesTab {...filters} />}
      {tab === 'trainers' && <TrainersTab {...filters} />}
      {tab === 'sales' && <SalesTab {...filters} />}
      {tab === 'subscriptions' && <SubscriptionsTab {...filters} />}
    </Card>
  );
}
