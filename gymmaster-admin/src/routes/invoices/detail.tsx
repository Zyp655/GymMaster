import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Result,
  Row,
  Skeleton,
  Space,
  Table,
  Typography,
} from 'antd';
import {
  CloseCircleOutlined,
  DollarOutlined,
  PrinterOutlined,
  RollbackOutlined,
} from '@ant-design/icons';
import { Link, useNavigate, useParams } from 'react-router-dom';
import invoiceApi, {
  INVOICE_ITEM_TYPE_LABEL,
  INVOICE_SOURCE_LABEL,
  PAYMENT_METHOD_LABEL,
} from '@/api/invoice';
import type { InvoiceItem, Payment, Refund } from '@/api/invoice';
import MoneyText from '@/components/Money/MoneyText';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { INVOICE_STATUS_MAP } from '@/constants/status';
import { usePermission } from '@/hooks/usePermission';
import { openInvoicePrint } from '@/routes/subscriptions/utils';
import { getErrorMessage } from '@/utils/errors';
import { formatDateTime, formatPhone } from '@/utils/format';
import { CollectModal, RefundModal, VoidModal } from './InvoiceActionModals';
import { canCollect, canRefund, canVoid } from './utils';

type ModalKey = 'collect' | 'refund' | 'void' | null;

/** Mục cuối chiếm phần còn lại của hàng (tránh cảnh báo tổng span lệch số cột). */
const fillLast = <T extends object>(items: T[]) =>
  items.map((it, i) => (i === items.length - 1 ? { ...it, span: 'filled' as const } : it));

/** Chi tiết hóa đơn `/app/invoices/:id`: dòng hàng, thanh toán, hoàn tiền + thao tác. */
export default function InvoiceDetailPage() {
  const { id: idParam } = useParams();
  const id = Number(idParam);
  const navigate = useNavigate();
  const { can } = usePermission();
  const [modal, setModal] = useState<ModalKey>(null);

  const query = useQuery({
    queryKey: [invoiceApi.key, 'detail', id],
    queryFn: () => invoiceApi.get(id),
    enabled: Number.isInteger(id) && id > 0,
  });
  const inv = query.data;

  if (query.isLoading) {
    return (
      <Card variant="borderless" className="page-card">
        <Skeleton active paragraph={{ rows: 8 }} />
      </Card>
    );
  }
  if (query.isError || !inv) {
    return (
      <Card variant="borderless" className="page-card">
        <Result
          status="warning"
          title={getErrorMessage(query.error, 'Không tìm thấy hóa đơn')}
          extra={
            <Space>
              <Button onClick={() => navigate('/app/invoices')}>Về danh sách</Button>
              <Button type="primary" onClick={() => query.refetch()}>
                Thử lại
              </Button>
            </Space>
          }
        />
      </Card>
    );
  }

  const subRef = inv.items.find((i) => i.item_type === 'MEMBERSHIP' || i.item_type === 'FEE');

  return (
    <>
      <Card variant="borderless" className="page-card">
        <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
          <Col>
            <PageTitle
              title={`Hóa đơn ${inv.invoice_no}`}
              back="/app/invoices"
              breadcrumb={[{ title: 'Hóa đơn', path: '/app/invoices' }, { title: inv.invoice_no }]}
            />
          </Col>
          <Col>
            <Space wrap>
              <Button icon={<PrinterOutlined />} onClick={() => openInvoicePrint(inv.id)}>
                In hóa đơn
              </Button>
              {can(PERM.INVOICE_COLLECT) && canCollect(inv) && (
                <Button
                  color="orange"
                  variant="solid"
                  icon={<DollarOutlined />}
                  onClick={() => setModal('collect')}
                >
                  Thu tiền
                </Button>
              )}
              {can(PERM.INVOICE_REFUND) && canRefund(inv) && (
                <Button icon={<RollbackOutlined />} onClick={() => setModal('refund')}>
                  Hoàn tiền
                </Button>
              )}
              {can(PERM.INVOICE_VOID) && canVoid(inv) && (
                <Button danger icon={<CloseCircleOutlined />} onClick={() => setModal('void')}>
                  Hủy hóa đơn
                </Button>
              )}
            </Space>
          </Col>
        </Row>

        {inv.status === 'VOID' && (
          <Alert
            type="error"
            showIcon
            style={{ marginBottom: 16 }}
            message={`Hóa đơn đã hủy lúc ${formatDateTime(inv.voided_at)}${inv.voided_by_name ? ` bởi ${inv.voided_by_name}` : ''}`}
            description={inv.void_reason ? `Lý do: ${inv.void_reason}` : undefined}
          />
        )}

        <Descriptions
          size="small"
          bordered
          column={{ xs: 1, sm: 2, lg: 3 }}
          items={fillLast([
            {
              key: 'status',
              label: 'Trạng thái',
              children: <StatusTag status={inv.status} map={INVOICE_STATUS_MAP} />,
            },
            { key: 'issued', label: 'Ngày lập', children: formatDateTime(inv.issued_at) },
            { key: 'by', label: 'Người lập', children: inv.issued_by_name || '—' },
            {
              key: 'member',
              label: 'Hội viên',
              children: inv.member ? (
                <Link to={`/app/members/${inv.member.id}`}>
                  {inv.member.full_name} · {inv.member.member_code} ·{' '}
                  {formatPhone(inv.member.phone)}
                </Link>
              ) : (
                'Khách lẻ'
              ),
            },
            { key: 'branch', label: 'Chi nhánh', children: inv.branch_name },
            { key: 'source', label: 'Nguồn', children: INVOICE_SOURCE_LABEL[inv.source] },
            ...(inv.promotion_code
              ? [{ key: 'promo', label: 'Khuyến mãi', children: inv.promotion_code }]
              : []),
            ...(subRef?.ref_id
              ? [
                  {
                    key: 'sub',
                    label: 'Gói tập',
                    children: (
                      <Link to={`/app/subscriptions?sid=${subRef.ref_id}`}>
                        Xem gói #{subRef.ref_id}
                      </Link>
                    ),
                  },
                ]
              : []),
            ...(inv.note ? [{ key: 'note', label: 'Ghi chú', children: inv.note }] : []),
          ])}
        />
      </Card>

      <Card variant="borderless" className="page-card" style={{ marginTop: 16 }} title="Chi tiết">
        <Table<InvoiceItem>
          size="small"
          rowKey="id"
          pagination={false}
          dataSource={inv.items}
          scroll={{ x: 'max-content' }}
          columns={[
            { title: '#', key: 'idx', width: 40, render: (_, __, i) => i + 1 },
            {
              title: 'Loại',
              dataIndex: 'item_type',
              render: (v: InvoiceItem['item_type']) => INVOICE_ITEM_TYPE_LABEL[v] ?? v,
            },
            { title: 'Mô tả', dataIndex: 'description' },
            { title: 'SL', dataIndex: 'quantity', align: 'right' },
            {
              title: 'Đơn giá',
              dataIndex: 'unit_price',
              align: 'right',
              render: (v: number) => <MoneyText value={v} />,
            },
            {
              title: 'Giảm',
              dataIndex: 'discount',
              align: 'right',
              render: (v: number) => (v ? <MoneyText value={-v} colored /> : '—'),
            },
            {
              title: 'Thành tiền',
              dataIndex: 'amount',
              align: 'right',
              render: (v: number) => <MoneyText value={v} strong />,
            },
          ]}
          summary={() => (
            <>
              {[
                ['Tạm tính', inv.subtotal],
                ['Giảm giá', -inv.discount],
                ['Tổng cộng', inv.total],
                ['Đã thanh toán', inv.paid_amount],
                ...(inv.refunded_amount ? [['Đã hoàn', -inv.refunded_amount] as const] : []),
                ['Còn nợ', inv.due],
              ].map(([label, value]) => (
                <Table.Summary.Row key={label as string}>
                  <Table.Summary.Cell index={0} colSpan={6} align="right">
                    <Typography.Text strong={label === 'Tổng cộng' || label === 'Còn nợ'}>
                      {label}
                    </Typography.Text>
                  </Table.Summary.Cell>
                  <Table.Summary.Cell index={1} align="right">
                    <MoneyText
                      value={value as number}
                      strong={label === 'Tổng cộng' || label === 'Còn nợ'}
                      style={
                        label === 'Còn nợ' && (value as number) > 0
                          ? { color: '#cf1322' }
                          : undefined
                      }
                    />
                  </Table.Summary.Cell>
                </Table.Summary.Row>
              ))}
            </>
          )}
        />
      </Card>

      <Row gutter={16}>
        <Col xs={24} lg={14}>
          <Card
            variant="borderless"
            className="page-card"
            style={{ marginTop: 16 }}
            title="Thanh toán"
          >
            <Table<Payment>
              size="small"
              rowKey="id"
              pagination={false}
              dataSource={inv.payments}
              scroll={{ x: 'max-content' }}
              locale={{ emptyText: 'Chưa thu tiền' }}
              columns={[
                {
                  title: 'Thời gian',
                  dataIndex: 'paid_at',
                  render: (v: string) => formatDateTime(v),
                },
                {
                  title: 'Số tiền',
                  dataIndex: 'amount',
                  align: 'right',
                  render: (v: number) => <MoneyText value={v} strong />,
                },
                {
                  title: 'Phương thức',
                  dataIndex: 'method',
                  render: (v: Payment['method']) => PAYMENT_METHOD_LABEL[v] ?? v,
                },
                { title: 'Mã GD', dataIndex: 'reference', render: (v: string) => v || '—' },
                { title: 'Người thu', dataIndex: 'received_by_name' },
                { title: 'Ghi chú', dataIndex: 'note', render: (v: string) => v || '—' },
              ]}
            />
          </Card>
        </Col>
        <Col xs={24} lg={10}>
          <Card
            variant="borderless"
            className="page-card"
            style={{ marginTop: 16 }}
            title="Hoàn tiền"
          >
            <Table<Refund>
              size="small"
              rowKey="id"
              pagination={false}
              dataSource={inv.refunds}
              scroll={{ x: 'max-content' }}
              locale={{ emptyText: 'Chưa hoàn tiền' }}
              columns={[
                {
                  title: 'Thời gian',
                  dataIndex: 'created_at',
                  render: (v: string) => formatDateTime(v),
                },
                {
                  title: 'Số tiền',
                  dataIndex: 'amount',
                  align: 'right',
                  render: (v: number) => <MoneyText value={v} />,
                },
                {
                  title: 'Phương thức',
                  dataIndex: 'method',
                  render: (v: Refund['method']) => PAYMENT_METHOD_LABEL[v] ?? v,
                },
                { title: 'Lý do', dataIndex: 'reason' },
                { title: 'Người hoàn', dataIndex: 'refunded_by_name' },
              ]}
            />
          </Card>
        </Col>
      </Row>

      <CollectModal open={modal === 'collect'} invoice={inv} onClose={() => setModal(null)} />
      <RefundModal open={modal === 'refund'} invoice={inv} onClose={() => setModal(null)} />
      <VoidModal open={modal === 'void'} invoice={inv} onClose={() => setModal(null)} />
    </>
  );
}
