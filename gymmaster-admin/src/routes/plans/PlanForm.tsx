import { useEffect } from 'react';
import { App, Col, Form, Input, InputNumber, Modal, Row, Segmented, Select, Switch } from 'antd';
import type { MembershipPlan, MembershipPlanInput } from '@/api/plan';
import { MoneyInput } from '@/components/Money';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { setFormError, showError } from '@/utils/errors';
import { ACCESS_SCOPE_LABELS, PLAN_TYPE_OPTIONS } from './constants';
import { useSavePlan } from './queries';

type FormValues = Omit<MembershipPlanInput, 'session_count'> & { session_count?: number | null };

export interface PlanFormProps {
  open: boolean;
  record: MembershipPlan | null;
  onClose: () => void;
}

/** Modal thêm/sửa gói tập – validate giống api/plan/v1/plan.go. */
export default function PlanForm({ open, record, onClose }: PlanFormProps) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const { can } = usePermission();
  const save = useSavePlan();
  const isEdit = !!record;
  const readOnly = !can(PERM.PLAN_MANAGE);
  const type = Form.useWatch('type', form);

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue(
      record
        ? { ...record }
        : {
            type: 'TIME',
            duration_days: 30,
            price: 0,
            access_scope: 'SINGLE_BRANCH',
            max_freeze_days: 0,
            max_freeze_times: 0,
            includes_classes: false,
            transfer_fee: 0,
            status: 'ACTIVE',
            sort_order: 0,
          },
    );
  }, [open, record, form]);

  const onFinish = (v: FormValues) => {
    const data: MembershipPlanInput = {
      code: v.code.trim().toUpperCase(),
      name: v.name.trim(),
      description: v.description?.trim() ?? '',
      type: v.type,
      duration_days: v.duration_days,
      session_count: v.type === 'SESSION' ? (v.session_count ?? null) : null,
      price: v.price ?? 0,
      access_scope: v.access_scope,
      max_freeze_days: v.max_freeze_days ?? 0,
      max_freeze_times: v.max_freeze_times ?? 0,
      includes_classes: !!v.includes_classes,
      transfer_fee: v.transfer_fee ?? 0,
      status: v.status,
      sort_order: v.sort_order ?? 0,
    };
    save.mutate(
      { id: record?.id, data },
      {
        onSuccess: () => {
          message.success(isEdit ? 'Đã cập nhật gói tập' : 'Đã thêm gói tập');
          onClose();
        },
        onError: (e) => {
          const mapped = setFormError(form, e, {
            code: /mã/i,
            name: /tên/i,
            session_count: /số buổi/i,
            max_freeze_days: /bảo lưu/i,
            duration_days: /thời hạn/i,
            price: /giá/i,
            transfer_fee: /chuyển nhượng/i,
            description: /mô tả/i,
          });
          if (!mapped) showError(e, 'Không thể lưu gói tập');
        },
      },
    );
  };

  const nonNeg = (label: string) => ({
    type: 'number' as const,
    min: 0,
    message: `${label} không được âm`,
  });

  return (
    <Modal
      open={open}
      title={isEdit ? `Gói tập ${record?.code}` : 'Thêm gói tập'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm gói'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={760}
      forceRender
    >
      <Form<FormValues>
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
              <Input maxLength={30} className="code-text" placeholder="VD: M3" />
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
              <Input maxLength={150} placeholder="VD: Gói 3 tháng" />
            </Form.Item>
          </Col>

          <Col xs={24} sm={8}>
            <Form.Item name="type" label="Loại gói" rules={[{ required: true }]}>
              <Segmented block options={PLAN_TYPE_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={type === 'SESSION' ? 8 : 16}>
            <Form.Item
              name="duration_days"
              label={type === 'SESSION' ? 'Hạn sử dụng (ngày)' : 'Thời hạn (ngày)'}
              tooltip={
                type === 'SESSION'
                  ? 'Số buổi phải dùng hết trong số ngày này.'
                  : 'Tập không giới hạn trong số ngày này.'
              }
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
          {type === 'SESSION' && (
            <Col xs={12} sm={8}>
              <Form.Item
                name="session_count"
                label="Số buổi"
                rules={[
                  { required: true, message: 'Gói theo buổi phải có số buổi lớn hơn 0' },
                  { type: 'number', min: 1, max: 10000, message: 'Số buổi từ 1 đến 10000' },
                ]}
              >
                <InputNumber
                  min={1}
                  max={10000}
                  precision={0}
                  suffix="buổi"
                  style={{ width: '100%' }}
                />
              </Form.Item>
            </Col>
          )}

          <Col xs={24} sm={8}>
            <Form.Item name="price" label="Giá bán" rules={[nonNeg('Giá')]}>
              <MoneyInput />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item name="access_scope" label="Phạm vi chi nhánh">
              <Select
                options={Object.entries(ACCESS_SCOPE_LABELS).map(([value, label]) => ({
                  value,
                  label,
                }))}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item
              name="transfer_fee"
              label="Phí chuyển nhượng"
              rules={[nonNeg('Phí chuyển nhượng')]}
            >
              <MoneyInput />
            </Form.Item>
          </Col>

          <Col xs={12} sm={8}>
            <Form.Item
              name="max_freeze_days"
              label="Bảo lưu tối đa (ngày)"
              dependencies={['max_freeze_times', 'duration_days']}
              tooltip="Số ngày và số lần bảo lưu cùng = 0 nghĩa là không cho bảo lưu."
              rules={[
                nonNeg('Số ngày bảo lưu'),
                ({ getFieldValue }) => ({
                  validator: (_, value?: number) => {
                    const times = getFieldValue('max_freeze_times') ?? 0;
                    const days = value ?? 0;
                    if ((days === 0) !== (times === 0))
                      return Promise.reject(
                        new Error('Số ngày và số lần bảo lưu phải cùng bằng 0 hoặc cùng lớn hơn 0'),
                      );
                    if (days > (getFieldValue('duration_days') ?? Infinity))
                      return Promise.reject(
                        new Error('Số ngày bảo lưu tối đa không được vượt quá thời hạn gói'),
                      );
                    return Promise.resolve();
                  },
                }),
              ]}
            >
              <InputNumber min={0} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item
              name="max_freeze_times"
              label="Bảo lưu tối đa (lần)"
              rules={[nonNeg('Số lần bảo lưu')]}
            >
              <InputNumber min={0} precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item name="sort_order" label="Thứ tự hiển thị" tooltip="Số nhỏ hiển thị trước.">
              <InputNumber precision={0} style={{ width: '100%' }} />
            </Form.Item>
          </Col>

          <Col xs={12} sm={8}>
            <Form.Item name="includes_classes" label="Gồm lớp tập nhóm" valuePropName="checked">
              <Switch checkedChildren="Có" unCheckedChildren="Không" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
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
