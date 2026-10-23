import { useQuery } from '@tanstack/react-query';
import { App, Button, Descriptions, Modal, Result, Skeleton, Table, Typography } from 'antd';
import { PrinterOutlined } from '@ant-design/icons';
import inventoryApi from '@/api/inventory';
import type { StockTransaction } from '@/api/inventory';
import MoneyText from '@/components/Money/MoneyText';
import StatusTag from '@/components/StatusTag';
import { STOCK_TX_TYPE_MAP, VOUCHER_KIND_MAP } from '@/constants/status';
import { getErrorMessage } from '@/utils/errors';
import { formatDateTime, formatNumber } from '@/utils/format';
import { printVoucher } from './printVoucher';
import QtyText from './QtyText';

export interface VoucherModalProps {
  /** Mã phiếu cần xem; `null` = đóng. */
  voucherNo: string | null;
  onClose: () => void;
}

/** Chi tiết phiếu kho theo `voucher_no` (NK/DC/KK/TH/HD) + in phiếu. */
export default function VoucherModal({ voucherNo, onClose }: VoucherModalProps) {
  const { message } = App.useApp();
  const query = useQuery({
    queryKey: [inventoryApi.key, 'voucher', voucherNo],
    queryFn: () => inventoryApi.voucher(voucherNo!),
    enabled: !!voucherNo,
  });
  const v = query.data;

  return (
    <Modal
      open={!!voucherNo}
      title={
        v ? (
          <span>
            {VOUCHER_KIND_MAP[v.kind]?.label ?? 'Phiếu'}{' '}
            <span className="code-text">{v.voucher_no}</span>
          </span>
        ) : (
          `Phiếu ${voucherNo ?? ''}`
        )
      }
      onCancel={onClose}
      width={980}
      footer={[
        <Button
          key="print"
          icon={<PrinterOutlined />}
          disabled={!v}
          onClick={() => {
            if (v && !printVoucher(v)) message.warning('Trình duyệt chặn cửa sổ in – hãy cho phép popup');
          }}
        >
          In phiếu
        </Button>,
        <Button key="close" type="primary" onClick={onClose}>
          Đóng
        </Button>,
      ]}
      destroyOnHidden
    >
      {query.isLoading ? (
        <Skeleton active />
      ) : query.isError || !v ? (
        <Result
          status="warning"
          title={getErrorMessage(query.error, 'Không tải được phiếu kho')}
          extra={<Button onClick={() => query.refetch()}>Thử lại</Button>}
        />
      ) : (
        <>
          <Descriptions
            size="small"
            column={{ xs: 1, sm: 2 }}
            style={{ marginBottom: 12 }}
            items={[
              {
                label: 'Loại',
                children: <StatusTag status={v.kind} map={VOUCHER_KIND_MAP} />,
              },
              { label: 'Thời gian', children: formatDateTime(v.created_at) },
              {
                label: v.kind === 'TRANSFER' ? 'Chi nhánh xuất' : 'Chi nhánh',
                children: v.branch_name,
              },
              ...(v.kind === 'TRANSFER'
                ? [{ label: 'Chi nhánh nhận', children: v.to_branch_name }]
                : []),
              ...(v.supplier_name ? [{ label: 'Nhà cung cấp', children: v.supplier_name }] : []),
              { label: 'Người lập', children: v.created_by_name || '—' },
              {
                label: 'Tổng số lượng',
                children: formatNumber(v.total_quantity),
              },
              { label: 'Tổng giá trị', children: <MoneyText value={v.total_value} strong /> },
              ...(v.note ? [{ label: 'Ghi chú', children: v.note, span: 'filled' as const }] : []),
            ]}
          />
          <Table<StockTransaction>
            size="small"
            rowKey="id"
            pagination={false}
            dataSource={v.lines}
            scroll={{ x: 'max-content' }}
            columns={[
              {
                title: 'Sản phẩm',
                key: 'product',
                render: (_, r) => (
                  <span>
                    {r.product_name}
                    <br />
                    <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
                      {r.product_sku}
                    </Typography.Text>
                  </span>
                ),
              },
              ...(v.kind === 'TRANSFER'
                ? [{ title: 'Chi nhánh', dataIndex: 'branch_name' }]
                : []),
              {
                title: 'Loại',
                dataIndex: 'type',
                render: (t: string) => <StatusTag status={t} map={STOCK_TX_TYPE_MAP} />,
              },
              {
                title: 'Số lượng',
                dataIndex: 'quantity',
                align: 'right',
                render: (q: number, r) => <QtyText value={q} unit={r.unit} />,
              },
              {
                title: 'Tồn sau',
                dataIndex: 'balance_after',
                align: 'right',
                render: (q: number) => formatNumber(q),
              },
              {
                title: 'Đơn giá',
                dataIndex: 'unit_cost',
                align: 'right',
                render: (c: number) => <MoneyText value={c} />,
              },
              {
                title: 'Thành tiền',
                key: 'value',
                align: 'right',
                render: (_, r) => <MoneyText value={Math.round(Math.abs(r.quantity) * r.unit_cost)} />,
              },
              {
                title: 'Ghi chú',
                dataIndex: 'note',
                render: (n: string) => (
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    {n}
                  </Typography.Text>
                ),
              },
            ]}
          />
        </>
      )}
    </Modal>
  );
}
