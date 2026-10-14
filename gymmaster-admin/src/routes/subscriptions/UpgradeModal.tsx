import { useEffect, useState } from 'react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Descriptions,
  Divider,
  Form,
  Input,
  Modal,
  Spin,
  Typography,
} from 'antd';
import membershipPlanApi from '@/api/plan';
import subscriptionApi from '@/api/subscription';
import type { Subscription, SubscriptionDetail } from '@/api/subscription';
import MoneyText from '@/components/Money/MoneyText';
import { getErrorMessage, showError } from '@/utils/errors';
import { formatDate } from '@/utils/format';
import PaymentFields from './PaymentFields';
import { DEFAULT_PAYMENT, toPaymentInput } from './payment';
import type { PaymentFormValues } from './payment';
import PlanCards from './PlanCards';
import SaleResult from './SaleResult';
import { useUpgradeSubscription } from './queries';

export interface UpgradeModalProps {
  open: boolean;
  sub: Subscription | null;
  onClose: () => void;
}

/** Nâng cấp: chọn gói mới → upgrade/quote (khấu trừ giá trị còn lại) → thu tiền. */
export default function UpgradeModal({ open, sub, onClose }: UpgradeModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<{ payment: PaymentFormValues; note?: string }>();
  const [planId, setPlanId] = useState<number>();
  const [done, setDone] = useState<SubscriptionDetail | null>(null);
  const upgrade = useUpgradeSubscription();

  useEffect(() => {
    if (!open) return;
    setPlanId(undefined);
    setDone(null);
    form.setFieldsValue({ payment: { ...DEFAULT_PAYMENT }, note: '' });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, sub?.id]);

  const plans = useQuery({
    queryKey: [membershipPlanApi.key, 'options'],
    queryFn: () => membershipPlanApi.options(),
    enabled: open,
    staleTime: 60_000,
  });

  const quoteQuery = useQuery({
    queryKey: [subscriptionApi.key, 'upgrade-quote', sub?.id, planId],
    queryFn: () => subscriptionApi.upgradeQuote(sub!.id, planId!),
    enabled: open && !!sub && !!planId,
    placeholderData: keepPreviousData,
    retry: false,
    staleTime: 0,
  });
  const quote = quoteQuery.isError || !planId ? undefined : quoteQuery.data;

  useEffect(() => {
    if (quote) form.setFieldValue(['payment', 'amount'], quote.amount_due);
  }, [quote, form]);

  const submit = async () => {
    if (!sub || !quote || !planId) return;
    const v = await form.validateFields();
    upgrade.mutate(
      {
        id: sub.id,
        data: { plan_id: planId, note: v.note?.trim() || '', payment: toPaymentInput(v.payment) },
      },
      {
        onSuccess: (res) => {
          message.success('Đã nâng cấp gói tập');
          setDone(res);
        },
        onError: (e) => showError(e, 'Không thể nâng cấp'),
      },
    );
  };

  return (
    <Modal
      open={open}
      title={`Nâng cấp gói – ${sub?.member_name ?? ''}`}
      width={860}
      onCancel={() => !upgrade.isPending && onClose()}
      maskClosable={false}
      destroyOnHidden
      footer={
        done
          ? null
          : [
              <Button key="c" onClick={onClose} disabled={upgrade.isPending}>
                Hủy
              </Button>,
              <Button
                key="ok"
                color="orange"
                variant="solid"
                loading={upgrade.isPending}
                disabled={!quote || quoteQuery.isFetching}
                onClick={submit}
              >
                Xác nhận nâng cấp
              </Button>,
            ]
      }
    >
      {done ? (
        <SaleResult title="Nâng cấp thành công" sub={done} onClose={onClose} />
      ) : (
        <Form form={form} layout="vertical" initialValues={{ payment: DEFAULT_PAYMENT }}>
          <Typography.Paragraph type="secondary">
            Gói hiện tại: <b>{sub?.plan_name}</b>. Giá trị còn lại của gói cũ được khấu trừ vào giá
            gói mới; gói cũ chuyển sang “Đã hủy”, gói mới bắt đầu từ hôm nay.
          </Typography.Paragraph>
          <PlanCards
            plans={plans.data ?? []}
            loading={plans.isLoading}
            value={planId}
            onChange={(p) => setPlanId(p.id)}
            disabledIds={sub ? [sub.plan_id] : []}
          />
          {planId && (
            <Spin spinning={quoteQuery.isFetching}>
              <Divider style={{ margin: '16px 0' }} />
              {quoteQuery.isError ? (
                <Alert type="error" showIcon message={getErrorMessage(quoteQuery.error)} />
              ) : quote ? (
                <Descriptions
                  size="small"
                  bordered
                  column={{ xs: 1, sm: 2 }}
                  items={[
                    {
                      key: 'cur',
                      label: 'Gói cũ',
                      children: `${quote.current_plan.code} – ${quote.current_plan.name}`,
                    },
                    {
                      key: 'new',
                      label: 'Gói mới',
                      children: `${quote.new_plan.code} – ${quote.new_plan.name}`,
                    },
                    {
                      key: 'cprice',
                      label: 'Giá trị gói cũ',
                      children: <MoneyText value={quote.current_price} />,
                    },
                    {
                      key: 'remain',
                      label: 'Còn lại',
                      children:
                        quote.total_sessions !== null
                          ? `${quote.remaining_sessions}/${quote.total_sessions} buổi`
                          : `${quote.remaining_days}/${quote.total_days} ngày`,
                    },
                    {
                      key: 'nprice',
                      label: 'Giá gói mới',
                      children: <MoneyText value={quote.new_price} />,
                    },
                    {
                      key: 'credit',
                      label: 'Khấu trừ giá trị còn lại',
                      children: <MoneyText value={-quote.remaining_value} colored />,
                    },
                    {
                      key: 'period',
                      label: 'Thời gian gói mới',
                      children: `${formatDate(quote.start_date)} – ${formatDate(quote.end_date)}`,
                    },
                    {
                      key: 'due',
                      label: 'Phải thu',
                      children: (
                        <Typography.Text strong style={{ fontSize: 18, color: '#F97316' }}>
                          <MoneyText value={quote.amount_due} />
                        </Typography.Text>
                      ),
                    },
                  ]}
                />
              ) : null}
            </Spin>
          )}
          {quote && (
            <div style={{ marginTop: 16 }}>
              {quote.amount_due > 0 ? (
                <PaymentFields form={form} max={quote.amount_due} />
              ) : (
                <Alert
                  type="success"
                  showIcon
                  style={{ marginBottom: 16 }}
                  message="Giá trị còn lại đủ bù gói mới – không cần thu thêm."
                />
              )}
              <Form.Item
                name="note"
                label="Ghi chú"
                rules={[{ max: 500, message: 'Ghi chú tối đa 500 ký tự' }]}
              >
                <Input.TextArea rows={2} />
              </Form.Item>
            </div>
          )}
        </Form>
      )}
    </Modal>
  );
}
