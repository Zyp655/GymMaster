import { useCallback, useMemo } from 'react';
import { Button, Card, Col, DatePicker, Row } from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { Link, useNavigate } from 'react-router-dom';
import invoiceApi, { INVOICE_SOURCE_OPTIONS, PAYMENT_METHOD_OPTIONS } from '@/api/invoice';
import type { Invoice, InvoiceListParams, InvoiceSource, InvoiceStatus } from '@/api/invoice';
import ActionBar from '@/components/ActionBar';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import PageTitle from '@/components/PageTitle';
import { INVOICE_STATUS_MAP, INVOICE_STATUS_OPTIONS } from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { DATE_FORMAT, formatDateTime, formatMoney, toApiDate } from '@/utils/format';
import { invoiceColumns } from './columns';

const TITLE = 'Hóa đơn';
const SUBTITLE = 'Hóa đơn bán gói, PT, sản phẩm – thu tiền, hủy, hoàn tiền, VNPay';

function InvoicesDesktop() {
  const url = useUrlParams();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const urlBranch = url.get('branch_id');

  const table = useTableQuery<Invoice, InvoiceListParams>({
    queryKey: [invoiceApi.key, 'list'],
    queryFn: (p) => invoiceApi.list(p),
    filterKeys: ['status', 'source', 'method', 'branch_id', 'member_id', 'from', 'to'],
    numberKeys: ['branch_id', 'member_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: { branch_id: urlBranch ? undefined : (branchId ?? undefined) },
  });
  const columns = useMemo(() => invoiceColumns(), []);
  const from = url.get('from');
  const to = url.get('to');

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title={TITLE} subtitle={SUBTITLE} />
        </Col>
        <Col>
          <ActionBar onRefresh={() => table.refetch()} refreshing={table.isFetching}>
            <Link to="/app/payments">
              <Button>Sổ thu & VNPay</Button>
            </Link>
          </ActionBar>
        </Col>
      </Row>
      <FilterBar
        searchPlaceholder="Số HĐ, tên/SĐT/mã hội viên"
        colTextSearch={4}
        filters={[
          { name: 'status', placeholder: 'Trạng thái', options: INVOICE_STATUS_OPTIONS, col: 3 },
          { name: 'source', placeholder: 'Nguồn', options: INVOICE_SOURCE_OPTIONS, col: 3 },
          { name: 'method', placeholder: 'Phương thức', options: PAYMENT_METHOD_OPTIONS, col: 3 },
          {
            name: 'branch_id',
            placeholder: 'Chi nhánh',
            options: branches.map((b) => ({ label: b.name, value: b.id })),
            col: 4,
          },
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
      <DataTable<Invoice> table={table} columns={columns} emptyText="Chưa có hóa đơn nào" />
    </Card>
  );
}

function InvoicesMobile() {
  const navigate = useNavigate();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const fetchPage = useCallback(
    ({ page, limit, keyword, filters }: MobileFetchArgs) => {
      const range = filters.range as [Dayjs | null, Dayjs | null] | undefined;
      return invoiceApi.list({
        page,
        limit,
        keyword: keyword || undefined,
        order_by: 'id',
        order_dir: 'DESC',
        status: (filters.status as InvoiceStatus) || undefined,
        source: (filters.source as InvoiceSource) || undefined,
        branch_id: (filters.branch_id as number | undefined) ?? branchId ?? undefined,
        from: toApiDate(range?.[0]) ?? undefined,
        to: toApiDate(range?.[1]) ?? undefined,
      });
    },
    [branchId],
  );
  return (
    <MobileList<Invoice>
      title={TITLE}
      queryKey={[invoiceApi.key, 'list', branchId]}
      fetchPage={fetchPage}
      onOpen={(r) => navigate(`/app/invoices/${r.id}`)}
      searchPlaceholder="Số HĐ, tên/SĐT hội viên"
      emptyText="Chưa có hóa đơn nào"
      filters={[
        { name: 'status', label: 'Trạng thái', options: INVOICE_STATUS_OPTIONS },
        { name: 'source', label: 'Nguồn', options: INVOICE_SOURCE_OPTIONS },
        {
          name: 'branch_id',
          label: 'Chi nhánh',
          options: branches.map((b) => ({ label: b.name, value: b.id })),
        },
        { name: 'range', label: 'Ngày lập', type: 'daterange' },
      ]}
      row={{
        title: (r) => `${r.invoice_no} · ${formatMoney(r.total)}`,
        subtitle: (r) => r.member?.full_name ?? 'Khách lẻ',
        meta: (r) => [
          formatDateTime(r.issued_at),
          r.branch_name,
          r.due > 0 && `Nợ ${formatMoney(r.due)}`,
        ],
        tags: (r) => [
          {
            label: INVOICE_STATUS_MAP[r.status]?.label,
            color: INVOICE_STATUS_MAP[r.status]?.color,
          },
        ],
      }}
    />
  );
}

export default function InvoicesPage() {
  return useIsMobile() ? <InvoicesMobile /> : <InvoicesDesktop />;
}
