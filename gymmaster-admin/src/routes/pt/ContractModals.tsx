import { useEffect, useState } from 'react';
import {
  Alert,
  App,
  Col,
  Form,
  Input,
  Modal,
  Row,
  Segmented,
  Select,
  Spin,
  Switch,
  Table,
  Typography,
} from 'antd';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import { COUNTER_PAYMENT_METHOD_OPTIONS } from '@/api/invoice';
import type { PaymentMethod } from '@/api/invoice';
import ptApi from '@/api/pt';
import type { PtContractDetail, SessionConflict } from '@/api/pt';
import MoneyInput from '@/components/Money/MoneyInput';
import MoneyText from '@/components/Money/MoneyText';
import { getErrorMessage, setFormError, showError, validateForm } from '@/utils/errors';
import { formatMoney, normalizeText } from '@/utils/format';
import { useCancelContract, useChangeTrainer, usePtTrainerOptions } from './queries';

export interface ContractModalProps {
  contract: PtContractDetail | null;
  onClose: () => void;
}

const filterByLabel = (input: string, option?: { label?: unknown }) =>
  normalizeText(String(option?.label ?? '')).includes(normalizeText(input));

/* ================================================================== Đổi HLV */

/** Đổi HLV: chọn HLV mới → kiểm tra trùng lịch (`/change-trainer/check`) → xác nhận. */
export function ChangeTrainerModal({ contract, onClose }: ContractModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<{ trainer_id?: number; note?: string }>();
  const [trainerId, setTrainerId] = useState<number>();
  const trainers = usePtTrainerOptions(contract?.branch_id, !!contract);
  const mutation = useChangeTrainer();

  useEffect(() => {
    if (!contract) return;
    form.resetFields();
    setTrainerId(undefined);
  }, [contract, form]);

  const check = useQuery({
    queryKey: [ptApi.key, 'change-trainer-check', contract?.id, trainerId],
    queryFn: () => ptApi.checkChangeTrainer(contract!.id, { trainer_id: trainerId! }),
    enabled: !!contract && !!trainerId,
    retry: false,
    staleTime: 0,
  });

  const submit = async () => {
    if (!contract) return;
    const v = await validateForm(form);
    if (!v) return;
    if (!v.trainer_id) return;
    mutation.mutate(
      { id: contract.id, data: { trainer_id: v.trainer_id, note: v.note?.trim() ?? '' } },
      {
        onSuccess: (res) => {
          message.success(`Đã đổi HLV sang ${res.trainer_name}`);
          onClose();
        },
        onError: (e) => {
          if (!setFormError(form, e, { trainer_id: /HLV|trùng/i }))
            showError(e, 'Không thể đổi HLV');
        },
      },
    );
  };

  const result = check.data;
  const conflicts = result?.conflicts ?? [];

  return (
    <Modal
      open={!!contract}
      title="Đổi huấn luyện viên"
      okText="Xác nhận đổi HLV"
      onOk={submit}
      okButtonProps={{ disabled: !result?.ok || check.isFetching }}
      confirmLoading={mutation.isPending}
      onCancel={onClose}
      destroyOnHidden
      width={640}
    >
      {contract && (
        <Typography.Paragraph type="secondary">
          Hợp đồng #{contract.id} · {contract.member_name} · {contract.package_name} · HLV hiện tại:{' '}
          <b>{contract.trainer_name}</b>
        </Typography.Paragraph>
      )}
      <Form form={form} layout="vertical">
        <Form.Item
          name="trainer_id"
          label="HLV mới"
          rules={[{ required: true, message: 'Vui lòng chọn huấn luyện viên mới' }]}
        >
          <Select
            showSearch
            placeholder="Chọn HLV tại chi nhánh của hợp đồng"
            loading={trainers.isLoading}
            filterOption={filterByLabel}
            options={(trainers.data ?? [])
              .filter((t) => t.id !== contract?.trainer_id)
              .map((t) => ({ value: t.id, label: t.full_name }))}
            onChange={(v: number) => setTrainerId(v)}
          />
        </Form.Item>
        <Form.Item
          name="note"
          label="Ghi chú"
          rules={[{ max: 255, message: 'Ghi chú tối đa 255 ký tự' }]}
        >
          <Input placeholder="VD: HLV cũ nghỉ phép dài" />
        </Form.Item>
      </Form>
      {trainerId && (
        <Spin spinning={check.isFetching}>
          {check.isError ? (
            <Alert type="error" showIcon message={getErrorMessage(check.error)} />
          ) : result ? (
            result.ok ? (
              <Alert
                type="success"
                showIcon
                message={
                  result.moved_sessions > 0
                    ? `Không trùng lịch – ${result.moved_sessions} buổi sắp tới sẽ chuyển sang HLV ${result.trainer_name}.`
                    : `Không có buổi sắp tới cần chuyển – có thể đổi sang HLV ${result.trainer_name}.`
                }
              />
            ) : (
              <>
                <Alert
                  type="error"
                  showIcon
                  style={{ marginBottom: 8 }}
                  message={`${conflicts.length}/${result.moved_sessions} buổi sắp tới bị trùng lịch của HLV ${result.trainer_name}`}
                  description="Hãy đổi giờ hoặc hủy các buổi này trước khi đổi HLV."
                />
                <Table<SessionConflict>
                  rowKey="session_id"
                  size="small"
                  pagination={false}
                  dataSource={conflicts}
                  columns={[
                    { title: 'Buổi', dataIndex: 'session_id', render: (v: number) => `#${v}` },
                    {
                      title: 'Thời gian',
                      key: 'time',
                      render: (_, c) =>
                        `${dayjs(c.start_at).format('HH:mm')}–${dayjs(c.end_at).format('HH:mm')} ${dayjs(c.start_at).format('DD/MM/YYYY')}`,
                    },
                    { title: 'Lý do', dataIndex: 'reason' },
                  ]}
                />
              </>
            )
          ) : null}
        </Spin>
      )}
    </Modal>
  );
}

/* ================================================================== Hủy hợp đồng */

interface CancelForm {
  reason: string;
  refund: boolean;
  amount?: number | null;
  method: PaymentMethod;
}

/** Hủy hợp đồng PT: lý do + hoàn tiền tùy chọn (gợi ý = giá/buổi × số buổi còn lại). */
export function CancelContractModal({ contract, onClose }: ContractModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<CancelForm>();
  const mutation = useCancelContract();
  const inv = contract?.invoice;
  const refundable = inv ? Math.max(0, inv.paid_amount - inv.refunded_amount) : 0;
  const suggested = contract
    ? Math.min(refundable, contract.price_per_session * contract.remaining_sessions)
    : 0;
  const refund = Form.useWatch('refund', form);

  useEffect(() => {
    if (contract)
      form.setFieldsValue({ reason: '', refund: false, amount: suggested, method: 'CASH' });
  }, [contract, form, suggested]);

  const submit = async () => {
    if (!contract) return;
    const v = await validateForm(form);
    if (!v) return;
    mutation.mutate(
      {
        id: contract.id,
        data: {
          reason: v.reason.trim(),
          refund: v.refund && v.amount ? { amount: v.amount, method: v.method } : undefined,
        },
      },
      {
        onSuccess: () => {
          message.success('Đã hủy hợp đồng PT');
          onClose();
        },
        onError: (e) => {
          if (!setFormError(form, e, { amount: /hoàn|số tiền/i }))
            showError(e, 'Không thể hủy hợp đồng');
        },
      },
    );
  };

  return (
    <Modal
      open={!!contract}
      title="Hủy hợp đồng PT"
      okText="Xác nhận hủy hợp đồng"
      okButtonProps={{ danger: true }}
      onOk={submit}
      confirmLoading={mutation.isPending}
      onCancel={onClose}
      destroyOnHidden
      width={600}
    >
      {contract && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          message={`${contract.member_name} · ${contract.package_name} · còn ${contract.remaining_sessions}/${contract.total_sessions} buổi`}
          description={
            <>
              Mọi buổi đã đặt chưa xác nhận ({contract.scheduled_sessions} buổi) sẽ bị hủy. Muốn
              tính công cho buổi đã dạy, hãy xác nhận hoàn thành trước.
              {inv && (
                <div>
                  Hóa đơn {inv.invoice_no}: đã thu <MoneyText value={inv.paid_amount} />
                  {inv.refunded_amount > 0 && (
                    <>
                      , đã hoàn <MoneyText value={inv.refunded_amount} />
                    </>
                  )}
                  .
                </div>
              )}
            </>
          }
        />
      )}
      <Form form={form} layout="vertical">
        <Form.Item
          name="reason"
          label="Lý do hủy"
          rules={[
            { required: true, whitespace: true, message: 'Vui lòng nhập lý do hủy' },
            { max: 255, message: 'Lý do tối đa 255 ký tự' },
          ]}
        >
          <Input.TextArea rows={2} placeholder="VD: Khách chuyển nơi ở" autoFocus />
        </Form.Item>
        <Form.Item name="refund" label="Hoàn tiền" valuePropName="checked">
          <Switch disabled={refundable <= 0} />
        </Form.Item>
        {refundable <= 0 && (
          <Typography.Paragraph type="secondary" style={{ marginTop: -16 }}>
            Hóa đơn chưa thu tiền – không có gì để hoàn
            {contract?.status === 'PENDING' ? ' (hóa đơn sẽ được hủy).' : '.'}
          </Typography.Paragraph>
        )}
        {refund && (
          <Row gutter={12}>
            <Col xs={24} sm={12}>
              <Form.Item
                name="amount"
                label="Số tiền hoàn"
                rules={[
                  { required: true, message: 'Vui lòng nhập số tiền hoàn' },
                  {
                    validator: (_, v: number | null | undefined) =>
                      v !== null && v !== undefined && (v <= 0 || v > refundable)
                        ? Promise.reject(
                            new Error(`Số tiền phải lớn hơn 0 và ≤ ${formatMoney(refundable)}`),
                          )
                        : Promise.resolve(),
                  },
                ]}
                extra={`Gợi ý: ${formatMoney(contract?.price_per_session)} × ${contract?.remaining_sessions} buổi còn lại = ${formatMoney(suggested)}`}
              >
                <MoneyInput max={refundable} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item name="method" label="Phương thức hoàn">
                <Segmented block options={COUNTER_PAYMENT_METHOD_OPTIONS} />
              </Form.Item>
            </Col>
          </Row>
        )}
      </Form>
    </Modal>
  );
}
