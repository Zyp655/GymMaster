import { useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Descriptions,
  Divider,
  Empty,
  Form,
  Input,
  List,
  Modal,
  Result,
  Row,
  Select,
  Skeleton,
  Space,
  Spin,
  Steps,
  Tag,
  Typography,
} from 'antd';
import { CheckCircleFilled, PrinterOutlined, UserOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { useNavigate } from 'react-router-dom';
import type { MemberOption } from '@/api/member';
import ptApi from '@/api/pt';
import type { PtContractDetail, PtQuote } from '@/api/pt';
import BranchSelect from '@/components/BranchSelect';
import MemberSelect from '@/components/MemberSelect';
import MoneyText from '@/components/Money/MoneyText';
import StatusTag from '@/components/StatusTag';
import { INVOICE_STATUS_MAP } from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import PaymentFields from '@/routes/subscriptions/PaymentFields';
import { DEFAULT_PAYMENT, toPaymentInput } from '@/routes/subscriptions/payment';
import type { PaymentFormValues } from '@/routes/subscriptions/payment';
import { openInvoicePrint } from '@/routes/subscriptions/utils';
import { getErrorMessage, showError, validateForm } from '@/utils/errors';
import { DATE_FORMAT, formatDate, formatMoney, normalizeText, toApiDate } from '@/utils/format';
import { PT_CONTRACT_STATUS_MAP, contractRemainText, isLiveContract } from './constants';
import {
  useMemberPtContracts,
  usePtPackageOptions,
  usePtTrainerOptions,
  useSellPt,
} from './queries';

export interface SellPtModalProps {
  open: boolean;
  onClose: () => void;
  /** Hội viên chọn sẵn (mở từ hồ sơ hội viên). */
  member?: MemberOption | null;
  /** Mở chi tiết hợp đồng vừa bán. */
  onOpenContract?: (id: number) => void;
}

interface PayForm {
  payment: PaymentFormValues;
  note?: string;
}

const filterByLabel = (input: string, option?: { label?: unknown }) =>
  normalizeText(String(option?.label ?? '')).includes(normalizeText(input));

/** Hợp đồng PT đang có của hội viên (bước 1). */
function MemberCurrentContracts({ memberId }: { memberId: number }) {
  const query = useMemberPtContracts(memberId);
  const live = (query.data ?? []).filter(isLiveContract);
  if (query.isLoading) return <Spin style={{ display: 'block', margin: '16px auto' }} />;
  if (query.isError)
    return <Alert type="warning" showIcon message={getErrorMessage(query.error)} />;
  return (
    <>
      <Typography.Text strong>Hợp đồng PT đang có</Typography.Text>
      {live.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="Hội viên chưa có hợp đồng PT nào"
        />
      ) : (
        <List
          size="small"
          dataSource={live}
          renderItem={(c) => (
            <List.Item>
              <Space wrap size={[8, 4]}>
                <StatusTag status={c.status} map={PT_CONTRACT_STATUS_MAP} />
                <b>{c.package_name}</b>
                <span>HLV {c.trainer_name}</span>
                <Typography.Text type="secondary">Còn {contractRemainText(c)}</Typography.Text>
                <Typography.Text type="secondary">{c.branch_name}</Typography.Text>
              </Space>
            </List.Item>
          )}
        />
      )}
    </>
  );
}

/** Tóm tắt báo giá gói PT. */
function PtQuoteSummary({ quote }: { quote: PtQuote }) {
  return (
    <Descriptions
      size="small"
      bordered
      column={{ xs: 1, sm: 2, md: 2, lg: 2, xl: 2, xxl: 2 }}
      items={[
        {
          key: 'pkg',
          label: 'Gói PT',
          span: 'filled',
          children: (
            <>
              <span className="code-text">{quote.package.code}</span> {quote.package.name} ·{' '}
              {quote.total_sessions} buổi × {quote.session_minutes} phút
            </>
          ),
        },
        { key: 'trainer', label: 'HLV', children: quote.trainer_name },
        {
          key: 'period',
          label: 'Thời hạn',
          children: `${formatDate(quote.start_date)} – ${formatDate(quote.end_date)}`,
        },
        { key: 'subtotal', label: 'Giá gói', children: <MoneyText value={quote.subtotal} /> },
        {
          key: 'discount',
          label: 'Giảm giá',
          children: quote.discount ? (
            <>
              <MoneyText value={-quote.discount} colored />
              {quote.promotion && (
                <Tag color="orange" bordered={false} style={{ marginLeft: 8 }}>
                  {quote.promotion.code}
                </Tag>
              )}
            </>
          ) : (
            '—'
          ),
        },
        {
          key: 'pps',
          label: 'Giá/buổi',
          children: <MoneyText value={quote.price_per_session} />,
        },
        {
          key: 'total',
          label: 'Tổng thanh toán',
          children: (
            <Typography.Text strong style={{ fontSize: 18, color: '#F97316' }}>
              <MoneyText value={quote.total} />
            </Typography.Text>
          ),
        },
      ]}
    />
  );
}

/**
 * Bán gói PT 3 bước: hội viên → gói PT + chi nhánh + HLV + ngày bắt đầu + mã KM (báo giá) → thanh toán.
 */
export default function SellPtModal({ open, onClose, member, onOpenContract }: SellPtModalProps) {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const { branchId: workingBranch } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const [step, setStep] = useState(0);
  const [memberId, setMemberId] = useState<number | undefined>(member?.id);
  const [memberObj, setMemberObj] = useState<MemberOption | undefined>(member ?? undefined);
  const [packageId, setPackageId] = useState<number>();
  const [branchId, setBranchId] = useState<number>();
  const [trainerId, setTrainerId] = useState<number>();
  const [startDate, setStartDate] = useState<Dayjs | null>(null);
  const [promoInput, setPromoInput] = useState('');
  const [promoCode, setPromoCode] = useState('');
  const [done, setDone] = useState<PtContractDetail | null>(null);
  const [form] = Form.useForm<PayForm>();
  const sell = useSellPt();

  const defaultBranch =
    workingBranch ?? branches.find((b) => b.status !== 'INACTIVE')?.id ?? undefined;

  useEffect(() => {
    if (!open) return;
    setStep(0);
    setMemberId(member?.id);
    setMemberObj(member ?? undefined);
    setPackageId(undefined);
    setBranchId(undefined);
    setTrainerId(undefined);
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

  const packages = usePtPackageOptions(open);
  const trainers = usePtTrainerOptions(branchId, open && !!branchId);

  const quoteBody = useMemo(
    () =>
      memberId && packageId && branchId && trainerId
        ? {
            member_id: memberId,
            pt_package_id: packageId,
            trainer_id: trainerId,
            branch_id: branchId,
            start_date: toApiDate(startDate) ?? '',
            promotion_code: promoCode,
          }
        : null,
    [memberId, packageId, branchId, trainerId, startDate, promoCode],
  );

  const quoteQuery = useQuery({
    queryKey: [ptApi.key, 'quote', quoteBody],
    queryFn: () => ptApi.quote(quoteBody!),
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
    const values = await validateForm(form);
    if (!values) return;
    sell.mutate(
      { ...quoteBody, note: values.note?.trim() || '', payment: toPaymentInput(values.payment) },
      {
        onSuccess: (res) => {
          message.success('Đã bán gói PT');
          setDone(res);
        },
        onError: (e) => showError(e, 'Không thể bán gói PT'),
      },
    );
  };

  let body: ReactNode;
  if (done) {
    const inv = done.invoice;
    body = (
      <Result
        status="success"
        title="Bán gói PT thành công"
        subTitle={
          <Space direction="vertical" size={4}>
            <span>
              {done.member_name} · <b>{done.package_name}</b> · HLV {done.trainer_name} ·{' '}
              {formatDate(done.start_date)} – {formatDate(done.end_date)}{' '}
              <StatusTag status={done.status} map={PT_CONTRACT_STATUS_MAP} />
            </span>
            {inv && (
              <span>
                Hóa đơn <b className="code-text">{inv.invoice_no}</b>: tổng{' '}
                <MoneyText value={inv.total} strong />, đã thu <MoneyText value={inv.paid_amount} />
                {inv.due > 0 && (
                  <>
                    , còn nợ <MoneyText value={inv.due} style={{ color: '#cf1322' }} />
                  </>
                )}{' '}
                <StatusTag status={inv.status} map={INVOICE_STATUS_MAP} />
              </span>
            )}
            {done.status === 'PENDING' && (
              <Typography.Text type="warning">
                Hợp đồng sẽ được kích hoạt khi hóa đơn được thu đủ.
              </Typography.Text>
            )}
          </Space>
        }
        extra={
          <Space wrap style={{ justifyContent: 'center' }}>
            {inv && (
              <Button
                type="primary"
                icon={<PrinterOutlined />}
                onClick={() => openInvoicePrint(inv.id)}
              >
                In hóa đơn
              </Button>
            )}
            {onOpenContract && (
              <Button
                onClick={() => {
                  onClose();
                  onOpenContract(done.id);
                }}
              >
                Xem hợp đồng
              </Button>
            )}
            <Button
              icon={<UserOutlined />}
              onClick={() => {
                onClose();
                navigate(`/app/members/${done.member_id}?tab=classes&part=pt`);
              }}
            >
              Đến hồ sơ hội viên
            </Button>
            <Button onClick={onClose}>Đóng</Button>
          </Space>
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
        {memberId ? <MemberCurrentContracts memberId={memberId} /> : null}
      </>
    );
  } else if (step === 1) {
    const list = packages.data ?? [];
    body = (
      <>
        {packages.isLoading ? (
          <Skeleton active paragraph={{ rows: 3 }} />
        ) : list.length === 0 ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Chưa có gói PT đang bán" />
        ) : (
          <Row gutter={[12, 12]}>
            {list.map((p) => {
              const selected = p.id === packageId;
              return (
                <Col xs={24} sm={12} md={8} key={p.id}>
                  <Card
                    size="small"
                    hoverable
                    onClick={() => setPackageId(p.id)}
                    style={{
                      height: '100%',
                      cursor: 'pointer',
                      borderColor: selected ? '#F97316' : undefined,
                      boxShadow: selected ? '0 0 0 2px rgba(249,115,22,0.25)' : undefined,
                    }}
                  >
                    <Space direction="vertical" size={2} style={{ width: '100%' }}>
                      <Space
                        style={{ justifyContent: 'space-between', width: '100%' }}
                        align="start"
                      >
                        <Typography.Text strong>{p.name}</Typography.Text>
                        {selected && (
                          <CheckCircleFilled style={{ color: '#F97316', fontSize: 18 }} />
                        )}
                      </Space>
                      <Typography.Text
                        type="secondary"
                        className="code-text"
                        style={{ fontSize: 12 }}
                      >
                        {p.code}
                      </Typography.Text>
                      <Typography.Text strong style={{ fontSize: 18, color: '#0A3150' }}>
                        <MoneyText value={p.price} />
                      </Typography.Text>
                      <Typography.Text style={{ fontSize: 13 }}>
                        {p.session_count} buổi × {p.session_minutes} phút · hạn {p.validity_days}{' '}
                        ngày
                      </Typography.Text>
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        {formatMoney(p.price_per_session)}/buổi
                      </Typography.Text>
                    </Space>
                  </Card>
                </Col>
              );
            })}
          </Row>
        )}
        <Divider style={{ margin: '16px 0' }} />
        <Form layout="vertical">
          <Row gutter={12}>
            <Col xs={24} sm={12}>
              <Form.Item label="Chi nhánh bán" required>
                <BranchSelect
                  activeOnly
                  value={branchId}
                  onChange={(v) => {
                    setBranchId((v as number | null) ?? undefined);
                    setTrainerId(undefined);
                  }}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item label="Huấn luyện viên" required>
                <Select
                  showSearch
                  placeholder={branchId ? 'Chọn HLV của chi nhánh' : 'Chọn chi nhánh trước'}
                  disabled={!branchId}
                  value={trainerId}
                  loading={trainers.isLoading}
                  filterOption={filterByLabel}
                  notFoundContent={
                    <Empty
                      image={Empty.PRESENTED_IMAGE_SIMPLE}
                      description="Chi nhánh chưa có HLV"
                    />
                  }
                  options={(trainers.data ?? []).map((t) => ({
                    value: t.id,
                    label: t.full_name,
                  }))}
                  onChange={(v: number) => setTrainerId(v)}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item label="Ngày bắt đầu" tooltip="Bỏ trống = hôm nay">
                <DatePicker
                  style={{ width: '100%' }}
                  format={DATE_FORMAT}
                  value={startDate}
                  onChange={setStartDate}
                  placeholder="Hôm nay"
                  disabledDate={(d) => d.isBefore(dayjs(), 'day')}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
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
                    placeholder="VD: PT200K"
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
                    disabled={!packageId || !promoInput.trim()}
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
              <PtQuoteSummary quote={quote} />
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
                {memberObj?.full_name} · <b>{quote.package.name}</b> · HLV {quote.trainer_name}
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
        {quote.total > 0 ? (
          <PaymentFields form={form} max={quote.total} />
        ) : (
          <Alert
            type="success"
            showIcon
            style={{ marginBottom: 16 }}
            message="Hóa đơn 0 ₫ – hợp đồng được kích hoạt ngay, không cần thu tiền."
          />
        )}
        <Form.Item
          name="note"
          label="Ghi chú hợp đồng"
          rules={[{ max: 500, message: 'Ghi chú tối đa 500 ký tự' }]}
        >
          <Input.TextArea rows={2} placeholder="VD: Khách được giới thiệu" />
        </Form.Item>
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
            Xác nhận bán gói PT
          </Button>
        ),
      ];

  return (
    <Modal
      open={open}
      title="Bán gói PT"
      width={860}
      onCancel={() => !sell.isPending && onClose()}
      footer={footer}
      maskClosable={false}
      destroyOnHidden
    >
      {!done && (
        <Steps
          size="small"
          current={step}
          items={[{ title: 'Hội viên' }, { title: 'Gói PT & HLV' }, { title: 'Thanh toán' }]}
          style={{ marginBottom: 20 }}
        />
      )}
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
