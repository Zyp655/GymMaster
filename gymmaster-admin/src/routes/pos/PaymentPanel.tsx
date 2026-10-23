import { Button, Input, Select, Space, Tooltip, Typography } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { COUNTER_PAYMENT_METHOD_OPTIONS } from '@/api/invoice';
import type { PaymentMethod } from '@/api/invoice';
import MoneyInput from '@/components/Money/MoneyInput';
import { formatMoney, formatNumber } from '@/utils/format';
import { newPaymentLine, paymentTotals } from './cart';
import type { PaymentLine } from './cart';

const DENOMINATIONS = [50_000, 100_000, 200_000, 500_000];

export interface PaymentPanelProps {
  total: number;
  payments: PaymentLine[];
  onChange: (payments: PaymentLine[]) => void;
  disabled?: boolean;
}

/** Thanh toán nhiều phương thức: các dòng (phương thức + số tiền + mã GD), nút nhanh, tiền thừa. */
export default function PaymentPanel({ total, payments, onChange, disabled }: PaymentPanelProps) {
  const t = paymentTotals(payments, total);

  const update = (key: string, patch: Partial<PaymentLine>) =>
    onChange(payments.map((p) => (p.key === key ? { ...p, ...patch } : p)));

  /** Cộng mệnh giá vào dòng tiền mặt (tạo nếu chưa có). */
  const addCash = (value: number) => {
    const cashLine = payments.find((p) => p.method === 'CASH');
    if (cashLine) update(cashLine.key, { amount: (cashLine.amount ?? 0) + value });
    else onChange([...payments, newPaymentLine('CASH', value)]);
  };

  /** Tiền mặt đủ: một dòng tiền mặt = tổng (giữ các dòng khác nếu đã nhập, điền phần còn lại). */
  const cashExact = () => {
    const others = payments.filter((p) => p.method !== 'CASH' && (p.amount ?? 0) > 0);
    const othersSum = others.reduce((s, p) => s + (p.amount ?? 0), 0);
    const rest = Math.max(0, total - othersSum);
    onChange([...others, newPaymentLine('CASH', rest)]);
  };

  /** Dòng mới mặc định chuyển khoản với phần còn thiếu. */
  const addLine = () => {
    const used = new Set(payments.map((p) => p.method));
    const method: PaymentMethod = !used.has('TRANSFER')
      ? 'TRANSFER'
      : !used.has('CARD')
        ? 'CARD'
        : 'CASH';
    onChange([...payments, newPaymentLine(method, t.missing || null)]);
  };

  return (
    <div className="pos-pay">
      {payments.map((p) => (
        <div key={p.key} className="pos-pay__row">
          <Select<PaymentMethod>
            value={p.method}
            options={COUNTER_PAYMENT_METHOD_OPTIONS}
            onChange={(method) => update(p.key, { method })}
            disabled={disabled}
            style={{ width: 130 }}
            aria-label="Phương thức"
          />
          <MoneyInput
            value={p.amount ?? undefined}
            onChange={(v) => update(p.key, { amount: v ?? null })}
            placeholder="Số tiền"
            disabled={disabled}
            style={{ flex: 1, minWidth: 0 }}
            aria-label="Số tiền"
          />
          {payments.length > 1 && (
            <Tooltip title="Bỏ dòng">
              <Button
                type="text"
                danger
                icon={<DeleteOutlined />}
                onClick={() => onChange(payments.filter((x) => x.key !== p.key))}
                disabled={disabled}
                aria-label="Bỏ dòng thanh toán"
              />
            </Tooltip>
          )}
          {p.method !== 'CASH' && (
            <Input
              className="pos-pay__ref"
              value={p.reference}
              maxLength={100}
              placeholder="Mã giao dịch (không bắt buộc)"
              onChange={(e) => update(p.key, { reference: e.target.value })}
              disabled={disabled}
            />
          )}
        </div>
      ))}

      <Space wrap size={6} style={{ marginBottom: 8 }}>
        <Button size="small" type="primary" ghost onClick={cashExact} disabled={disabled}>
          Tiền mặt đủ
        </Button>
        {DENOMINATIONS.map((d) => (
          <Button key={d} size="small" onClick={() => addCash(d)} disabled={disabled}>
            +{formatNumber(d / 1000)}k
          </Button>
        ))}
        <Button
          size="small"
          icon={<PlusOutlined />}
          onClick={addLine}
          disabled={disabled || payments.length >= 4}
        >
          Thêm phương thức
        </Button>
        {t.received > 0 && (
          <Button
            size="small"
            type="text"
            onClick={() => onChange([newPaymentLine()])}
            disabled={disabled}
          >
            Nhập lại
          </Button>
        )}
      </Space>

      <div className="pos-sum">
        <span>Khách đưa</span>
        <b>{formatMoney(t.received)}</b>
        {t.missing > 0 ? (
          <>
            <span>Còn thiếu</span>
            <b style={{ color: '#cf1322' }}>{formatMoney(t.missing)}</b>
          </>
        ) : (
          <>
            <span>Tiền thừa trả khách</span>
            <b className="pos-change">{formatMoney(t.change)}</b>
          </>
        )}
      </div>
      {t.error && (
        <Typography.Text type="danger" style={{ fontSize: 12 }}>
          {t.error}
        </Typography.Text>
      )}
    </div>
  );
}
