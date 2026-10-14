import { useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Col,
  DatePicker,
  Divider,
  Empty,
  Form,
  Input,
  List,
  Modal,
  Row,
  Space,
  Spin,
  Steps,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { MemberOption } from '@/api/member';
import membershipPlanApi from '@/api/plan';
import subscriptionApi from '@/api/subscription';
import type { SubscriptionDetail } from '@/api/subscription';
import BranchSelect from '@/components/BranchSelect';
import MemberSelect from '@/components/MemberSelect';
import MoneyText from '@/components/Money/MoneyText';
import StatusTag from '@/components/StatusTag';
import { SUBSCRIPTION_STATUS_MAP } from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { getErrorMessage, showError } from '@/utils/errors';
import { DATE_FORMAT, formatDate, formatMoney, toApiDate } from '@/utils/format';
import PaymentFields from './PaymentFields';
import { DEFAULT_PAYMENT, toPaymentInput } from './payment';
import type { PaymentFormValues } from './payment';
import PlanCards from './PlanCards';
import QuoteSummary from './QuoteSummary';
import SaleResult from './SaleResult';
import { useSellSubscription } from './queries';
import { isLiveSubscription, remainingText } from './utils';

export interface SellSubscriptionModalProps {
  open: boolean;
  onClose: () => void;
  /** Hội viên chọn sẵn (mở từ hồ sơ hội viên). */
  member?: MemberOption | null;
}

interface PayForm {
  payment: PaymentFormValues;
  note?: string;
}

/** Danh sách gói còn hiệu lực của hội viên (bước 1). */
function MemberCurrentSubscriptions({ memberId }: { memberId: number }) {
  const query = useQuery({
    queryKey: [subscriptionApi.key, 'member', memberId],
    queryFn: () => subscriptionApi.byMember(memberId),
  });
  const live = (query.data ?? []).filter(isLiveSubscription);

  if (query.isLoading) return <Spin style={{ display: 'block', margin: '16px auto' }} />;
  if (query.isError)
    return <Alert type="warning" showIcon message={getErrorMessage(query.error)} />;
  return (
    <>
      <Typography.Text strong>Gói đang có</Typography.Text>
      {live.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="Hội viên chưa có gói còn hiệu lực"
        />
      ) : (
        <List
          size="small"
          dataSource={live}
          renderItem={(s) => (
            <List.Item>
              <Space wrap size={[8, 4]}>
                <StatusTag status={s.status} map={SUBSCRIPTION_STATUS_MAP} />
                <b>{s.plan_name}</b>
                <span>
                  {formatDate(s.start_date)} – {formatDate(s.end_date)}
                </span>
                <Typography.Text type="secondary">Còn {remainingText(s)}</Typography.Text>
                <Typography.Text type="secondary">{s.branch_name}</Typography.Text>
              </Space>
            </List.Item>
          )}
        />
      )}
      {live.some((s) => s.status !== 'PENDING') && (
        <Alert
          type="info"
          showIcon
          style={{ marginTop: 8 }}
          message="Nếu không chọn ngày bắt đầu, gói mới cùng phạm vi sẽ tự nối tiếp sau gói đang chạy."
        />
      )}
    </>
  );
}

/**
 * Bán gói tập 3 bước: hội viên → gói (+ chi nhánh, ngày bắt đầu, mã KM, báo giá) → thanh toán.
 * Mở từ trang Đăng ký gói tập, hồ sơ hội viên, nút "Bán gói" ở header.
 */
export default function SellSubscriptionModal({
  open,
  onClose,
  member,
}: SellSubscriptionModalProps) {
  const { message } = App.useApp();
  const { branchId: workingBranch } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const [step, setStep] = useState(0);
  const [memberId, setMemberId] = useState<number | undefined>(member?.id);
  const [memberObj, setMemberObj] = useState<MemberOption | undefined>(member ?? undefined);
  const [planId, setPlanId] = useState<number>();
  const [branchId, setBranchId] = useState<number | undefined>();
  const [startDate, setStartDate] = useState<Dayjs | null>(null);
  const [promoInput, setPromoInput] = useState('');
  const [promoCode, setPromoCode] = useState('');
  const [done, setDone] = useState<SubscriptionDetail | null>(null);
  const [form] = Form.useForm<PayForm>();
  const sell = useSellSubscription();

  const defaultBranch =
    workingBranch ?? branches.find((b) => b.status !== 'INACTIVE')?.id ?? undefined;

  // Mở lại → đặt lại toàn bộ.
  useEffect(() => {
    if (!open) return;
    setStep(0);
    setMemberId(member?.id);
    setMemberObj(member ?? undefined);
    setPlanId(undefined);
    setBranchId(undefined);
    setStartDate(null);
    setPromoInput('');
    setPromoCode('');
    setDone(null);
    sell.reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, member?.id]);

  useEffect(() => {
    if (open && branchId === undefined && defaultBranch) setBranchId(defaultBranch);
  }, [open, branchId, defaultBranch]);

  const plansQuery = useQuery({
    queryKey: [membershipPlanApi.key, 'options'],
    queryFn: () => membershipPlanApi.options(),
    enabled: open,
    staleTime: 60_000,
  });

  const quoteBody = useMemo(
    () =>
      memberId && planId && branchId
        ? {
            member_id: memberId,
            plan_id: planId,
            branch_id: branchId,
            start_date: toApiDate(startDate) ?? '',
            promotion_code: promoCode,
          }
        : null,
    [memberId, planId, branchId, startDate, promoCode],
  );

  const quoteQuery = useQuery({
    queryKey: [subscriptionApi.key, 'quote', quoteBody],
    queryFn: () => subscriptionApi.quote(quoteBody!),
    enabled: open && !!quoteBody,
    placeholderData: keepPreviousData,
    retry: false,
    staleTime: 0,
  });
  const quote = quoteQuery.isError ? undefined : quoteQuery.data;
  const quoteError = quoteQuery.isError ? getErrorMessage(quoteQuery.error) : '';
  const promoError = !!promoCode && /khuyến mãi|mã|giá trị tối thiểu/i.test(quoteError);

  const goPayment = () => {
    if (!quote) return;
    form.setFieldsValue({ payment: { ...DEFAULT_PAYMENT, amount: quote.total }, note: '' });
    setStep(2);
  };

  const submit = async () => {
    if (!quoteBody || !quote) return;
    const values = await form.validateFields();
    sell.mutate(
      { ...quoteBody, note: values.note?.trim() || '', payment: toPaymentInput(values.payment) },
      {
        onSuccess: (res) => {
          message.success('Đã bán gói tập');
          setDone(res);
        },
        onError: (e) => showError(e, 'Không thể bán gói'),
      },
    );
  };

  const steps = [{ title: 'Hội viên' }, { title: 'Chọn gói' }, { title: 'Thanh toán' }];

  let body: ReactNode;
  if (done) {
    body = (
      <SaleResult
        title="Bán gói thành công"
        sub={done}
        onClose={onClose}
        extra={
          <Button
            onClick={() => {
              setDone(null);
              setStep(1);
              setPlanId(undefined);
              setPromoCode('');
              setPromoInput('');
              setStartDate(null);
            }}
          >
            Bán thêm gói
          </Button>
        }
      />
    );
  } else if (step === 0) {
    body = (
      <>
        <Form layout="vertical">
          <Form.Item label="Hội viên" required>
            <MemberSelect
              value={memberId}
              initialMember={memberObj}
              activeOnly
              autoFocus
              onChange={(id, m) => {
                setMemberId(id);
                setMemberObj(m);
              }}
            />
          </Form.Item>
        </Form>
        {memberId ? <MemberCurrentSubscriptions memberId={memberId} /> : null}
      </>
    );
  } else if (step === 1) {
    body = (
      <>
        <PlanCards
          plans={plansQuery.data ?? []}
          loading={plansQuery.isLoading}
          value={planId}
          onChange={(p) => setPlanId(p.id)}
        />
        <Divider style={{ margin: '16px 0' }} />
        <Form layout="vertical">
          <Row gutter={12}>
            <Col xs={24} sm={8}>
              <Form.Item label="Chi nhánh bán" required>
                <BranchSelect
                  activeOnly
                  value={branchId}
                  onChange={(v) => setBranchId((v as number | null) ?? undefined)}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}>
              <Form.Item
                label="Ngày bắt đầu"
                tooltip="Bỏ trống = hôm nay hoặc nối tiếp gói đang chạy"
              >
                <DatePicker
                  style={{ width: '100%' }}
                  format={DATE_FORMAT}
                  value={startDate}
                  onChange={setStartDate}
                  placeholder="Tự động"
                  disabledDate={(d) => d.isBefore(dayjs(), 'day')}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}>
              <Form.Item
                label="Mã khuyến mãi"
                validateStatus={promoError ? 'error' : quote?.promotion ? 'success' : undefined}
                help={
                  promoError
                    ? quoteError
                    : quote?.promotion
                      ? `${quote.promotion.name} (−${formatMoney(quote.promotion.discount)})`
                      : undefined
                }
              >
                <Space.Compact style={{ width: '100%' }}>
                  <Input
                    value={promoInput}
                    maxLength={30}
                    placeholder="VD: CHAODONG"
                    onChange={(e) => {
                      const v = e.target.value.toUpperCase();
                      setPromoInput(v);
                      if (!v) setPromoCode('');
                    }}
                    onPressEnter={() => setPromoCode(promoInput.trim())}
                    allowClear
                  />
                  <Button
                    onClick={() => setPromoCode(promoInput.trim())}
                    disabled={!planId || !promoInput.trim()}
                  >
                    Áp dụng
                  </Button>
                </Space.Compact>
              </Form.Item>
            </Col>
          </Row>
        </Form>
        {quoteBody && (
          <Spin spinning={quoteQuery.isFetching}>
            {quoteError && !promoError ? (
              <Alert type="error" showIcon message={quoteError} />
            ) : quote ? (
              <QuoteSummary quote={quote} />
            ) : null}
          </Spin>
        )}
      </>
    );
  } else {
    body = quote ? (
      <>
        <Alert
          type="info"
          style={{ marginBottom: 16 }}
          message={
            <Space wrap>
              <span>
                {memberObj?.full_name} · <b>{quote.plan.name}</b>
              </span>
              <span>
                {formatDate(quote.start_date)} – {formatDate(quote.end_date)}
              </span>
              <span>
                Tổng: <MoneyText value={quote.total} strong />
              </span>
            </Space>
          }
        />
        <>
          {quote.total > 0 ? (
            <PaymentFields form={form} max={quote.total} />
          ) : (
            <Alert
              type="success"
              showIcon
              style={{ marginBottom: 16 }}
              message="Hóa đơn 0 ₫ – gói được kích hoạt ngay, không cần thu tiền."
            />
          )}
          <Form.Item
            name="note"
            label="Ghi chú bán gói"
            rules={[{ max: 500, message: 'Ghi chú tối đa 500 ký tự' }]}
          >
            <Input.TextArea rows={2} placeholder="VD: Khách được giới thiệu" />
          </Form.Item>
        </>
      </>
    ) : null;
  }

  const footer = done
    ? null
    : [
        step > 0 && (
          <Button key="back" onClick={() => setStep(step - 1)} disabled={sell.isPending}>
            Quay lại
          </Button>
        ),
        <Button key="cancel" onClick={onClose} disabled={sell.isPending}>
          Hủy
        </Button>,
        step === 0 && (
          <Button key="next" type="primary" disabled={!memberId} onClick={() => setStep(1)}>
            Tiếp tục
          </Button>
        ),
        step === 1 && (
          <Button
            key="next"
            type="primary"
            disabled={!quote || quoteQuery.isFetching || !!quoteError}
            onClick={goPayment}
          >
            Tiếp tục
          </Button>
        ),
        step === 2 && (
          <Button
            key="submit"
            color="orange"
            variant="solid"
            loading={sell.isPending}
            onClick={submit}
          >
            Xác nhận bán gói
          </Button>
        ),
      ];

  return (
    <Modal
      open={open}
      title="Bán gói tập"
      width={860}
      onCancel={() => !sell.isPending && onClose()}
      footer={footer}
      maskClosable={false}
      destroyOnHidden
    >
      {!done && <Steps size="small" current={step} items={steps} style={{ marginBottom: 20 }} />}
      {/* Form ngoài (không render thẻ <form>) giữ giá trị thanh toán xuyên các bước. */}
      <Form
        form={form}
        layout="vertical"
        component={false}
        initialValues={{ payment: DEFAULT_PAYMENT }}
      >
        {body}
      </Form>
    </Modal>
  );
}
