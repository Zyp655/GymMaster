import { useEffect } from 'react';
import { Alert, App, Form, Input, Modal, Radio, Space, Typography } from 'antd';
import dayjs from 'dayjs';
import type { CancelScope, ClassSchedule } from '@/api/class';
import { setFormError, showError } from '@/utils/errors';
import { weekdayName } from './constants';
import { useCancelSchedule } from './queries';

interface FormValues {
  reason: string;
  scope: CancelScope;
}

export interface CancelScheduleModalProps {
  schedule: ClassSchedule | null;
  open: boolean;
  onClose: () => void;
  onDone?: () => void;
}

/** Hủy buổi lớp: lý do + phạm vi (chỉ buổi này / cả các buổi sau trong chuỗi). */
export default function CancelScheduleModal({
  schedule,
  open,
  onClose,
  onDone,
}: CancelScheduleModalProps) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const cancel = useCancelSchedule();
  const isSeries = !!schedule?.series_id;
  const affected = (schedule?.booked_count ?? 0) + (schedule?.waitlist_count ?? 0);

  useEffect(() => {
    if (open) {
      form.resetFields();
      form.setFieldsValue({ scope: 'one' });
    }
  }, [open, form]);

  const onFinish = (v: FormValues) => {
    if (!schedule) return;
    cancel.mutate(
      { id: schedule.id, data: { reason: v.reason.trim(), scope: isSeries ? v.scope : 'one' } },
      {
        onSuccess: (res) => {
          message.success(
            `Đã hủy ${res.cancelled_schedules} buổi lớp` +
              (res.cancelled_bookings ? `, hủy ${res.cancelled_bookings} lượt đặt chỗ` : ''),
          );
          onClose();
          onDone?.();
        },
        onError: (e) => {
          if (!setFormError(form, e, { reason: /lý do/i })) showError(e, 'Không thể hủy buổi lớp');
        },
      },
    );
  };

  const start = schedule ? dayjs(schedule.start_at) : null;

  return (
    <Modal
      open={open}
      title="Hủy buổi lớp"
      okText="Hủy buổi lớp"
      okButtonProps={{ danger: true }}
      cancelText="Đóng"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={cancel.isPending}
      forceRender
    >
      {schedule && start && (
        <Typography.Paragraph style={{ marginTop: 8 }}>
          <b>{schedule.class_type_name}</b> – {weekdayName(start)},{' '}
          {start.format('DD/MM/YYYY HH:mm')} · {schedule.room_name}
        </Typography.Paragraph>
      )}
      {affected > 0 && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 12 }}
          message={`${schedule?.booked_count} lượt đặt chỗ và ${schedule?.waitlist_count} lượt chờ của buổi này sẽ bị hủy. Hội viên có tài khoản sẽ nhận thông báo.`}
        />
      )}
      <Form<FormValues> form={form} layout="vertical" onFinish={onFinish}>
        <Form.Item
          name="reason"
          label="Lý do hủy"
          rules={[
            { required: true, whitespace: true, message: 'Vui lòng nhập lý do hủy' },
            { max: 255, message: 'Lý do tối đa 255 ký tự' },
          ]}
        >
          <Input.TextArea rows={3} maxLength={255} showCount placeholder="VD: HLV nghỉ phép" />
        </Form.Item>
        {isSeries && (
          <Form.Item name="scope" label="Phạm vi">
            <Radio.Group>
              <Space direction="vertical">
                <Radio value="one">Chỉ buổi này</Radio>
                <Radio value="following">Buổi này và các buổi sau trong chuỗi lịch lặp</Radio>
              </Space>
            </Radio.Group>
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
}
