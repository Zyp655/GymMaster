import { useEffect, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Col,
  DatePicker,
  Form,
  Input,
  Modal,
  Row,
  Select,
  TimePicker,
} from 'antd';
import { DeleteOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { Shift, ShiftInput } from '@/api/shift';
import type { StaffOption } from '@/api/staff';
import ConfirmButton from '@/components/ConfirmButton';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { getErrorMessage, isFormError, setFormError, showError } from '@/utils/errors';
import { API_DATE_FORMAT, DATE_FORMAT, TIME_FORMAT } from '@/utils/format';
import { POSITION_META } from '../staff/constants';
import { useDeleteShift, useSaveShift } from './queries';
import { shiftDate, shiftTime } from './utils';

interface FormValues {
  staff_id: number;
  date: Dayjs;
  start: Dayjs;
  end: Dayjs;
  note?: string;
}

export interface ShiftFormState {
  open: boolean;
  record: Shift | null;
  /** Gợi ý khi thêm mới từ ô trống. */
  staffId?: number;
  date?: Dayjs;
}

interface Props extends ShiftFormState {
  branchId: number;
  branchName?: string;
  staffOptions: StaffOption[];
  staffLoading?: boolean;
  onClose: () => void;
}

/** Modal thêm/sửa/xóa ca làm. Giờ gửi dạng `YYYY-MM-DDTHH:mm` (backend hiểu là giờ VN). */
export default function ShiftForm({
  open,
  record,
  staffId,
  date,
  branchId,
  branchName,
  staffOptions,
  staffLoading,
  onClose,
}: Props) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const { can } = usePermission();
  const save = useSaveShift();
  const remove = useDeleteShift();
  const [conflict, setConflict] = useState<string | null>(null);
  const readOnly = !can(PERM.SHIFT_MANAGE);
  const isEdit = !!record;

  const start = Form.useWatch('start', form);
  const end = Form.useWatch('end', form);
  const overnight = !!start && !!end && end.format(TIME_FORMAT) < start.format(TIME_FORMAT);

  useEffect(() => {
    if (!open) return;
    setConflict(null);
    form.resetFields();
    if (record) {
      form.setFieldsValue({
        staff_id: record.staff_id,
        date: dayjs(shiftDate(record.start_at)),
        start: dayjs(shiftTime(record.start_at), TIME_FORMAT),
        end: dayjs(shiftTime(record.end_at), TIME_FORMAT),
        note: record.note,
      });
    } else {
      form.setFieldsValue({
        staff_id: staffId,
        date: date ?? dayjs(),
        start: dayjs('06:00', TIME_FORMAT),
        end: dayjs('14:00', TIME_FORMAT),
      });
    }
  }, [open, record, staffId, date, form]);

  // Nhân viên của ca cũ có thể không còn trong danh sách (đã khóa / rời chi nhánh) → vẫn hiển thị tên.
  const options = [
    ...staffOptions.map((s) => ({
      value: s.id,
      label: `${s.full_name} (${s.staff_code} · ${POSITION_META[s.position]?.label ?? s.position})`,
    })),
    ...(record && !staffOptions.some((s) => s.id === record.staff_id)
      ? [{ value: record.staff_id, label: `${record.staff_name} (${record.staff_code})` }]
      : []),
  ];

  const onFinish = (v: FormValues) => {
    setConflict(null);
    const day = v.date.format(API_DATE_FORMAT);
    const endDay = overnight ? v.date.add(1, 'day').format(API_DATE_FORMAT) : day;
    const data: ShiftInput = {
      branch_id: branchId,
      staff_id: v.staff_id,
      start_at: `${day}T${v.start.format(TIME_FORMAT)}`,
      end_at: `${endDay}T${v.end.format(TIME_FORMAT)}`,
      note: v.note?.trim() ?? '',
    };
    save.mutate(
      { id: record?.id, data },
      {
        onSuccess: () => {
          message.success(isEdit ? 'Đã cập nhật ca làm' : 'Đã thêm ca làm');
          onClose();
        },
        onError: (e) => {
          if (isFormError(e) && /trùng/i.test(getErrorMessage(e))) {
            setConflict(getErrorMessage(e));
            form.setFields([{ name: 'staff_id', errors: [getErrorMessage(e)] }]);
            return;
          }
          const mapped = setFormError(form, e, {
            staff_id: /nhân viên/i,
            end: /kết thúc|16 giờ/i,
            start: /bắt đầu/i,
            note: /ghi chú/i,
          });
          if (!mapped) showError(e, 'Không thể lưu ca làm');
        },
      },
    );
  };

  return (
    <Modal
      open={open}
      title={isEdit ? 'Ca làm việc' : 'Thêm ca làm'}
      onCancel={onClose}
      width={520}
      forceRender
      footer={
        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', flexWrap: 'wrap' }}>
          {isEdit && record && (
            <ConfirmButton
              perm={PERM.SHIFT_MANAGE}
              danger
              icon={<DeleteOutlined />}
              style={{ marginRight: 'auto' }}
              confirmTitle="Xóa ca làm này?"
              confirmContent={`${record.staff_name}: ${shiftTime(record.start_at)}–${shiftTime(record.end_at)} ngày ${dayjs(shiftDate(record.start_at)).format(DATE_FORMAT)}`}
              onConfirm={() =>
                remove.mutateAsync(record.id, {
                  onSuccess: () => {
                    message.success('Đã xóa ca làm');
                    onClose();
                  },
                  onError: (e) => showError(e, 'Không thể xóa ca làm'),
                })
              }
            >
              Xóa ca
            </ConfirmButton>
          )}
          <Button onClick={onClose}>{readOnly ? 'Đóng' : 'Hủy'}</Button>
          {!readOnly && (
            <Button type="primary" loading={save.isPending} onClick={() => form.submit()}>
              {isEdit ? 'Lưu thay đổi' : 'Thêm ca'}
            </Button>
          )}
        </div>
      }
    >
      {conflict && (
        <Alert
          type="error"
          showIcon
          style={{ margin: '12px 0' }}
          message={conflict}
          description="Nhân viên đã có ca khác giao với khung giờ này (tính trên mọi chi nhánh). Hãy đổi giờ hoặc chọn nhân viên khác."
        />
      )}
      <Form<FormValues>
        form={form}
        layout="vertical"
        disabled={readOnly}
        onFinish={onFinish}
        onValuesChange={() => conflict && setConflict(null)}
        style={{ marginTop: 12 }}
      >
        {branchName && (
          <Form.Item label="Chi nhánh" style={{ marginBottom: 12 }}>
            <Input value={branchName} disabled />
          </Form.Item>
        )}
        <Form.Item
          name="staff_id"
          label="Nhân viên"
          rules={[{ required: true, message: 'Vui lòng chọn nhân viên' }]}
        >
          <Select
            showSearch
            optionFilterProp="label"
            placeholder="Chọn nhân viên"
            loading={staffLoading}
            options={options}
            notFoundContent="Chi nhánh chưa có nhân viên đang làm việc"
          />
        </Form.Item>
        <Row gutter={12}>
          <Col xs={24} sm={10}>
            <Form.Item
              name="date"
              label="Ngày"
              rules={[{ required: true, message: 'Vui lòng chọn ngày' }]}
            >
              <DatePicker format={DATE_FORMAT} style={{ width: '100%' }} allowClear={false} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={7}>
            <Form.Item
              name="start"
              label="Bắt đầu"
              rules={[{ required: true, message: 'Nhập giờ bắt đầu' }]}
            >
              <TimePicker
                format={TIME_FORMAT}
                minuteStep={15}
                needConfirm={false}
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
          <Col xs={12} sm={7}>
            <Form.Item
              name="end"
              label="Kết thúc"
              dependencies={['start']}
              extra={overnight ? 'Kết thúc vào ngày hôm sau' : undefined}
              rules={[
                { required: true, message: 'Nhập giờ kết thúc' },
                ({ getFieldValue }) => ({
                  validator: (_, value?: Dayjs) => {
                    const s = getFieldValue('start') as Dayjs | undefined;
                    if (!value || !s) return Promise.resolve();
                    const sm = s.hour() * 60 + s.minute();
                    let em = value.hour() * 60 + value.minute();
                    if (em === sm)
                      return Promise.reject(new Error('Giờ kết thúc phải sau giờ bắt đầu'));
                    if (em < sm) em += 24 * 60;
                    return em - sm > 16 * 60
                      ? Promise.reject(new Error('Một ca làm không được dài quá 16 giờ'))
                      : Promise.resolve();
                  },
                }),
              ]}
            >
              <TimePicker
                format={TIME_FORMAT}
                minuteStep={15}
                needConfirm={false}
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
        </Row>
        <Form.Item
          name="note"
          label="Ghi chú"
          rules={[{ max: 255, message: 'Ghi chú tối đa 255 ký tự' }]}
        >
          <Input maxLength={255} placeholder="VD: Ca sáng" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
