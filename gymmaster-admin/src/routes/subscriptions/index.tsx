import { useCallback, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button, Card, Col, DatePicker, Row, Space, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { EyeOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { Link } from 'react-router-dom';
import membershipPlanApi from '@/api/plan';
import subscriptionApi from '@/api/subscription';
import type { Subscription, SubscriptionListParams, SubscriptionStatus } from '@/api/subscription';
import ActionBar from '@/components/ActionBar';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import MoneyText from '@/components/Money/MoneyText';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import {
  INVOICE_STATUS_MAP,
  SUBSCRIPTION_STATUS_MAP,
  SUBSCRIPTION_STATUS_OPTIONS,
} from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { DATE_FORMAT, formatDate, formatMoney, formatPhone, toApiDate } from '@/utils/format';
import SellSubscriptionModal from './SellSubscriptionModal';
import SubscriptionDrawer from './SubscriptionDrawer';
import { remainingText } from './utils';

const TITLE = 'Đăng ký gói tập';
const SUBTITLE = 'Gói tập đã bán: vòng đời, gia hạn, nâng cấp, bảo lưu, chuyển nhượng';

const EXPIRING_OPTIONS = [3, 7, 15, 30].map((n) => ({
  value: n,
  label: `Hết hạn trong ${n} ngày`,
}));

function usePlanFilterOptions() {
  const q = useQuery({
    queryKey: [membershipPlanApi.key, 'filter-options'],
    // Lọc cần cả gói đã ngừng bán → dùng danh sách đầy đủ (cần plan.view), lỗi thì dùng options.
    queryFn: async () => {
      try {
        return (
          await membershipPlanApi.list({ limit: 100, order_by: 'sort_order', order_dir: 'ASC' })
        ).items;
      } catch {
        return membershipPlanApi.options();
      }
    },
    staleTime: 5 * 60_000,
  });
  return (q.data ?? []).map((p) => ({ value: p.id, label: `${p.code} – ${p.name}` }));
}

function SubscriptionsDesktop() {
  const url = useUrlParams();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const planOptions = usePlanFilterOptions();
  const [selling, setSelling] = useState(false);
  const viewId = Number(url.get('sid')) || null;
  const urlBranch = url.get('branch_id');

  const table = useTableQuery<Subscription, SubscriptionListParams>({
    queryKey: [subscriptionApi.key, 'list'],
    queryFn: (p) => subscriptionApi.list(p),
    filterKeys: ['status', 'plan_id', 'branch_id', 'expiring_in_days', 'from', 'to'],
    numberKeys: ['plan_id', 'branch_id', 'expiring_in_days'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: { branch_id: urlBranch ? undefined : (branchId ?? undefined) },
  });

  const open = useCallback((id: number) => url.set({ sid: id }), [url]);
  const from = url.get('from');
  const to = url.get('to');

  const columns = useMemo<ColumnsType<Subscription>>(
    () => [
      {
        title: 'Hội viên',
        key: 'member',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <Button type="link" size="small" style={{ padding: 0 }} onClick={() => open(r.id)}>
              {r.member_name}
            </Button>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {r.member_code} · {formatPhone(r.member_phone)}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Gói',
        key: 'plan',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <span>{r.plan_name}</span>
            <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
              {r.plan_code}
            </Typography.Text>
          </Space>
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
      {
        title: 'Còn lại',
        key: 'left',
        className: 'text-nowrap',
        render: (_, r) =>
          r.days_left !== null && r.days_left <= 7 && r.status !== 'PENDING' ? (
            <Typography.Text type="warning">{remainingText(r)}</Typography.Text>
          ) : (
            remainingText(r)
          ),
      },
      {
        title: 'Giá trị',
        dataIndex: 'price',
        sorter: true,
        align: 'right',
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        sorter: true,
        render: (v: string) => <StatusTag status={v} map={SUBSCRIPTION_STATUS_MAP} />,
      },
      {
        title: 'Hóa đơn',
        key: 'invoice',
        render: (_, r) =>
          r.invoice_id ? (
            <Space direction="vertical" size={0}>
              <Link to={`/app/invoices/${r.invoice_id}`} className="code-text">
                {r.invoice_no}
              </Link>
              <StatusTag status={r.invoice_status} map={INVOICE_STATUS_MAP} />
            </Space>
          ) : (
            '—'
          ),
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 56,
        render: (_, r) => (
          <Button
            type="text"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => open(r.id)}
            aria-label="Chi tiết"
          />
        ),
      },
    ],
    [open],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title={TITLE} subtitle={SUBTITLE} />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => setSelling(true)}
            addPerm={PERM.SUBSCRIPTION_SELL}
            addText="Bán gói"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          >
            <Link to="/app/subscription-freezes">
              <Button>Duyệt bảo lưu</Button>
            </Link>
          </ActionBar>
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tên, SĐT, mã hội viên, số HĐ..."
        colTextSearch={5}
        filters={[
          {
            name: 'status',
            placeholder: 'Trạng thái',
            options: SUBSCRIPTION_STATUS_OPTIONS,
            col: 3,
          },
          { name: 'plan_id', placeholder: 'Gói tập', options: planOptions, col: 3 },
          {
            name: 'branch_id',
            placeholder: 'Chi nhánh',
            options: branches.map((b) => ({ label: b.name, value: b.id })),
            col: 3,
          },
          {
            name: 'expiring_in_days',
            placeholder: 'Sắp hết hạn',
            options: EXPIRING_OPTIONS,
            col: 3,
          },
        ]}
      >
        <Col xs={24} sm={12} md={6}>
          <DatePicker.RangePicker
            style={{ width: '100%' }}
            format={DATE_FORMAT}
            placeholder={['Hết hạn từ', 'Đến ngày']}
            value={from || to ? [from ? dayjs(from) : null, to ? dayjs(to) : null] : null}
            allowEmpty={[true, true]}
            onChange={(v) =>
              url.set(
                { from: toApiDate(v?.[0]) ?? undefined, to: toApiDate(v?.[1]) ?? undefined },
                { resetPage: true },
              )
            }
          />
        </Col>
      </FilterBar>

      <DataTable<Subscription> table={table} columns={columns} emptyText="Chưa có gói tập nào" />

      <SellSubscriptionModal open={selling} onClose={() => setSelling(false)} />
      <SubscriptionDrawer id={viewId} onClose={() => url.set({ sid: undefined })} />
    </Card>
  );
}

function SubscriptionsMobile() {
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const planOptions = usePlanFilterOptions();
  const [selling, setSelling] = useState(false);
  const [viewId, setViewId] = useState<number | null>(null);

  const fetchPage = useCallback(
    ({ page, limit, keyword, filters }: MobileFetchArgs) => {
      const f = filters as Record<string, unknown>;
      const range = f.range as [Dayjs | null, Dayjs | null] | undefined;
      return subscriptionApi.list({
        page,
        limit,
        keyword: keyword || undefined,
        order_by: 'id',
        order_dir: 'DESC',
        status: (f.status as SubscriptionStatus) || undefined,
        plan_id: (f.plan_id as number) || undefined,
        branch_id: (f.branch_id as number | undefined) ?? branchId ?? undefined,
        expiring_in_days: (f.expiring_in_days as number) || undefined,
        from: toApiDate(range?.[0]) ?? undefined,
        to: toApiDate(range?.[1]) ?? undefined,
      });
    },
    [branchId],
  );

  return (
    <>
      <MobileList<Subscription>
        title={TITLE}
        queryKey={[subscriptionApi.key, 'list', branchId]}
        fetchPage={fetchPage}
        onAdd={() => setSelling(true)}
        addPerm={PERM.SUBSCRIPTION_SELL}
        onOpen={(r) => setViewId(r.id)}
        searchPlaceholder="Tên, SĐT, mã hội viên"
        emptyText="Chưa có gói tập nào"
        filters={[
          { name: 'status', label: 'Trạng thái', options: SUBSCRIPTION_STATUS_OPTIONS },
          { name: 'plan_id', label: 'Gói tập', options: planOptions },
          {
            name: 'branch_id',
            label: 'Chi nhánh',
            options: branches.map((b) => ({ label: b.name, value: b.id })),
          },
          { name: 'expiring_in_days', label: 'Sắp hết hạn', options: EXPIRING_OPTIONS },
          { name: 'range', label: 'Ngày hết hạn', type: 'daterange' },
        ]}
        row={{
          title: (r) => r.member_name,
          subtitle: (r) =>
            `${r.plan_name} · ${formatDate(r.start_date)} – ${formatDate(r.end_date)}`,
          meta: (r) => [r.member_code, remainingText(r), formatMoney(r.price)],
          tags: (r) => [
            {
              label: SUBSCRIPTION_STATUS_MAP[r.status]?.label ?? r.status,
              color: SUBSCRIPTION_STATUS_MAP[r.status]?.color,
            },
          ],
        }}
        actions={(r) => [
          {
            key: 'view',
            label: 'Xem chi tiết',
            icon: <EyeOutlined />,
            onClick: () => setViewId(r.id),
          },
        ]}
        details={(r) => [
          ['Chi nhánh', r.branch_name],
          ['Hóa đơn', r.invoice_no || '—'],
          ['Người bán', r.sold_by_name || '—'],
        ]}
      />
      <SellSubscriptionModal open={selling} onClose={() => setSelling(false)} />
      <SubscriptionDrawer id={viewId} onClose={() => setViewId(null)} />
    </>
  );
}

export default function SubscriptionsPage() {
  return useIsMobile() ? <SubscriptionsMobile /> : <SubscriptionsDesktop />;
}
