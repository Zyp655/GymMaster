import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  AlertOutlined,
  CheckCircleOutlined,
  DeleteOutlined,
  DollarOutlined,
  EditOutlined,
  ToolOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import equipmentApi, {
  EQUIPMENT_CATEGORY_LABELS,
  EQUIPMENT_CATEGORY_OPTIONS,
  MAINTENANCE_TYPE_COLORS,
  MAINTENANCE_TYPE_LABELS,
  MAINTENANCE_TYPE_OPTIONS,
} from '@/api/equipment';
import type {
  Equipment,
  EquipmentCategory,
  EquipmentInput,
  EquipmentListParams,
  EquipmentStatus,
  MaintenanceListParams,
  MaintenanceLog,
  MaintenanceLogInput,
  MaintenanceStatus,
  MaintenanceType,
} from '@/api/equipment';
import roomApi from '@/api/room';
import ActionBar from '@/components/ActionBar';
import BranchSelect from '@/components/BranchSelect';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MoneyInput from '@/components/Money/MoneyInput';
import MoneyText from '@/components/Money/MoneyText';
import PageTitle from '@/components/PageTitle';
import StatCard from '@/components/StatCard';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import {
  EQUIPMENT_STATUS_MAP,
  EQUIPMENT_STATUS_OPTIONS,
  MAINTENANCE_STATUS_MAP,
  MAINTENANCE_STATUS_OPTIONS,
} from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import { DATE_FORMAT, formatDate, formatDateTime, formatNumber, toApiDate } from '@/utils/format';

interface EquipmentFormValues {
  branch_id: number;
  room_id?: number | null;
  code: string;
  name: string;
  category: EquipmentCategory;
  purchase_date?: Dayjs | null;
  price?: number;
  warranty_until?: Dayjs | null;
  status: EquipmentStatus;
  maintenance_interval_days?: number;
  next_maintenance_date?: Dayjs | null;
  note?: string;
}

function EquipmentFormModal({
  open,
  record,
  readOnly,
  onClose,
}: {
  open: boolean;
  record: Equipment | null;
  readOnly: boolean;
  onClose: () => void;
}) {
  const [form] = Form.useForm<EquipmentFormValues>();
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const selectedBranchId = Form.useWatch('branch_id', form);
  const isEdit = !!record;

  const roomsQuery = useQuery({
    queryKey: [roomApi.key, 'options', selectedBranchId ?? branchId],
    queryFn: () => roomApi.options(selectedBranchId ?? branchId),
    enabled: open,
  });

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    if (record) {
      form.setFieldsValue({
        branch_id: record.branch_id,
        room_id: record.room_id ?? undefined,
        code: record.code,
        name: record.name,
        category: record.category,
        purchase_date: record.purchase_date ? dayjs(record.purchase_date) : null,
        price: record.price,
        warranty_until: record.warranty_until ? dayjs(record.warranty_until) : null,
        status: record.status,
        maintenance_interval_days: record.maintenance_interval_days || 90,
        next_maintenance_date: record.next_maintenance_date
          ? dayjs(record.next_maintenance_date)
          : null,
        note: record.note,
      });
    } else {
      form.setFieldsValue({
        branch_id: branchId ?? undefined,
        category: 'CARDIO',
        status: 'ACTIVE',
        price: 0,
        maintenance_interval_days: 90,
        next_maintenance_date: dayjs().add(90, 'day'),
      });
    }
  }, [open, record, form, branchId]);

  const save = useMutation({
    mutationFn: (v: EquipmentFormValues) => {
      const payload: EquipmentInput = {
        branch_id: v.branch_id,
        room_id: v.room_id ?? null,
        code: v.code.trim().toUpperCase(),
        name: v.name.trim(),
        category: v.category,
        purchase_date: toApiDate(v.purchase_date) ?? null,
        price: Number(v.price) || 0,
        warranty_until: toApiDate(v.warranty_until) ?? null,
        status: v.status,
        maintenance_interval_days: Number(v.maintenance_interval_days) || 90,
        next_maintenance_date: toApiDate(v.next_maintenance_date) ?? null,
        note: v.note?.trim() || '',
      };
      return record ? equipmentApi.update(record.id, payload) : equipmentApi.create(payload);
    },
    onSuccess: () => {
      message.success(isEdit ? 'Đã cập nhật thiết bị' : 'Đã thêm thiết bị mới');
      void qc.invalidateQueries({ queryKey: [equipmentApi.key] });
      onClose();
    },
    onError: (err) => showError(err, 'Không thể lưu thiết bị'),
  });

  return (
    <Modal
      open={open}
      title={isEdit ? `Thiết bị: ${record?.code} – ${record?.name}` : 'Thêm thiết bị phòng tập'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm thiết bị'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={680}
      forceRender
    >
      <Form<EquipmentFormValues>
        form={form}
        layout="vertical"
        disabled={readOnly}
        onFinish={(v) => save.mutate(v)}
        style={{ marginTop: 12 }}
      >
        <Row gutter={12}>
          <Col xs={24} sm={8}>
            <Form.Item
              name="code"
              label="Mã thiết bị"
              rules={[{ required: true, whitespace: true, message: 'Vui lòng nhập mã thiết bị' }]}
            >
              <Input placeholder="VD: TB-TM01" maxLength={30} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={16}>
            <Form.Item
              name="name"
              label="Tên thiết bị"
              rules={[{ required: true, whitespace: true, message: 'Vui lòng nhập tên thiết bị' }]}
            >
              <Input placeholder="VD: Máy chạy bộ Technogym Excite Run" maxLength={150} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="branch_id"
              label="Chi nhánh"
              rules={[{ required: true, message: 'Vui lòng chọn chi nhánh' }]}
            >
              <BranchSelect activeOnly />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="room_id" label="Phòng đặt thiết bị">
              <Select
                allowClear
                placeholder="Khu vực chung / Chọn phòng"
                options={(roomsQuery.data ?? []).map((r) => ({ value: r.id, label: r.name }))}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="category"
              label="Danh mục"
              rules={[{ required: true, message: 'Chọn danh mục' }]}
            >
              <Select options={EQUIPMENT_CATEGORY_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="status"
              label="Trạng thái"
              rules={[{ required: true, message: 'Chọn trạng thái' }]}
            >
              <Select options={EQUIPMENT_STATUS_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={8}>
            <Form.Item name="price" label="Giá mua (VND)">
              <MoneyInput />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item name="purchase_date" label="Ngày mua">
              <DatePicker style={{ width: '100%' }} format={DATE_FORMAT} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={8}>
            <Form.Item name="warranty_until" label="Hạn bảo hành">
              <DatePicker style={{ width: '100%' }} format={DATE_FORMAT} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={12}>
            <Form.Item name="maintenance_interval_days" label="Chu kỳ bảo trì (ngày)">
              <InputNumber min={7} max={730} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={12}>
            <Form.Item name="next_maintenance_date" label="Ngày bảo trì tiếp theo">
              <DatePicker style={{ width: '100%' }} format={DATE_FORMAT} />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item name="note" label="Ghi chú">
              <Input.TextArea rows={2} maxLength={500} placeholder="Số serial, nhà cung cấp..." />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}

function ReportIssueModal({
  open,
  equipment,
  onClose,
}: {
  open: boolean;
  equipment: Equipment | null;
  onClose: () => void;
}) {
  const [form] = Form.useForm<{ description: string; cost?: number }>();
  const { message } = App.useApp();
  const qc = useQueryClient();

  useEffect(() => {
    if (open) form.resetFields();
  }, [open, form]);

  const reportMut = useMutation({
    mutationFn: (v: { description: string; cost?: number }) =>
      equipmentApi.reportIssue(equipment!.id, v.description.trim(), v.cost),
    onSuccess: () => {
      message.success(`Đã ghi nhận báo hỏng cho thiết bị ${equipment?.code}`);
      void qc.invalidateQueries({ queryKey: [equipmentApi.key] });
      void qc.invalidateQueries({ queryKey: [equipmentApi.logKey] });
      onClose();
    },
    onError: (err) => showError(err, 'Không thể gửi báo hỏng thiết bị'),
  });

  return (
    <Modal
      open={open}
      title={`Báo hỏng thiết bị: ${equipment?.code ?? ''} – ${equipment?.name ?? ''}`}
      okText="Xác nhận báo hỏng"
      okButtonProps={{ danger: true }}
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={reportMut.isPending}
      width={500}
      forceRender
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={(v) => reportMut.mutate(v)}
        style={{ marginTop: 12 }}
      >
        <Form.Item
          name="description"
          label="Mô tả sự cố / tình trạng hỏng"
          rules={[{ required: true, whitespace: true, message: 'Vui lòng mô tả tình trạng hỏng' }]}
        >
          <Input.TextArea
            rows={3}
            placeholder="VD: Băng tải bị trượt, phát tiếng kêu lớn ở motor..."
            autoFocus
          />
        </Form.Item>
        <Form.Item name="cost" label="Chi phí sửa chữa dự kiến (nếu có)">
          <MoneyInput />
        </Form.Item>
      </Form>
    </Modal>
  );
}

function MaintenanceModal({
  open,
  record,
  onClose,
}: {
  open: boolean;
  record: MaintenanceLog | null;
  onClose: () => void;
}) {
  const [form] = Form.useForm<MaintenanceLogInput>();
  const { message } = App.useApp();
  const qc = useQueryClient();
  const isEdit = !!record;

  const equipmentsQuery = useQuery({
    queryKey: [equipmentApi.key, 'options'],
    queryFn: () => equipmentApi.list({ limit: 200 }),
    enabled: open && !isEdit,
  });

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    if (record) {
      form.setFieldsValue({
        equipment_id: record.equipment_id,
        type: record.type,
        description: record.description,
        cost: record.cost,
        status: record.status,
      });
    } else {
      form.setFieldsValue({
        type: 'PERIODIC',
        status: 'OPEN',
        cost: 0,
      });
    }
  }, [open, record, form]);

  const save = useMutation({
    mutationFn: (v: MaintenanceLogInput) => {
      const payload = {
        equipment_id: v.equipment_id,
        type: v.type,
        description: v.description.trim(),
        cost: Number(v.cost) || 0,
        status: v.status,
      };
      return record
        ? equipmentApi.updateLog(record.id, payload)
        : equipmentApi.createLog(payload);
    },
    onSuccess: () => {
      message.success(isEdit ? 'Đã cập nhật phiếu bảo trì' : 'Đã tạo phiếu bảo trì');
      void qc.invalidateQueries({ queryKey: [equipmentApi.logKey] });
      void qc.invalidateQueries({ queryKey: [equipmentApi.key] });
      onClose();
    },
    onError: (err) => showError(err, 'Không thể lưu phiếu bảo trì'),
  });

  return (
    <Modal
      open={open}
      title={isEdit ? `Cập nhật phiếu bảo trì #${record?.id}` : 'Tạo phiếu bảo trì / sửa chữa'}
      okText={isEdit ? 'Lưu cập nhật' : 'Tạo phiếu'}
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={540}
      forceRender
    >
      <Form<MaintenanceLogInput>
        form={form}
        layout="vertical"
        onFinish={(v) => save.mutate(v)}
        style={{ marginTop: 12 }}
      >
        {!isEdit && (
          <Form.Item
            name="equipment_id"
            label="Thiết bị"
            rules={[{ required: true, message: 'Vui lòng chọn thiết bị' }]}
          >
            <Select
              showSearch
              optionFilterProp="label"
              placeholder="Chọn thiết bị cần bảo trì"
              options={(equipmentsQuery.data?.items ?? []).map((e) => ({
                value: e.id,
                label: `${e.code} – ${e.name}`,
              }))}
            />
          </Form.Item>
        )}
        <Row gutter={12}>
          <Col xs={12}>
            <Form.Item
              name="type"
              label="Loại bảo trì"
              rules={[{ required: true, message: 'Chọn loại bảo trì' }]}
            >
              <Select options={MAINTENANCE_TYPE_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={12}>
            <Form.Item
              name="status"
              label="Trạng thái xử lý"
              rules={[{ required: true, message: 'Chọn trạng thái' }]}
            >
              <Select options={MAINTENANCE_STATUS_OPTIONS} />
            </Form.Item>
          </Col>
        </Row>
        <Form.Item name="cost" label="Chi phí thực tế (VND)">
          <MoneyInput />
        </Form.Item>
        <Form.Item
          name="description"
          label="Nội dung công việc / Mô tả"
          rules={[{ required: true, whitespace: true, message: 'Nhập nội dung bảo trì' }]}
        >
          <Input.TextArea rows={3} placeholder="Chi tiết hạng mục bảo dưỡng, thay thế linh kiện..." />
        </Form.Item>
      </Form>
    </Modal>
  );
}

function EquipmentsTab() {
  const { message } = App.useApp();
  const { can } = usePermission();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const canManage = can(PERM.EQUIPMENT_MANAGE);

  const [modal, setModal] = useState<{ open: boolean; record: Equipment | null }>({
    open: false,
    record: null,
  });
  const [issueTarget, setIssueTarget] = useState<Equipment | null>(null);

  const table = useTableQuery<Equipment, EquipmentListParams>({
    queryKey: [equipmentApi.key, 'list'],
    queryFn: (p) => equipmentApi.list(p),
    filterKeys: ['branch_id', 'category', 'status'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const remove = useMutation({
    mutationFn: (id: number) => equipmentApi.remove(id),
    onSuccess: () => {
      message.success('Đã xóa thiết bị');
      void qc.invalidateQueries({ queryKey: [equipmentApi.key] });
    },
    onError: (err) => showError(err, 'Không thể xóa thiết bị'),
  });

  const columns = useMemo<ColumnsType<Equipment>>(
    () => [
      {
        title: 'Mã TB',
        dataIndex: 'code',
        width: 110,
        render: (code: string, r) => (
          <Button
            type="link"
            size="small"
            className="code-text"
            style={{ padding: 0 }}
            onClick={() => setModal({ open: true, record: r })}
          >
            {code}
          </Button>
        ),
      },
      {
        title: 'Tên thiết bị',
        dataIndex: 'name',
        render: (name: string, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>{name}</Typography.Text>
            {r.room_name && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                Phòng: {r.room_name}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Danh mục',
        dataIndex: 'category',
        render: (c: EquipmentCategory) => EQUIPMENT_CATEGORY_LABELS[c] ?? c,
      },
      {
        title: 'Chi nhánh',
        dataIndex: 'branch_name',
        render: (v: string | undefined, r) => v || `CN #${r.branch_id}`,
      },
      {
        title: 'Giá mua',
        dataIndex: 'price',
        align: 'right',
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        title: 'Bảo trì tiếp theo',
        dataIndex: 'next_maintenance_date',
        className: 'text-nowrap',
        render: (v: string | null) => {
          if (!v) return '—';
          const due = dayjs(v).isBefore(dayjs(), 'day');
          return (
            <Typography.Text type={due ? 'danger' : undefined}>
              {formatDate(v)} {due ? '(Quá hạn)' : ''}
            </Typography.Text>
          );
        },
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        render: (s: string) => <StatusTag status={s} map={EQUIPMENT_STATUS_MAP} />,
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 140,
        render: (_, r) => (
          <Space size={4}>
            {canManage && r.status === 'ACTIVE' && (
              <Tooltip title="Báo hỏng thiết bị">
                <Button
                  size="small"
                  danger
                  icon={<AlertOutlined />}
                  onClick={() => setIssueTarget(r)}
                >
                  Báo hỏng
                </Button>
              </Tooltip>
            )}
            {canManage && (
              <Tooltip title="Sửa">
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => setModal({ open: true, record: r })}
                />
              </Tooltip>
            )}
            <ConfirmButton
              perm={PERM.EQUIPMENT_MANAGE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa thiết bị ${r.code} – ${r.name}?`}
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
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col />
        <Col>
          <ActionBar
            onAdd={() => setModal({ open: true, record: null })}
            addPerm={PERM.EQUIPMENT_MANAGE}
            addText="Thêm thiết bị"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo mã, tên thiết bị..."
        colTextSearch={5}
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
            name: 'category',
            placeholder: 'Danh mục thiết bị',
            options: EQUIPMENT_CATEGORY_OPTIONS,
            col: 5,
          },
          {
            name: 'status',
            placeholder: 'Trạng thái',
            options: EQUIPMENT_STATUS_OPTIONS,
            col: 4,
          },
        ]}
      />

      <DataTable<Equipment>
        table={table}
        columns={columns}
        emptyText="Chưa có thiết bị nào"
      />

      <EquipmentFormModal
        open={modal.open}
        record={modal.record}
        readOnly={!canManage}
        onClose={() => setModal({ open: false, record: null })}
      />

      <ReportIssueModal
        open={!!issueTarget}
        equipment={issueTarget}
        onClose={() => setIssueTarget(null)}
      />
    </>
  );
}

function MaintenanceLogsTab() {
  const { message } = App.useApp();
  const { can } = usePermission();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const canManage = can(PERM.EQUIPMENT_MANAGE);

  const [modal, setModal] = useState<{ open: boolean; record: MaintenanceLog | null }>({
    open: false,
    record: null,
  });

  const table = useTableQuery<MaintenanceLog, MaintenanceListParams>({
    queryKey: [equipmentApi.logKey, 'list'],
    queryFn: (p) => equipmentApi.listLogs(p),
    filterKeys: ['status', 'type'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const summaryQuery = useQuery({
    queryKey: [equipmentApi.logKey, 'summary', branchId],
    queryFn: () => equipmentApi.summary({ branch_id: branchId ?? undefined }),
  });

  const quickDone = useMutation({
    mutationFn: (r: MaintenanceLog) =>
      equipmentApi.updateLog(r.id, { status: 'DONE', cost: r.cost }),
    onSuccess: () => {
      message.success('Đã hoàn tất bảo trì, thiết bị hoạt động trở lại');
      void qc.invalidateQueries({ queryKey: [equipmentApi.logKey] });
      void qc.invalidateQueries({ queryKey: [equipmentApi.key] });
    },
    onError: (err) => showError(err, 'Không thể cập nhật trạng thái bảo trì'),
  });

  const summary = summaryQuery.data;
  const computedCost = useMemo(
    () =>
      summary?.total_cost ??
      table.items.reduce((acc, item) => acc + (Number(item.cost) || 0), 0),
    [summary?.total_cost, table.items],
  );

  const columns = useMemo<ColumnsType<MaintenanceLog>>(
    () => [
      {
        title: 'Ngày tạo',
        dataIndex: 'created_at',
        className: 'text-nowrap',
        render: (v: string) => formatDateTime(v),
      },
      {
        title: 'Thiết bị',
        key: 'equipment',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>
              {r.equipment_name || `Thiết bị #${r.equipment_id}`}
            </Typography.Text>
            {r.equipment_code && (
              <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
                {r.equipment_code}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Loại',
        dataIndex: 'type',
        render: (t: MaintenanceType) => (
          <Tag color={MAINTENANCE_TYPE_COLORS[t] ?? 'default'}>
            {MAINTENANCE_TYPE_LABELS[t] ?? t}
          </Tag>
        ),
      },
      {
        title: 'Mô tả công việc / sự cố',
        dataIndex: 'description',
      },
      {
        title: 'Chi phí',
        dataIndex: 'cost',
        align: 'right',
        render: (v: number) => <MoneyText value={v} strong />,
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        render: (s: MaintenanceStatus) => <StatusTag status={s} map={MAINTENANCE_STATUS_MAP} />,
      },
      {
        title: 'Hoàn thành',
        dataIndex: 'finished_at',
        className: 'text-nowrap',
        render: (v: string | null | undefined) => (v ? formatDateTime(v) : '—'),
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 130,
        render: (_, r) => (
          <Space size={4}>
            {canManage && r.status !== 'DONE' && (
              <Tooltip title="Đánh dấu hoàn tất bảo trì">
                <Button
                  size="small"
                  type="primary"
                  ghost
                  icon={<CheckCircleOutlined />}
                  loading={quickDone.isPending}
                  onClick={() => quickDone.mutate(r)}
                >
                  Xong
                </Button>
              </Tooltip>
            )}
            {canManage && (
              <Tooltip title="Cập nhật chi phí / trạng thái">
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => setModal({ open: true, record: r })}
                />
              </Tooltip>
            )}
          </Space>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [canManage],
  );

  return (
    <>
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={12} md={8}>
          <StatCard
            label="Tổng chi phí bảo trì"
            value={computedCost}
            currency
            caption={`${formatNumber(summary?.total_count ?? table.total)} phiếu bảo trì`}
            icon={<DollarOutlined />}
            loading={summaryQuery.isLoading && table.isLoading}
          />
        </Col>
        <Col xs={12} md={8}>
          <StatCard
            label="Đang chờ / Đang xử lý"
            value={
              (summary?.open_count ??
                table.items.filter((x) => x.status === 'OPEN').length) +
              (summary?.in_progress_count ??
                table.items.filter((x) => x.status === 'IN_PROGRESS').length)
            }
            caption="Cần kỹ thuật viên kiểm tra"
            icon={<WarningOutlined />}
            loading={table.isLoading}
          />
        </Col>
        <Col xs={24} md={8}>
          <StatCard
            label="Đã hoàn tất"
            value={
              summary?.done_count ??
              table.items.filter((x) => x.status === 'DONE').length
            }
            caption="Thiết bị đã đưa vào vận hành"
            icon={<ToolOutlined />}
            loading={table.isLoading}
          />
        </Col>
      </Row>

      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col />
        <Col>
          <ActionBar
            onAdd={() => setModal({ open: true, record: null })}
            addPerm={PERM.EQUIPMENT_MANAGE}
            addText="Tạo phiếu bảo trì"
            onRefresh={() => {
              void table.refetch();
              void summaryQuery.refetch();
            }}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo mô tả, tên/mã thiết bị..."
        colTextSearch={6}
        filters={[
          {
            name: 'status',
            placeholder: 'Trạng thái phiếu',
            options: MAINTENANCE_STATUS_OPTIONS,
            col: 5,
          },
          {
            name: 'type',
            placeholder: 'Loại bảo trì',
            options: MAINTENANCE_TYPE_OPTIONS,
            col: 5,
          },
        ]}
      />

      <DataTable<MaintenanceLog>
        table={table}
        columns={columns}
        emptyText="Chưa có nhật ký bảo trì nào"
      />

      <MaintenanceModal
        open={modal.open}
        record={modal.record}
        onClose={() => setModal({ open: false, record: null })}
      />
    </>
  );
}

/** Trang Thiết bị & Bảo trì (`/app/equipments`). */
export default function EquipmentsPage() {
  return (
    <Card variant="borderless" className="page-card">
      <div className="page-head">
        <PageTitle
          title="Thiết bị & bảo trì"
          subtitle="Quản lý danh mục máy tập, lịch bảo dưỡng định kỳ, báo hỏng và chi phí sửa chữa"
        />
      </div>

      <Tabs
        defaultActiveKey="equipments"
        items={[
          {
            key: 'equipments',
            label: 'Danh sách thiết bị',
            children: <EquipmentsTab />,
          },
          {
            key: 'maintenance',
            label: 'Nhật ký bảo trì & Chi phí',
            children: <MaintenanceLogsTab />,
          },
        ]}
      />
    </Card>
  );
}
