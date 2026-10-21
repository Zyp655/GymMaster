import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Divider,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Spin,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  BookOutlined,
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  MinusCircleOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { Link } from 'react-router-dom';
import staffApi from '@/api/staff';
import workoutApi, { MUSCLE_GROUP_LABELS } from '@/api/workout';
import type {
  WorkoutPlan,
  WorkoutPlanCloneInput,
  WorkoutPlanDay,
  WorkoutPlanInput,
  WorkoutPlanListParams,
  WorkoutPlanStatus,
} from '@/api/workout';
import ActionBar from '@/components/ActionBar';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MemberSelect from '@/components/MemberSelect';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { WORKOUT_PLAN_STATUS_MAP, WORKOUT_PLAN_STATUS_OPTIONS } from '@/constants/status';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useAppSelector } from '@/store';
import { showError } from '@/utils/errors';
import { DATE_FORMAT, formatDate, toApiDate } from '@/utils/format';

interface PlanFormValues {
  member_id?: number | null;
  trainer_id?: number;
  name: string;
  goal?: string;
  date_range?: [Dayjs | null, Dayjs | null];
  status: WorkoutPlanStatus;
  days: WorkoutPlanDay[];
}

interface PlanEditorModalProps {
  open: boolean;
  planId: number | null;
  initialRecord: WorkoutPlan | null;
  readOnly: boolean;
  onClose: () => void;
}

function PlanEditorModal({ open, planId, initialRecord, readOnly, onClose }: PlanEditorModalProps) {
  const [form] = Form.useForm<PlanFormValues>();
  const { message } = App.useApp();
  const qc = useQueryClient();
  const currentUser = useAppSelector((s) => s.auth.user);
  const isEdit = !!planId;

  const detailQuery = useQuery({
    queryKey: [workoutApi.planKey, 'detail', planId],
    queryFn: () => workoutApi.getPlan(planId!),
    enabled: open && !!planId,
  });

  const exercisesQuery = useQuery({
    queryKey: [workoutApi.exerciseKey, 'all'],
    queryFn: () => workoutApi.listExercises({ limit: 100 }),
    enabled: open,
    staleTime: 60_000,
  });

  const trainersQuery = useQuery({
    queryKey: [staffApi.key, 'trainers', 'workout'],
    queryFn: () => staffApi.trainerOptions(),
    enabled: open,
    staleTime: 60_000,
  });

  const exerciseOptions = useMemo(
    () =>
      (exercisesQuery.data?.items ?? []).map((e) => ({
        value: e.id,
        label: `${e.name} (${MUSCLE_GROUP_LABELS[e.muscle_group] ?? e.muscle_group})`,
      })),
    [exercisesQuery.data],
  );

  const trainerOptions = useMemo(
    () => (trainersQuery.data ?? []).map((t) => ({ value: t.id, label: t.full_name })),
    [trainersQuery.data],
  );

  useEffect(() => {
    if (!open) return;
    const src = detailQuery.data ?? initialRecord;
    if (src) {
      form.setFieldsValue({
        member_id: src.member_id ?? undefined,
        trainer_id: src.trainer_id || currentUser?.id,
        name: src.name,
        goal: src.goal,
        status: src.status || 'ACTIVE',
        date_range:
          src.start_date || src.end_date
            ? [
                src.start_date ? dayjs(src.start_date) : null,
                src.end_date ? dayjs(src.end_date) : null,
              ]
            : undefined,
        days:
          src.days && src.days.length > 0
            ? src.days
            : [
                {
                  day_index: 1,
                  title: 'Buổi 1 – Thân trên',
                  items: [
                    {
                      exercise_id: exerciseOptions[0]?.value ?? 1,
                      sets: 3,
                      reps: 10,
                      weight: 20,
                      rest_sec: 60,
                      note: '',
                    },
                  ],
                },
              ],
      });
    } else {
      form.resetFields();
      form.setFieldsValue({
        trainer_id: currentUser?.id,
        status: 'ACTIVE',
        days: [
          {
            day_index: 1,
            title: 'Buổi 1 – Ngực & Tay sau',
            items: [],
          },
        ],
      });
    }
  }, [open, detailQuery.data, initialRecord, form, currentUser?.id, exerciseOptions]);

  const save = useMutation({
    mutationFn: (values: PlanFormValues) => {
      const payload: WorkoutPlanInput = {
        member_id: values.member_id ?? null,
        trainer_id: values.trainer_id ?? currentUser?.id,
        name: values.name.trim(),
        goal: values.goal?.trim() || '',
        start_date: toApiDate(values.date_range?.[0]) ?? null,
        end_date: toApiDate(values.date_range?.[1]) ?? null,
        status: values.status,
        days: (values.days ?? []).map((d, dIdx) => ({
          day_index: d.day_index || dIdx + 1,
          title: d.title?.trim() || `Buổi ${dIdx + 1}`,
          items: (d.items ?? []).map((it, iIdx) => ({
            exercise_id: it.exercise_id,
            sets: Number(it.sets) || 3,
            reps: Number(it.reps) || 10,
            weight: Number(it.weight) || 0,
            rest_sec: Number(it.rest_sec) || 60,
            note: it.note?.trim() || '',
            sort: iIdx + 1,
          })),
        })),
      };
      return planId ? workoutApi.updatePlan(planId, payload) : workoutApi.createPlan(payload);
    },
    onSuccess: () => {
      message.success(isEdit ? 'Đã cập nhật giáo án' : 'Đã tạo giáo án mới');
      void qc.invalidateQueries({ queryKey: [workoutApi.planKey] });
      onClose();
    },
    onError: (err) => showError(err, 'Không thể lưu giáo án tập luyện'),
  });

  const srcRecord = detailQuery.data ?? initialRecord;

  return (
    <Modal
      open={open}
      title={isEdit ? `Giáo án: ${srcRecord?.name ?? ''}` : 'Soạn giáo án tập luyện'}
      okText={isEdit ? 'Lưu giáo án' : 'Tạo giáo án'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={940}
      forceRender
    >
      {isEdit && detailQuery.isLoading ? (
        <div style={{ textAlign: 'center', padding: 36 }}>
          <Spin />
        </div>
      ) : (
        <Form<PlanFormValues>
          form={form}
          layout="vertical"
          disabled={readOnly}
          onFinish={(v) => save.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Row gutter={12}>
            <Col xs={24} sm={10}>
              <Form.Item
                name="name"
                label="Tên giáo án"
                rules={[{ required: true, whitespace: true, message: 'Vui lòng nhập tên giáo án' }]}
              >
                <Input placeholder="VD: Giáo án tăng cơ giảm mỡ 4 buổi/tuần" maxLength={150} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}>
              <Form.Item name="member_id" label="Hội viên (để trống = Giáo án mẫu)">
                <MemberSelect
                  placeholder="Chọn hội viên hoặc để trống làm mẫu"
                  initialMember={
                    srcRecord?.member_id
                      ? {
                          id: srcRecord.member_id,
                          full_name: srcRecord.member_name || `HV #${srcRecord.member_id}`,
                          member_code: srcRecord.member_code || '',
                          phone: '',
                          avatar_url: '',
                          status: 'ACTIVE',
                        }
                      : null
                  }
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={6}>
              <Form.Item name="trainer_id" label="Huấn luyện viên">
                <Select allowClear options={trainerOptions} placeholder="Chọn HLV phụ trách" />
              </Form.Item>
            </Col>
            <Col xs={24} sm={10}>
              <Form.Item name="goal" label="Mục tiêu tập luyện">
                <Input
                  placeholder="VD: Tăng cơ ngực/vai, cải thiện sức bền tim mạch"
                  maxLength={255}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}>
              <Form.Item name="date_range" label="Thời gian áp dụng">
                <DatePicker.RangePicker
                  style={{ width: '100%' }}
                  format={DATE_FORMAT}
                  placeholder={['Từ ngày', 'Đến ngày']}
                  allowEmpty={[true, true]}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={6}>
              <Form.Item name="status" label="Trạng thái" rules={[{ required: true }]}>
                <Select options={WORKOUT_PLAN_STATUS_OPTIONS} />
              </Form.Item>
            </Col>
          </Row>

          <Divider orientation="left" style={{ margin: '8px 0 16px' }}>
            Các buổi tập theo ngày
          </Divider>

          <Form.List name="days">
            {(dayFields, { add: addDay, remove: removeDay }) => (
              <Space direction="vertical" size={14} style={{ width: '100%' }}>
                {dayFields.map((dayField, dayIdx) => (
                  <Card
                    key={dayField.key}
                    size="small"
                    style={{ background: 'var(--gm-hover)', borderColor: 'var(--gm-border)' }}
                    title={
                      <Row gutter={8} align="middle">
                        <Col flex="130px">
                          <Form.Item
                            name={[dayField.name, 'day_index']}
                            style={{ marginBottom: 0 }}
                            rules={[{ required: true, message: 'Chọn ngày' }]}
                          >
                            <Select
                              size="small"
                              options={[1, 2, 3, 4, 5, 6, 7].map((d) => ({
                                value: d,
                                label: `Ngày / Buổi ${d}`,
                              }))}
                            />
                          </Form.Item>
                        </Col>
                        <Col flex="auto">
                          <Form.Item
                            name={[dayField.name, 'title']}
                            style={{ marginBottom: 0 }}
                            rules={[{ required: true, message: 'Nhập tiêu đề buổi tập' }]}
                          >
                            <Input size="small" placeholder="Tiêu đề buổi (VD: Ngực & Tay sau)" />
                          </Form.Item>
                        </Col>
                      </Row>
                    }
                    extra={
                      !readOnly && dayFields.length > 1 ? (
                        <Button
                          type="text"
                          danger
                          size="small"
                          icon={<DeleteOutlined />}
                          onClick={() => removeDay(dayField.name)}
                        >
                          Xóa buổi
                        </Button>
                      ) : null
                    }
                  >
                    <Form.List name={[dayField.name, 'items']}>
                      {(itemFields, { add: addItem, remove: removeItem }) => (
                        <Space direction="vertical" size={8} style={{ width: '100%' }}>
                          {itemFields.map((itemField) => (
                            <Row key={itemField.key} gutter={8} align="middle">
                              <Col xs={24} md={8}>
                                <Form.Item
                                  name={[itemField.name, 'exercise_id']}
                                  style={{ marginBottom: 0 }}
                                  rules={[{ required: true, message: 'Chọn bài tập' }]}
                                >
                                  <Select
                                    showSearch
                                    optionFilterProp="label"
                                    placeholder="Chọn bài tập từ thư viện"
                                    options={exerciseOptions}
                                  />
                                </Form.Item>
                              </Col>
                              <Col xs={6} md={3}>
                                <Form.Item
                                  name={[itemField.name, 'sets']}
                                  style={{ marginBottom: 0 }}
                                >
                                  <InputNumber
                                    min={1}
                                    max={20}
                                    prefix="Set"
                                    style={{ width: '100%' }}
                                  />
                                </Form.Item>
                              </Col>
                              <Col xs={6} md={3}>
                                <Form.Item
                                  name={[itemField.name, 'reps']}
                                  style={{ marginBottom: 0 }}
                                >
                                  <InputNumber
                                    min={1}
                                    max={200}
                                    prefix="Rep"
                                    style={{ width: '100%' }}
                                  />
                                </Form.Item>
                              </Col>
                              <Col xs={6} md={3}>
                                <Form.Item
                                  name={[itemField.name, 'weight']}
                                  style={{ marginBottom: 0 }}
                                >
                                  <InputNumber
                                    min={0}
                                    max={500}
                                    step={2.5}
                                    suffix="kg"
                                    style={{ width: '100%' }}
                                  />
                                </Form.Item>
                              </Col>
                              <Col xs={6} md={3}>
                                <Form.Item
                                  name={[itemField.name, 'rest_sec']}
                                  style={{ marginBottom: 0 }}
                                >
                                  <InputNumber
                                    min={0}
                                    max={600}
                                    step={15}
                                    suffix="s"
                                    style={{ width: '100%' }}
                                  />
                                </Form.Item>
                              </Col>
                              <Col xs={21} md={3}>
                                <Form.Item
                                  name={[itemField.name, 'note']}
                                  style={{ marginBottom: 0 }}
                                >
                                  <Input placeholder="Ghi chú" />
                                </Form.Item>
                              </Col>
                              <Col xs={3} md={1} style={{ textAlign: 'right' }}>
                                {!readOnly && (
                                  <Button
                                    type="text"
                                    danger
                                    icon={<MinusCircleOutlined />}
                                    onClick={() => removeItem(itemField.name)}
                                  />
                                )}
                              </Col>
                            </Row>
                          ))}
                          {!readOnly && (
                            <Button
                              type="dashed"
                              size="small"
                              icon={<PlusOutlined />}
                              onClick={() =>
                                addItem({
                                  exercise_id: exerciseOptions[0]?.value,
                                  sets: 3,
                                  reps: 10,
                                  weight: 0,
                                  rest_sec: 60,
                                  note: '',
                                })
                              }
                            >
                              Thêm bài tập vào buổi {dayIdx + 1}
                            </Button>
                          )}
                        </Space>
                      )}
                    </Form.List>
                  </Card>
                ))}
                {!readOnly && (
                  <Button
                    type="dashed"
                    block
                    icon={<PlusOutlined />}
                    onClick={() =>
                      addDay({
                        day_index: Math.min(7, dayFields.length + 1),
                        title: `Buổi ${dayFields.length + 1}`,
                        items: [],
                      })
                    }
                  >
                    Thêm buổi tập theo ngày
                  </Button>
                )}
              </Space>
            )}
          </Form.List>
        </Form>
      )}
    </Modal>
  );
}

interface CloneModalProps {
  open: boolean;
  plan: WorkoutPlan | null;
  onClose: () => void;
}

function ClonePlanModal({ open, plan, onClose }: CloneModalProps) {
  const [form] = Form.useForm<{
    member_id: number;
    name?: string;
    date_range?: [Dayjs | null, Dayjs | null] | null;
  }>();
  const { message } = App.useApp();
  const qc = useQueryClient();

  useEffect(() => {
    if (!open || !plan) return;
    form.resetFields();
    form.setFieldsValue({
      name: `${plan.name}`,
      date_range: [dayjs(), dayjs().add(30, 'day')],
    });
  }, [open, plan, form]);

  const cloneMut = useMutation({
    mutationFn: (v: {
      member_id: number;
      name?: string;
      date_range?: [Dayjs | null, Dayjs | null] | null;
    }) => {
      const payload: WorkoutPlanCloneInput = {
        member_id: v.member_id,
        name: v.name?.trim() || plan?.name,
        start_date: toApiDate(v.date_range?.[0]) ?? null,
        end_date: toApiDate(v.date_range?.[1]) ?? null,
      };
      return workoutApi.clonePlan(plan!.id, payload);
    },
    onSuccess: () => {
      message.success('Đã nhân bản giáo án cho hội viên');
      void qc.invalidateQueries({ queryKey: [workoutApi.planKey] });
      onClose();
    },
    onError: (err) => showError(err, 'Không thể nhân bản giáo án'),
  });

  return (
    <Modal
      open={open}
      title={`Nhân bản giáo án "${plan?.name ?? ''}"`}
      okText="Nhân bản giáo án"
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={cloneMut.isPending}
      width={520}
      forceRender
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={(v) => cloneMut.mutate(v)}
        style={{ marginTop: 12 }}
      >
        <Form.Item
          name="member_id"
          label="Hội viên nhận giáo án"
          rules={[{ required: true, message: 'Vui lòng chọn hội viên nhận giáo án' }]}
        >
          <MemberSelect activeOnly />
        </Form.Item>
        <Form.Item name="name" label="Tên giáo án mới">
          <Input placeholder="Nhập tên giáo án cho hội viên" maxLength={150} />
        </Form.Item>
        <Form.Item name="date_range" label="Thời gian áp dụng">
          <DatePicker.RangePicker
            style={{ width: '100%' }}
            format={DATE_FORMAT}
            placeholder={['Từ ngày', 'Đến ngày']}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}

/** Trang Giáo án tập luyện (`/app/workout-plans`). */
export default function WorkoutPlansPage() {
  const { message } = App.useApp();
  const { can } = usePermission();
  const qc = useQueryClient();
  const canManage = can(PERM.WORKOUT_MANAGE);

  const [editor, setEditor] = useState<{ open: boolean; record: WorkoutPlan | null }>({
    open: false,
    record: null,
  });
  const [cloneTarget, setCloneTarget] = useState<WorkoutPlan | null>(null);

  const trainersQuery = useQuery({
    queryKey: [staffApi.key, 'trainers', 'filter'],
    queryFn: () => staffApi.trainerOptions(),
    staleTime: 5 * 60_000,
  });
  const trainerOptions = (trainersQuery.data ?? []).map((t) => ({
    value: t.id,
    label: t.full_name,
  }));

  const table = useTableQuery<WorkoutPlan, WorkoutPlanListParams>({
    queryKey: [workoutApi.planKey, 'list'],
    queryFn: (p) => workoutApi.listPlans(p),
    filterKeys: ['member_id', 'trainer_id', 'status'],
    numberKeys: ['member_id', 'trainer_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
  });

  const remove = useMutation({
    mutationFn: (id: number) => workoutApi.deletePlan(id),
    onSuccess: () => {
      message.success('Đã xóa giáo án');
      void qc.invalidateQueries({ queryKey: [workoutApi.planKey] });
    },
    onError: (err) => showError(err, 'Không thể xóa giáo án'),
  });

  const columns = useMemo<ColumnsType<WorkoutPlan>>(
    () => [
      {
        title: 'Tên giáo án',
        dataIndex: 'name',
        render: (name: string, r) => (
          <Space direction="vertical" size={2}>
            <Space size={6}>
              <Button
                type="link"
                size="small"
                style={{ padding: 0, fontWeight: 600 }}
                onClick={() => setEditor({ open: true, record: r })}
              >
                {name}
              </Button>
              {r.member_id == null && <Tag color="purple">Giáo án mẫu</Tag>}
            </Space>
            {r.goal && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                Mục tiêu: {r.goal}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Hội viên',
        key: 'member',
        render: (_, r) =>
          r.member_id ? (
            <Space direction="vertical" size={0}>
              <Link to={`/app/members/${r.member_id}`}>
                {r.member_name || `Hội viên #${r.member_id}`}
              </Link>
              {r.member_code && (
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {r.member_code}
                </Typography.Text>
              )}
            </Space>
          ) : (
            <Tag color="purple">Giáo án mẫu</Tag>
          ),
      },
      {
        title: 'HLV phụ trách',
        dataIndex: 'trainer_name',
        render: (v: string | undefined, r) => v || `HLV #${r.trainer_id}`,
      },
      {
        title: 'Thời gian áp dụng',
        key: 'dates',
        className: 'text-nowrap',
        render: (_, r) =>
          r.start_date || r.end_date
            ? `${formatDate(r.start_date)} → ${formatDate(r.end_date)}`
            : '—',
      },
      {
        title: 'Số buổi',
        key: 'days',
        align: 'center',
        render: (_, r) => `${r.day_count ?? r.days?.length ?? 0} buổi`,
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        render: (s: string) => <StatusTag status={s} map={WORKOUT_PLAN_STATUS_MAP} />,
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 120,
        render: (_, r) => (
          <Space size={0}>
            {canManage && (
              <>
                <Tooltip title="Nhân bản (Clone) cho hội viên">
                  <Button
                    type="text"
                    size="small"
                    icon={<CopyOutlined />}
                    onClick={() => setCloneTarget(r)}
                  />
                </Tooltip>
                <Tooltip title="Soạn / Sửa giáo án">
                  <Button
                    type="text"
                    size="small"
                    icon={<EditOutlined />}
                    onClick={() => setEditor({ open: true, record: r })}
                  />
                </Tooltip>
              </>
            )}
            <ConfirmButton
              perm={PERM.WORKOUT_MANAGE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa giáo án "${r.name}"?`}
              onConfirm={() => remove.mutateAsync(r.id)}
              aria-label="Xóa"
            />
          </Space>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage],
  );

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle
            title="Giáo án tập luyện"
            subtitle="Thiết kế lộ trình tập luyện theo buổi cho hội viên hoặc tạo giáo án mẫu để nhân bản"
          />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => setEditor({ open: true, record: null })}
            addPerm={PERM.WORKOUT_MANAGE}
            addText="Soạn giáo án"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          >
            <Link to="/app/exercises">
              <Button icon={<BookOutlined />}>Thư viện bài tập</Button>
            </Link>
          </ActionBar>
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên giáo án, mục tiêu..."
        colTextSearch={5}
        filters={[
          {
            name: 'trainer_id',
            placeholder: 'Huấn luyện viên',
            options: trainerOptions,
            col: 4,
          },
          {
            name: 'status',
            placeholder: 'Trạng thái',
            options: WORKOUT_PLAN_STATUS_OPTIONS,
            col: 4,
          },
        ]}
      >
        <Col xs={24} sm={12} md={6}>
          <MemberSelect
            placeholder="Lọc theo hội viên"
            value={table.params.member_id ?? null}
            onChange={(id) => table.setFilters({ member_id: id ? String(id) : undefined })}
          />
        </Col>
      </FilterBar>

      <DataTable<WorkoutPlan>
        table={table}
        columns={columns}
        emptyText="Chưa có giáo án tập luyện nào"
      />

      <PlanEditorModal
        open={editor.open}
        planId={editor.record?.id ?? null}
        initialRecord={editor.record}
        readOnly={!canManage}
        onClose={() => setEditor({ open: false, record: null })}
      />

      <ClonePlanModal
        open={!!cloneTarget}
        plan={cloneTarget}
        onClose={() => setCloneTarget(null)}
      />
    </Card>
  );
}
