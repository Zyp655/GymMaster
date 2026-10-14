import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Card,
  Checkbox,
  Col,
  DatePicker,
  Divider,
  Drawer,
  Empty,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Spin,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CalendarOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  ExclamationCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
  SaveOutlined,
  ScheduleOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import {
  DOW_LABELS,
  DOW_ORDER,
  trainerSchedulesApi,
} from '@/api/operations-ext';
import type {
  BranchHoliday,
  BranchHolidayInput,
  LeaveRequest,
  LeaveRequestStatus,
  TrainerAvailabilitySlot,
} from '@/api/operations-ext';
import staffApi from '@/api/staff';
import ActionBar from '@/components/ActionBar';
import BranchSelect from '@/components/BranchSelect';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { LEAVE_STATUS_MAP, LEAVE_STATUS_OPTIONS } from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import { DATE_FORMAT, formatDate, formatDateTime, toApiDate } from '@/utils/format';

/* --------------------------------------------------------------------------
 * TAB 1: Lịch rảnh HLV theo tuần (GET/PUT /admin/trainers/:id/availability)
 * -------------------------------------------------------------------------- */
function TrainerAvailabilityTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();

  const [selectedTrainer, setSelectedTrainer] = useState<number | undefined>();
  const [selectedBranch, setSelectedBranch] = useState<number | undefined>(
    branchId ?? branches[0]?.id ?? 1,
  );
  const [slots, setSlots] = useState<TrainerAvailabilitySlot[]>([]);

  const trainersQuery = useQuery({
    queryKey: [staffApi.key, 'options', 'trainers', branchId],
    queryFn: async () => {
      const list = await staffApi.options(branchId);
      const trainers = list.filter((s) => s.position === 'TRAINER');
      return trainers.length > 0 ? trainers : list;
    },
  });

  useEffect(() => {
    if (!selectedTrainer && trainersQuery.data && trainersQuery.data.length > 0) {
      setSelectedTrainer(trainersQuery.data[0].id);
    }
  }, [trainersQuery.data, selectedTrainer]);

  useEffect(() => {
    if (branchId) setSelectedBranch(branchId);
    else if (!selectedBranch && branches.length > 0) setSelectedBranch(branches[0].id);
  }, [branchId, branches, selectedBranch]);

  const availQuery = useQuery({
    queryKey: [trainerSchedulesApi.availabilityKey, selectedTrainer, selectedBranch],
    queryFn: () => trainerSchedulesApi.getAvailability(selectedTrainer!, selectedBranch),
    enabled: !!selectedTrainer,
  });

  useEffect(() => {
    if (availQuery.data) {
      setSlots(availQuery.data);
    }
  }, [availQuery.data]);

  const saveMut = useMutation({
    mutationFn: () =>
      trainerSchedulesApi.updateAvailability(
        selectedTrainer!,
        slots.map((s) => ({
          branch_id: s.branch_id || selectedBranch || 1,
          dow: s.dow,
          start_time: s.start_time,
          end_time: s.end_time,
        })),
        selectedBranch,
      ),
    onSuccess: () => {
      message.success('Đã lưu lịch rảnh theo tuần của HLV!');
      void qc.invalidateQueries({ queryKey: [trainerSchedulesApi.availabilityKey] });
    },
    onError: (err) => showError(err, 'Không thể lưu lịch rảnh HLV'),
  });

  const addSlotForDow = (dow: number) => {
    setSlots((prev) => [
      ...prev,
      {
        branch_id: selectedBranch || 1,
        dow,
        start_time: '08:00',
        end_time: '17:00',
      },
    ]);
  };

  const updateSlotTime = (idx: number, field: 'start_time' | 'end_time', value: string) => {
    setSlots((prev) => prev.map((s, i) => (i === idx ? { ...s, [field]: value } : s)));
  };

  const removeSlot = (idx: number) => {
    setSlots((prev) => prev.filter((_, i) => i !== idx));
  };

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card size="small">
        <Row justify="space-between" align="middle" gutter={[12, 12]}>
          <Col>
            <Space wrap>
              <Select
                showSearch
                optionFilterProp="label"
                placeholder="Chọn Huấn luyện viên (HLV)"
                style={{ width: 260 }}
                value={selectedTrainer}
                onChange={(v) => setSelectedTrainer(v)}
                options={(trainersQuery.data ?? []).map((t) => ({
                  value: t.id,
                  label: `${t.full_name} (${t.staff_code})`,
                }))}
              />
              {!branchId && (
                <Select
                  placeholder="Chi nhánh"
                  style={{ width: 200 }}
                  value={selectedBranch}
                  onChange={(v) => setSelectedBranch(v)}
                  options={branches.map((b) => ({ value: b.id, label: b.name }))}
                />
              )}
            </Space>
          </Col>
          <Col>
            <Space>
              {canManage && selectedTrainer && (
                <Button
                  color="orange"
                  variant="solid"
                  icon={<SaveOutlined />}
                  loading={saveMut.isPending}
                  onClick={() => saveMut.mutate()}
                >
                  Lưu lịch rảnh HLV
                </Button>
              )}
              <Button
                icon={<ReloadOutlined spin={availQuery.isFetching} />}
                onClick={() => availQuery.refetch()}
              >
                Làm mới
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {availQuery.isLoading ? (
        <div style={{ textAlign: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      ) : !selectedTrainer ? (
        <Card>
          <Empty description="Vui lòng chọn Huấn luyện viên để cấu hình lịch rảnh" />
        </Card>
      ) : (
        <Row gutter={[12, 12]}>
          {DOW_ORDER.map((dow) => {
            const daySlots = slots
              .map((s, idx) => ({ ...s, _idx: idx }))
              .filter((s) => s.dow === dow);

            return (
              <Col xs={24} sm={12} md={8} lg={6} key={dow}>
                <Card
                  size="small"
                  title={<Typography.Text strong>{DOW_LABELS[dow]}</Typography.Text>}
                  extra={
                    canManage && (
                      <Button
                        size="small"
                        type="link"
                        icon={<PlusOutlined />}
                        onClick={() => addSlotForDow(dow)}
                      >
                        Thêm ca
                      </Button>
                    )
                  }
                  style={{ height: '100%' }}
                >
                  {daySlots.length === 0 ? (
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      Nghỉ / Không nhận lịch PT
                    </Typography.Text>
                  ) : (
                    <Space direction="vertical" size={8} style={{ width: '100%' }}>
                      {daySlots.map((slot) => (
                        <div
                          key={slot._idx}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 6,
                            background: '#f8fafc',
                            padding: '6px 8px',
                            borderRadius: 6,
                          }}
                        >
                          <Input
                            size="small"
                            value={slot.start_time}
                            disabled={!canManage}
                            onChange={(e) => updateSlotTime(slot._idx, 'start_time', e.target.value)}
                            placeholder="08:00"
                            style={{ width: 70 }}
                          />
                          <span>→</span>
                          <Input
                            size="small"
                            value={slot.end_time}
                            disabled={!canManage}
                            onChange={(e) => updateSlotTime(slot._idx, 'end_time', e.target.value)}
                            placeholder="17:00"
                            style={{ width: 70 }}
                          />
                          {canManage && (
                            <Button
                              size="small"
                              type="text"
                              danger
                              icon={<DeleteOutlined />}
                              onClick={() => removeSlot(slot._idx)}
                            />
                          )}
                        </div>
                      ))}
                    </Space>
                  )}
                </Card>
              </Col>
            );
          })}
        </Row>
      )}
    </Space>
  );
}

/* --------------------------------------------------------------------------
 * TAB 2: Đơn xin nghỉ phép & Lịch bị ảnh hưởng (/admin/leave-requests, /impacts)
 * -------------------------------------------------------------------------- */
function LeaveRequestsTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();

  const [createOpen, setCreateOpen] = useState(false);
  const [inspectLeave, setInspectLeave] = useState<LeaveRequest | null>(null);
  const [subMap, setSubMap] = useState<Record<string, number>>({});
  const [form] = Form.useForm<{
    staff_id: number;
    branch_id?: number;
    range: [Dayjs, Dayjs];
    reason: string;
  }>();

  const staffQuery = useQuery({
    queryKey: [staffApi.key, 'options', 'leave', branchId],
    queryFn: () => staffApi.options(branchId),
  });

  const table = useTableQuery<
    LeaveRequest,
    { branch_id?: number; status?: LeaveRequestStatus; page?: number; limit?: number }
  >({
    queryKey: [trainerSchedulesApi.leaveKey, 'list'],
    queryFn: (p) => trainerSchedulesApi.listLeaveRequests(p),
    filterKeys: ['branch_id', 'status'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const impactsQuery = useQuery({
    queryKey: [trainerSchedulesApi.leaveKey, 'impacts', inspectLeave?.id],
    queryFn: () => trainerSchedulesApi.getLeaveImpacts(inspectLeave!.id),
    enabled: !!inspectLeave,
  });

  const createMut = useMutation({
    mutationFn: (v: {
      staff_id: number;
      branch_id?: number;
      range: [Dayjs, Dayjs];
      reason: string;
    }) =>
      trainerSchedulesApi.createLeaveRequest({
        staff_id: v.staff_id,
        branch_id: v.branch_id ?? branchId ?? null,
        from_at: v.range[0].toISOString(),
        to_at: v.range[1].toISOString(),
        reason: v.reason.trim(),
      }),
    onSuccess: () => {
      message.success('Đã tạo đơn xin nghỉ phép!');
      void qc.invalidateQueries({ queryKey: [trainerSchedulesApi.leaveKey] });
      setCreateOpen(false);
    },
    onError: (err) => showError(err, 'Không thể gửi đơn nghỉ phép'),
  });

  const approveMut = useMutation({
    mutationFn: (id: number) => trainerSchedulesApi.approveLeaveRequest(id),
    onSuccess: () => {
      message.success('Đã phê duyệt đơn xin nghỉ phép!');
      void qc.invalidateQueries({ queryKey: [trainerSchedulesApi.leaveKey] });
      setInspectLeave(null);
    },
    onError: (err) => showError(err, 'Không thể duyệt đơn nghỉ phép'),
  });

  const rejectMut = useMutation({
    mutationFn: (id: number) => trainerSchedulesApi.rejectLeaveRequest(id),
    onSuccess: () => {
      message.success('Đã từ chối đơn xin nghỉ phép');
      void qc.invalidateQueries({ queryKey: [trainerSchedulesApi.leaveKey] });
      setInspectLeave(null);
    },
    onError: (err) => showError(err, 'Không thể từ chối đơn'),
  });

  const substituteClassMut = useMutation({
    mutationFn: ({ scheduleId, trainerId }: { scheduleId: number; trainerId: number }) =>
      trainerSchedulesApi.substituteClass(scheduleId, trainerId),
    onSuccess: () => {
      message.success('Đã phân công HLV dạy thay lớp học!');
      void impactsQuery.refetch();
    },
    onError: (err) => showError(err, 'Không thể gán HLV dạy thay (Kiểm tra trùng lịch)'),
  });

  const reassignPtMut = useMutation({
    mutationFn: ({ sessionId, trainerId }: { sessionId: number; trainerId: number }) =>
      trainerSchedulesApi.reassignPTSession(sessionId, trainerId),
    onSuccess: () => {
      message.success('Đã chuyển giao buổi dạy PT cho HLV thay thế!');
      void impactsQuery.refetch();
    },
    onError: (err) => showError(err, 'Không thể chuyển buổi PT (Kiểm tra trùng lịch)'),
  });

  const columns = useMemo<ColumnsType<LeaveRequest>>(
    () => [
      {
        title: 'Nhân viên / HLV',
        key: 'staff',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>{r.staff_name || `Nhân sự #${r.staff_id}`}</Typography.Text>
            {r.staff_code && (
              <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
                {r.staff_code}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Thời gian xin nghỉ',
        key: 'period',
        render: (_, r) => `${formatDateTime(r.from_at)} → ${formatDateTime(r.to_at)}`,
      },
      {
        title: 'Lý do nghỉ phép',
        dataIndex: 'reason',
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        render: (s: LeaveRequestStatus) => <StatusTag status={s} map={LEAVE_STATUS_MAP} />,
      },
      {
        title: '',
        key: 'actions',
        width: 220,
        render: (_, r) => (
          <Button
            size="small"
            type="primary"
            ghost
            icon={<ExclamationCircleOutlined />}
            onClick={() => setInspectLeave(r)}
          >
            Xem lịch ảnh hưởng & Duyệt
          </Button>
        ),
      },
    ],
    [],
  );

  const trainerOptions = (staffQuery.data ?? []).map((s) => ({
    value: s.id,
    label: `${s.full_name} (${s.staff_code})`,
  }));

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col />
        <Col>
          <ActionBar
            onAdd={() => {
              form.resetFields();
              form.setFieldsValue({
                branch_id: branchId ?? branches[0]?.id,
                range: [dayjs().add(1, 'day').startOf('day'), dayjs().add(1, 'day').endOf('day')],
              });
              setCreateOpen(true);
            }}
            addPerm={PERM.LEAVE_MANAGE}
            addText="Tạo đơn xin nghỉ phép"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên nhân sự, lý do nghỉ..."
        colTextSearch={6}
        filters={[
          ...(branchId
            ? []
            : [
                {
                  name: 'branch_id',
                  placeholder: 'Chi nhánh',
                  options: branches.map((b) => ({ label: b.name, value: b.id })),
                  col: 4,
                },
              ]),
          {
            name: 'status',
            placeholder: 'Trạng thái duyệt',
            options: LEAVE_STATUS_OPTIONS,
            col: 5,
          },
        ]}
      />

      <DataTable<LeaveRequest>
        table={table}
        columns={columns}
        emptyText="Chưa có đơn xin nghỉ phép nào"
      />

      <Modal
        open={createOpen}
        title="Tạo đơn xin nghỉ phép (Nhân sự / HLV)"
        okText="Gửi đơn nghỉ phép"
        cancelText="Hủy"
        onOk={() => form.submit()}
        onCancel={() => setCreateOpen(false)}
        confirmLoading={createMut.isPending}
        width={520}
        forceRender
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(v) => createMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Form.Item
            name="staff_id"
            label="Nhân viên / HLV xin nghỉ"
            rules={[{ required: true, message: 'Vui lòng chọn nhân sự' }]}
          >
            <Select showSearch optionFilterProp="label" options={trainerOptions} />
          </Form.Item>
          <Form.Item
            name="range"
            label="Khoảng thời gian nghỉ"
            rules={[{ required: true, message: 'Chọn thời gian nghỉ' }]}
          >
            <DatePicker.RangePicker showTime style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item
            name="reason"
            label="Lý do xin nghỉ"
            rules={[{ required: true, whitespace: true, message: 'Nhập lý do nghỉ' }]}
          >
            <Input.TextArea rows={3} placeholder="VD: Nghỉ phép cá nhân, đi công tác..." />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        open={!!inspectLeave}
        onClose={() => setInspectLeave(null)}
        width={680}
        title={
          inspectLeave ? (
            <Space>
              <span>Đơn nghỉ phép #{inspectLeave.id}</span>
              <StatusTag status={inspectLeave.status} map={LEAVE_STATUS_MAP} />
            </Space>
          ) : (
            'Chi tiết đơn nghỉ phép'
          )
        }
        extra={
          inspectLeave && canManage && inspectLeave.status === 'PENDING' ? (
            <Space>
              <ConfirmButton
                perm={PERM.LEAVE_MANAGE}
                type="primary"
                icon={<CheckCircleOutlined />}
                confirmTitle="Xác nhận phê duyệt đơn xin nghỉ phép này?"
                onConfirm={() => approveMut.mutateAsync(inspectLeave.id)}
              >
                Duyệt đơn
              </ConfirmButton>
              <ConfirmButton
                perm={PERM.LEAVE_MANAGE}
                danger
                icon={<CloseCircleOutlined />}
                confirmTitle="Từ chối đơn xin nghỉ phép này?"
                onConfirm={() => rejectMut.mutateAsync(inspectLeave.id)}
              >
                Từ chối
              </ConfirmButton>
            </Space>
          ) : null
        }
      >
        {inspectLeave && (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Alert
              type="warning"
              showIcon
              message={`Thời gian nghỉ: ${formatDateTime(inspectLeave.from_at)} → ${formatDateTime(inspectLeave.to_at)}`}
              description={`Lý do: ${inspectLeave.reason || 'Không ghi chú'}`}
            />

            <Divider orientation="left" style={{ margin: '4px 0' }}>
              Các ca Lớp học nhóm bị ảnh hưởng ({impactsQuery.data?.classes.length ?? 0})
            </Divider>

            <Table
              size="small"
              rowKey="id"
              loading={impactsQuery.isLoading}
              pagination={false}
              dataSource={impactsQuery.data?.classes ?? []}
              locale={{ emptyText: 'Không có lịch lớp học nào bị trùng' }}
              columns={[
                {
                  title: 'Lớp học',
                  key: 'name',
                  render: (_, r) => r.class_type_name || r.title || `Lớp #${r.id}`,
                },
                {
                  title: 'Thời gian',
                  dataIndex: 'start_at',
                  render: (v: string) => formatDateTime(v),
                },
                {
                  title: 'Phân công HLV dạy thay',
                  key: 'sub',
                  render: (_, r) => (
                    <Space>
                      <Select
                        size="small"
                        placeholder="Chọn HLV thay"
                        style={{ width: 170 }}
                        value={subMap[`class_${r.id}`] ?? r.substitute_trainer_id ?? undefined}
                        onChange={(val) =>
                          setSubMap((prev) => ({ ...prev, [`class_${r.id}`]: val }))
                        }
                        options={trainerOptions.filter((o) => o.value !== inspectLeave.staff_id)}
                      />
                      {canManage && (
                        <Button
                          size="small"
                          type="primary"
                          icon={<SwapOutlined />}
                          disabled={!subMap[`class_${r.id}`]}
                          loading={substituteClassMut.isPending}
                          onClick={() =>
                            substituteClassMut.mutate({
                              scheduleId: r.id,
                              trainerId: subMap[`class_${r.id}`],
                            })
                          }
                        >
                          Gán thay
                        </Button>
                      )}
                    </Space>
                  ),
                },
              ]}
            />

            <Divider orientation="left" style={{ margin: '4px 0' }}>
              Các buổi dạy PT 1-1 bị ảnh hưởng ({impactsQuery.data?.pt_sessions.length ?? 0})
            </Divider>

            <Table
              size="small"
              rowKey="id"
              loading={impactsQuery.isLoading}
              pagination={false}
              dataSource={impactsQuery.data?.pt_sessions ?? []}
              locale={{ emptyText: 'Không có buổi PT nào bị trùng' }}
              columns={[
                {
                  title: 'Hội viên',
                  key: 'member',
                  render: (_, r) => r.member_name || `Hội viên #${r.member_id ?? r.id}`,
                },
                {
                  title: 'Thời gian',
                  dataIndex: 'start_at',
                  render: (v: string) => formatDateTime(v),
                },
                {
                  title: 'Bàn giao HLV thay thế',
                  key: 'reassign',
                  render: (_, r) => (
                    <Space>
                      <Select
                        size="small"
                        placeholder="Chọn HLV nhận"
                        style={{ width: 170 }}
                        value={subMap[`pt_${r.id}`]}
                        onChange={(val) =>
                          setSubMap((prev) => ({ ...prev, [`pt_${r.id}`]: val }))
                        }
                        options={trainerOptions.filter((o) => o.value !== inspectLeave.staff_id)}
                      />
                      {canManage && (
                        <Button
                          size="small"
                          type="primary"
                          icon={<SwapOutlined />}
                          disabled={!subMap[`pt_${r.id}`]}
                          loading={reassignPtMut.isPending}
                          onClick={() =>
                            reassignPtMut.mutate({
                              sessionId: r.id,
                              trainerId: subMap[`pt_${r.id}`],
                            })
                          }
                        >
                          Bàn giao
                        </Button>
                      )}
                    </Space>
                  ),
                },
              ]}
            />
          </Space>
        )}
      </Drawer>
    </>
  );
}

/* --------------------------------------------------------------------------
 * TAB 3: Lịch nghỉ lễ phòng tập (/admin/branch-holidays)
 * -------------------------------------------------------------------------- */
function BranchHolidaysTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();

  const [modal, setModal] = useState<{ open: boolean; record: BranchHoliday | null }>({
    open: false,
    record: null,
  });
  const [form] = Form.useForm<{
    branch_id?: number | null;
    date: Dayjs;
    name: string;
    closed: boolean;
    open_time?: string;
    close_time?: string;
    extend_subs: boolean;
  }>();

  const isClosed = Form.useWatch('closed', form) ?? true;

  const table = useTableQuery<BranchHoliday, { branch_id?: number; page?: number; limit?: number }>({
    queryKey: [trainerSchedulesApi.holidayKey, 'list'],
    queryFn: (p) => trainerSchedulesApi.listHolidays(p),
    filterKeys: ['branch_id'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'date', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  useEffect(() => {
    if (!modal.open) return;
    form.resetFields();
    if (modal.record) {
      form.setFieldsValue({
        branch_id: modal.record.branch_id ?? undefined,
        date: dayjs(modal.record.date),
        name: modal.record.name,
        closed: modal.record.closed,
        open_time: modal.record.open_time ?? '08:00',
        close_time: modal.record.close_time ?? '17:00',
        extend_subs: modal.record.extend_subs,
      });
    } else {
      form.setFieldsValue({
        branch_id: branchId ?? undefined,
        date: dayjs().add(7, 'day'),
        closed: true,
        open_time: '08:00',
        close_time: '17:00',
        extend_subs: true,
      });
    }
  }, [modal, form, branchId]);

  const saveMut = useMutation({
    mutationFn: (v: {
      branch_id?: number | null;
      date: Dayjs;
      name: string;
      closed: boolean;
      open_time?: string;
      close_time?: string;
      extend_subs: boolean;
    }) => {
      const payload: BranchHolidayInput = {
        branch_id: v.branch_id ?? null,
        date: toApiDate(v.date) ?? dayjs().format('YYYY-MM-DD'),
        name: v.name.trim(),
        closed: !!v.closed,
        open_time: v.closed ? null : v.open_time || '08:00',
        close_time: v.closed ? null : v.close_time || '17:00',
        extend_subs: !!v.extend_subs,
      };
      return modal.record
        ? trainerSchedulesApi.updateHoliday(modal.record.id, payload)
        : trainerSchedulesApi.createHoliday(payload);
    },
    onSuccess: (_, v) => {
      message.success(
        modal.record
          ? 'Đã cập nhật lịch nghỉ lễ'
          : `Đã thêm ngày nghỉ lễ${v.extend_subs ? ' và kích hoạt cộng bù ngày cho gói tập' : ''}!`,
      );
      void qc.invalidateQueries({ queryKey: [trainerSchedulesApi.holidayKey] });
      setModal({ open: false, record: null });
    },
    onError: (err) => showError(err, 'Không thể lưu lịch nghỉ lễ'),
  });

  const deleteMut = useMutation({
    mutationFn: (id: number) => trainerSchedulesApi.deleteHoliday(id),
    onSuccess: () => {
      message.success('Đã xóa ngày nghỉ lễ');
      void qc.invalidateQueries({ queryKey: [trainerSchedulesApi.holidayKey] });
    },
    onError: (err) => showError(err, 'Không thể xóa ngày nghỉ lễ'),
  });

  const columns = useMemo<ColumnsType<BranchHoliday>>(
    () => [
      {
        title: 'Ngày lễ',
        dataIndex: 'date',
        width: 130,
        render: (v: string) => <Typography.Text strong>{formatDate(v)}</Typography.Text>,
      },
      {
        title: 'Tên ngày nghỉ lễ / Sự kiện',
        dataIndex: 'name',
      },
      {
        title: 'Phạm vi áp dụng',
        dataIndex: 'branch_name',
        render: (v: string | undefined, r) =>
          r.branch_id ? v || `CN #${r.branch_id}` : <Tag color="blue">Toàn hệ thống</Tag>,
      },
      {
        title: 'Chế độ mở cửa',
        key: 'open_mode',
        render: (_, r) =>
          r.closed ? (
            <Tag color="red">Đóng cửa cả ngày</Tag>
          ) : (
            <Tag color="gold">
              Giờ đặc biệt: {r.open_time || '08:00'} – {r.close_time || '17:00'}
            </Tag>
          ),
      },
      {
        title: 'Cộng bù ngày gói tập',
        dataIndex: 'extend_subs',
        render: (ext: boolean) =>
          ext ? <Tag color="green">Tự động cộng +1 ngày</Tag> : <Tag>Không cộng bù</Tag>,
      },
      {
        title: '',
        key: 'actions',
        width: 100,
        render: (_, r) =>
          canManage ? (
            <Space size={4}>
              <Button
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => setModal({ open: true, record: r })}
              />
              <ConfirmButton
                perm={[PERM.HOLIDAY_MANAGE, PERM.LEAVE_MANAGE]}
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
                confirmTitle={`Xóa lịch nghỉ lễ "${r.name}"?`}
                onConfirm={() => deleteMut.mutateAsync(r.id)}
                aria-label="Xóa"
              />
            </Space>
          ) : null,
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage],
  );

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col />
        <Col>
          <ActionBar
            onAdd={() => setModal({ open: true, record: null })}
            addPerm={[PERM.HOLIDAY_MANAGE, PERM.LEAVE_MANAGE]}
            addText="Thêm ngày nghỉ lễ"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên ngày lễ..."
        colTextSearch={6}
        filters={
          branchId
            ? []
            : [
                {
                  name: 'branch_id',
                  placeholder: 'Chi nhánh',
                  options: branches.map((b) => ({ label: b.name, value: b.id })),
                  col: 5,
                },
              ]
        }
      />

      <DataTable<BranchHoliday>
        table={table}
        columns={columns}
        emptyText="Chưa có lịch nghỉ lễ nào được cấu hình"
      />

      <Modal
        open={modal.open}
        title={modal.record ? 'Cập nhật ngày nghỉ lễ' : 'Thiết lập lịch nghỉ lễ phòng tập'}
        okText="Lưu lịch nghỉ lễ"
        cancelText="Hủy"
        onOk={() => form.submit()}
        onCancel={() => setModal({ open: false, record: null })}
        confirmLoading={saveMut.isPending}
        width={520}
        forceRender
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(v) => saveMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Row gutter={12}>
            <Col xs={12}>
              <Form.Item
                name="date"
                label="Ngày nghỉ lễ"
                rules={[{ required: true, message: 'Chọn ngày' }]}
              >
                <DatePicker style={{ width: '100%' }} format={DATE_FORMAT} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="branch_id" label="Chi nhánh (Trống = Toàn hệ thống)">
                <BranchSelect activeOnly allowClear />
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item
                name="name"
                label="Tên dịp nghỉ lễ"
                rules={[{ required: true, whitespace: true, message: 'Nhập tên dịp lễ' }]}
              >
                <Input placeholder="VD: Nghỉ Tết Nguyên Đán / Bảo trì định kỳ toàn cơ sở" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="closed" valuePropName="checked" style={{ marginBottom: 8 }}>
            <Checkbox>Đóng cửa toàn bộ thời gian trong ngày</Checkbox>
          </Form.Item>

          {!isClosed && (
            <Row gutter={12}>
              <Col xs={12}>
                <Form.Item name="open_time" label="Giờ mở cửa đặc biệt">
                  <Input placeholder="08:00" />
                </Form.Item>
              </Col>
              <Col xs={12}>
                <Form.Item name="close_time" label="Giờ đóng cửa đặc biệt">
                  <Input placeholder="17:00" />
                </Form.Item>
              </Col>
            </Row>
          )}

          <Form.Item name="extend_subs" valuePropName="checked">
            <Checkbox>
              Tự động cộng bù +1 ngày hạn sử dụng cho toàn bộ gói tập theo thời gian (TIME) đang ACTIVE
            </Checkbox>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

export default function TrainerSchedulesPage() {
  const { can } = usePermission();
  const canLeave = can([PERM.LEAVE_MANAGE, PERM.HOLIDAY_MANAGE]);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <PageTitle
        title="Lịch rảnh HLV, Nghỉ phép & Ngày lễ"
        subtitle="Thiết lập khung giờ rảnh hàng tuần của HLV, duyệt đơn nghỉ phép kèm phân công dạy thay và quản lý lịch nghỉ lễ phòng tập"
      />

      <Tabs
        defaultActiveKey="availability"
        items={[
          {
            key: 'availability',
            label: (
              <Space size={6}>
                <ScheduleOutlined />
                <span>Lịch rảnh HLV theo tuần</span>
              </Space>
            ),
            children: <TrainerAvailabilityTab canManage={canLeave} />,
          },
          {
            key: 'leave-requests',
            label: (
              <Space size={6}>
                <SwapOutlined />
                <span>Đơn xin nghỉ phép & Lịch bị ảnh hưởng</span>
              </Space>
            ),
            children: <LeaveRequestsTab canManage={canLeave} />,
          },
          {
            key: 'holidays',
            label: (
              <Space size={6}>
                <CalendarOutlined />
                <span>Lịch nghỉ lễ phòng tập</span>
              </Space>
            ),
            children: <BranchHolidaysTab canManage={canLeave} />,
          },
        ]}
      />
    </Space>
  );
}
