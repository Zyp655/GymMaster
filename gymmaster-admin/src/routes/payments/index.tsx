import { useCallback, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Button,
  Card,
  Col,
  DatePicker,
  Popconfirm,
  Row,
  Segmented,
  Space,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  BankOutlined,
  CreditCardOutlined,
  DollarOutlined,
  GlobalOutlined,
  RollbackOutlined,
  WalletOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { Link } from 'react-router-dom';
import { cmsSystemApi } from '@/api/cms-system';
import invoiceApi, { PAYMENT_METHOD_LABEL, PAYMENT_METHOD_OPTIONS } from '@/api/invoice';
import type {
  OnlinePaymentListParams,
  OnlinePaymentTxn,
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
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import {
  ONLINE_TXN_STATUS_MAP,
  ONLINE_TXN_STATUS_OPTIONS,
  PAYMENT_METHOD_MAP,
} from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { DATE_FORMAT, formatDateTime, formatMoney, formatNumber, toApiDate } from '@/utils/format';

const TITLE = 'Sổ thu tiền & Giao dịch VNPay';
const SUBTITLE = 'Các lần thu tiền theo hóa đơn và giao dịch thanh toán VNPay trực tuyến';

const METHOD_ICON: Record<PaymentMethod, ReactNode> = {
  CASH: <WalletOutlined />,
  TRANSFER: <BankOutlined />,
  CARD: <CreditCardOutlined />,
  VNPAY: <GlobalOutlined />,
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
  const methods: PaymentMethod[] = ['CASH', 'TRANSFER', 'CARD', 'VNPAY'];
  return (
    <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
      <Col xs={12} md={6} lg={4}>
        <StatCard
          label="Tổng đã thu"
          value={s?.total_amount ?? 0}
          currency
          caption={`${formatNumber(s?.count ?? 0)} lần thu`}
          icon={<DollarOutlined />}
          loading={loading}
        />
      </Col>
      {methods.map((m) => {
        const row = s?.by_method.find((x) => x.method === m);
        return (
          <Col xs={12} md={6} lg={5} key={m}>
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

function OnlinePaymentsSection() {
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const { can } = usePermission();

  const refundMut = useMutation({
    mutationFn: (id: number) =>
      cmsSystemApi.refundOnlineTxn(id, { reason: 'Hoàn tiền giao dịch VNPay trực tuyến' }),
    onSuccess: (res) => {
      message.success(`Đã hoàn tiền VNPay thành công (${res.gateway_ref})`);
      qc.invalidateQueries({ queryKey: [invoiceApi.key] });
    },
  });

  const table = useTableQuery<OnlinePaymentTxn, OnlinePaymentListParams>({
    queryKey: [invoiceApi.key, 'online-payments'],
    queryFn: (p) => invoiceApi.onlinePayments(p),
    filterKeys: ['branch_id', 'status'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const columns = useMemo<ColumnsType<OnlinePaymentTxn>>(
    () => [
      {
        title: 'Thời gian tạo',
        dataIndex: 'created_at',
        className: 'text-nowrap',
        render: (v: string) => formatDateTime(v),
      },
      {
        title: 'Mã tham chiếu',
        dataIndex: 'txn_ref',
        render: (v: string) => <span className="code-text">{v}</span>,
      },
      {
        title: 'Hóa đơn',
        dataIndex: 'invoice_no',
        render: (v: string | undefined, r) => (
          <Link to={`/app/invoices/${r.invoice_id}`} className="code-text">
            {v || `#${r.invoice_id}`}
          </Link>
        ),
      },
      {
        title: 'Hội viên',
        key: 'member',
        render: (_, r) =>
          r.member_id ? (
            <Space direction="vertical" size={0}>
              <Link to={`/app/members/${r.member_id}`}>
                {r.member_name || `HV #${r.member_id}`}
              </Link>
              {r.member_code && (
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {r.member_code}
                </Typography.Text>
              )}
            </Space>
          ) : (
            '—'
          ),
      },
      {
        title: 'Số tiền',
        dataIndex: 'amount',
        align: 'right',
        render: (v: number) => <MoneyText value={v} strong />,
      },
      {
        title: 'Cổng / Ngân hàng',
        key: 'bank',
        render: (_, r) => (
          <Space size={6}>
            <Tag color="geekblue">VNPay</Tag>
            {r.bank_code ? <Tag>{r.bank_code}</Tag> : null}
          </Space>
        ),
      },
      {
        title: 'Mã GD VNPay',
        dataIndex: 'provider_txn_no',
        render: (v: string) => (v ? <span className="code-text">{v}</span> : '—'),
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        render: (v: string) => <StatusTag status={v} map={ONLINE_TXN_STATUS_MAP} />,
      },
      {
        title: 'Thanh toán lúc',
        dataIndex: 'paid_at',
        className: 'text-nowrap',
        render: (v: string | null) => (v ? formatDateTime(v) : '—'),
      },
      {
        title: 'Thao tác',
        key: 'actions',
        render: (_, r) =>
          r.status === 'SUCCESS' && can(PERM.INVOICE_REFUND) ? (
            <Popconfirm
              title="Hoàn tiền giao dịch VNPay trực tuyến này?"
              onConfirm={() => refundMut.mutate(r.id)}
            >
              <Button size="small" danger icon={<RollbackOutlined />}>
                Hoàn tiền Online
              </Button>
            </Popconfirm>
          ) : (
            '—'
          ),
      },
    ],
    [can, refundMut],
  );

  return (
    <>
      <FilterBar
        searchPlaceholder="Mã tham chiếu (txn_ref), số HĐ, mã GD VNPay..."
        colTextSearch={6}
        filters={[
          {
            name: 'branch_id',
            placeholder: 'Chi nhánh',
            options: branches.map((b) => ({ label: b.name, value: b.id })),
          },
          {
            name: 'status',
            placeholder: 'Trạng thái GD',
            options: ONLINE_TXN_STATUS_OPTIONS,
            col: 4,
          },
        ]}
      />
      <DataTable<OnlinePaymentTxn>
        table={table}
        columns={columns}
        emptyText="Chưa có giao dịch VNPay trực tuyến nào"
      />
    </>
  );
}

function PaymentsDesktop() {
  const url = useUrlParams();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const staffOptions = useStaffOptions();
  const urlBranch = url.get('branch_id');
  const [tab, setTab] = useState<'ledger' | 'online'>('ledger');

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
        render: (v: PaymentMethod) => <StatusTag status={v} map={PAYMENT_METHOD_MAP} />,
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
          <Space wrap>
            <Segmented
              value={tab}
              onChange={(v) => setTab(v as 'ledger' | 'online')}
              options={[
                { label: 'Sổ thu tiền', value: 'ledger' },
                { label: 'Giao dịch VNPay trực tuyến', value: 'online' },
              ]}
            />
            <ActionBar onRefresh={() => table.refetch()} refreshing={table.isFetching} />
          </Space>
        </Col>
      </Row>

      {tab === 'online' ? (
        <OnlinePaymentsSection />
      ) : (
        <>
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
              {
                name: 'method',
                placeholder: 'Phương thức',
                options: PAYMENT_METHOD_OPTIONS,
                col: 3,
              },
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
          <DataTable<PaymentItem>
            table={table}
            columns={columns}
            emptyText="Chưa có lần thu nào"
          />
        </>
      )}
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
        tags: (r) => [
          {
            label: PAYMENT_METHOD_MAP[r.method]?.label ?? r.method,
            color: PAYMENT_METHOD_MAP[r.method]?.color,
          },
        ],
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
