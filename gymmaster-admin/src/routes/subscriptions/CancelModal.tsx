import { useEffect } from 'react';
import { Alert, App, Checkbox, Form, Input, Modal } from 'antd';
import type { SubscriptionDetail } from '@/api/subscription';
import { showError, validateForm } from '@/utils/errors';
import { formatMoney } from '@/utils/format';
import PaymentFields from './PaymentFields';
import { DEFAULT_PAYMENT } from './payment';
import type { PaymentFormValues } from './payment';
import { useCancelSubscription } from './queries';

export interface CancelModalProps {
  open: boolean;
  sub: SubscriptionDetail | null;
  onClose: () => void;
}

interface CancelForm {
  reason: string;
  withRefund?: boolean;
  refund: PaymentFormValues;
}

/** Hủy gói (lý do bắt buộc) + hoàn tiền tùy chọn trên hóa đơn của gói. */
export default function CancelModal({ open, sub, onClose }: CancelModalProps) {
  const { message, modal } = App.useApp();
  const [form] = Form.useForm<CancelForm>();
  const cancel = useCancelSubscription();
  const withRefund = Form.useWatch('withRefund', form);

  const inv = sub?.invoice;
  const refundable = inv ? Math.max(0, inv.paid_amount - inv.refunded_amount) : 0;
  // Hóa đơn đã thu một phần (hoặc chưa thu nhưng gói không còn PENDING) → phần còn nợ sẽ được xóa khi hủy gói.
  const willVoid = sub?.status === 'PENDING' && !!inv && inv.paid_amount === 0;
  const writeOff =
    inv && !willVoid && inv.status !== 'VOID' ? Math.max(0, inv.total - inv.paid_amount) : 0;

  useEffect(() => {
    if (open) form.resetFields();
  }, [open, form]);

  const submit = async () => {
    if (!sub) return;
    const v = await validateForm(form);
    if (!v) return;
    const refund =
      v.withRefund && v.refund?.amount
        ? { amount: v.refund.amount, method: v.refund.method ?? 'CASH' }
        : undefined;
    modal.confirm({
      title: `Hủy gói ${sub.plan_name} của ${sub.member_name}?`,
      content: [
        refund ? `Hoàn ${formatMoney(refund.amount)} cho hội viên.` : 'Gói sẽ chuyển sang “Đã hủy”.',
        writeOff > 0 ? `Sẽ xóa nợ ${formatMoney(writeOff)} trên hóa đơn ${inv?.invoice_no}.` : '',
        'Thao tác không thể hoàn tác.',
      ]
        .filter(Boolean)
        .join(' '),
      okText: 'Hủy gói',
      okButtonProps: { danger: true },
      cancelText: 'Không',
      onOk: () =>
        cancel
          .mutateAsync({ id: sub.id, data: { reason: v.reason.trim(), refund } })
          .then(() => {
            message.success('Đã hủy gói tập');
            onClose();
          })
          .catch((e: unknown) => showError(e, 'Không thể hủy gói')),
    });
  };

  return (
    <Modal
      open={open}
      title={`Hủy gói – ${sub?.member_name ?? ''}`}
      okText="Hủy gói"
      okButtonProps={{ danger: true }}
      cancelText="Đóng"
      onOk={submit}
      confirmLoading={cancel.isPending}
      onCancel={onClose}
      forceRender
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={{ withRefund: false, refund: { ...DEFAULT_PAYMENT, amount: refundable } }}
      >
        {writeOff > 0 && (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
            message={`Sẽ xóa nợ ${formatMoney(writeOff)} trên hóa đơn ${inv?.invoice_no}`}
            description="Phần chưa thu được ghi là “Xóa nợ do hủy gói” (không tính là giảm giá) và hóa đơn không thu tiếp được nữa. Các kỳ trả góp chưa thu (nếu có) bị hủy."
          />
        )}
        {willVoid && inv && (
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            message={`Hóa đơn ${inv.invoice_no} chưa thu tiền sẽ bị hủy theo, lượt dùng mã khuyến mãi (nếu có) được trả lại.`}
          />
        )}
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
        {refundable > 0 && (
          <>
            <Form.Item name="withRefund" valuePropName="checked">
              <Checkbox>
                Hoàn tiền cho hội viên (tối đa {formatMoney(refundable)} trên hóa đơn{' '}
                {inv?.invoice_no})
              </Checkbox>
            </Form.Item>
            {withRefund && (
              <PaymentFields
                form={form}
                name={['refund']}
                max={refundable}
                allowZero={false}
                amountLabel="Số tiền hoàn"
                fillLabel="Hoàn toàn bộ"
                compact
              />
            )}
          </>
        )}
      </Form>
    </Modal>
  );
}
