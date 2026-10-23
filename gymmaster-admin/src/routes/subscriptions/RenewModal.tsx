import { useEffect, useState } from 'react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { Alert, App, Button, Col, Form, Input, Modal, Row, Select, Space, Spin } from 'antd';
import membershipPlanApi from '@/api/plan';
import subscriptionApi from '@/api/subscription';
import type { Subscription, SubscriptionDetail } from '@/api/subscription';
import { getErrorMessage, showError, validateForm } from '@/utils/errors';
import { formatMoney } from '@/utils/format';
import PaymentFields from './PaymentFields';
import { DEFAULT_PAYMENT, toPaymentInput } from './payment';
import type { PaymentFormValues } from './payment';
import QuoteSummary from './QuoteSummary';
import SaleResult from './SaleResult';
import { useRenewSubscription } from './queries';

export interface RenewModalProps {
  open: boolean;
  sub: Subscription | null;
  onClose: () => void;
}

/** Gia hạn: chọn gói (mặc định gói hiện tại) + mã KM → báo giá → thu tiền. */
export default function RenewModal({ open, sub, onClose }: RenewModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<{ payment: PaymentFormValues; note?: string }>();
  const [planId, setPlanId] = useState<number>();
  const [promoInput, setPromoInput] = useState('');
  const [promoCode, setPromoCode] = useState('');
  const [done, setDone] = useState<SubscriptionDetail | null>(null);
  const renew = useRenewSubscription();

  useEffect(() => {
    if (!open || !sub) return;
    setPlanId(sub.plan_id);
    setPromoInput('');
    setPromoCode('');
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

  const body = { plan_id: planId, promotion_code: promoCode };
  const quoteQuery = useQuery({
    queryKey: [subscriptionApi.key, 'renew-quote', sub?.id, body],
    queryFn: () => subscriptionApi.renewQuote(sub!.id, body),
    enabled: open && !!sub && !!planId,
    placeholderData: keepPreviousData,
    retry: false,
    staleTime: 0,
  });
  const quote = quoteQuery.isError ? undefined : quoteQuery.data;
  const quoteError = quoteQuery.isError ? getErrorMessage(quoteQuery.error) : '';
  const promoError = !!promoCode && /khuyến mãi|mã|giá trị tối thiểu/i.test(quoteError);

  // Báo giá đổi → số tiền thu mặc định = tổng.
  useEffect(() => {
    if (quote) form.setFieldValue(['payment', 'amount'], quote.total);
  }, [quote?.total, form, quote]);

  const submit = async () => {
    if (!sub || !quote) return;
    const v = await validateForm(form);
    if (!v) return;
    renew.mutate(
      {
        id: sub.id,
        data: { ...body, note: v.note?.trim() || '', payment: toPaymentInput(v.payment) },
      },
      {
        onSuccess: (res) => {
          message.success('Đã gia hạn gói tập');
          setDone(res);
        },
        onError: (e) => showError(e, 'Không thể gia hạn'),
      },
    );
  };

  // Gói đang chọn có thể đã ngừng bán (không còn trong options) → vẫn hiện để gia hạn đúng gói.
  const planOptions = (plans.data ?? []).map((p) => ({
    value: p.id,
    label: `${p.code} – ${p.name} (${formatMoney(p.price)})`,
  }));
  if (sub && !planOptions.some((o) => o.value === sub.plan_id)) {
    planOptions.unshift({ value: sub.plan_id, label: `${sub.plan_code} – ${sub.plan_name}` });
  }

  return (
    <Modal
      open={open}
      title={`Gia hạn gói – ${sub?.member_name ?? ''}`}
      width={720}
      onCancel={() => !renew.isPending && onClose()}
      maskClosable={false}
      destroyOnHidden
      footer={
        done
          ? null
          : [
              <Button key="c" onClick={onClose} disabled={renew.isPending}>
                Hủy
              </Button>,
              <Button
                key="ok"
                color="orange"
                variant="solid"
                loading={renew.isPending}
                disabled={!quote || quoteQuery.isFetching}
                onClick={submit}
              >
                Xác nhận gia hạn
              </Button>,
            ]
      }
    >
      {done ? (
        <SaleResult title="Gia hạn thành công" sub={done} onClose={onClose} />
      ) : (
        <Form form={form} layout="vertical" initialValues={{ payment: DEFAULT_PAYMENT }}>
          <Row gutter={12}>
            <Col xs={24} sm={14}>
              <Form.Item label="Gói gia hạn" required>
                <Select
                  value={planId}
                  onChange={setPlanId}
                  options={planOptions}
                  loading={plans.isLoading}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={10}>
              <Form.Item
                label="Mã khuyến mãi"
                validateStatus={promoError ? 'error' : quote?.promotion ? 'success' : undefined}
                help={promoError ? quoteError : quote?.promotion?.name}
              >
                <Space.Compact style={{ width: '100%' }}>
                  <Input
                    value={promoInput}
                    maxLength={30}
                    allowClear
                    onChange={(e) => {
                      const val = e.target.value.toUpperCase();
                      setPromoInput(val);
                      if (!val) setPromoCode('');
                    }}
                    onPressEnter={() => setPromoCode(promoInput.trim())}
                  />
                  <Button
                    onClick={() => setPromoCode(promoInput.trim())}
                    disabled={!promoInput.trim()}
                  >
                    Áp dụng
                  </Button>
                </Space.Compact>
              </Form.Item>
            </Col>
          </Row>
          <Spin spinning={quoteQuery.isFetching}>
            {quoteError && !promoError ? (
              <Alert type="error" showIcon message={quoteError} />
            ) : quote ? (
              <QuoteSummary quote={quote} />
            ) : null}
          </Spin>
          {quote && (
            <div style={{ marginTop: 16 }}>
              {quote.total > 0 && <PaymentFields form={form} max={quote.total} />}
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
