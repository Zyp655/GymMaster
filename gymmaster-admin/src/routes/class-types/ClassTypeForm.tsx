import { useEffect } from 'react';
import {
  App,
  Col,
  ColorPicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Segmented,
  Select,
} from 'antd';
import type { Color } from 'antd/es/color-picker';
import type { ClassType, ClassTypeInput } from '@/api/class';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { setFormError, showError } from '@/utils/errors';
import { LEVEL_OPTIONS } from '../classes/constants';
import { useSaveClassType } from './queries';

interface FormValues extends Omit<ClassTypeInput, 'color'> {
  color: Color | string;
}

export interface ClassTypeFormProps {
  open: boolean;
  /** `null` = thêm mới. */
  record: ClassType | null;
  onClose: () => void;
}

const PRESET_COLORS = [
  '#F97316',
  '#EF4444',
  '#EC4899',
  '#A855F7',
  '#6366F1',
  '#3B82F6',
  '#0EA5E9',
  '#14B8A6',
  '#22C55E',
  '#84CC16',
  '#EAB308',
  '#0A3150',
];

const toHex = (c: Color | string) =>
  (typeof c === 'string' ? c : c.toHexString()).slice(0, 7).toUpperCase();

/** Modal thêm/sửa loại lớp – validate giống backend (api/class/v1/class_type.go). */
export default function ClassTypeForm({ open, record, onClose }: ClassTypeFormProps) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const { can } = usePermission();
  const save = useSaveClassType();
  const isEdit = !!record;
  const readOnly = !can(PERM.CLASS_MANAGE);

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue(
      record
        ? { ...record }
        : {
            color: '#F97316',
            default_duration_min: 60,
            default_capacity: 20,
            level: 'ALL',
            status: 'ACTIVE',
          },
    );
  }, [open, record, form]);

  const onFinish = (values: FormValues) => {
    const data: ClassTypeInput = {
      code: values.code.trim().toUpperCase(),
      name: values.name.trim(),
      description: values.description?.trim() ?? '',
      color: toHex(values.color),
      default_duration_min: values.default_duration_min,
      default_capacity: values.default_capacity,
      level: values.level,
      status: values.status,
    };
    save.mutate(
      { id: record?.id, data },
      {
        onSuccess: () => {
          message.success(isEdit ? 'Đã cập nhật loại lớp' : 'Đã thêm loại lớp');
          onClose();
        },
        onError: (error) => {
          const mapped = setFormError(form, error, {
            code: /mã/i,
            name: /tên/i,
            color: /màu/i,
            default_duration_min: /thời lượng/i,
            default_capacity: /sức chứa/i,
            level: /cấp độ/i,
          });
          if (!mapped) showError(error, 'Không thể lưu loại lớp');
        },
      },
    );
  };

  return (
    <Modal
      open={open}
      title={isEdit ? `Loại lớp ${record?.code}` : 'Thêm loại lớp'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm loại lớp'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={680}
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
              label="Mã loại lớp"
              tooltip="Tối đa 30 ký tự: chữ, số, '-' hoặc '_'. Không trùng."
              normalize={(v: string) => v?.toUpperCase()}
              rules={[
                { required: true, whitespace: true, message: 'Vui lòng nhập mã loại lớp' },
                { max: 30, message: 'Mã tối đa 30 ký tự' },
                { pattern: /^[A-Za-z0-9_-]+$/, message: "Mã chỉ gồm chữ, số, '-' và '_'" },
              ]}
            >
              <Input placeholder="VD: YOGA-BASIC" maxLength={30} className="code-text" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={16}>
            <Form.Item
              name="name"
              label="Tên loại lớp"
              rules={[
                { required: true, whitespace: true, message: 'Vui lòng nhập tên loại lớp' },
                { max: 100, message: 'Tên tối đa 100 ký tự' },
              ]}
            >
              <Input placeholder="VD: Yoga cơ bản" maxLength={100} />
            </Form.Item>
          </Col>

          <Col xs={24} sm={8}>
            <Form.Item
              name="color"
              label="Màu trên lịch"
              rules={[{ required: true, message: 'Vui lòng chọn màu' }]}
            >
              <ColorPicker
                showText
                format="hex"
                presets={[{ label: 'Gợi ý', colors: PRESET_COLORS }]}
              />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item
              name="default_duration_min"
              label="Thời lượng mặc định"
              rules={[
                { required: true, message: 'Vui lòng nhập thời lượng' },
                {
                  type: 'number',
                  min: 15,
                  max: 240,
                  message: 'Thời lượng mặc định từ 15 đến 240 phút',
                },
              ]}
            >
              <InputNumber min={15} max={240} step={15} suffix="phút" style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item
              name="default_capacity"
              label="Sức chứa mặc định"
              rules={[
                { required: true, message: 'Vui lòng nhập sức chứa' },
                { type: 'number', min: 1, max: 500, message: 'Sức chứa mặc định từ 1 đến 500' },
              ]}
            >
              <InputNumber min={1} max={500} suffix="người" style={{ width: '100%' }} />
            </Form.Item>
          </Col>

          <Col xs={24} sm={12}>
            <Form.Item name="level" label="Cấp độ" rules={[{ required: true }]}>
              <Select options={LEVEL_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="status" label="Trạng thái" rules={[{ required: true }]}>
              <Segmented
                block
                options={[
                  { label: 'Hoạt động', value: 'ACTIVE' },
                  { label: 'Ngừng', value: 'INACTIVE' },
                ]}
              />
            </Form.Item>
          </Col>

          <Col xs={24}>
            <Form.Item name="description" label="Mô tả">
              <Input.TextArea
                rows={3}
                placeholder="Nội dung, đối tượng phù hợp, dụng cụ cần mang..."
                maxLength={1000}
                showCount
              />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}
