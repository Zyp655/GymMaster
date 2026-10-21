import { useEffect } from 'react';
import {
  App,
  Col,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Segmented,
  Select,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { Promotion, PromotionInput, PromotionType } from '@/api/promotion';
import { MoneyInput } from '@/components/Money';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { setFormError, showError } from '@/utils/errors';
import { DATETIME_FORMAT } from '@/utils/format';
import { APPLIES_TO_OPTIONS } from './constants';
import { useSavePromotion } from './queries';

interface FormValues extends Omit<
  PromotionInput,
  'start_at' | 'end_at' | 'max_discount' | 'usage_limit'
> {
  period: [Dayjs, Dayjs];
  max_discount?: number | null;
  usage_limit?: number | null;
}

const RFC3339 = 'YYYY-MM-DDTHH:mm:ssZ';

export interface PromotionFormProps {
  open: boolean;
  record: Promotion | null;
  onClose: () => void;
}

/** Modal thêm/sửa khuyến mãi – validate giống api/promotion/v1/promotion.go. */
export default function PromotionForm({ open, record, onClose }: PromotionFormProps) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const { can } = usePermission();
  const save = useSavePromotion();
  const isEdit = !!record;
  const readOnly = !can(PERM.PROMOTION_MANAGE);
  const type = Form.useWatch('type', form) as PromotionType | undefined;

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue(
      record
        ? {
            ...record,
            period: [dayjs(record.start_at), dayjs(record.end_at)],
          }
        : {
            type: 'PERCENT',
            applies_to: 'ALL',
            min_amount: 0,
            status: 'ACTIVE',
            period: [dayjs().startOf('day'), dayjs().add(1, 'month').endOf('day')],
          },
    );
  }, [open, record, form]);

  const onFinish = (v: FormValues) => {
    const data: PromotionInput = {
      code: v.code.trim().toUpperCase(),
      name: v.name.trim(),
      type: v.type,
      value: v.value,
      max_discount: v.type === 'PERCENT' && v.max_discount ? v.max_discount : null,
      min_amount: v.min_amount ?? 0,
      applies_to: v.applies_to,
      start_at: v.period[0].second(0).format(RFC3339),
      end_at: v.period[1].second(59).format(RFC3339),
      usage_limit: v.usage_limit || null,
      status: v.status,
    };
    save.mutate(
      { id: record?.id, data },
      {
        onSuccess: () => {
          message.success(isEdit ? 'Đã cập nhật khuyến mãi' : 'Đã thêm khuyến mãi');
          onClose();
        },
        onError: (e) => {
          const mapped = setFormError(form, e, {
            code: /mã/i,
            name: /tên/i,
            value: /phần trăm|giá trị giảm/i,
            max_discount: /trần|tối đa/i,
            min_amount: /tối thiểu/i,
            period: /thời gian/i,
            usage_limit: /lượt/i,
          });
          if (!mapped) showError(e, 'Không thể lưu khuyến mãi');
        },
      },
    );
  };

  return (
    <Modal
      open={open}
      title={isEdit ? `Khuyến mãi ${record?.code}` : 'Thêm khuyến mãi'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm khuyến mãi'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={720}
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
              label="Mã khuyến mãi"
              normalize={(v: string) => v?.toUpperCase()}
              rules={[
                { required: true, message: 'Vui lòng nhập mã khuyến mãi' },
                {
                  pattern: /^[A-Za-z0-9_-]{3,30}$/,
                  message: "Mã gồm 3-30 ký tự chữ, số, '-' hoặc '_'",
                },
              ]}
            >
              <Input maxLength={30} className="code-text" placeholder="VD: CHAODONG" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={16}>
            <Form.Item
              name="name"
              label="Tên chương trình"
              rules={[
                { required: true, whitespace: true, message: 'Vui lòng nhập tên chương trình' },
                { max: 150, message: 'Tên tối đa 150 ký tự' },
              ]}
            >
              <Input maxLength={150} />
            </Form.Item>
          </Col>

          <Col xs={24} sm={8}>
            <Form.Item name="type" label="Loại giảm giá" rules={[{ required: true }]}>
              <Segmented
                block
                options={[
                  { label: 'Phần trăm', value: 'PERCENT' },
                  { label: 'Số tiền', value: 'FIXED' },
                ]}
                onChange={() => form.setFieldsValue({ value: undefined, max_discount: null })}
              />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item
              name="value"
              label="Giá trị giảm"
              rules={[
                { required: true, message: 'Vui lòng nhập giá trị giảm' },
                {
                  validator: (_, v?: number) =>
                    v === undefined || v === null
                      ? Promise.resolve()
                      : v <= 0
                        ? Promise.reject(new Error('Giá trị giảm phải lớn hơn 0'))
                        : type === 'PERCENT' && v > 100
                          ? Promise.reject(new Error('Phần trăm giảm không được vượt quá 100%'))
                          : Promise.resolve(),
                },
              ]}
            >
              {type === 'FIXED' ? (
                <MoneyInput min={0} />
              ) : (
                <InputNumber min={0} max={100} step={5} suffix="%" style={{ width: '100%' }} />
              )}
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            {type === 'PERCENT' ? (
              <Form.Item
                name="max_discount"
                label="Giảm tối đa"
                tooltip="Bỏ trống = không giới hạn số tiền giảm."
                rules={[{ type: 'number', min: 1, message: 'Mức giảm tối đa phải lớn hơn 0' }]}
              >
                <MoneyInput placeholder="Không giới hạn" />
              </Form.Item>
            ) : (
              <Form.Item label="Giảm tối đa">
                <Typography.Text type="secondary">Không áp dụng</Typography.Text>
              </Form.Item>
            )}
          </Col>

          <Col xs={24} sm={8}>
            <Form.Item name="applies_to" label="Áp dụng cho">
              <Select options={APPLIES_TO_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item
              name="min_amount"
              label="Giá trị đơn tối thiểu"
              rules={[{ type: 'number', min: 0, message: 'Giá trị đơn tối thiểu không được âm' }]}
            >
              <MoneyInput />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item
              name="usage_limit"
              label="Giới hạn lượt dùng"
              tooltip="Bỏ trống = không giới hạn."
              rules={[
                { type: 'number', min: 1, message: 'Giới hạn lượt dùng phải lớn hơn 0' },
                {
                  validator: (_, v?: number | null) =>
                    v && record && v < record.used_count
                      ? Promise.reject(
                          new Error(
                            `Giới hạn lượt dùng không được nhỏ hơn số lượt đã dùng (${record.used_count})`,
                          ),
                        )
                      : Promise.resolve(),
                },
              ]}
            >
              <InputNumber
                min={1}
                precision={0}
                placeholder="Không giới hạn"
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>

          <Col xs={24} sm={16}>
            <Form.Item
              name="period"
              label="Thời gian áp dụng"
              rules={[
                { required: true, message: 'Vui lòng chọn thời gian áp dụng' },
                {
                  validator: (_, v?: [Dayjs, Dayjs]) =>
                    v && !v[1].isAfter(v[0])
                      ? Promise.reject(new Error('Thời gian kết thúc phải sau thời gian bắt đầu'))
                      : Promise.resolve(),
                },
              ]}
            >
              <DatePicker.RangePicker
                showTime={{ format: 'HH:mm' }}
                format={DATETIME_FORMAT}
                style={{ width: '100%' }}
                placeholder={['Bắt đầu', 'Kết thúc']}
                // Form bị disabled khi chỉ có quyền xem; antd cảnh báo nếu RangePicker disabled mà rỗng.
                allowEmpty={readOnly ? [true, true] : undefined}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item name="status" label="Trạng thái">
              <Segmented
                block
                options={[
                  { label: 'Áp dụng', value: 'ACTIVE' },
                  { label: 'Ngừng', value: 'INACTIVE' },
                ]}
              />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}
