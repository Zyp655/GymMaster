import { useEffect } from 'react';
import { Alert, App, Form, Input, Modal } from 'antd';
import PaymentFields from '@/routes/subscriptions/PaymentFields';
import { DEFAULT_PAYMENT, toPaymentInput } from '@/routes/subscriptions/payment';
import type { PaymentFormValues } from '@/routes/subscriptions/payment';
import {
  useCollectPayment,
  useRefundInvoice,
  useVoidInvoice,
} from '@/routes/subscriptions/queries';
import { setFormError, showError } from '@/utils/errors';
import { formatMoney } from '@/utils/format';
import type { InvoiceLike } from './utils';

export interface InvoiceModalProps {
  open: boolean;
  invoice: InvoiceLike | null;
  onClose: () => void;
}

/** Thu tiền hóa đơn – không vượt quá số còn nợ. */
export function CollectModal({ open, invoice, onClose }: InvoiceModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<PaymentFormValues>();
  const collect = useCollectPayment();
  const due = invoice?.due ?? 0;

  useEffect(() => {
    if (open) form.setFieldsValue({ ...DEFAULT_PAYMENT, amount: due });
  }, [open, due, form]);

  const submit = async () => {
    if (!invoice) return;
    const v = await form.validateFields();
    const data = toPaymentInput(v);
    if (!data) return;
    collect.mutate(
      { id: invoice.id, data },
      {
        onSuccess: (res) => {
          message.success(
            res.status === 'PAID'
              ? `Đã thu đủ hóa đơn ${res.invoice_no}`
              : `Đã thu ${formatMoney(data.amount)}, còn nợ ${formatMoney(res.due)}`,
          );
          onClose();
        },
        onError: (e) => {
          if (!setFormError(form, e, { amount: /số tiền|nợ/i })) showError(e, 'Không thể thu tiền');
        },
      },
    );
  };

  return (
    <Modal
      open={open}
      title={`Thu tiền hóa đơn ${invoice?.invoice_no ?? ''}`}
      okText="Thu tiền"
      cancelText="Hủy"
      onOk={submit}
      confirmLoading={collect.isPending}
      onCancel={onClose}
      forceRender
      width={600}
    >
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message={`Tổng ${formatMoney(invoice?.total)} · đã thu ${formatMoney(invoice?.paid_amount)} · còn nợ ${formatMoney(due)}`}
      />
      <Form form={form} layout="vertical">
        <PaymentFields form={form} name={[]} max={due} allowZero={false} />
      </Form>
    </Modal>
  );
}

/** Hoàn tiền – tối đa số đã thu chưa hoàn. Xác nhận trước khi gửi. */
export function RefundModal({ open, invoice, onClose }: InvoiceModalProps) {
  const { message, modal } = App.useApp();
  const [form] = Form.useForm<PaymentFormValues & { reason: string }>();
  const refund = useRefundInvoice();
  const max = invoice ? Math.max(0, invoice.paid_amount - invoice.refunded_amount) : 0;

  useEffect(() => {
    if (open) form.setFieldsValue({ ...DEFAULT_PAYMENT, amount: max, reason: '' });
  }, [open, max, form]);

  const submit = async () => {
    if (!invoice) return;
    const v = await form.validateFields();
    modal.confirm({
      title: `Hoàn ${formatMoney(v.amount)} cho hóa đơn ${invoice.invoice_no}?`,
      content: 'Hoàn tiền không tự hủy gói tập. Muốn hủy gói kèm hoàn tiền, dùng “Hủy gói”.',
      okText: 'Hoàn tiền',
      okButtonProps: { danger: true },
      cancelText: 'Không',
      onOk: () =>
        refund
          .mutateAsync({
            id: invoice.id,
            data: { amount: v.amount!, method: v.method ?? 'CASH', reason: v.reason.trim() },
          })
          .then(() => {
            message.success('Đã hoàn tiền');
            onClose();
          })
          .catch((e: unknown) => {
            if (!setFormError(form, e, { amount: /số tiền|hoàn/i }))
              showError(e, 'Không thể hoàn tiền');
          }),
    });
  };

  return (
    <Modal
      open={open}
      title={`Hoàn tiền hóa đơn ${invoice?.invoice_no ?? ''}`}
      okText="Hoàn tiền"
      okButtonProps={{ danger: true }}
      cancelText="Hủy"
      onOk={submit}
      confirmLoading={refund.isPending}
      onCancel={onClose}
      forceRender
      width={600}
    >
      <Alert
        type="warning"
        showIcon
        style={{ marginBottom: 16 }}
        message={`Đã thu ${formatMoney(invoice?.paid_amount)} · đã hoàn ${formatMoney(invoice?.refunded_amount)} · có thể hoàn ${formatMoney(max)}`}
      />
      <Form form={form} layout="vertical">
        <PaymentFields
          form={form}
          name={[]}
          max={max}
          allowZero={false}
          amountLabel="Số tiền hoàn"
          fillLabel="Hoàn toàn bộ"
          compact
        />
        <Form.Item
          name="reason"
          label="Lý do hoàn tiền"
          rules={[
            { required: true, whitespace: true, message: 'Vui lòng nhập lý do hoàn tiền' },
            { max: 255, message: 'Lý do tối đa 255 ký tự' },
          ]}
        >
          <Input.TextArea rows={2} />
        </Form.Item>
      </Form>
    </Modal>
  );
}

/** Hủy hóa đơn – chỉ khi chưa thu đồng nào. */
export function VoidModal({ open, invoice, onClose }: InvoiceModalProps) {
  const { message, modal } = App.useApp();
  const [form] = Form.useForm<{ reason: string }>();
  const voidInv = useVoidInvoice();

  useEffect(() => {
    if (open) form.resetFields();
  }, [open, form]);

  const submit = async () => {
    if (!invoice) return;
    const v = await form.validateFields();
    modal.confirm({
      title: `Hủy hóa đơn ${invoice.invoice_no}?`,
      content:
        'Gói tập chờ thanh toán của hóa đơn sẽ bị hủy (khôi phục gói cũ nếu là nâng cấp/chuyển nhượng), lượt khuyến mãi được trả lại.',
      okText: 'Hủy hóa đơn',
      okButtonProps: { danger: true },
      cancelText: 'Không',
      onOk: () =>
        voidInv
          .mutateAsync({ id: invoice.id, reason: v.reason.trim() })
          .then(() => {
            message.success('Đã hủy hóa đơn');
            onClose();
          })
          .catch((e: unknown) => showError(e, 'Không thể hủy hóa đơn')),
    });
  };

  return (
    <Modal
      open={open}
      title={`Hủy hóa đơn ${invoice?.invoice_no ?? ''}`}
      okText="Hủy hóa đơn"
      okButtonProps={{ danger: true }}
      cancelText="Đóng"
      onOk={submit}
      confirmLoading={voidInv.isPending}
      onCancel={onClose}
      forceRender
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="reason"
          label="Lý do hủy"
          rules={[
            { required: true, whitespace: true, message: 'Vui lòng nhập lý do hủy' },
            { max: 255, message: 'Lý do tối đa 255 ký tự' },
          ]}
        >
          <Input.TextArea rows={2} autoFocus />
        </Form.Item>
      </Form>
    </Modal>
  );
}
