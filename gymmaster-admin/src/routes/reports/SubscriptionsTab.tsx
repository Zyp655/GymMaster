import { useQuery } from '@tanstack/react-query';
import { Button, Card, Col, Progress, Result, Row, Table, Tag, Typography } from 'antd';
import reportApi from '@/api/report';
import type { SubscriptionEventCounts } from '@/api/report';
import MoneyText from '@/components/Money/MoneyText';
import StatCard, { SectionHeader } from '@/components/StatCard';
import { getErrorMessage } from '@/utils/errors';
import { formatNumber } from '@/utils/format';
import { ExportButton } from './shared';
import { rangeParams } from './types';
import type { ReportFilters } from './types';

const EVENT_ITEMS: { key: keyof SubscriptionEventCounts; label: string }[] = [
  { key: 'new', label: 'Đăng ký mới' },
  { key: 'renewed', label: 'Gia hạn' },
  { key: 'upgraded', label: 'Nâng cấp' },
  { key: 'transferred', label: 'Chuyển nhượng' },
  { key: 'frozen', label: 'Bảo lưu' },
  { key: 'expired', label: 'Hết hạn' },
  { key: 'cancelled', label: 'Hủy gói' },
];

export default function SubscriptionsTab(f: ReportFilters) {
  const query = useQuery({
    queryKey: [reportApi.key, 'subscriptions', f.branchId ?? null, f.from, f.to, f.groupBy],
    queryFn: () => reportApi.subscriptions({ ...rangeParams(f), group_by: f.groupBy }),
  });

  if (query.isError) {
    return (
      <Result
        status="warning"
        title="Không tải được báo cáo gói tập"
        subTitle={getErrorMessage(query.error)}
        extra={<Button onClick={() => void query.refetch()}>Thử lại</Button>}
      />
    );
  }

  const d = query.data;
  const counts = d?.counts;

  return (
    <div>
      <div className="rp-tab-head">
        <span className="rp-tab-head__note">
          Biến động vòng đời gói tập và doanh số theo từng gói tập trong kỳ.
        </span>
        <ExportButton type="subscriptions" branchId={f.branchId} from={f.from} to={f.to} />
      </div>

      <SectionHeader title="Biến động đăng ký gói tập trong kỳ" />
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        {EVENT_ITEMS.map((item) => (
          <Col key={item.key} xs={12} sm={8} md={6} lg={3}>
            <StatCard
              label={item.label}
              value={counts?.[item.key] ?? 0}
              loading={query.isLoading}
            />
          </Col>
        ))}
      </Row>

      <Card
        size="small"
        title="Doanh số theo gói tập"
        loading={query.isLoading}
        style={{ marginBottom: 16 }}
      >
        <Table
          size="small"
          rowKey="plan_id"
          pagination={false}
          dataSource={d?.by_plan ?? []}
          scroll={{ x: 'max-content' }}
          locale={{ emptyText: 'Chưa có gói tập nào bán ra trong kỳ' }}
          columns={[
            {
              title: 'Mã gói',
              dataIndex: 'plan_code',
              render: (v: string) => <span className="code-text">{v}</span>,
            },
            {
              title: 'Tên gói tập',
              dataIndex: 'plan_name',
              render: (v: string) => <b>{v}</b>,
            },
            {
              title: 'Loại',
              dataIndex: 'plan_type',
              render: (v: string) => (
                <Tag color={v === 'TIME' ? 'blue' : 'purple'}>
                  {v === 'TIME' ? 'Theo thời hạn' : 'Theo buổi'}
                </Tag>
              ),
            },
            {
              title: 'Tổng bán',
              dataIndex: 'sold',
              align: 'right',
              render: (v: number) => <b>{formatNumber(v)}</b>,
            },
            {
              title: 'Bán mới',
              dataIndex: 'new_sales',
              align: 'right',
              render: (v: number) => formatNumber(v),
            },
            {
              title: 'Nối tiếp (GH/NC/CN)',
              dataIndex: 'continued',
              align: 'right',
              render: (v: number) => formatNumber(v),
            },
            {
              title: 'Giá trị bán',
              dataIndex: 'sold_amount',
              align: 'right',
              render: (v: number) => <MoneyText value={v} />,
            },
            {
              title: 'Thực thu',
              dataIndex: 'collected',
              align: 'right',
              render: (v: number) => <MoneyText value={v} strong />,
            },
            {
              title: 'Tỷ trọng thu',
              dataIndex: 'pct',
              width: 150,
              render: (pct: number) => <Progress percent={pct} size="small" />,
            },
          ]}
        />
      </Card>

      <Card size="small" title="Biến động gói tập theo thời gian" loading={query.isLoading}>
        <Table
          size="small"
          rowKey="period"
          dataSource={d?.series ?? []}
          pagination={(d?.series.length ?? 0) > 15 ? { pageSize: 15, size: 'small' } : false}
          scroll={{ x: 'max-content' }}
          locale={{ emptyText: 'Chưa có dữ liệu' }}
          columns={[
            {
              title: 'Kỳ',
              dataIndex: 'label',
              render: (v: string) => <Typography.Text strong>{v}</Typography.Text>,
            },
            ...EVENT_ITEMS.map((item) => ({
              title: item.label,
              dataIndex: item.key,
              align: 'right' as const,
              render: (v: number) => formatNumber(v),
            })),
          ]}
        />
      </Card>
    </div>
  );
}
