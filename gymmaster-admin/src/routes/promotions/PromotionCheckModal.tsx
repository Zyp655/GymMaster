import { useEffect } from 'react';
import { Alert, Button, Col, Descriptions, Form, Input, Modal, Row, Select } from 'antd';
import type { PromotionCheckInput } from '@/api/promotion';
import { MoneyInput, MoneyText } from '@/components/Money';
import { getErrorMessage } from '@/utils/errors';
import { ORDER_TYPE_OPTIONS, discountText } from './constants';
import { useCheckPromotion } from './queries';

/** Thử mã khuyến mãi với một giá trị đơn (không tăng lượt dùng). */
export default function PromotionCheckModal({
  open,
  initialCode,
  onClose,
}: {
  open: boolean;
  initialCode?: string;
  onClose: () => void;
}) {
  const [form] = Form.useForm<PromotionCheckInput>();
  const check = useCheckPromotion();

  useEffect(() => {
    if (!open) return;
    check.reset();
    form.setFieldsValue({
      code: initialCode ?? form.getFieldValue('code'),
      applies_to: form.getFieldValue('applies_to') ?? 'MEMBERSHIP',
      amount: form.getFieldValue('amount') ?? 1000000,
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, initialCode]);

  return (
    <Modal
      open={open}
      title="Kiểm tra mã khuyến mãi"
      footer={null}
      onCancel={onClose}
      width={560}
      forceRender
    >
      <Form<PromotionCheckInput>
        form={form}
        layout="vertical"
        style={{ marginTop: 12 }}
        onValuesChange={() => check.reset()}
        onFinish={(v) => check.mutate({ ...v, code: v.code.trim(), amount: v.amount ?? 0 })}
      >
        <Row gutter={12}>
          <Col xs={24} sm={12}>
            <Form.Item
              name="code"
              label="Mã khuyến mãi"
              normalize={(v: string) => v?.toUpperCase()}
              rules={[{ required: true, whitespace: true, message: 'Vui lòng nhập mã khuyến mãi' }]}
            >
              <Input className="code-text" placeholder="VD: CHAODONG" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="applies_to" label="Loại đơn" rules={[{ required: true }]}>
              <Select options={ORDER_TYPE_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={16}>
            <Form.Item
              name="amount"
              label="Giá trị đơn thử"
              rules={[{ required: true, message: 'Vui lòng nhập giá trị đơn' }]}
            >
              <MoneyInput />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item label=" ">
              <Button type="primary" block htmlType="submit" loading={check.isPending}>
                Kiểm tra
              </Button>
            </Form.Item>
          </Col>
        </Row>
      </Form>

      {check.isError && (
        <Alert type="error" showIcon message={getErrorMessage(check.error, 'Mã không hợp lệ')} />
      )}
      {check.data && (
        <Alert
          type="success"
          showIcon
          message={`Mã hợp lệ – ${check.data.promotion.name}`}
          description={
            <Descriptions size="small" column={1} style={{ marginTop: 8 }}>
              <Descriptions.Item label="Ưu đãi">
                {discountText(check.data.promotion)}
              </Descriptions.Item>
              <Descriptions.Item label="Số tiền giảm">
                <MoneyText value={check.data.discount} strong style={{ color: '#12805C' }} />
              </Descriptions.Item>
              <Descriptions.Item label="Còn phải trả">
                <MoneyText value={check.data.final_amount} strong />
              </Descriptions.Item>
            </Descriptions>
          }
        />
      )}
    </Modal>
  );
}
