import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Descriptions,
  Divider,
  Drawer,
  Form,
  Input,
  InputNumber,
  Result,
  Segmented,
  Skeleton,
  Space,
  Table,
  Typography,
} from 'antd';
import { PrinterOutlined, RollbackOutlined, StopOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';
import invoiceApi, { PAYMENT_METHOD_LABEL, PAYMENT_METHOD_OPTIONS } from '@/api/invoice';
import type { PaymentMethod } from '@/api/invoice';
import inventoryApi from '@/api/inventory';
import posApi from '@/api/pos';
import type { PosOrderLine } from '@/api/pos';
import productApi from '@/api/product';
import MoneyText from '@/components/Money/MoneyText';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { INVOICE_STATUS_MAP } from '@/constants/status';
import { usePermission } from '@/hooks/usePermission';
import { VoidModal } from '@/routes/invoices/InvoiceActionModals';
import { canVoid } from '@/routes/invoices/utils';
import { printReceipt80 } from '@/routes/pos/cart';
import { getErrorMessage, showError } from '@/utils/errors';
import { formatDateTime, formatMoney, formatNumber } from '@/utils/format';

export interface OrderDrawerProps {
  invoiceId: number | null;
  onClose: () => void;
}

/** Ước tính tiền hoàn giống backend: round(amount × (đã_trả + SL)/đã_bán) − round(amount × đã_trả/đã_bán). */
function estimateRefund(l: PosOrderLine, qty: number) {
  if (!qty || !l.sold_quantity) return 0;
  return (
    Math.round((l.amount * (l.returned_quantity + qty)) / l.sold_quantity) -
    Math.round((l.amount * l.returned_quantity) / l.sold_quantity)
  );
}

/** Chi tiết đơn POS + trả hàng (chọn SL ≤ còn trả được, phương thức hoàn, lý do). */
export default function OrderDrawer({ invoiceId, onClose }: OrderDrawerProps) {
  const { message, modal } = App.useApp();
  const { can } = usePermission();
  const queryClient = useQueryClient();
  const [qty, setQty] = useState<Record<number, number | null>>({});
  const [method, setMethod] = useState<PaymentMethod>('CASH');
  const [reasonForm] = Form.useForm<{ reason: string }>();
  // Đổi key = form gắn lại từ đầu (xóa lý do) mà không gọi resetFields khi Form chưa mount.
  const [formKey, setFormKey] = useState(0);
  const [voidOpen, setVoidOpen] = useState(false);

  const query = useQuery({
    queryKey: [posApi.key, 'order', invoiceId],
    queryFn: () => posApi.order(invoiceId!),
    enabled: !!invoiceId,
  });

  useEffect(() => {
    setQty({});
    setMethod('CASH');
  }, [invoiceId]);

  const ret = useMutation({
    mutationFn: (data: Parameters<typeof posApi.returnItems>[1]) =>
      posApi.returnItems(invoiceId!, data),
    onSuccess: (res) => {
      queryClient.setQueryData([posApi.key, 'order', invoiceId], {
        invoice: res.invoice,
        lines: res.lines,
      });
      void queryClient.invalidateQueries({ queryKey: [posApi.key] });
      void queryClient.invalidateQueries({ queryKey: [invoiceApi.key] });
      void queryClient.invalidateQueries({ queryKey: [inventoryApi.key] });
      void queryClient.invalidateQueries({ queryKey: [productApi.key] });
    },
  });

  const data = query.data;
  const inv = data?.invoice;
  const lines = data?.lines ?? [];
  const selected = lines.filter((l) => (qty[l.product_id] ?? 0) > 0);
  const remainingPaid = inv ? Math.max(0, inv.paid_amount - inv.refunded_amount) : 0;
  const refundEstimate = Math.min(
    remainingPaid,
    selected.reduce((s, l) => s + estimateRefund(l, qty[l.product_id] ?? 0), 0),
  );
  const canReturn =
    !!inv &&
    can(PERM.POS_SELL) &&
    (inv.status === 'PAID' || inv.status === 'REFUNDED') &&
    lines.some((l) => l.returnable_quantity > 0);

  const submitReturn = async () => {
    if (!inv || selected.length === 0) return;
    const { reason } = await reasonForm.validateFields();
    modal.confirm({
      title: `Trả ${selected.length} sản phẩm của hóa đơn ${inv.invoice_no}?`,
      content: (
        <div>
          <ul style={{ paddingLeft: 18 }}>
            {selected.map((l) => (
              <li key={l.product_id}>
                {l.description} × {qty[l.product_id]}
              </li>
            ))}
          </ul>
          Hàng được nhập lại kho. Hoàn khách khoảng <b>{formatMoney(refundEstimate)}</b> bằng{' '}
          {PAYMENT_METHOD_LABEL[method].toLowerCase()}.
        </div>
      ),
      okText: 'Trả hàng & hoàn tiền',
      okButtonProps: { danger: true },
      cancelText: 'Không',
      onOk: () =>
        ret
          .mutateAsync({
            items: selected.map((l) => ({ product_id: l.product_id, quantity: qty[l.product_id]! })),
            refund_method: method,
            reason: reason.trim(),
          })
          .then((res) => {
            message.success(
              `Đã trả hàng (${res.voucher_no})${res.refund_amount ? `, hoàn ${formatMoney(res.refund_amount)}` : ''}`,
            );
            setQty({});
            setFormKey((k) => k + 1);
          })
          .catch((e: unknown) => showError(e, 'Không thể trả hàng')),
    });
  };

  return (
    <Drawer
      open={!!invoiceId}
      onClose={onClose}
      width={760}
      title={inv ? `Đơn POS ${inv.invoice_no}` : 'Đơn POS'}
      extra={
        inv && (
          <Space>
            <Button icon={<PrinterOutlined />} onClick={() => printReceipt80(inv.id)}>
              In 80mm
            </Button>
            {canVoid(inv) && can(PERM.INVOICE_VOID) && (
              <Button danger icon={<StopOutlined />} onClick={() => setVoidOpen(true)}>
                Hủy hóa đơn
              </Button>
            )}
          </Space>
        )
      }
    >
      {query.isLoading ? (
        <Skeleton active />
      ) : query.isError || !inv ? (
        <Result
          status="warning"
          title={getErrorMessage(query.error, 'Không tải được đơn hàng')}
          extra={<Button onClick={() => query.refetch()}>Thử lại</Button>}
        />
      ) : (
        <>
          <Descriptions
            size="small"
            column={{ xs: 1, sm: 2 }}
            items={[
              {
                label: 'Trạng thái',
                children: <StatusTag status={inv.status} map={INVOICE_STATUS_MAP} />,
              },
              { label: 'Thời gian', children: formatDateTime(inv.issued_at) },
              {
                label: 'Khách hàng',
                children: inv.member ? (
                  <Link to={`/app/members/${inv.member.id}`}>
                    {inv.member.full_name} ({inv.member.member_code})
                  </Link>
                ) : (
                  'Khách lẻ'
                ),
              },
              { label: 'Chi nhánh', children: inv.branch_name },
              { label: 'Thu ngân', children: inv.issued_by_name || '—' },
              {
                label: 'Khuyến mãi',
                children: inv.promotion_code ? (
                  <span className="code-text">{inv.promotion_code}</span>
                ) : (
                  '—'
                ),
              },
              { label: 'Tạm tính', children: <MoneyText value={inv.subtotal} /> },
              { label: 'Giảm giá', children: <MoneyText value={inv.discount} /> },
              { label: 'Tổng cộng', children: <MoneyText value={inv.total} strong /> },
              {
                label: 'Đã thu / đã hoàn',
                children: (
                  <span>
                    <MoneyText value={inv.paid_amount} /> /{' '}
                    <MoneyText value={inv.refunded_amount} />
                  </span>
                ),
              },
              ...(inv.note ? [{ label: 'Ghi chú', children: inv.note, span: 2 }] : []),
              ...(inv.status === 'VOID'
                ? [{ label: 'Lý do hủy', children: inv.void_reason || '—', span: 2 }]
                : []),
            ]}
          />

          <Divider orientation="left" plain>
            Sản phẩm {canReturn && '– chọn số lượng trả'}
          </Divider>
          <Table<PosOrderLine>
            size="small"
            rowKey="product_id"
            pagination={false}
            dataSource={lines}
            scroll={{ x: 'max-content' }}
            columns={[
              { title: 'Sản phẩm', dataIndex: 'description' },
              { title: 'Đã bán', dataIndex: 'sold_quantity', align: 'right' },
              {
                title: 'Đã trả',
                dataIndex: 'returned_quantity',
                align: 'right',
                render: (v: number) => (v ? formatNumber(v) : '—'),
              },
              {
                title: 'Thành tiền',
                dataIndex: 'amount',
                align: 'right',
                render: (v: number) => <MoneyText value={v} />,
              },
              ...(canReturn
                ? [
                    {
                      title: 'SL trả',
                      key: 'qty',
                      width: 130,
                      render: (_: unknown, l: PosOrderLine) =>
                        l.returnable_quantity > 0 ? (
                          <InputNumber
                            size="small"
                            min={0}
                            max={l.returnable_quantity}
                            precision={0}
                            value={qty[l.product_id] ?? null}
                            placeholder={`≤ ${l.returnable_quantity}`}
                            onChange={(v) => setQty((q) => ({ ...q, [l.product_id]: v }))}
                            style={{ width: '100%' }}
                            aria-label={`Số lượng trả ${l.description}`}
                          />
                        ) : (
                          <Typography.Text type="secondary">Đã trả hết</Typography.Text>
                        ),
                    },
                    {
                      title: 'Hoàn (ước tính)',
                      key: 'refund',
                      align: 'right' as const,
                      render: (_: unknown, l: PosOrderLine) =>
                        qty[l.product_id] ? (
                          <MoneyText value={estimateRefund(l, qty[l.product_id] ?? 0)} />
                        ) : (
                          '—'
                        ),
                    },
                  ]
                : []),
            ]}
          />

          {canReturn && (
            <div style={{ marginTop: 16 }}>
              <Form form={reasonForm} layout="vertical" preserve={false} key={`${invoiceId}-${formKey}`}>
                <Form.Item label="Phương thức hoàn tiền" required>
                  <Segmented<PaymentMethod>
                    options={PAYMENT_METHOD_OPTIONS}
                    value={method}
                    onChange={setMethod}
                  />
                </Form.Item>
                <Form.Item
                  name="reason"
                  label="Lý do trả hàng"
                  rules={[
                    { required: true, whitespace: true, message: 'Vui lòng nhập lý do trả hàng' },
                    { max: 255, message: 'Lý do tối đa 255 ký tự' },
                  ]}
                >
                  <Input maxLength={255} placeholder="VD: Khách đổi ý, hàng lỗi..." />
                </Form.Item>
              </Form>
              {inv.status === 'REFUNDED' && (
                <Alert
                  type="info"
                  showIcon
                  style={{ marginBottom: 12 }}
                  message="Hóa đơn đã hoàn hết tiền – trả hàng chỉ nhập lại kho, không hoàn thêm tiền."
                />
              )}
              <Space wrap>
                <Button
                  danger
                  type="primary"
                  icon={<RollbackOutlined />}
                  disabled={selected.length === 0}
                  loading={ret.isPending}
                  onClick={submitReturn}
                >
                  Trả hàng{refundEstimate > 0 ? ` – hoàn ${formatMoney(refundEstimate)}` : ''}
                </Button>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  Số tiền hoàn chính xác do hệ thống tính theo tỷ lệ thành tiền.
                </Typography.Text>
              </Space>
            </div>
          )}

          {inv.payments.length > 0 && (
            <>
              <Divider orientation="left" plain>
                Thanh toán
              </Divider>
              {inv.payments.map((p) => (
                <div key={p.id}>
                  {PAYMENT_METHOD_LABEL[p.method]}: <MoneyText value={p.amount} />
                  {p.reference && ` · ${p.reference}`} · {formatDateTime(p.paid_at)}
                </div>
              ))}
            </>
          )}
          {inv.refunds.length > 0 && (
            <>
              <Divider orientation="left" plain>
                Hoàn tiền
              </Divider>
              {inv.refunds.map((r) => (
                <div key={r.id}>
                  {PAYMENT_METHOD_LABEL[r.method]}: <MoneyText value={r.amount} /> · {r.reason} ·{' '}
                  {formatDateTime(r.created_at)} · {r.refunded_by_name}
                </div>
              ))}
            </>
          )}
        </>
      )}
      <VoidModal
        open={voidOpen}
        invoice={inv ?? null}
        onClose={() => {
          setVoidOpen(false);
          void queryClient.invalidateQueries({ queryKey: [posApi.key] });
          void queryClient.invalidateQueries({ queryKey: [inventoryApi.key] });
        }}
      />
    </Drawer>
  );
}
