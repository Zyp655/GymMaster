import { useCallback, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Button,
  Card,
  Col,
  DatePicker,
  Divider,
  Empty,
  Row,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  DollarOutlined,
  EyeOutlined,
  PrinterOutlined,
  RollbackOutlined,
  ShoppingCartOutlined,
  ShoppingOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { Link, useNavigate } from 'react-router-dom';
import { PAYMENT_METHOD_LABEL, PAYMENT_METHOD_OPTIONS } from '@/api/invoice';
import type { Invoice, InvoiceStatus, PaymentMethod } from '@/api/invoice';
import posApi from '@/api/pos';
import type { PosOrderListParams, PosSummary } from '@/api/pos';
import ActionBar from '@/components/ActionBar';
import DataTable from '@/components/DataTable';
import DateText from '@/components/DateText';
import FilterBar from '@/components/FilterBar';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import MoneyText from '@/components/Money/MoneyText';
import HelpLabel from '@/components/HelpLabel';
import PageTitle from '@/components/PageTitle';
import StatCard, { SectionHeader } from '@/components/StatCard';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { INVOICE_STATUS_MAP, INVOICE_STATUS_OPTIONS } from '@/constants/status';
import { REVENUE_HELP } from '@/constants/revenueBasis';
import { useIsMobile } from '@/hooks/useIsMobile';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { printReceipt80 } from '@/routes/pos/cart';
import {
  API_DATE_FORMAT,
  DATE_FORMAT,
  formatDateTime,
  formatMoney,
  formatNumber,
  toApiDate,
} from '@/utils/format';
import OrderDrawer from './OrderDrawer';

const TITLE = 'Đơn hàng POS';
const SUBTITLE = 'Danh sách hóa đơn bán hàng tại quầy, tổng kết ngày và trả hàng';

function PosDailySummary({
  branchId,
  selectedDate,
  onDateChange,
}: {
  branchId: number | null;
  selectedDate: string;
  onDateChange: (d: string) => void;
}) {
  const summaryQuery = useQuery({
    queryKey: [posApi.key, 'summary', branchId ?? null, selectedDate],
    queryFn: () => posApi.summary({ branch_id: branchId ?? undefined, date: selectedDate }),
  });

  const s: PosSummary | undefined = summaryQuery.data;
  const loading = summaryQuery.isLoading;

  return (
    <div style={{ marginBottom: 20 }}>
      <SectionHeader
        title={`Tổng kết bán hàng ngày ${dayjs(selectedDate).format(DATE_FORMAT)}`}
        extra={
          <DatePicker
            allowClear={false}
            format={DATE_FORMAT}
            value={dayjs(selectedDate)}
            onChange={(d) => d && onDateChange(d.format(API_DATE_FORMAT))}
          />
        }
      />
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        <Col xs={12} sm={8} lg={6}>
          <StatCard
            label="Số đơn / SL bán"
            value={s?.order_count ?? 0}
            valueSuffix="đơn"
            caption={`SL bán thuần: ${formatNumber(s?.items_sold ?? 0)} sản phẩm`}
            icon={<ShoppingOutlined />}
            loading={loading}
          />
        </Col>
        <Col xs={12} sm={8} lg={6}>
          <StatCard
            label="Doanh thu gộp"
            value={s?.gross_revenue ?? 0}
            currency
            caption={`Giảm giá: ${formatMoney(s?.discount ?? 0)}`}
            icon={<DollarOutlined />}
            loading={loading}
          />
        </Col>
        <Col xs={12} sm={8} lg={6}>
          <StatCard
            label={<HelpLabel help={REVENUE_HELP.posDaily}>Doanh thu thuần</HelpLabel>}
            value={s?.net_revenue ?? 0}
            currency
            caption={`Hoàn trả: ${formatMoney(s?.refund_amount ?? 0)}`}
            icon={<RollbackOutlined />}
            loading={loading}
          />
        </Col>
        <Col xs={12} sm={8} lg={6}>
          <StatCard
            label="Lãi gộp"
            value={s?.gross_profit ?? 0}
            currency
            caption={`Giá vốn: ${formatMoney(s?.cost_of_goods ?? 0)}`}
            icon={<DollarOutlined />}
            loading={loading}
          />
        </Col>
      </Row>

      <Row gutter={[12, 12]}>
        <Col xs={24} md={10}>
          <Card
            size="small"
            title={
              <HelpLabel help={REVENUE_HELP.posByMethod}>Thu theo phương thức (gộp)</HelpLabel>
            }
            loading={loading}
          >
            <Space direction="vertical" size={6} style={{ width: '100%' }}>
              {(s?.by_method ?? []).map((m) => (
                <Row key={m.method} justify="space-between" align="middle">
                  <Col>
                    <Typography.Text>{PAYMENT_METHOD_LABEL[m.method] ?? m.method}</Typography.Text>{' '}
                    <Tag bordered={false}>{formatNumber(m.count)} lần</Tag>
                  </Col>
                  <Col>
                    <MoneyText value={m.amount} strong />
                  </Col>
                </Row>
              ))}
              <Divider style={{ margin: '4px 0' }} />
              <Row justify="space-between" align="middle">
                <Col>
                  <Typography.Text strong>Tổng thu (chưa trừ hoàn trả)</Typography.Text>
                </Col>
                <Col>
                  <MoneyText
                    value={(s?.by_method ?? []).reduce((t, m) => t + m.amount, 0)}
                    strong
                  />
                </Col>
              </Row>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                Hoàn trả trong ngày: {formatMoney(s?.refund_amount ?? 0)} – xem thẻ Doanh thu thuần.
              </Typography.Text>
            </Space>
          </Card>
        </Col>
        <Col xs={24} md={14}>
          <Card size="small" title="Sản phẩm bán chạy trong ngày" loading={loading}>
            {!s?.top_products?.length ? (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="Chưa có sản phẩm bán ra trong ngày"
              />
            ) : (
              <Table
                size="small"
                rowKey="product_id"
                pagination={false}
                dataSource={s.top_products}
                columns={[
                  {
                    title: 'Sản phẩm',
                    key: 'name',
                    render: (_, p) => (
                      <span>
                        {p.name}{' '}
                        <Typography.Text
                          type="secondary"
                          className="code-text"
                          style={{ fontSize: 12 }}
                        >
                          ({p.sku})
                        </Typography.Text>
                      </span>
                    ),
                  },
                  {
                    title: 'SL bán (thuần)',
                    dataIndex: 'quantity',
                    align: 'right',
                    width: 80,
                    render: (v: number) => formatNumber(v),
                  },
                  {
                    title: 'Doanh thu',
                    dataIndex: 'revenue',
                    align: 'right',
                    width: 130,
                    render: (v: number) => <MoneyText value={v} strong />,
                  },
                ]}
              />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
}

function PosOrdersDesktop() {
  const navigate = useNavigate();
  const { can } = usePermission();
  const url = useUrlParams();
  const { branchId } = useWorkingBranch();
  const [selectedDate, setSelectedDate] = useState(() => dayjs().format(API_DATE_FORMAT));

  const idParam = url.get('id');
  const selectedId = idParam && Number(idParam) > 0 ? Number(idParam) : null;
  const openOrder = useCallback((id: number) => url.set({ id }), [url]);
  const closeOrder = useCallback(() => url.set({ id: undefined }), [url]);

  const table = useTableQuery<Invoice, PosOrderListParams>({
    queryKey: [posApi.key, 'orders'],
    queryFn: (p) => posApi.orders(p),
    filterKeys: ['status', 'method', 'from', 'to'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: { branch_id: branchId ?? undefined },
  });

  const from = url.get('from');
  const to = url.get('to');

  const columns = useMemo<ColumnsType<Invoice>>(
    () => [
      {
        title: 'Số HĐ',
        dataIndex: 'invoice_no',
        sorter: true,
        width: 145,
        render: (v: string, r) => (
          <Button
            type="link"
            style={{ padding: 0 }}
            className="code-text"
            onClick={() => openOrder(r.id)}
          >
            {v}
          </Button>
        ),
      },
      {
        title: 'Ngày lập',
        dataIndex: 'issued_at',
        sorter: true,
        width: 135,
        render: (v: string) => <DateText value={v} withTime />,
      },
      {
        title: 'Hội viên / Khách',
        key: 'member',
        width: 150,
        ellipsis: true,
        render: (_, r) =>
          r.member ? (
            <Space direction="vertical" size={0}>
              <Link to={`/app/members/${r.member.id}`}>{r.member.full_name}</Link>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {r.member.member_code}
              </Typography.Text>
            </Space>
          ) : (
            <Typography.Text type="secondary">Khách lẻ</Typography.Text>
          ),
      },
      { title: 'Chi nhánh', dataIndex: 'branch_name', width: 140, ellipsis: true },
      {
        title: 'Giảm giá',
        dataIndex: 'discount',
        align: 'right',
        width: 95,
        responsive: ['lg'],
        render: (v: number) => (v > 0 ? <MoneyText value={v} /> : '—'),
      },
      {
        title: 'Tổng cộng',
        dataIndex: 'total',
        sorter: true,
        align: 'right',
        width: 110,
        render: (v: number) => <MoneyText value={v} strong />,
      },
      {
        title: 'Đã thu',
        dataIndex: 'paid_amount',
        sorter: true,
        align: 'right',
        width: 115,
        render: (v: number, r) => (
          <Space direction="vertical" size={0} align="end">
            <MoneyText value={v} />
            {r.refunded_amount > 0 && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                hoàn {formatMoney(r.refunded_amount)}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        sorter: true,
        width: 135,
        render: (v: string) => <StatusTag status={v} map={INVOICE_STATUS_MAP} />,
      },
      {
        title: 'Thu ngân',
        dataIndex: 'issued_by_name',
        width: 130,
        ellipsis: true,
        // Chỉ hiện trên màn rất rộng – ở 1440px các cột chính (trạng thái) không bị cột thao tác cố định che.
        responsive: ['xxl'],
        render: (v: string) => v || '—',
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 80,
        render: (_, r) => (
          <Space size={4}>
            <Tooltip title="Chi tiết / Trả hàng">
              <Button
                type="text"
                size="small"
                icon={<EyeOutlined />}
                onClick={() => openOrder(r.id)}
                aria-label="Chi tiết đơn"
              />
            </Tooltip>
            <Tooltip title="In hóa đơn 80mm">
              <Button
                type="text"
                size="small"
                icon={<PrinterOutlined />}
                onClick={() => printReceipt80(r.id)}
                aria-label="In hóa đơn 80mm"
              />
            </Tooltip>
          </Space>
        ),
      },
    ],
    [openOrder],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title={TITLE} subtitle={SUBTITLE} />
        </Col>
        <Col>
          <ActionBar onRefresh={() => table.refetch()} refreshing={table.isFetching}>
            {can(PERM.POS_SELL) && (
              <Button
                color="orange"
                variant="solid"
                icon={<ShoppingCartOutlined />}
                onClick={() => navigate('/app/pos')}
              >
                Bán hàng POS
              </Button>
            )}
          </ActionBar>
        </Col>
      </Row>

      <PosDailySummary
        branchId={branchId}
        selectedDate={selectedDate}
        onDateChange={setSelectedDate}
      />

      <FilterBar
        searchPlaceholder="Tìm mã HĐ, tên/mã/SĐT hội viên..."
        colTextSearch={6}
        filters={[
          { name: 'status', placeholder: 'Trạng thái', options: INVOICE_STATUS_OPTIONS, col: 4 },
          { name: 'method', placeholder: 'Phương thức', options: PAYMENT_METHOD_OPTIONS, col: 4 },
        ]}
      >
        <Col xs={24} sm={12} md={6}>
          <DatePicker.RangePicker
            style={{ width: '100%' }}
            format={DATE_FORMAT}
            placeholder={['Lập từ ngày', 'Đến ngày']}
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

      <DataTable<Invoice>
        table={table}
        columns={columns}
        emptyText="Chưa có đơn hàng POS nào"
        // Độ rộng cột cố định: ở 1440px cột trạng thái không bị cột thao tác cố định che/cắt chữ.
        tableLayout="fixed"
        scroll={{ x: 1105 }}
      />
      <OrderDrawer invoiceId={selectedId} onClose={closeOrder} />
    </Card>
  );
}

function PosOrdersMobile() {
  const navigate = useNavigate();
  const { branchId } = useWorkingBranch();
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const today = dayjs().format(API_DATE_FORMAT);

  const summaryQuery = useQuery({
    queryKey: [posApi.key, 'summary', branchId ?? null, today],
    queryFn: () => posApi.summary({ branch_id: branchId ?? undefined, date: today }),
  });

  const fetchPage = useCallback(
    ({ page, limit, keyword, filters }: MobileFetchArgs) => {
      const range = filters.range as [Dayjs | null, Dayjs | null] | undefined;
      return posApi.orders({
        page,
        limit,
        keyword: keyword || undefined,
        order_by: 'id',
        order_dir: 'DESC',
        branch_id: branchId ?? undefined,
        status: (filters.status as InvoiceStatus) || undefined,
        method: (filters.method as PaymentMethod) || undefined,
        from: toApiDate(range?.[0]) ?? undefined,
        to: toApiDate(range?.[1]) ?? undefined,
      });
    },
    [branchId],
  );

  const s = summaryQuery.data;

  return (
    <>
      <MobileList<Invoice>
        title={TITLE}
        queryKey={[posApi.key, 'orders', branchId]}
        fetchPage={fetchPage}
        onOpen={(r) => setSelectedId(r.id)}
        onAdd={() => navigate('/app/pos')}
        addPerm={PERM.POS_SELL}
        searchPlaceholder="Tìm mã HĐ, tên/mã/SĐT hội viên..."
        emptyText="Chưa có đơn hàng POS nào"
        stats={[
          [
            'Đơn hôm nay',
            `${formatNumber(s?.order_count ?? 0)} (${formatNumber(s?.items_sold ?? 0)} SP)`,
          ],
          ['DT thuần', formatMoney(s?.net_revenue ?? 0)],
          ['Lãi gộp', formatMoney(s?.gross_profit ?? 0)],
        ]}
        filters={[
          { name: 'status', label: 'Trạng thái', options: INVOICE_STATUS_OPTIONS },
          { name: 'method', label: 'Phương thức', options: PAYMENT_METHOD_OPTIONS },
          { name: 'range', label: 'Ngày lập', type: 'daterange' },
        ]}
        row={{
          title: (r) => `${r.invoice_no} · ${formatMoney(r.total)}`,
          subtitle: (r) => r.member?.full_name ?? 'Khách lẻ',
          meta: (r) => [
            formatDateTime(r.issued_at),
            r.branch_name,
            r.refunded_amount > 0 && `Hoàn ${formatMoney(r.refunded_amount)}`,
          ],
          tags: (r) => [
            {
              label: INVOICE_STATUS_MAP[r.status]?.label,
              color: INVOICE_STATUS_MAP[r.status]?.color,
            },
          ],
        }}
        actions={(r) => [
          {
            key: 'detail',
            label: 'Chi tiết / Trả hàng',
            icon: <EyeOutlined />,
            onClick: () => setSelectedId(r.id),
          },
          {
            key: 'print',
            label: 'In hóa đơn 80mm',
            icon: <PrinterOutlined />,
            onClick: () => printReceipt80(r.id),
          },
        ]}
      />
      <OrderDrawer invoiceId={selectedId} onClose={() => setSelectedId(null)} />
    </>
  );
}

export default function PosOrdersPage() {
  return useIsMobile() ? <PosOrdersMobile /> : <PosOrdersDesktop />;
}
