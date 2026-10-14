import { Button, Space, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PrinterOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';
import { INVOICE_SOURCE_LABEL } from '@/api/invoice';
import type { Invoice, InvoiceSource } from '@/api/invoice';
import DateText from '@/components/DateText';
import MoneyText from '@/components/Money/MoneyText';
import StatusTag from '@/components/StatusTag';
import { INVOICE_STATUS_MAP } from '@/constants/status';
import { openInvoicePrint } from '@/routes/subscriptions/utils';
import { formatMoney } from '@/utils/format';

/** Cột dùng chung (trang Hóa đơn + tab hóa đơn của hội viên). */
export function invoiceColumns({ showMember = true } = {}): ColumnsType<Invoice> {
  return [
    {
      title: 'Số HĐ',
      dataIndex: 'invoice_no',
      sorter: true,
      render: (v: string, r) => (
        <Link to={`/app/invoices/${r.id}`} className="code-text">
          {v}
        </Link>
      ),
    },
    {
      title: 'Ngày lập',
      dataIndex: 'issued_at',
      sorter: true,
      render: (v: string) => <DateText value={v} withTime />,
    },
    ...(showMember
      ? [
          {
            title: 'Hội viên',
            key: 'member',
            render: (_: unknown, r: Invoice) =>
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
        ]
      : []),
    { title: 'Chi nhánh', dataIndex: 'branch_name' },
    {
      title: 'Nguồn',
      dataIndex: 'source',
      render: (v: InvoiceSource) => INVOICE_SOURCE_LABEL[v] ?? v,
    },
    {
      title: 'Tổng',
      dataIndex: 'total',
      sorter: true,
      align: 'right',
      render: (v: number) => <MoneyText value={v} strong />,
    },
    {
      title: 'Đã trả',
      dataIndex: 'paid_amount',
      sorter: true,
      align: 'right',
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
      title: 'Còn nợ',
      dataIndex: 'due',
      align: 'right',
      render: (v: number) =>
        v > 0 ? <MoneyText value={v} style={{ color: '#cf1322' }} /> : <MoneyText value={0} />,
    },
    {
      title: 'Trạng thái',
      dataIndex: 'status',
      sorter: true,
      render: (v: string) => <StatusTag status={v} map={INVOICE_STATUS_MAP} />,
    },
    {
      title: '',
      key: 'actions',
      fixed: 'right',
      width: 48,
      render: (_, r) => (
        <Tooltip title="In hóa đơn">
          <Button
            type="text"
            size="small"
            icon={<PrinterOutlined />}
            onClick={() => openInvoicePrint(r.id)}
            aria-label="In hóa đơn"
          />
        </Tooltip>
      ),
    },
  ];
}
