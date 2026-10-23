import { useEffect } from 'react';
import { App, Col, Form, Input, Modal, Row, Segmented, TimePicker } from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { Branch, BranchInput } from '@/api/types';
import { STATUS_PRESETS } from '@/constants/status';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { ErrorCode, isApiError } from '@/utils/api';
import { getErrorMessage, setFormError, showError } from '@/utils/errors';
import { TIME_FORMAT } from '@/utils/format';
import { useSaveBranch } from './queries';

interface FormValues extends Omit<BranchInput, 'open_time' | 'close_time'> {
  open_time: Dayjs;
  close_time: Dayjs;
}

export interface BranchFormProps {
  open: boolean;
  /** `null` = thêm mới. */
  record: Branch | null;
  onClose: () => void;
}

const toTime = (v: string | undefined, fallback: string) => dayjs(v || fallback, TIME_FORMAT);

/** Modal thêm/sửa chi nhánh – validate giống backend (api/branch/v1/branch.go). */
export default function BranchForm({ open, record, onClose }: BranchFormProps) {
  const [form] = Form.useForm<FormValues>();
  const { message, modal } = App.useApp();
  const { can } = usePermission();
  const save = useSaveBranch();
  const isEdit = !!record;
  const readOnly = isEdit ? !can(PERM.BRANCH_UPDATE) : !can(PERM.BRANCH_CREATE);

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue(
      record
        ? {
            ...record,
            open_time: toTime(record.open_time, '05:30'),
            close_time: toTime(record.close_time, '22:00'),
          }
        : {
            status: 'ACTIVE',
            open_time: toTime(undefined, '05:30'),
            close_time: toTime(undefined, '22:00'),
          },
    );
  }, [open, record, form]);

  const onFinish = (values: FormValues) => {
    const data: BranchInput = {
      code: values.code.trim().toUpperCase(),
      name: values.name.trim(),
      address: values.address?.trim() ?? '',
      phone: values.phone?.trim() ?? '',
      email: values.email?.trim() ?? '',
      open_time: values.open_time.format(TIME_FORMAT),
      close_time: values.close_time.format(TIME_FORMAT),
      status: values.status,
    };
    submit(data);
  };

  const submit = (data: BranchInput) => {
    save.mutate(
      { id: record?.id, data },
      {
        onSuccess: () => {
          message.success(isEdit ? 'Đã cập nhật chi nhánh' : 'Đã thêm chi nhánh');
          onClose();
        },
        onError: (error) => {
          // Đổi giờ mở cửa khi còn lớp/PT tương lai ngoài khung giờ mới → 409: hỏi xác nhận rồi gửi lại force.
          if (
            isEdit &&
            !data.force &&
            isApiError(error) &&
            error.code === ErrorCode.CONFLICT &&
            /lớp nhóm|buổi PT/i.test(error.message)
          ) {
            modal.confirm({
              title: 'Vẫn đổi giờ mở cửa?',
              content: getErrorMessage(error),
              okText: 'Vẫn lưu',
              cancelText: 'Xem lại',
              onOk: () => submit({ ...data, force: true }),
            });
            return;
          }
          const mapped = setFormError(form, error, {
            code: /mã/i,
            name: /tên/i,
            email: /email/i,
            phone: /điện thoại/i,
            close_time: /đóng cửa|sau giờ mở/i,
            open_time: /mở cửa/i,
            address: /địa chỉ/i,
          });
          if (!mapped) showError(error, 'Không thể lưu chi nhánh');
        },
      },
    );
  };

  return (
    <Modal
      open={open}
      title={isEdit ? `Chi nhánh ${record?.code}` : 'Thêm chi nhánh'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm chi nhánh'}
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
              label="Mã chi nhánh"
              tooltip="2–20 ký tự in hoa, số, '-' hoặc '_'. Không trùng."
              normalize={(v: string) => v?.toUpperCase()}
              rules={[
                { required: true, message: 'Vui lòng nhập mã chi nhánh' },
                {
                  pattern: /^[A-Z0-9_-]{2,20}$/,
                  message: "Mã gồm 2-20 ký tự in hoa, số, '-' hoặc '_'",
                },
              ]}
            >
              <Input placeholder="VD: HN01" maxLength={20} className="code-text" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={16}>
            <Form.Item
              name="name"
              label="Tên chi nhánh"
              rules={[
                { required: true, whitespace: true, message: 'Vui lòng nhập tên chi nhánh' },
                { max: 150, message: 'Tên tối đa 150 ký tự' },
              ]}
            >
              <Input placeholder="VD: GymMaster Cầu Giấy" maxLength={150} />
            </Form.Item>
          </Col>

          <Col xs={24}>
            <Form.Item
              name="address"
              label="Địa chỉ"
              rules={[{ max: 255, message: 'Địa chỉ tối đa 255 ký tự' }]}
            >
              <Input placeholder="Số nhà, đường, quận/huyện, tỉnh/thành" maxLength={255} />
            </Form.Item>
          </Col>

          <Col xs={24} sm={12}>
            <Form.Item
              name="phone"
              label="Số điện thoại"
              rules={[{ pattern: /^[0-9+ ]{8,20}$/, message: 'Số điện thoại không hợp lệ' }]}
            >
              <Input placeholder="VD: 0241234567" maxLength={20} inputMode="tel" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="email"
              label="Email"
              rules={[{ type: 'email', message: 'Email không hợp lệ' }]}
            >
              <Input placeholder="VD: caugiay@gymmaster.vn" inputMode="email" />
            </Form.Item>
          </Col>

          <Col xs={12} sm={8}>
            <Form.Item
              name="open_time"
              label="Giờ mở cửa"
              rules={[{ required: true, message: 'Vui lòng nhập giờ mở cửa' }]}
            >
              <TimePicker
                format={TIME_FORMAT}
                minuteStep={5}
                needConfirm={false}
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item
              name="close_time"
              label="Giờ đóng cửa"
              dependencies={['open_time']}
              rules={[
                { required: true, message: 'Vui lòng nhập giờ đóng cửa' },
                ({ getFieldValue }) => ({
                  validator: (_, value?: Dayjs) => {
                    const openTime = getFieldValue('open_time') as Dayjs | undefined;
                    if (!value || !openTime) return Promise.resolve();
                    return value.format(TIME_FORMAT) > openTime.format(TIME_FORMAT)
                      ? Promise.resolve()
                      : Promise.reject(new Error('Giờ đóng cửa phải sau giờ mở cửa'));
                  },
                }),
              ]}
            >
              <TimePicker
                format={TIME_FORMAT}
                minuteStep={5}
                needConfirm={false}
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item
              name="status"
              label="Trạng thái"
              rules={[{ required: true, message: 'Vui lòng chọn trạng thái' }]}
            >
              <Segmented
                block
                options={[
                  { label: 'Hoạt động', value: 'ACTIVE' },
                  { label: 'Ngừng', value: 'INACTIVE', title: STATUS_PRESETS.INACTIVE.label },
                ]}
              />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}
