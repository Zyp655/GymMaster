import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  App,
  Checkbox,
  DatePicker,
  Empty,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { MemberOption } from '@/api/member';
import type { PtContract, PtSession } from '@/api/pt';
import MemberSelect from '@/components/MemberSelect';
import StatusTag from '@/components/StatusTag';
import { toApiDateTime, weekdayName } from '@/routes/classes/constants';
import { setFormError, showError, validateForm } from '@/utils/errors';
import { DATETIME_FORMAT, formatDate } from '@/utils/format';
import { PT_CONTRACT_STATUS_MAP, isBookable } from './constants';
import {
  useBookSession,
  useCancelBeforeMinutes,
  useCancelSession,
  useCompleteSession,
  useMemberPtContracts,
  useNoShowSession,
  useRescheduleSession,
} from './queries';

const START_RULES = [{ required: true, message: 'Vui lòng chọn giờ bắt đầu' }];

/** Ô chọn ngày giờ bắt đầu (bước 15 phút, không chọn ngày quá khứ). */
function StartPicker(props: { value?: Dayjs | null; onChange?: (v: Dayjs | null) => void }) {
  return (
    <DatePicker
      {...props}
      showTime={{ format: 'HH:mm', minuteStep: 15, hideDisabledOptions: true }}
      format={DATETIME_FORMAT}
      style={{ width: '100%' }}
      placeholder="Chọn ngày giờ"
      disabledDate={(d) => d.isBefore(dayjs(), 'day')}
      needConfirm={false}
    />
  );
}

const sessionLabel = (s: Pick<PtSession, 'start_at' | 'end_at'>) => {
  const st = dayjs(s.start_at);
  return `${st.format('HH:mm')}–${dayjs(s.end_at).format('HH:mm')} · ${weekdayName(st)}, ${st.format('DD/MM/YYYY')}`;
};

/* ================================================================== Đặt buổi */

export interface BookSessionModalProps {
  open: boolean;
  onClose: () => void;
  /** Hợp đồng chọn sẵn (mở từ chi tiết hợp đồng). */
  contract?: PtContract | null;
  /** Hội viên chọn sẵn. */
  member?: MemberOption | null;
  /** Giờ bắt đầu chọn sẵn (bấm ô trống trên lịch). */
  initialAt?: Dayjs | null;
  /** Ưu tiên hợp đồng của HLV này (lịch đang lọc theo HLV). */
  trainerId?: number;
  onBooked?: (s: PtSession) => void;
}

interface BookForm {
  contract_id?: number;
  start_at?: Dayjs | null;
}

export function BookSessionModal({
  open,
  onClose,
  contract,
  member,
  initialAt,
  trainerId,
  onBooked,
}: BookSessionModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<BookForm>();
  const [memberId, setMemberId] = useState<number | undefined>();
  const [memberObj, setMemberObj] = useState<MemberOption | undefined>();
  const book = useBookSession();
  const fixedContract = !!contract;

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    setMemberId(contract?.member_id ?? member?.id);
    setMemberObj(member ?? undefined);
    form.setFieldsValue({ contract_id: contract?.id, start_at: initialAt ?? null });
    book.reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, contract?.id, member?.id, initialAt]);

  const contracts = useMemberPtContracts(fixedContract ? null : memberId);
  const options = useMemo(() => {
    const list = (contracts.data ?? []).filter(isBookable);
    return [...list].sort(
      (a, b) => Number(b.trainer_id === trainerId) - Number(a.trainer_id === trainerId),
    );
  }, [contracts.data, trainerId]);

  // Chỉ có 1 hợp đồng đặt được → chọn sẵn.
  useEffect(() => {
    if (!fixedContract && options.length === 1 && !form.getFieldValue('contract_id'))
      form.setFieldValue('contract_id', options[0].id);
  }, [options, fixedContract, form]);

  const contractId = Form.useWatch('contract_id', form);
  const selected = contract ?? options.find((c) => c.id === contractId);
  const startAt = Form.useWatch('start_at', form);

  const submit = async () => {
    const v = await validateForm(form);
    if (!v) return;
    if (!v.contract_id || !v.start_at) return;
    book.mutate(
      { contractId: v.contract_id, startAt: toApiDateTime(v.start_at) },
      {
        onSuccess: (s) => {
          message.success(`Đã đặt buổi PT ${sessionLabel(s)}`);
          onBooked?.(s);
          onClose();
        },
        onError: (e) => {
          if (!setFormError(form, e, { start_at: /giờ|trùng|thời hạn|tương lai|mở cửa/i }))
            showError(e, 'Không thể đặt buổi');
        },
      },
    );
  };

  return (
    <Modal
      open={open}
      title="Đặt buổi PT"
      okText="Đặt buổi"
      onOk={submit}
      confirmLoading={book.isPending}
      onCancel={onClose}
      destroyOnHidden
      maskClosable={false}
    >
      <Form form={form} layout="vertical">
        {!fixedContract && (
          <Form.Item label="Hội viên" required>
            <MemberSelect
              value={memberId}
              initialMember={memberObj}
              activeOnly
              autoFocus={!member}
              onChange={(id, m) => {
                setMemberId(id);
                setMemberObj(m);
                form.setFieldValue('contract_id', undefined);
              }}
            />
          </Form.Item>
        )}
        <Form.Item
          name="contract_id"
          label="Hợp đồng PT"
          rules={[{ required: true, message: 'Vui lòng chọn hợp đồng PT' }]}
          hidden={fixedContract}
        >
          <Select
            placeholder={memberId ? 'Chọn hợp đồng đang tập' : 'Chọn hội viên trước'}
            disabled={!memberId}
            loading={contracts.isFetching}
            notFoundContent={
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="Hội viên không có hợp đồng PT còn buổi để đặt"
              />
            }
            options={options.map((c) => ({
              value: c.id,
              label: `${c.package_name} · HLV ${c.trainer_name} · còn đặt ${c.bookable_sessions} buổi`,
            }))}
          />
        </Form.Item>
        {selected && (
          <Alert
            type="info"
            style={{ marginBottom: 16 }}
            message={
              <Space direction="vertical" size={0}>
                <span>
                  <b>{selected.member_name}</b> · {selected.package_name}{' '}
                  <StatusTag status={selected.status} map={PT_CONTRACT_STATUS_MAP} />
                </span>
                <span>
                  HLV {selected.trainer_name} · {selected.branch_name} · {selected.session_minutes}{' '}
                  phút/buổi
                </span>
                <span>
                  Thời hạn {formatDate(selected.start_date)} – {formatDate(selected.end_date)} · còn
                  đặt được <b>{selected.bookable_sessions}</b> buổi
                </span>
              </Space>
            }
          />
        )}
        <Form.Item
          name="start_at"
          label="Giờ bắt đầu"
          rules={START_RULES}
          extra={
            selected && startAt
              ? `Kết thúc lúc ${startAt.add(selected.session_minutes, 'minute').format('HH:mm')}`
              : undefined
          }
        >
          <StartPicker />
        </Form.Item>
      </Form>
    </Modal>
  );
}

/* ================================================================== Đổi giờ */

export interface SessionModalProps {
  session: PtSession | null;
  onClose: () => void;
}

export function RescheduleModal({ session, onClose }: SessionModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<{ start_at?: Dayjs | null }>();
  const mutation = useRescheduleSession();

  useEffect(() => {
    if (session) form.setFieldsValue({ start_at: dayjs(session.start_at) });
  }, [session, form]);

  const submit = async () => {
    if (!session) return;
    const v = await validateForm(form);
    if (!v) return;
    if (!v.start_at) return;
    mutation.mutate(
      { id: session.id, startAt: toApiDateTime(v.start_at) },
      {
        onSuccess: (s) => {
          message.success(`Đã đổi giờ sang ${sessionLabel(s)}`);
          onClose();
        },
        onError: (e) => {
          if (!setFormError(form, e, { start_at: /giờ|trùng|thời hạn|tương lai|mở cửa/i }))
            showError(e, 'Không thể đổi giờ');
        },
      },
    );
  };

  return (
    <Modal
      open={!!session}
      title="Đổi giờ buổi PT"
      okText="Lưu giờ mới"
      onOk={submit}
      confirmLoading={mutation.isPending}
      onCancel={onClose}
      destroyOnHidden
    >
      {session && (
        <Typography.Paragraph type="secondary">
          {session.member_name} · HLV {session.trainer_name} · hiện tại {sessionLabel(session)}
        </Typography.Paragraph>
      )}
      <Form form={form} layout="vertical">
        <Form.Item name="start_at" label="Giờ bắt đầu mới" rules={START_RULES}>
          <StartPicker />
        </Form.Item>
      </Form>
    </Modal>
  );
}

/* ================================================================== Hủy buổi */

export function CancelSessionModal({ session, onClose }: SessionModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<{ reason: string; count_as_used: boolean }>();
  const mutation = useCancelSession();
  const beforeMin = useCancelBeforeMinutes();

  useEffect(() => {
    if (session) form.setFieldsValue({ reason: '', count_as_used: false });
  }, [session, form]);

  const deadline = session ? dayjs(session.start_at).subtract(beforeMin, 'minute') : null;
  const late = !!deadline && dayjs().isAfter(deadline);

  const submit = async () => {
    if (!session) return;
    const v = await validateForm(form);
    if (!v) return;
    mutation.mutate(
      {
        id: session.id,
        data: { reason: v.reason.trim(), count_as_used: late && !!v.count_as_used },
      },
      {
        onSuccess: (s) => {
          message.success(
            s.status === 'NO_SHOW' ? 'Đã hủy buổi (tính là đã dùng buổi)' : 'Đã hủy buổi PT',
          );
          onClose();
        },
        onError: (e) => showError(e, 'Không thể hủy buổi'),
      },
    );
  };

  return (
    <Modal
      open={!!session}
      title="Hủy buổi PT"
      okText="Xác nhận hủy buổi"
      okButtonProps={{ danger: true }}
      onOk={submit}
      confirmLoading={mutation.isPending}
      onCancel={onClose}
      destroyOnHidden
    >
      {session && deadline && (
        <Alert
          showIcon
          type={late ? 'warning' : 'success'}
          style={{ marginBottom: 16 }}
          message={late ? 'Hủy trễ hạn' : 'Hủy đúng hạn – không trừ buổi'}
          description={
            <>
              Buổi {sessionLabel(session)} của {session.member_name}. Hạn hủy:{' '}
              {deadline.format('HH:mm DD/MM/YYYY')} ({Math.round(beforeMin / 60)} giờ trước giờ
              tập).
              {late && ' Có thể châm chước (không trừ buổi) hoặc tính là đã dùng buổi.'}
            </>
          }
        />
      )}
      <Form form={form} layout="vertical">
        <Form.Item
          name="reason"
          label="Lý do hủy"
          rules={[
            { required: true, whitespace: true, message: 'Vui lòng nhập lý do hủy' },
            { max: 255, message: 'Lý do tối đa 255 ký tự' },
          ]}
        >
          <Input.TextArea rows={2} placeholder="VD: Khách báo ốm" autoFocus />
        </Form.Item>
        {late && (
          <Form.Item name="count_as_used" valuePropName="checked">
            <Checkbox>Tính là đã dùng buổi (trừ 1 buổi, không tính hoa hồng HLV)</Checkbox>
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
}

/* ================================================================== Hoàn thành */

export function CompleteSessionModal({ session, onClose }: SessionModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<{ workout_note: string; trainer_note: string }>();
  const mutation = useCompleteSession();

  useEffect(() => {
    if (session)
      form.setFieldsValue({
        workout_note: session.workout_note ?? '',
        trainer_note: session.trainer_note ?? '',
      });
  }, [session, form]);

  const submit = async () => {
    if (!session) return;
    const v = await validateForm(form);
    if (!v) return;
    mutation.mutate(
      {
        id: session.id,
        data: {
          workout_note: v.workout_note?.trim() ?? '',
          trainer_note: v.trainer_note?.trim() ?? '',
        },
      },
      {
        onSuccess: () => {
          message.success('Đã xác nhận hoàn thành buổi PT');
          onClose();
        },
        onError: (e) => showError(e, 'Không thể xác nhận hoàn thành'),
      },
    );
  };

  return (
    <Modal
      open={!!session}
      title="Xác nhận hoàn thành buổi PT"
      okText="Hoàn thành"
      onOk={submit}
      confirmLoading={mutation.isPending}
      onCancel={onClose}
      destroyOnHidden
      width={640}
    >
      {session && (
        <Typography.Paragraph type="secondary">
          {session.member_name} · HLV {session.trainer_name} · {sessionLabel(session)}. Hoàn thành
          sẽ trừ 1 buổi của hợp đồng và ghi nhận hoa hồng cho HLV.
        </Typography.Paragraph>
      )}
      <Form form={form} layout="vertical">
        <Form.Item
          name="workout_note"
          label="Giáo án buổi tập"
          rules={[{ max: 5000, message: 'Giáo án tối đa 5000 ký tự' }]}
        >
          <Input.TextArea
            rows={5}
            placeholder={'VD:\nKhởi động 10 phút\nSquat 5x5 @60kg\nBench press 3x8'}
            autoFocus
          />
        </Form.Item>
        <Form.Item
          name="trainer_note"
          label="Nhận xét của HLV"
          rules={[{ max: 5000, message: 'Nhận xét tối đa 5000 ký tự' }]}
        >
          <Input.TextArea rows={3} placeholder="VD: Tiến bộ tốt, cần cải thiện kỹ thuật hít thở" />
        </Form.Item>
      </Form>
    </Modal>
  );
}

/* ================================================================== Vắng mặt */

export function NoShowModal({ session, onClose }: SessionModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm<{ note: string }>();
  const mutation = useNoShowSession();

  useEffect(() => {
    if (session) form.setFieldsValue({ note: '' });
  }, [session, form]);

  const submit = async () => {
    if (!session) return;
    const v = await validateForm(form);
    if (!v) return;
    mutation.mutate(
      { id: session.id, note: v.note?.trim() || undefined },
      {
        onSuccess: () => {
          message.success('Đã ghi nhận hội viên vắng mặt');
          onClose();
        },
        onError: (e) => showError(e, 'Không thể đánh dấu vắng mặt'),
      },
    );
  };

  return (
    <Modal
      open={!!session}
      title="Đánh dấu vắng mặt"
      okText="Xác nhận vắng mặt"
      okButtonProps={{ danger: true }}
      onOk={submit}
      confirmLoading={mutation.isPending}
      onCancel={onClose}
      destroyOnHidden
    >
      <Alert
        type="warning"
        showIcon
        style={{ marginBottom: 16 }}
        message="Buổi vắng mặt vẫn bị trừ vào hợp đồng nhưng HLV không được tính hoa hồng."
      />
      <Form form={form} layout="vertical">
        <Form.Item
          name="note"
          label="Ghi chú"
          rules={[{ max: 255, message: 'Ghi chú tối đa 255 ký tự' }]}
        >
          <Input placeholder="Mặc định: Hội viên vắng mặt" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
