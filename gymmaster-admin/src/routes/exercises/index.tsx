import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App, Button, Card, Col, Form, Input, Modal, Row, Select, Space, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  DeleteOutlined,
  EditOutlined,
  PlayCircleOutlined,
  ScheduleOutlined,
} from '@ant-design/icons';
import { Link } from 'react-router-dom';
import workoutApi, {
  MUSCLE_GROUP_COLORS,
  MUSCLE_GROUP_LABELS,
  MUSCLE_GROUP_OPTIONS,
} from '@/api/workout';
import type { Exercise, ExerciseInput, ExerciseListParams, MuscleGroup } from '@/api/workout';
import ActionBar from '@/components/ActionBar';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import PageTitle from '@/components/PageTitle';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { setFormError, showError } from '@/utils/errors';

interface ExerciseModalProps {
  open: boolean;
  record: Exercise | null;
  readOnly: boolean;
  onClose: () => void;
}

function ExerciseFormModal({ open, record, readOnly, onClose }: ExerciseModalProps) {
  const [form] = Form.useForm<ExerciseInput>();
  const { message } = App.useApp();
  const qc = useQueryClient();
  const isEdit = !!record;

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    if (record) {
      form.setFieldsValue({
        name: record.name,
        muscle_group: record.muscle_group,
        equipment: record.equipment,
        video_url: record.video_url,
        description: record.description,
      });
    } else {
      form.setFieldsValue({ muscle_group: 'CHEST', equipment: '', video_url: '', description: '' });
    }
  }, [open, record, form]);

  const save = useMutation({
    mutationFn: (values: ExerciseInput) => {
      const payload: ExerciseInput = {
        name: values.name.trim(),
        muscle_group: values.muscle_group,
        equipment: values.equipment?.trim() || '',
        video_url: values.video_url?.trim() || '',
        description: values.description?.trim() || '',
      };
      return record ? workoutApi.updateExercise(record.id, payload) : workoutApi.createExercise(payload);
    },
    onSuccess: () => {
      message.success(isEdit ? 'Đã cập nhật bài tập' : 'Đã thêm bài tập mới');
      void qc.invalidateQueries({ queryKey: [workoutApi.exerciseKey] });
      onClose();
    },
    onError: (err) => {
      const mapped = setFormError(form, err, {
        name: /tên bài/i,
        muscle_group: /nhóm cơ/i,
      });
      if (!mapped) showError(err, 'Không thể lưu bài tập');
    },
  });

  return (
    <Modal
      open={open}
      title={isEdit ? `Bài tập: ${record?.name}` : 'Thêm bài tập mới'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm bài tập'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={620}
      forceRender
    >
      <Form<ExerciseInput>
        form={form}
        layout="vertical"
        disabled={readOnly}
        onFinish={(v) => save.mutate(v)}
        style={{ marginTop: 12 }}
      >
        <Row gutter={16}>
          <Col xs={24} sm={14}>
            <Form.Item
              name="name"
              label="Tên bài tập"
              rules={[
                { required: true, whitespace: true, message: 'Vui lòng nhập tên bài tập' },
                { max: 150, message: 'Tên bài tập tối đa 150 ký tự' },
              ]}
            >
              <Input placeholder="VD: Đẩy ngực thanh đòn (Barbell Bench Press)" maxLength={150} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={10}>
            <Form.Item
              name="muscle_group"
              label="Nhóm cơ chính"
              rules={[{ required: true, message: 'Vui lòng chọn nhóm cơ' }]}
            >
              <Select options={MUSCLE_GROUP_OPTIONS} placeholder="Chọn nhóm cơ" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="equipment"
              label="Dụng cụ / Thiết bị"
              rules={[{ max: 100, message: 'Dụng cụ tối đa 100 ký tự' }]}
            >
              <Input placeholder="VD: Thanh đòn, Tạ đơn, Máy cáp..." maxLength={100} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="video_url"
              label="Link video hướng dẫn"
              rules={[{ max: 500, message: 'Link tối đa 500 ký tự' }]}
            >
              <Input placeholder="https://youtube.com/..." maxLength={500} />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item name="description" label="Mô tả kỹ thuật & lưu ý">
              <Input.TextArea
                rows={4}
                placeholder="Hướng dẫn tư thế chuẩn, nhịp thở, các lỗi thường gặp..."
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

/** Trang Thư viện bài tập (`/app/exercises`). */
export default function ExercisesPage() {
  const { message } = App.useApp();
  const { can } = usePermission();
  const qc = useQueryClient();
  const canManage = can(PERM.WORKOUT_MANAGE);
  const [modal, setModal] = useState<{ open: boolean; record: Exercise | null }>({
    open: false,
    record: null,
  });

  const table = useTableQuery<Exercise, ExerciseListParams>({
    queryKey: [workoutApi.exerciseKey, 'list'],
    queryFn: (p) => workoutApi.listExercises(p),
    filterKeys: ['muscle_group'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
  });

  const remove = useMutation({
    mutationFn: (id: number) => workoutApi.deleteExercise(id),
    onSuccess: () => {
      message.success('Đã xóa bài tập');
      void qc.invalidateQueries({ queryKey: [workoutApi.exerciseKey] });
    },
    onError: (err) => showError(err, 'Không thể xóa bài tập (có thể đang dùng trong giáo án)'),
  });

  const openForm = (record: Exercise | null = null) => setModal({ open: true, record });

  const columns = useMemo<ColumnsType<Exercise>>(
    () => [
      {
        title: 'Tên bài tập',
        dataIndex: 'name',
        sorter: true,
        render: (name: string, r) => (
          <Button type="link" size="small" style={{ padding: 0 }} onClick={() => openForm(r)}>
            {name}
          </Button>
        ),
      },
      {
        title: 'Nhóm cơ',
        dataIndex: 'muscle_group',
        width: 170,
        render: (m: MuscleGroup) => (
          <Tag color={MUSCLE_GROUP_COLORS[m] ?? 'default'}>{MUSCLE_GROUP_LABELS[m] ?? m}</Tag>
        ),
      },
      {
        title: 'Dụng cụ',
        dataIndex: 'equipment',
        width: 180,
        render: (v: string) => v || <Typography.Text type="secondary">Tự trọng / Không</Typography.Text>,
      },
      {
        title: 'Mô tả kỹ thuật',
        dataIndex: 'description',
        render: (v: string) =>
          v ? (
            <Typography.Text ellipsis={{ tooltip: v }} style={{ maxWidth: 340 }}>
              {v}
            </Typography.Text>
          ) : (
            <Typography.Text type="secondary">—</Typography.Text>
          ),
      },
      {
        title: 'Video',
        dataIndex: 'video_url',
        width: 110,
        align: 'center',
        render: (url: string) =>
          url ? (
            <a href={url} target="_blank" rel="noreferrer">
              <Button size="small" type="link" icon={<PlayCircleOutlined />}>
                Xem
              </Button>
            </a>
          ) : (
            <Typography.Text type="secondary">—</Typography.Text>
          ),
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 88,
        render: (_, r) => (
          <Space size={0}>
            {canManage && (
              <Tooltip title="Sửa">
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => openForm(r)}
                />
              </Tooltip>
            )}
            <ConfirmButton
              perm={PERM.WORKOUT_MANAGE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa bài tập "${r.name}"?`}
              confirmContent="Bài tập đang nằm trong giáo án sẽ không xóa được."
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
            title="Thư viện bài tập"
            subtitle="Danh mục bài tập chuẩn theo nhóm cơ để huấn luyện viên soạn giáo án"
          />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => openForm(null)}
            addPerm={PERM.WORKOUT_MANAGE}
            addText="Thêm bài tập"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          >
            <Link to="/app/workout-plans">
              <Button icon={<ScheduleOutlined />}>Giáo án tập luyện</Button>
            </Link>
          </ActionBar>
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên bài tập, dụng cụ, mô tả..."
        colTextSearch={6}
        filters={[
          {
            name: 'muscle_group',
            placeholder: 'Nhóm cơ',
            options: MUSCLE_GROUP_OPTIONS,
            col: 5,
          },
        ]}
      />

      <DataTable<Exercise>
        table={table}
        columns={columns}
        emptyText="Chưa có bài tập nào trong thư viện"
      />

      <ExerciseFormModal
        open={modal.open}
        record={modal.record}
        readOnly={!canManage}
        onClose={() => setModal((m) => ({ ...m, open: false }))}
      />
    </Card>
  );
}
