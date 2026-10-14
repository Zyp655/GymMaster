import { useEffect, useState } from 'react';
import { Alert, App, Button, Form, Input, Modal } from 'antd';
import type { SubscriptionDetail } from '@/api/subscription';
import MemberSelect from '@/components/MemberSelect';
import { formatDate, formatMoney } from '@/utils/format';
import { setFormError, showError } from '@/utils/errors';
import PaymentFields from './PaymentFields';
import { DEFAULT_PAYMENT, toPaymentInput } from './payment';
import type { PaymentFormValues } from './payment';
import SaleResult from './SaleResult';
import { useTransferSubscription } from './queries';

export interface TransferModalProps {
  open: boolean;
  sub: SubscriptionDetail | null;
  onClose: () => void;
}

interface TransferForm {
  to_member_id?: number;
  note?: string;
  payment: PaymentFormValues;
}

/** Chuyển nhượng phần còn lại của gói cho hội viên khác (phí = `plan.transfer_fee`, người nhận trả). */
export default function TransferModal({ open, sub, onClose }: TransferModalProps) {
  const { message, modal } = App.useApp();
  const [form] = Form.useForm<TransferForm>();
  const [done, setDone] = useState<SubscriptionDetail | null>(null);
  const transfer = useTransferSubscription();
  const fee = sub?.plan.transfer_fee ?? 0;

  useEffect(() => {
    if (!open) return;
    setDone(null);
    form.resetFields();
    form.setFieldsValue({ payment: { ...DEFAULT_PAYMENT, amount: fee } });
  }, [open, form, fee]);

  const submit = async () => {
    if (!sub) return;
    const v = await form.validateFields();
    modal.confirm({
      title: 'Xác nhận chuyển nhượng gói?',
      content: `Gói ${sub.plan_name} của ${sub.member_name} sẽ chuyển sang trạng thái “Đã chuyển nhượng”. Thao tác không thể hoàn tác (trừ khi hủy hóa đơn phí chưa thu).`,
      okText: 'Chuyển nhượng',
      cancelText: 'Hủy',
      onOk: () =>
        transfer
          .mutateAsync({
            id: sub.id,
            data: {
              to_member_id: v.to_member_id!,
              note: v.note?.trim() || '',
              payment: fee > 0 ? toPaymentInput(v.payment) : undefined,
            },
          })
          .then((res) => {
            message.success('Đã chuyển nhượng gói tập');
            setDone(res);
          })
          .catch((e: unknown) => {
            if (!setFormError(form, e, { to_member_id: /hội viên/i }))
              showError(e, 'Không thể chuyển nhượng');
          }),
    });
  };

  return (
    <Modal
      open={open}
      title={`Chuyển nhượng gói – ${sub?.member_name ?? ''}`}
      width={680}
      onCancel={() => !transfer.isPending && onClose()}
      maskClosable={false}
      forceRender
      footer={
        done
          ? null
          : [
              <Button key="c" onClick={onClose}>
                Hủy
              </Button>,
              <Button key="ok" type="primary" loading={transfer.isPending} onClick={submit}>
                Chuyển nhượng
              </Button>,
            ]
      }
    >
      {done ? (
        <SaleResult title="Chuyển nhượng thành công" sub={done} onClose={onClose} />
      ) : (
        sub && (
          <Form form={form} layout="vertical" initialValues={{ payment: DEFAULT_PAYMENT }}>
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
              message={`Người nhận được dùng phần còn lại đến ${formatDate(sub.end_date)}${
                sub.remaining_sessions !== null ? ` (${sub.remaining_sessions} buổi)` : ''
              }.`}
              description={
                fee > 0
                  ? `Phí chuyển nhượng ${formatMoney(fee)} – lập hóa đơn cho người nhận; gói mới chờ thanh toán đến khi thu đủ phí.`
                  : 'Gói không thu phí chuyển nhượng – gói mới kích hoạt ngay.'
              }
            />
            <Form.Item
              name="to_member_id"
              label="Hội viên nhận"
              rules={[
                { required: true, message: 'Vui lòng chọn hội viên nhận' },
                {
                  validator: (_, v) =>
                    v && v === sub.member_id
                      ? Promise.reject(new Error('Không thể chuyển cho chính chủ gói'))
                      : Promise.resolve(),
                },
              ]}
            >
              <MemberSelect activeOnly />
            </Form.Item>
            {fee > 0 && <PaymentFields form={form} max={fee} amountLabel="Thu phí chuyển nhượng" />}
            <Form.Item
              name="note"
              label="Ghi chú"
              rules={[{ max: 500, message: 'Ghi chú tối đa 500 ký tự' }]}
            >
              <Input.TextArea rows={2} />
            </Form.Item>
          </Form>
        )
      )}
    </Modal>
  );
}
