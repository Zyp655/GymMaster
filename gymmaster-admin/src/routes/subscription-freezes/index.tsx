import { useCallback, useMemo } from 'react';
import { Button, Card, Col, Row, Space, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Link } from 'react-router-dom';
import subscriptionApi from '@/api/subscription';
import type { Freeze, FreezeListParams, FreezeStatus } from '@/api/subscription';
import ActionBar from '@/components/ActionBar';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { FREEZE_STATUS_MAP, FREEZE_STATUS_OPTIONS } from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import FreezeActions from '@/routes/subscriptions/FreezeActions';
import { formatDate, formatDateTime } from '@/utils/format';

const TITLE = 'Duyệt bảo lưu';
const SUBTITLE = 'Yêu cầu bảo lưu gói tập: duyệt, từ chối, kết thúc sớm';

function FreezesDesktop() {
  const url = useUrlParams();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const urlBranch = url.get('branch_id');

  const table = useTableQuery<Freeze, FreezeListParams>({
    queryKey: [subscriptionApi.key, 'freezes'],
    queryFn: (p) => subscriptionApi.freezes(p),
    filterKeys: ['status', 'branch_id'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: { branch_id: urlBranch ? undefined : (branchId ?? undefined) },
  });

  const columns = useMemo<ColumnsType<Freeze>>(
    () => [
      {
        title: 'Hội viên',
        key: 'member',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <Link to={`/app/members/${r.member_id}`}>{r.member_name}</Link>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {r.member_code}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Gói',
        key: 'plan',
        render: (_, r) => (
          <Link to={`/app/subscriptions?sid=${r.subscription_id}`}>{r.plan_name}</Link>
        ),
      },
      { title: 'Chi nhánh', dataIndex: 'branch_name' },
      {
        title: 'Bắt đầu',
        dataIndex: 'start_date',
        sorter: true,
        className: 'text-nowrap',
        render: (v: string) => formatDate(v),
      },
      {
        title: 'Kết thúc',
        dataIndex: 'end_date',
        sorter: true,
        className: 'text-nowrap',
        render: (v: string) => formatDate(v),
      },
      { title: 'Số ngày', dataIndex: 'days', align: 'right' },
      {
        title: 'Lý do',
        dataIndex: 'reason',
        width: 220,
        onCell: () => ({ style: { minWidth: 160 } }),
        render: (v: string) =>
          v ? (
            <Typography.Text
              ellipsis={{ tooltip: v }}
              style={{ display: 'block', minWidth: 140, maxWidth: 220 }}
            >
              {v}
            </Typography.Text>
          ) : (
            '—'
          ),
      },
      {
        title: 'Người yêu cầu',
        key: 'req',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <span>{r.requested_by_name || '—'}</span>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {formatDateTime(r.created_at)}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        sorter: true,
        render: (v: string, r) => (
          <Space direction="vertical" size={0}>
            <StatusTag status={v} map={FREEZE_STATUS_MAP} />
            {r.reviewed_by_name && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {r.reviewed_by_name}
                {r.review_note ? `: ${r.review_note}` : ''}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        render: (_, r) => <FreezeActions freeze={r} />,
      },
    ],
    [],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle
            title={TITLE}
            subtitle={SUBTITLE}
            breadcrumb={[
              { title: 'Đăng ký gói tập', path: '/app/subscriptions' },
              { title: TITLE },
            ]}
          />
        </Col>
        <Col>
          <ActionBar onRefresh={() => table.refetch()} refreshing={table.isFetching}>
            <Button onClick={() => url.set({ status: 'PENDING' }, { resetPage: true })}>
              Chờ duyệt
            </Button>
          </ActionBar>
        </Col>
      </Row>
      <FilterBar
        searchPlaceholder="Tên, SĐT, mã hội viên"
        filters={[
          { name: 'status', placeholder: 'Trạng thái', options: FREEZE_STATUS_OPTIONS },
          {
            name: 'branch_id',
            placeholder: 'Chi nhánh',
            options: branches.map((b) => ({ label: b.name, value: b.id })),
          },
        ]}
      />
      <DataTable<Freeze> table={table} columns={columns} emptyText="Không có yêu cầu bảo lưu" />
    </Card>
  );
}

function FreezesMobile() {
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const fetchPage = useCallback(
    ({ page, limit, keyword, filters }: MobileFetchArgs) =>
      subscriptionApi.freezes({
        page,
        limit,
        keyword: keyword || undefined,
        order_by: 'id',
        order_dir: 'DESC',
        status: (filters.status as FreezeStatus) || undefined,
        branch_id: (filters.branch_id as number | undefined) ?? branchId ?? undefined,
      }),
    [branchId],
  );
  return (
    <MobileList<Freeze>
      title={TITLE}
      queryKey={[subscriptionApi.key, 'freezes', branchId]}
      fetchPage={fetchPage}
      searchPlaceholder="Tên, SĐT, mã hội viên"
      emptyText="Không có yêu cầu bảo lưu"
      filters={[
        { name: 'status', label: 'Trạng thái', options: FREEZE_STATUS_OPTIONS },
        {
          name: 'branch_id',
          label: 'Chi nhánh',
          options: branches.map((b) => ({ label: b.name, value: b.id })),
        },
      ]}
      row={{
        title: (r) => r.member_name,
        subtitle: (r) => `${r.plan_name} · ${formatDate(r.start_date)} – ${formatDate(r.end_date)}`,
        meta: (r) => [`${r.days} ngày`, r.branch_name, r.reason],
        tags: (r) => [
          { label: FREEZE_STATUS_MAP[r.status]?.label, color: FREEZE_STATUS_MAP[r.status]?.color },
        ],
      }}
      details={(r) => [
        ['Người yêu cầu', r.requested_by_name || '—'],
        ['Người duyệt', r.reviewed_by_name || '—'],
        ['Ghi chú duyệt', r.review_note || '—'],
        ['Thao tác', <FreezeActions key="a" freeze={r} size="middle" />],
      ]}
    />
  );
}

export default function SubscriptionFreezesPage() {
  return useIsMobile() ? <FreezesMobile /> : <FreezesDesktop />;
}
