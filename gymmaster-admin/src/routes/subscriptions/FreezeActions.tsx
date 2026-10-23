import { useState } from 'react';
import { App, Button, Form, Input, Modal, Space } from 'antd';
import { CheckOutlined, CloseOutlined, StopOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Freeze } from '@/api/subscription';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { showError, validateForm } from '@/utils/errors';
import { formatDate } from '@/utils/format';
import { useReviewFreeze } from './queries';

type Action = 'approve' | 'reject' | 'end';

const TEXT: Record<Action, { title: string; ok: string; done: string; danger?: boolean }> = {
  approve: { title: 'Duyệt bảo lưu', ok: 'Duyệt', done: 'Đã duyệt yêu cầu bảo lưu' },
  reject: { title: 'Từ chối bảo lưu', ok: 'Từ chối', done: 'Đã từ chối yêu cầu', danger: true },
  end: { title: 'Kết thúc bảo lưu', ok: 'Xác nhận', done: 'Đã kết thúc bảo lưu', danger: true },
};

/** Bảo lưu đang áp dụng (đã duyệt, đã tới ngày bắt đầu). */
const isApplied = (f: Freeze) =>
  f.status === 'APPROVED' && !dayjs(f.start_date).isAfter(dayjs(), 'day');

/** Nút Duyệt / Từ chối / Kết thúc sớm (hủy yêu cầu) cho một yêu cầu bảo lưu – kèm ô ghi chú. */
export default function FreezeActions({
  freeze,
  size = 'small',
}: {
  freeze: Freeze;
  size?: 'small' | 'middle';
}) {
  const { message } = App.useApp();
  const { can } = usePermission();
  const review = useReviewFreeze();
  const [action, setAction] = useState<Action | null>(null);
  const [form] = Form.useForm<{ note?: string }>();

  const canApprove = can(PERM.SUBSCRIPTION_APPROVE_FREEZE);
  const canEnd = can([PERM.SUBSCRIPTION_FREEZE, PERM.SUBSCRIPTION_APPROVE_FREEZE]);
  const pending = freeze.status === 'PENDING';
  const approved = freeze.status === 'APPROVED';
  const applied = isApplied(freeze);

  const open = (a: Action) => setAction(a);

  const submit = async () => {
    if (!action) return;
    const parsed = await validateForm(form);
    if (!parsed) return;
    const { note } = parsed;
    review.mutate(
      { id: freeze.id, action, note: note?.trim() || '' },
      {
        onSuccess: () => {
          message.success(TEXT[action].done);
          setAction(null);
        },
        onError: (e) => showError(e, 'Không thể xử lý yêu cầu'),
      },
    );
  };

  const endLabel = applied ? 'Kết thúc sớm' : 'Hủy yêu cầu';
  const endDesc = applied
    ? `Hội viên tập lại từ hôm nay; số ngày bảo lưu chưa dùng được trả lại và ngày kết thúc gói lùi tương ứng (bảo lưu ${formatDate(freeze.start_date)} – ${formatDate(freeze.end_date)}).`
    : 'Yêu cầu bảo lưu sẽ bị hủy, không ảnh hưởng gói tập.';

  return (
    <>
      <Space size={4} wrap>
        {pending && canApprove && (
          <>
            <Button
              size={size}
              type="primary"
              icon={<CheckOutlined />}
              onClick={() => open('approve')}
            >
              Duyệt
            </Button>
            <Button size={size} danger icon={<CloseOutlined />} onClick={() => open('reject')}>
              Từ chối
            </Button>
          </>
        )}
        {(pending || approved) && canEnd && (
          <Button size={size} icon={<StopOutlined />} onClick={() => open('end')}>
            {endLabel}
          </Button>
        )}
      </Space>
      <Modal
        open={!!action}
        title={
          action
            ? `${action === 'end' ? endLabel : TEXT[action].title} – ${freeze.member_name}`
            : ''
        }
        okText={action ? TEXT[action].ok : ''}
        okButtonProps={{ danger: action ? TEXT[action].danger : false }}
        cancelText="Hủy"
        confirmLoading={review.isPending}
        onOk={submit}
        onCancel={() => setAction(null)}
        destroyOnHidden
      >
        <p style={{ marginTop: 0 }}>
          {action === 'end'
            ? endDesc
            : `${freeze.plan_name}: bảo lưu ${freeze.days} ngày (${formatDate(freeze.start_date)} – ${formatDate(freeze.end_date)}). Lý do: ${freeze.reason || '—'}`}
        </p>
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item
            name="note"
            label="Ghi chú"
            rules={[{ max: 255, message: 'Ghi chú tối đa 255 ký tự' }]}
          >
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
