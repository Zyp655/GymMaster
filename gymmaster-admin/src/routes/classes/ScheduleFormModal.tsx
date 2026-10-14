import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  App,
  Checkbox,
  Col,
  DatePicker,
  Descriptions,
  Form,
  Input,
  InputNumber,
  Modal,
  Result,
  Row,
  Segmented,
  Select,
  Table,
  TimePicker,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import type { ClassSchedule, ClassScheduleRecurringResult } from '@/api/class';
import BranchSelect from '@/components/BranchSelect';
import { setFormError, showError } from '@/utils/errors';
import {
  API_DATE_FORMAT,
  DATE_FORMAT,
  TIME_FORMAT,
  formatDate,
  normalizeText,
} from '@/utils/format';
import { WEEKDAY_OPTIONS, toApiDateTime, weekdayName } from './constants';
import {
  useBranchHours,
  useClassTypeOptions,
  useCreateRecurring,
  useCreateSchedule,
  useRoomOptions,
  useTrainerOptions,
  useUpdateSchedule,
} from './queries';

type Mode = 'single' | 'recurring';

interface FormValues {
  branch_id?: number;
  class_type_id?: number;
  room_id?: number;
  trainer_id?: number;
  date?: Dayjs;
  range?: [Dayjs, Dayjs];
  weekdays?: number[];
  time?: Dayjs;
  duration_min?: number;
  capacity?: number;
  note?: string;
}

export interface ScheduleFormModalProps {
  open: boolean;
  /** Có = sửa buổi này; không = thêm lịch. */
  record?: ClassSchedule | null;
  /** Chi nhánh mặc định khi thêm (chi nhánh đang xem). */
  branchId?: number | null;
  /** Thời điểm điền sẵn (bấm ô trống trên lịch). */
  initialAt?: Dayjs | null;
  onClose: () => void;
}

const MAX_RECURRING_DAYS = 83;

const filterByLabel = (input: string, option?: { label?: unknown }) =>
  normalizeText(String(option?.label ?? '')).includes(normalizeText(input));

/** Modal thêm lịch (Một buổi / Lặp hàng tuần) và sửa một buổi lớp. */
export default function ScheduleFormModal({
  open,
  record,
  branchId,
  initialAt,
  onClose,
}: ScheduleFormModalProps) {
  const [form] = Form.useForm<FormValues>();
  const { message } = App.useApp();
  const isEdit = !!record;
  const [mode, setMode] = useState<Mode>('single');
  const [result, setResult] = useState<ClassScheduleRecurringResult | null>(null);

  const createOne = useCreateSchedule();
  const createRecurring = useCreateRecurring();
  const update = useUpdateSchedule();
  const saving = createOne.isPending || createRecurring.isPending || update.isPending;

  const selectedBranch = Form.useWatch('branch_id', form) ?? record?.branch_id;
  const selectedRoomId = Form.useWatch('room_id', form);
  const duration = Form.useWatch('duration_min', form);

  const types = useClassTypeOptions();
  const rooms = useRoomOptions(selectedBranch, open && !!selectedBranch);
  const trainers = useTrainerOptions(selectedBranch, open && !!selectedBranch);
  const hours = useBranchHours();

  const branchHours = hours.data?.find((b) => b.id === selectedBranch);
  const selectedRoom = rooms.data?.find((r) => r.id === selectedRoomId);

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    setMode('single');
    if (record) {
      const start = dayjs(record.start_at);
      form.setFieldsValue({
        branch_id: record.branch_id,
        class_type_id: record.class_type_id,
        room_id: record.room_id,
        trainer_id: record.trainer_id,
        date: start.startOf('day'),
        time: start,
        duration_min: record.duration_min,
        capacity: record.capacity,
        note: record.note,
      });
      return;
    }
    const at = initialAt ?? null;
    form.setFieldsValue({
      branch_id: branchId ?? undefined,
      date: at ? at.startOf('day') : dayjs().add(1, 'day').startOf('day'),
      time: at ?? undefined,
      range: at ? [at.startOf('day'), at.add(4, 'week').startOf('day')] : undefined,
      weekdays: at ? [((at.day() + 6) % 7) + 1] : [],
    });
  }, [open, record, branchId, initialAt, form]);

  const typeOptions = useMemo(
    () =>
      (types.data ?? []).map((t) => ({
        value: t.id,
        label: t.name,
        title: t.code,
        color: t.color,
      })),
    [types.data],
  );

  /** Chọn loại lớp → điền thời lượng + sức chứa mặc định (không vượt sức chứa phòng). */
  const onTypeChange = (id: number) => {
    const t = types.data?.find((x) => x.id === id);
    if (!t) return;
    const cap = selectedRoom
      ? Math.min(t.default_capacity, selectedRoom.capacity)
      : t.default_capacity;
    form.setFieldsValue({ duration_min: t.default_duration_min, capacity: cap });
  };

  const onRoomChange = (id: number) => {
    const room = rooms.data?.find((r) => r.id === id);
    const cap = form.getFieldValue('capacity') as number | undefined;
    if (room && (!cap || cap > room.capacity)) form.setFieldsValue({ capacity: room.capacity });
  };

  const handleError = (error: unknown) => {
    const mapped = setFormError(form, error, {
      capacity: /sức chứa/i,
      trainer_id: /HLV/i,
      room_id: /phòng/i,
      class_type_id: /loại lớp/i,
      range: /12 tuần|ngày kết thúc/i,
      time: /giờ|thời điểm|tương lai/i,
      note: /ghi chú/i,
    });
    if (!mapped)
      showError(error, isEdit ? 'Không thể cập nhật buổi lớp' : 'Không thể tạo lịch lớp');
  };

  const onFinish = (v: FormValues) => {
    const time = v.time as Dayjs;
    if (isEdit && record) {
      const startAt = (v.date as Dayjs).hour(time.hour()).minute(time.minute()).second(0);
      update.mutate(
        {
          id: record.id,
          data: {
            room_id: v.room_id as number,
            trainer_id: v.trainer_id as number,
            start_at: toApiDateTime(startAt),
            duration_min: v.duration_min as number,
            capacity: v.capacity as number,
            note: v.note?.trim() ?? '',
          },
        },
        {
          onSuccess: () => {
            message.success('Đã cập nhật buổi lớp');
            onClose();
          },
          onError: handleError,
        },
      );
      return;
    }

    const common = {
      class_type_id: v.class_type_id as number,
      branch_id: v.branch_id as number,
      room_id: v.room_id as number,
      trainer_id: v.trainer_id as number,
      duration_min: v.duration_min,
      capacity: v.capacity,
      note: v.note?.trim() ?? '',
    };

    if (mode === 'single') {
      const startAt = (v.date as Dayjs).hour(time.hour()).minute(time.minute()).second(0);
      createOne.mutate(
        { ...common, start_at: toApiDateTime(startAt) },
        {
          onSuccess: () => {
            message.success('Đã thêm buổi lớp');
            onClose();
          },
          onError: handleError,
        },
      );
      return;
    }

    const [from, to] = v.range as [Dayjs, Dayjs];
    createRecurring.mutate(
      {
        ...common,
        start_date: from.format(API_DATE_FORMAT),
        end_date: to.format(API_DATE_FORMAT),
        weekdays: [...(v.weekdays ?? [])].sort(),
        start_time: time.format(TIME_FORMAT),
      },
      {
        onSuccess: (res) => {
          onClose();
          setResult(res);
        },
        onError: handleError,
      },
    );
  };

  const hoursText = branchHours ? `${branchHours.open_time} – ${branchHours.close_time}` : null;

  /** Kiểm tra giờ học nằm trong giờ mở cửa (backend vẫn kiểm tra lại). */
  const timeValidator = (_: unknown, value?: Dayjs) => {
    if (!value || !branchHours) return Promise.resolve();
    const start = value.format(TIME_FORMAT);
    const end = value.add(duration ?? 0, 'minute');
    const endText = end.isSame(value, 'day') ? end.format(TIME_FORMAT) : '24:00';
    if (start < branchHours.open_time || endText > branchHours.close_time)
      return Promise.reject(
        new Error(`Lịch lớp phải nằm trong giờ mở cửa của chi nhánh (${hoursText})`),
      );
    return Promise.resolve();
  };

  return (
    <>
      <Modal
        open={open}
        title={isEdit ? `Sửa buổi lớp – ${record?.class_type_name}` : 'Thêm lịch lớp'}
        okText={isEdit ? 'Lưu thay đổi' : mode === 'single' ? 'Thêm buổi lớp' : 'Tạo lịch lặp'}
        cancelText="Hủy"
        onOk={() => form.submit()}
        onCancel={onClose}
        confirmLoading={saving}
        width={720}
        forceRender
        destroyOnHidden={false}
      >
        {!isEdit && (
          <Segmented<Mode>
            block
            value={mode}
            onChange={setMode}
            style={{ margin: '12px 0 4px' }}
            options={[
              { label: 'Một buổi', value: 'single' },
              { label: 'Lặp hàng tuần', value: 'recurring' },
            ]}
          />
        )}
        {isEdit && record && record.booked_count > 0 && (
          <Alert
            type="warning"
            showIcon
            style={{ marginTop: 12 }}
            message={`Buổi lớp đã có ${record.booked_count} chỗ được đặt – sức chứa không được nhỏ hơn số này. Tăng sức chứa sẽ tự đẩy người trong danh sách chờ lên.`}
          />
        )}

        <Form<FormValues>
          form={form}
          layout="vertical"
          onFinish={onFinish}
          style={{ marginTop: 12 }}
        >
          <Row gutter={16}>
            {isEdit && record ? (
              <Col xs={24}>
                <Descriptions
                  size="small"
                  column={{ xs: 1, sm: 2, md: 2, lg: 2, xl: 2, xxl: 2 }}
                  style={{ marginBottom: 12 }}
                  items={[
                    { key: 'type', label: 'Loại lớp', children: record.class_type_name },
                    { key: 'branch', label: 'Chi nhánh', children: record.branch_name },
                  ]}
                />
              </Col>
            ) : (
              <>
                <Col xs={24} sm={12}>
                  <Form.Item
                    name="branch_id"
                    label="Chi nhánh"
                    rules={[{ required: true, message: 'Vui lòng chọn chi nhánh' }]}
                    extra={hoursText ? `Giờ mở cửa: ${hoursText}` : undefined}
                  >
                    <BranchSelect
                      activeOnly
                      onChange={() =>
                        form.setFieldsValue({ room_id: undefined, trainer_id: undefined })
                      }
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12}>
                  <Form.Item
                    name="class_type_id"
                    label="Loại lớp"
                    rules={[{ required: true, message: 'Vui lòng chọn loại lớp' }]}
                  >
                    <Select
                      showSearch
                      placeholder="Chọn loại lớp"
                      loading={types.isLoading}
                      options={typeOptions}
                      filterOption={filterByLabel}
                      onChange={onTypeChange}
                      optionRender={(o) => (
                        <span>
                          <span
                            aria-hidden
                            style={{
                              display: 'inline-block',
                              width: 10,
                              height: 10,
                              borderRadius: 3,
                              marginRight: 8,
                              background: (o.data as { color: string }).color,
                            }}
                          />
                          {o.label}
                        </span>
                      )}
                      notFoundContent="Chưa có loại lớp đang hoạt động"
                    />
                  </Form.Item>
                </Col>
              </>
            )}

            <Col xs={24} sm={12}>
              <Form.Item
                name="room_id"
                label="Phòng"
                rules={[{ required: true, message: 'Vui lòng chọn phòng' }]}
              >
                <Select
                  showSearch
                  placeholder={selectedBranch ? 'Chọn phòng' : 'Chọn chi nhánh trước'}
                  disabled={!selectedBranch}
                  loading={rooms.isLoading}
                  options={(rooms.data ?? []).map((r) => ({
                    value: r.id,
                    label: `${r.name} (tối đa ${r.capacity})`,
                  }))}
                  filterOption={filterByLabel}
                  onChange={onRoomChange}
                  notFoundContent="Chi nhánh chưa có phòng đang hoạt động"
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="trainer_id"
                label="Huấn luyện viên"
                rules={[{ required: true, message: 'Vui lòng chọn HLV' }]}
              >
                <Select
                  showSearch
                  placeholder={selectedBranch ? 'Chọn HLV' : 'Chọn chi nhánh trước'}
                  disabled={!selectedBranch}
                  loading={trainers.isLoading}
                  options={(trainers.data ?? []).map((t) => ({
                    value: t.id,
                    label: t.full_name,
                    desc: t.specialties?.join(', '),
                  }))}
                  optionRender={(o) => (
                    <div>
                      {o.label}
                      {(o.data as { desc?: string }).desc && (
                        <Typography.Text type="secondary" style={{ fontSize: 12, marginLeft: 6 }}>
                          {(o.data as { desc?: string }).desc}
                        </Typography.Text>
                      )}
                    </div>
                  )}
                  filterOption={filterByLabel}
                  notFoundContent="Chi nhánh chưa có HLV"
                />
              </Form.Item>
            </Col>

            {isEdit || mode === 'single' ? (
              <Col xs={12} sm={8}>
                <Form.Item
                  name="date"
                  label="Ngày học"
                  rules={[{ required: true, message: 'Vui lòng chọn ngày' }]}
                >
                  <DatePicker
                    format={(d) => `${weekdayName(d)}, ${d.format(DATE_FORMAT)}`}
                    disabledDate={(d) => d.isBefore(dayjs(), 'day')}
                    allowClear={false}
                    style={{ width: '100%' }}
                  />
                </Form.Item>
              </Col>
            ) : (
              <>
                <Col xs={24} sm={16}>
                  <Form.Item
                    name="range"
                    label="Từ ngày – đến ngày"
                    tooltip="Tối đa 12 tuần"
                    rules={[
                      { required: true, message: 'Vui lòng chọn khoảng ngày' },
                      {
                        validator: (_, v?: [Dayjs, Dayjs]) =>
                          v && v[1].diff(v[0], 'day') > MAX_RECURRING_DAYS
                            ? Promise.reject(new Error('Lịch lặp tối đa 12 tuần'))
                            : Promise.resolve(),
                      },
                    ]}
                  >
                    <DatePicker.RangePicker
                      format={DATE_FORMAT}
                      disabledDate={(d) => d.isBefore(dayjs(), 'day')}
                      style={{ width: '100%' }}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24}>
                  <Form.Item
                    name="weekdays"
                    label="Lặp vào các thứ"
                    rules={[
                      {
                        required: true,
                        type: 'array',
                        min: 1,
                        message: 'Vui lòng chọn thứ trong tuần',
                      },
                    ]}
                  >
                    <Checkbox.Group options={WEEKDAY_OPTIONS} />
                  </Form.Item>
                </Col>
              </>
            )}

            <Col xs={12} sm={8}>
              <Form.Item
                name="time"
                label="Giờ bắt đầu"
                dependencies={['duration_min', 'branch_id']}
                rules={[
                  { required: true, message: 'Vui lòng nhập giờ bắt đầu' },
                  { validator: timeValidator },
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
            <Col xs={12} sm={8}>
              <Form.Item
                name="duration_min"
                label="Thời lượng"
                rules={[
                  { required: isEdit, message: 'Vui lòng nhập thời lượng' },
                  { type: 'number', min: 15, max: 240, message: 'Thời lượng từ 15 đến 240 phút' },
                ]}
              >
                <InputNumber
                  min={15}
                  max={240}
                  step={15}
                  suffix="phút"
                  placeholder="Theo loại lớp"
                  style={{ width: '100%' }}
                />
              </Form.Item>
            </Col>
            <Col xs={12} sm={8}>
              <Form.Item
                name="capacity"
                label="Sức chứa"
                tooltip={
                  selectedRoom ? `Phòng chứa tối đa ${selectedRoom.capacity} người` : undefined
                }
                rules={[
                  { required: isEdit, message: 'Vui lòng nhập sức chứa' },
                  {
                    validator: (_, v?: number) => {
                      if (v === undefined || v === null) return Promise.resolve();
                      if (v < 1) return Promise.reject(new Error('Sức chứa phải lớn hơn 0'));
                      if (selectedRoom && v > selectedRoom.capacity)
                        return Promise.reject(
                          new Error(`Sức chứa vượt quá sức chứa phòng (${selectedRoom.capacity})`),
                        );
                      if (record && v < record.booked_count)
                        return Promise.reject(
                          new Error(
                            `Sức chứa không được nhỏ hơn số chỗ đã đặt (${record.booked_count})`,
                          ),
                        );
                      return Promise.resolve();
                    },
                  },
                ]}
              >
                <InputNumber
                  min={1}
                  max={selectedRoom?.capacity ?? 500}
                  suffix="người"
                  placeholder="Theo loại lớp"
                  style={{ width: '100%' }}
                />
              </Form.Item>
            </Col>

            <Col xs={24}>
              <Form.Item
                name="note"
                label="Ghi chú"
                rules={[{ max: 255, message: 'Ghi chú tối đa 255 ký tự' }]}
              >
                <Input.TextArea rows={2} maxLength={255} showCount placeholder="VD: Mang thảm" />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      <RecurringResultModal result={result} onClose={() => setResult(null)} />
    </>
  );
}

/** Kết quả tạo lịch lặp: số buổi đã tạo + bảng buổi bị bỏ qua (ngày, lý do). */
function RecurringResultModal({
  result,
  onClose,
}: {
  result: ClassScheduleRecurringResult | null;
  onClose: () => void;
}) {
  const skipped = result?.skipped ?? [];
  const created = result?.created ?? 0;
  return (
    <Modal
      open={!!result}
      title="Kết quả tạo lịch lặp"
      onCancel={onClose}
      onOk={onClose}
      okText="Đóng"
      cancelButtonProps={{ style: { display: 'none' } }}
      width={600}
    >
      <Result
        status={created === 0 ? 'warning' : 'success'}
        title={`Đã tạo ${created} buổi${skipped.length ? `, bỏ qua ${skipped.length} buổi` : ''}`}
        subTitle={
          created === 0
            ? 'Không tạo được buổi nào – xem lý do bên dưới.'
            : 'Các buổi đã tạo thuộc cùng một chuỗi lịch lặp.'
        }
        style={{ padding: '16px 0' }}
      />
      {skipped.length > 0 && (
        <Table
          size="small"
          rowKey={(r) => `${r.date}-${r.reason}`}
          pagination={false}
          dataSource={skipped}
          scroll={{ y: 280 }}
          columns={[
            {
              title: 'Ngày',
              dataIndex: 'date',
              width: 180,
              render: (d: string) => `${weekdayName(dayjs(d))}, ${formatDate(d)}`,
            },
            { title: 'Lý do bỏ qua', dataIndex: 'reason' },
          ]}
        />
      )}
    </Modal>
  );
}
