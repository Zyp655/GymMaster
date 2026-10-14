import { Button, Col, Form, Input, Row, Segmented, Space, Typography } from 'antd';
import type { FormInstance } from 'antd';
import { PAYMENT_METHOD_OPTIONS } from '@/api/invoice';
import type { PaymentMethod } from '@/api/invoice';
import MoneyInput from '@/components/Money/MoneyInput';
import { formatMoney } from '@/utils/format';

export interface PaymentFieldsProps {
  /** Số tối đa được thu (tổng hóa đơn / số còn nợ). */
  max: number;
  form: FormInstance;
  /** Tiền tố tên trường – mặc định `['payment']`; `[]` = trường ở gốc form. */
  name?: string[];
  /** Cho phép để 0 = chưa thu tiền (bán gói, gia hạn...). Thu tiền hóa đơn thì bắt buộc > 0. */
  allowZero?: boolean;
  /** Nhãn số tiền. */
  amountLabel?: string;
  /** Ẩn mã giao dịch/ghi chú (vd. hoàn tiền). */
  compact?: boolean;
  /** Nhãn nút điền nhanh số tối đa. */
  fillLabel?: string;
}

/** Khối nhập thanh toán dùng chung: số tiền (mặc định = tổng, cho thu một phần), phương thức, mã GD, ghi chú. */
export default function PaymentFields({
  max,
  form,
  name = ['payment'],
  allowZero = true,
  amountLabel = 'Số tiền thu',
  compact,
  fillLabel = 'Thu đủ',
}: PaymentFieldsProps) {
  const path = (f: string) => [...name, f];
  const amount = Form.useWatch(path('amount'), form) as number | null | undefined;
  const method = Form.useWatch(path('method'), form) as PaymentMethod | undefined;
  const rest = Math.max(0, max - (amount ?? 0));

  return (
    <Row gutter={12}>
      <Col xs={24} sm={12}>
        <Form.Item
          name={path('amount')}
          label={amountLabel}
          rules={[
            ...(allowZero ? [] : [{ required: true, message: 'Vui lòng nhập số tiền' }]),
            {
              validator: (_, v: number | null | undefined) => {
                if (v === null || v === undefined) return Promise.resolve();
                if (!allowZero && v <= 0)
                  return Promise.reject(new Error('Số tiền phải lớn hơn 0'));
                if (v > max)
                  return Promise.reject(new Error(`Không được vượt quá ${formatMoney(max)}`));
                return Promise.resolve();
              },
            },
          ]}
          extra={
            <Space size={8} wrap style={{ fontSize: 12 }}>
              {amount !== undefined && amount !== null && amount < max && (
                <Typography.Text type="warning" style={{ fontSize: 12 }}>
                  {amount <= 0
                    ? 'Chưa thu tiền – hóa đơn ở trạng thái chưa thu'
                    : `Một phần – còn lại ${formatMoney(rest)}`}
                </Typography.Text>
              )}
              {amount !== max && (
                <Button
                  type="link"
                  size="small"
                  style={{ padding: 0, height: 'auto', fontSize: 12 }}
                  onClick={() => form.setFieldValue(path('amount'), max)}
                >
                  {fillLabel} {formatMoney(max)}
                </Button>
              )}
            </Space>
          }
        >
          <MoneyInput max={max} />
        </Form.Item>
      </Col>
      <Col xs={24} sm={12}>
        <Form.Item
          name={path('method')}
          label="Phương thức"
          rules={[{ required: true, message: 'Vui lòng chọn phương thức' }]}
        >
          <Segmented block options={PAYMENT_METHOD_OPTIONS} />
        </Form.Item>
      </Col>
      {!compact && (
        <>
          <Col xs={24} sm={12}>
            <Form.Item
              name={path('reference')}
              label="Mã giao dịch"
              rules={[{ max: 100, message: 'Mã giao dịch tối đa 100 ký tự' }]}
            >
              <Input
                placeholder={method === 'CASH' ? 'Không bắt buộc' : 'Mã chuyển khoản / POS thẻ'}
                allowClear
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name={path('note')}
              label="Ghi chú thanh toán"
              rules={[{ max: 255, message: 'Ghi chú tối đa 255 ký tự' }]}
            >
              <Input allowClear />
            </Form.Item>
          </Col>
        </>
      )}
    </Row>
  );
}
