import { useEffect } from 'react';
import { App, Col, Form, Input, InputNumber, Modal, Row, Segmented, Typography } from 'antd';
import type { PtPackage, PtPackageInput } from '@/api/ptPackage';
import { MoneyInput, MoneyText } from '@/components/Money';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { setFormError, showError } from '@/utils/errors';
import { useSavePtPackage } from './queries';

export interface PtPackageFormProps {
  open: boolean;
  record: PtPackage | null;
  onClose: () => void;
}

/** Modal thêm/sửa gói PT – validate giống api/ptpackage/v1/ptpackage.go. */
export default function PtPackageForm({ open, record, onClose }: PtPackageFormProps) {
  const [form] = Form.useForm<PtPackageInput>();
  const { message } = App.useApp();
  const { can } = usePermission();
  const save = useSavePtPackage();
  const isEdit = !!record;
  const readOnly = !can(PERM.PLAN_MANAGE);
  const price = Form.useWatch('price', form);
  const sessions = Form.useWatch('session_count', form);
  const perSession = price && sessions ? Math.round(price / sessions) : null;

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue(
      record
        ? { ...record }
        : { session_count: 10, session_minutes: 60, validity_days: 60, price: 0, status: 'ACTIVE' },
    );
  }, [open, record, form]);

  const onFinish = (v: PtPackageInput) => {
    const data: PtPackageInput = {
      code: v.code.trim().toUpperCase(),
      name: v.name.trim(),
      description: v.description?.trim() ?? '',
      session_count: v.session_count,
      session_minutes: v.session_minutes ?? 60,
      validity_days: v.validity_days,
      price: v.price ?? 0,
      status: v.status,
    };
    save.mutate(
      { id: record?.id, data },
      {
        onSuccess: () => {
          message.success(isEdit ? 'Đã cập nhật gói PT' : 'Đã thêm gói PT');
          onClose();
        },
        onError: (e) => {
          const mapped = setFormError(form, e, {
            code: /mã/i,
            name: /tên/i,
            session_count: /số buổi/i,
            session_minutes: /thời lượng/i,
            validity_days: /thời hạn/i,
            price: /giá/i,
            description: /mô tả/i,
          });
          if (!mapped) showError(e, 'Không thể lưu gói PT');
        },
      },
    );
  };

  return (
    <Modal
      open={open}
      title={isEdit ? `Gói PT ${record?.code}` : 'Thêm gói PT'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm gói'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={680}
      forceRender
    >
      <Form<PtPackageInput>
        form={form}
        layout="vertical"
        disabled={readOnly}
        onFinish={onFinish}
        style={{ marginTop: 12 }}
      >
        <Row gutter={16}>
          <Col xs={24} sm={8}>
            <Form.Item
              name="code"
              label="Mã gói"
              normalize={(v: string) => v?.toUpperCase()}
              rules={[
                { required: true, message: 'Vui lòng nhập mã gói' },
                {
                  pattern: /^[A-Z0-9_-]{2,30}$/,
                  message: "Mã gồm 2-30 ký tự in hoa, số, '-' hoặc '_'",
                },
              ]}
            >
              <Input maxLength={30} className="code-text" placeholder="VD: PT10" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={16}>
            <Form.Item
              name="name"
              label="Tên gói"
              rules={[
                { required: true, whitespace: true, message: 'Vui lòng nhập tên gói' },
                { max: 150, message: 'Tên tối đa 150 ký tự' },
              ]}
            >
              <Input maxLength={150} placeholder="VD: PT 10 buổi" />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item
              name="session_count"
              label="Số buổi"
              rules={[
                { required: true, message: 'Vui lòng nhập số buổi' },
                { type: 'number', min: 1, max: 1000, message: 'Số buổi từ 1 đến 1000' },
              ]}
            >
              <InputNumber
                min={1}
                max={1000}
                precision={0}
                suffix="buổi"
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item
              name="session_minutes"
              label="Thời lượng mỗi buổi"
              rules={[
                {
                  type: 'number',
                  min: 15,
                  max: 240,
                  message: 'Thời lượng mỗi buổi từ 15 đến 240 phút',
                },
              ]}
            >
              <InputNumber
                min={15}
                max={240}
                step={15}
                precision={0}
                suffix="phút"
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item
              name="validity_days"
              label="Hiệu lực"
              tooltip="Số ngày kể từ khi ký hợp đồng để dùng hết các buổi."
              rules={[
                { required: true, message: 'Vui lòng nhập thời hạn (ngày)' },
                { type: 'number', min: 1, max: 3650, message: 'Thời hạn từ 1 đến 3650 ngày' },
              ]}
            >
              <InputNumber
                min={1}
                max={3650}
                precision={0}
                suffix="ngày"
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="price"
              label="Giá bán"
              rules={[{ type: 'number', min: 0, message: 'Giá không được âm' }]}
              extra={
                perSession !== null ? (
                  <Typography.Text type="secondary">
                    ≈ <MoneyText value={perSession} /> / buổi
                  </Typography.Text>
                ) : undefined
              }
            >
              <MoneyInput />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="status" label="Trạng thái">
              <Segmented
                block
                options={[
                  { label: 'Đang bán', value: 'ACTIVE' },
                  { label: 'Ngừng bán', value: 'INACTIVE' },
                ]}
              />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item
              name="description"
              label="Mô tả"
              rules={[{ max: 2000, message: 'Mô tả tối đa 2000 ký tự' }]}
            >
              <Input.TextArea rows={3} maxLength={2000} showCount />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}
