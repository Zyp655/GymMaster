import { useEffect } from 'react';
import { Alert, App, DatePicker, Form, Input, Modal, Typography } from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { SubscriptionDetail } from '@/api/subscription';
import { setFormError, showError, validateForm } from '@/utils/errors';
import { DATE_FORMAT, formatDate, toApiDate } from '@/utils/format';
import { useRequestFreeze } from './queries';
import { freezeQuota } from './utils';

export interface FreezeModalProps {
  open: boolean;
  sub: SubscriptionDetail | null;
  onClose: () => void;
}

interface FreezeForm {
  range: [Dayjs, Dayjs];
  reason?: string;
}

/** Tạo yêu cầu bảo lưu: khoảng ngày + lý do, hiển thị hạn mức còn lại. */
export default function FreezeModal({ open, sub, onClose }: FreezeModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<FreezeForm>();
  const request = useRequestFreeze();
  const range = Form.useWatch('range', form);

  useEffect(() => {
    if (open) form.resetFields();
  }, [open, form]);

  if (!sub) return null;
  const q = freezeQuota(sub);
  const days = range?.[0] && range?.[1] ? range[1].diff(range[0], 'day') + 1 : 0;

  const submit = async () => {
    const v = await validateForm(form);
    if (!v) return;
    request.mutate(
      {
        id: sub.id,
        data: {
          start_date: toApiDate(v.range[0])!,
          end_date: toApiDate(v.range[1])!,
          reason: v.reason?.trim() || '',
        },
      },
      {
        onSuccess: () => {
          message.success('Đã gửi yêu cầu bảo lưu – chờ quản lý duyệt');
          onClose();
        },
        onError: (e) => {
          if (!setFormError(form, e, { range: /ngày|bảo lưu|trùng/i }))
            showError(e, 'Không thể tạo yêu cầu');
        },
      },
    );
  };

  return (
    <Modal
      open={open}
      title={`Bảo lưu gói – ${sub.member_name}`}
      okText="Gửi yêu cầu"
      cancelText="Hủy"
      onOk={submit}
      confirmLoading={request.isPending}
      onCancel={onClose}
      forceRender
    >
      <Alert
        type={q.timesLeft > 0 && q.daysLeft > 0 ? 'info' : 'warning'}
        showIcon
        style={{ marginBottom: 16 }}
        message={
          q.maxTimes > 0
            ? `Hạn mức còn lại: ${q.daysLeft}/${q.maxDays} ngày, ${q.timesLeft}/${q.maxTimes} lần`
            : 'Gói tập này không cho phép bảo lưu'
        }
        description={`Gói ${sub.plan_name}: ${formatDate(sub.start_date)} – ${formatDate(sub.end_date)}. Ngày kết thúc gói được cộng thêm số ngày bảo lưu khi được duyệt.`}
      />
      <Form form={form} layout="vertical">
        <Form.Item
          name="range"
          label="Khoảng bảo lưu"
          rules={[{ required: true, message: 'Vui lòng chọn khoảng ngày' }]}
          extra={
            days > 0 ? (
              <Typography.Text type={days > q.daysLeft ? 'danger' : 'secondary'}>
                {days} ngày
              </Typography.Text>
            ) : undefined
          }
        >
          <DatePicker.RangePicker
            style={{ width: '100%' }}
            format={DATE_FORMAT}
            placeholder={['Từ ngày', 'Đến ngày']}
            disabledDate={(d) =>
              d.isBefore(dayjs(), 'day') || d.isBefore(dayjs(sub.start_date), 'day')
            }
          />
        </Form.Item>
        <Form.Item
          name="reason"
          label="Lý do"
          rules={[{ max: 255, message: 'Lý do tối đa 255 ký tự' }]}
        >
          <Input.TextArea rows={2} placeholder="VD: Đi công tác" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
