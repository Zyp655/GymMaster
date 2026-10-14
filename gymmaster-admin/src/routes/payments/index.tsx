import { useCallback, useMemo } from 'react';
import type { ReactNode } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Card, Col, DatePicker, Row, Space, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  BankOutlined,
  CreditCardOutlined,
  DollarOutlined,
  WalletOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { Link } from 'react-router-dom';
import invoiceApi, { PAYMENT_METHOD_LABEL, PAYMENT_METHOD_OPTIONS } from '@/api/invoice';
import type {
  PaymentItem,
  PaymentListParams,
  PaymentListResult,
  PaymentMethod,
} from '@/api/invoice';
import staffApi from '@/api/staff';
import ActionBar from '@/components/ActionBar';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import MoneyText from '@/components/Money/MoneyText';
import PageTitle from '@/components/PageTitle';
import StatCard from '@/components/StatCard';
import { PERM } from '@/constants/permissions';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { DATE_FORMAT, formatDateTime, formatMoney, formatNumber, toApiDate } from '@/utils/format';

const TITLE = 'Sổ thu tiền';
const SUBTITLE = 'Các lần thu tiền theo hóa đơn – tổng hợp theo phương thức';

const METHOD_ICON: Record<PaymentMethod, ReactNode> = {
  CASH: <WalletOutlined />,
  TRANSFER: <BankOutlined />,
  CARD: <CreditCardOutlined />,
};

function useStaffOptions() {
  const { can } = usePermission();
  const q = useQuery({
    queryKey: [staffApi.key, 'options', 'payments'],
    queryFn: () => staffApi.options(),
    enabled: can(PERM.STAFF_VIEW),
    staleTime: 5 * 60_000,
  });
  return (q.data ?? []).map((s) => ({ value: s.id, label: `${s.full_name} (${s.staff_code})` }));
}

function SummaryCards({ data, loading }: { data?: PaymentListResult; loading?: boolean }) {
  const s = data?.summary;
  return (
    <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
      <Col xs={12} md={6}>
        <StatCard
          label="Tổng đã thu"
          value={s?.total_amount ?? 0}
          currency
          caption={`${formatNumber(s?.count ?? 0)} lần thu`}
          icon={<DollarOutlined />}
          loading={loading}
        />
      </Col>
      {(['CASH', 'TRANSFER', 'CARD'] as PaymentMethod[]).map((m) => {
        const row = s?.by_method.find((x) => x.method === m);
        return (
          <Col xs={12} md={6} key={m}>
            <StatCard
              label={PAYMENT_METHOD_LABEL[m]}
              value={row?.amount ?? 0}
              currency
              caption={`${formatNumber(row?.count ?? 0)} lần`}
              icon={METHOD_ICON[m]}
              progress={s?.total_amount ? ((row?.amount ?? 0) / s.total_amount) * 100 : null}
              loading={loading}
            />
          </Col>
        );
      })}
    </Row>
  );
}

function PaymentsDesktop() {
  const url = useUrlParams();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const staffOptions = useStaffOptions();
  const urlBranch = url.get('branch_id');

  const table = useTableQuery<PaymentItem, PaymentListParams>({
    queryKey: [invoiceApi.key, 'payments'],
    queryFn: (p) => invoiceApi.payments(p),
    filterKeys: ['branch_id', 'method', 'from', 'to', 'received_by'],
    numberKeys: ['branch_id', 'received_by'],
    defaults: { order_by: 'paid_at', order_dir: 'DESC' },
    extraParams: { branch_id: urlBranch ? undefined : (branchId ?? undefined) },
  });
  const from = url.get('from');
  const to = url.get('to');

  const columns = useMemo<ColumnsType<PaymentItem>>(
    () => [
      {
        title: 'Thời gian',
        dataIndex: 'paid_at',
        sorter: true,
        className: 'text-nowrap',
        render: (v: string) => formatDateTime(v),
      },
      {
        title: 'Hóa đơn',
        dataIndex: 'invoice_no',
        render: (v: string, r) => (
          <Link to={`/app/invoices/${r.invoice_id}`} className="code-text">
            {v}
          </Link>
        ),
      },
      {
        title: 'Hội viên',
        key: 'member',
        render: (_, r) =>
          r.member_id ? (
            <Space direction="vertical" size={0}>
              <Link to={`/app/members/${r.member_id}`}>{r.member_name}</Link>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {r.member_code}
              </Typography.Text>
            </Space>
          ) : (
            <Typography.Text type="secondary">Khách lẻ</Typography.Text>
          ),
      },
      { title: 'Chi nhánh', dataIndex: 'branch_name' },
      {
        title: 'Số tiền',
        dataIndex: 'amount',
        sorter: true,
        align: 'right',
        render: (v: number) => <MoneyText value={v} strong />,
      },
      {
        title: 'Phương thức',
        dataIndex: 'method',
        sorter: true,
        render: (v: PaymentMethod) => PAYMENT_METHOD_LABEL[v] ?? v,
      },
      { title: 'Mã GD', dataIndex: 'reference', render: (v: string) => v || '—' },
      { title: 'Người thu', dataIndex: 'received_by_name', render: (v: string) => v || '—' },
      { title: 'Ghi chú', dataIndex: 'note', render: (v: string) => v || '—' },
    ],
    [],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title={TITLE} subtitle={SUBTITLE} />
        </Col>
        <Col>
          <ActionBar onRefresh={() => table.refetch()} refreshing={table.isFetching} />
        </Col>
      </Row>
      <SummaryCards
        data={table.query.data as PaymentListResult | undefined}
        loading={table.isLoading}
      />
      <FilterBar
        searchPlaceholder="Số HĐ, hội viên, mã GD"
        colTextSearch={5}
        filters={[
          {
            name: 'branch_id',
            placeholder: 'Chi nhánh',
            options: branches.map((b) => ({ label: b.name, value: b.id })),
          },
          { name: 'method', placeholder: 'Phương thức', options: PAYMENT_METHOD_OPTIONS, col: 3 },
          ...(staffOptions.length
            ? [{ name: 'received_by', placeholder: 'Người thu', options: staffOptions }]
            : []),
        ]}
      >
        <Col xs={24} sm={12} md={6}>
          <DatePicker.RangePicker
            style={{ width: '100%' }}
            format={DATE_FORMAT}
            placeholder={['Thu từ ngày', 'Đến ngày']}
            value={from || to ? [from ? dayjs(from) : null, to ? dayjs(to) : null] : null}
            allowEmpty={[true, true]}
            presets={[
              { label: 'Hôm nay', value: [dayjs(), dayjs()] },
              { label: '7 ngày qua', value: [dayjs().subtract(6, 'day'), dayjs()] },
              { label: 'Tháng này', value: [dayjs().startOf('month'), dayjs()] },
            ]}
            onChange={(v) =>
              url.set(
                { from: toApiDate(v?.[0]) ?? undefined, to: toApiDate(v?.[1]) ?? undefined },
                { resetPage: true },
              )
            }
          />
        </Col>
      </FilterBar>
      <DataTable<PaymentItem> table={table} columns={columns} emptyText="Chưa có lần thu nào" />
    </Card>
  );
}

function PaymentsMobile() {
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const today = dayjs().format('YYYY-MM-DD');
  // Thẻ tổng hôm nay cho bản điện thoại.
  const todayQuery = useQuery({
    queryKey: [invoiceApi.key, 'payments', 'today', branchId, today],
    queryFn: () =>
      invoiceApi.payments({ limit: 1, from: today, to: today, branch_id: branchId ?? undefined }),
  });
  const fetchPage = useCallback(
    ({ page, limit, keyword, filters }: MobileFetchArgs) => {
      const range = filters.range as [Dayjs | null, Dayjs | null] | undefined;
      return invoiceApi.payments({
        page,
        limit,
        keyword: keyword || undefined,
        order_by: 'paid_at',
        order_dir: 'DESC',
        method: (filters.method as PaymentMethod) || undefined,
        branch_id: (filters.branch_id as number | undefined) ?? branchId ?? undefined,
        from: toApiDate(range?.[0]) ?? undefined,
        to: toApiDate(range?.[1]) ?? undefined,
      });
    },
    [branchId],
  );
  const s = todayQuery.data?.summary;
  return (
    <MobileList<PaymentItem>
      title={TITLE}
      queryKey={[invoiceApi.key, 'payments', branchId]}
      fetchPage={fetchPage}
      searchPlaceholder="Số HĐ, hội viên, mã GD"
      emptyText="Chưa có lần thu nào"
      stats={[
        ['Hôm nay', formatMoney(s?.total_amount ?? 0)],
        ...(s?.by_method ?? []).map(
          (m) => [PAYMENT_METHOD_LABEL[m.method], formatMoney(m.amount)] as [string, string],
        ),
      ]}
      filters={[
        {
          name: 'branch_id',
          label: 'Chi nhánh',
          options: branches.map((b) => ({ label: b.name, value: b.id })),
        },
        { name: 'method', label: 'Phương thức', options: PAYMENT_METHOD_OPTIONS },
        { name: 'range', label: 'Ngày thu', type: 'daterange' },
      ]}
      row={{
        title: (r) => `${formatMoney(r.amount)} · ${PAYMENT_METHOD_LABEL[r.method]}`,
        subtitle: (r) => `${r.invoice_no} · ${r.member_name || 'Khách lẻ'}`,
        meta: (r) => [formatDateTime(r.paid_at), r.branch_name, r.received_by_name],
      }}
      details={(r) => [
        ['Mã GD', r.reference || '—'],
        ['Ghi chú', r.note || '—'],
      ]}
    />
  );
}

export default function PaymentsPage() {
  return useIsMobile() ? <PaymentsMobile /> : <PaymentsDesktop />;
}
