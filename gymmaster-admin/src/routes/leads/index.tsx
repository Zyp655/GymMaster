import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Badge,
  Button,
  Card,
  Col,
  DatePicker,
  Descriptions,
  Divider,
  Drawer,
  Empty,
  Form,
  Input,
  Modal,
  Progress,
  Row,
  Segmented,
  Select,
  Space,
  Spin,
  Table,
  Tabs,
  Tag,
  Timeline,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  AppstoreOutlined,
  CheckCircleOutlined,
  FilterOutlined,
  PhoneOutlined,
  QrcodeOutlined,
  SendOutlined,
  SwapRightOutlined,
  TableOutlined,
  TeamOutlined,
  UserSwitchOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { Link, useNavigate } from 'react-router-dom';
import leadsApi, {
  LEAD_ACTIVITY_LABELS,
  LEAD_ACTIVITY_OPTIONS,
  LEAD_SOURCE_COLORS,
  LEAD_SOURCE_LABELS,
  LEAD_SOURCE_OPTIONS,
  LEAD_STATUS_ORDER,
} from '@/api/leads';
import type {
  AtRiskMember,
  Lead,
  LeadActivityType,
  LeadInput,
  LeadListParams,
  LeadSource,
  LeadStatus,
  TrialPass,
} from '@/api/leads';
import membershipPlanApi from '@/api/plan';
import staffApi from '@/api/staff';
import ActionBar from '@/components/ActionBar';
import BranchSelect from '@/components/BranchSelect';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import PageTitle from '@/components/PageTitle';
import StatCard from '@/components/StatCard';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { LEAD_STATUS_MAP, LEAD_STATUS_OPTIONS } from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import { DATE_FORMAT, formatDate, formatDateTime, formatNumber, formatPhone, toApiDate } from '@/utils/format';

interface LeadFormValues {
  branch_id: number;
  full_name: string;
  phone: string;
  email?: string;
  source: LeadSource;
  interest_plan_id?: number;
  status: LeadStatus;
  assigned_to?: number;
  trial_date?: Dayjs | null;
  note?: string;
}

function LeadFormModal({
  open,
  record,
  onClose,
}: {
  open: boolean;
  record: Lead | null;
  onClose: () => void;
}) {
  const [form] = Form.useForm<LeadFormValues>();
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const isEdit = !!record;

  const plansQuery = useQuery({
    queryKey: [membershipPlanApi.key, 'options', 'leads'],
    queryFn: () => membershipPlanApi.options(),
    enabled: open,
  });

  const staffQuery = useQuery({
    queryKey: [staffApi.key, 'options', 'leads'],
    queryFn: () => staffApi.options(branchId),
    enabled: open,
  });

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    if (record) {
      form.setFieldsValue({
        branch_id: record.branch_id,
        full_name: record.full_name,
        phone: record.phone,
        email: record.email,
        source: record.source,
        interest_plan_id: record.interest_plan_id ?? undefined,
        status: record.status,
        assigned_to: record.assigned_to ?? undefined,
        trial_date: record.trial_date ? dayjs(record.trial_date) : null,
        note: record.note,
      });
    } else {
      form.setFieldsValue({
        branch_id: branchId ?? undefined,
        source: 'WALK_IN',
        status: 'NEW',
      });
    }
  }, [open, record, form, branchId]);

  const save = useMutation({
    mutationFn: (v: LeadFormValues) => {
      const payload: LeadInput = {
        branch_id: v.branch_id,
        full_name: v.full_name.trim(),
        phone: v.phone.trim(),
        email: v.email?.trim() || '',
        source: v.source,
        interest_plan_id: v.interest_plan_id ?? null,
        status: v.status,
        assigned_to: v.assigned_to ?? null,
        trial_date: toApiDate(v.trial_date) ?? null,
        note: v.note?.trim() || '',
      };
      return record ? leadsApi.update(record.id, payload) : leadsApi.create(payload);
    },
    onSuccess: () => {
      message.success(isEdit ? 'Đã cập nhật khách tiềm năng' : 'Đã thêm khách tiềm năng');
      void qc.invalidateQueries({ queryKey: [leadsApi.key] });
      onClose();
    },
    onError: (err) => showError(err, 'Không thể lưu khách tiềm năng'),
  });

  return (
    <Modal
      open={open}
      title={isEdit ? `Cập nhật Lead: ${record?.full_name}` : 'Thêm khách tiềm năng (Lead)'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm Lead'}
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={620}
      forceRender
    >
      <Form<LeadFormValues>
        form={form}
        layout="vertical"
        onFinish={(v) => save.mutate(v)}
        style={{ marginTop: 12 }}
      >
        <Row gutter={12}>
          <Col xs={24} sm={12}>
            <Form.Item
              name="full_name"
              label="Họ và tên khách"
              rules={[{ required: true, whitespace: true, message: 'Vui lòng nhập họ tên' }]}
            >
              <Input placeholder="Nguyễn Văn A" maxLength={150} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="phone"
              label="Số điện thoại"
              rules={[{ required: true, whitespace: true, message: 'Vui lòng nhập số điện thoại' }]}
            >
              <Input placeholder="0901234567" maxLength={20} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="email"
              label="Email"
              rules={[{ type: 'email', message: 'Email không hợp lệ' }]}
            >
              <Input placeholder="khachhang@email.com" maxLength={150} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="branch_id"
              label="Chi nhánh quan tâm"
              rules={[{ required: true, message: 'Vui lòng chọn chi nhánh' }]}
            >
              <BranchSelect activeOnly />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="source" label="Nguồn khách" rules={[{ required: true }]}>
              <Select options={LEAD_SOURCE_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="status" label="Trạng thái" rules={[{ required: true }]}>
              <Select options={LEAD_STATUS_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="interest_plan_id" label="Gói tập quan tâm">
              <Select
                allowClear
                placeholder="Chọn gói tập"
                options={(plansQuery.data ?? []).map((p) => ({
                  value: p.id,
                  label: `${p.code} – ${p.name}`,
                }))}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="assigned_to" label="Nhân viên phụ trách">
              <Select
                allowClear
                placeholder="Chọn nhân viên CSKH"
                options={(staffQuery.data ?? []).map((s) => ({
                  value: s.id,
                  label: `${s.full_name} (${s.staff_code})`,
                }))}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="trial_date" label="Ngày hẹn tập thử">
              <DatePicker style={{ width: '100%' }} format={DATE_FORMAT} />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item name="note" label="Ghi chú nhu cầu">
              <Input.TextArea
                rows={2}
                placeholder="Khách muốn giảm cân, tập khung giờ 18h-20h..."
              />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}

function LeadDetailDrawer({
  lead,
  canManage,
  onClose,
  onEdit,
}: {
  lead: Lead | null;
  canManage: boolean;
  onClose: () => void;
  onEdit: (lead: Lead) => void;
}) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [activityForm] = Form.useForm<{ type: LeadActivityType; content: string }>();
  const [trialPass, setTrialPass] = useState<TrialPass | null>(null);

  useEffect(() => {
    setTrialPass(lead?.trial_pass ?? null);
  }, [lead]);

  const detailQuery = useQuery({
    queryKey: [leadsApi.key, 'detail', lead?.id],
    queryFn: () => leadsApi.get(lead!.id),
    enabled: !!lead,
  });

  const activitiesQuery = useQuery({
    queryKey: [leadsApi.key, 'activities', lead?.id],
    queryFn: async () => {
      try {
        return await leadsApi.listActivities(lead!.id);
      } catch {
        return detailQuery.data?.activities ?? [];
      }
    },
    enabled: !!lead,
  });

  const currentLead = detailQuery.data ?? lead;
  const activities = activitiesQuery.data ?? currentLead?.activities ?? [];

  const updateStatusMut = useMutation({
    mutationFn: (status: LeadStatus) => leadsApi.update(currentLead!.id, { status }),
    onSuccess: () => {
      message.success('Đã cập nhật trạng thái khách tiềm năng');
      void qc.invalidateQueries({ queryKey: [leadsApi.key] });
    },
    onError: (err) => showError(err, 'Không thể đổi trạng thái'),
  });

  const addActivityMut = useMutation({
    mutationFn: (v: { type: LeadActivityType; content: string }) =>
      leadsApi.addActivity(currentLead!.id, { type: v.type, content: v.content.trim() }),
    onSuccess: () => {
      message.success('Đã lưu nhật ký chăm sóc');
      activityForm.resetFields();
      void qc.invalidateQueries({ queryKey: [leadsApi.key] });
    },
    onError: (err) => showError(err, 'Không thể lưu lịch sử chăm sóc'),
  });

  const trialPassMut = useMutation({
    mutationFn: () =>
      leadsApi.issueTrialPass(currentLead!.id, dayjs().add(3, 'day').format('YYYY-MM-DD')),
    onSuccess: (pass) => {
      setTrialPass(pass);
      message.success('Đã cấp vé tập thử QR thành công');
      void qc.invalidateQueries({ queryKey: [leadsApi.key] });
    },
    onError: (err) => showError(err, 'Không thể cấp vé tập thử'),
  });

  const convertMut = useMutation({
    mutationFn: () => leadsApi.convert(currentLead!.id),
    onSuccess: (res) => {
      message.success('Đã chuyển đổi khách tiềm năng thành Hội viên!');
      void qc.invalidateQueries({ queryKey: [leadsApi.key] });
      onClose();
      const memberId = res?.member_id ?? res?.id;
      navigate(memberId ? `/app/subscriptions?member_id=${memberId}` : '/app/subscriptions');
    },
    onError: (err) => showError(err, 'Không thể chuyển đổi thành hội viên'),
  });

  const activeTrialPass = trialPass ?? currentLead?.trial_pass;

  return (
    <Drawer
      open={!!lead}
      onClose={onClose}
      width={620}
      title={
        currentLead ? (
          <Space>
            <span>{currentLead.full_name}</span>
            <StatusTag status={currentLead.status} map={LEAD_STATUS_MAP} />
          </Space>
        ) : (
          'Chi tiết khách tiềm năng'
        )
      }
      extra={
        currentLead && canManage ? (
          <Button size="small" onClick={() => onEdit(currentLead)}>
            Sửa thông tin
          </Button>
        ) : null
      }
    >
      {currentLead && (
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          {canManage && (
            <Card size="small" style={{ background: '#fafbfc' }}>
              <Space direction="vertical" size={10} style={{ width: '100%' }}>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  Chuyển bước chăm sóc:
                </Typography.Text>
                <Segmented
                  block
                  value={currentLead.status}
                  options={LEAD_STATUS_OPTIONS}
                  disabled={updateStatusMut.isPending}
                  onChange={(st) => updateStatusMut.mutate(st as LeadStatus)}
                />
                <Space wrap style={{ marginTop: 4 }}>
                  <Button
                    icon={<QrcodeOutlined />}
                    loading={trialPassMut.isPending}
                    onClick={() => trialPassMut.mutate()}
                  >
                    Cấp vé tập thử QR
                  </Button>
                  {currentLead.status !== 'CONVERTED' && (
                    <Button
                      color="orange"
                      variant="solid"
                      icon={<UserSwitchOutlined />}
                      loading={convertMut.isPending}
                      onClick={() => convertMut.mutate()}
                    >
                      Chuyển đổi thành Hội viên & Bán gói
                    </Button>
                  )}
                  {currentLead.member_id && (
                    <Link to={`/app/members/${currentLead.member_id}`}>
                      <Button type="primary" ghost icon={<CheckCircleOutlined />}>
                        Xem hồ sơ Hội viên #{currentLead.member_id}
                      </Button>
                    </Link>
                  )}
                </Space>
              </Space>
            </Card>
          )}

          {activeTrialPass && (
            <Alert
              type="info"
              showIcon
              icon={<QrcodeOutlined />}
              message="Vé tập thử QR (quét tại quầy Check-in)"
              description={
                <Space direction="vertical" size={2}>
                  <div>
                    Mã QR Token: <Tag color="geekblue" className="code-text">{activeTrialPass.qr_token}</Tag>
                  </div>
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    Hiệu lực đến: {formatDate(activeTrialPass.valid_date)}
                    {activeTrialPass.used_at
                      ? ` · Đã sử dụng lúc ${formatDateTime(activeTrialPass.used_at)}`
                      : ' · Chưa sử dụng'}
                  </Typography.Text>
                </Space>
              }
            />
          )}

          <Descriptions size="small" bordered column={2}>
            <Descriptions.Item label="Họ tên">{currentLead.full_name}</Descriptions.Item>
            <Descriptions.Item label="Số điện thoại">
              {formatPhone(currentLead.phone)}
            </Descriptions.Item>
            <Descriptions.Item label="Email">{currentLead.email || '—'}</Descriptions.Item>
            <Descriptions.Item label="Nguồn">
              <Tag color={LEAD_SOURCE_COLORS[currentLead.source] ?? 'default'}>
                {LEAD_SOURCE_LABELS[currentLead.source] ?? currentLead.source}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Chi nhánh">
              {currentLead.branch_name || `CN #${currentLead.branch_id}`}
            </Descriptions.Item>
            <Descriptions.Item label="Ngày hẹn tập thử">
              {formatDate(currentLead.trial_date)}
            </Descriptions.Item>
            <Descriptions.Item label="Ghi chú" span={2}>
              {currentLead.note || '—'}
            </Descriptions.Item>
          </Descriptions>

          <Divider orientation="left" style={{ margin: '4px 0' }}>
            Lịch sử chăm sóc ({activities.length})
          </Divider>

          {canManage && (
            <Form
              form={activityForm}
              layout="vertical"
              initialValues={{ type: 'CALL', content: '' }}
              onFinish={(v) => addActivityMut.mutate(v)}
            >
              <Row gutter={8}>
                <Col xs={24} sm={7}>
                  <Form.Item name="type" style={{ marginBottom: 8 }} rules={[{ required: true }]}>
                    <Select options={LEAD_ACTIVITY_OPTIONS} />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={13}>
                  <Form.Item
                    name="content"
                    style={{ marginBottom: 8 }}
                    rules={[{ required: true, whitespace: true, message: 'Nhập nội dung trao đổi' }]}
                  >
                    <Input placeholder="Nhập kết quả cuộc gọi, ghi chú hẹn tập thử..." />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={4}>
                  <Button
                    type="primary"
                    htmlType="submit"
                    block
                    icon={<SendOutlined />}
                    loading={addActivityMut.isPending}
                  >
                    Lưu
                  </Button>
                </Col>
              </Row>
            </Form>
          )}

          {activitiesQuery.isLoading ? (
            <div style={{ textAlign: 'center', padding: 24 }}>
              <Spin />
            </div>
          ) : activities.length === 0 ? (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="Chưa có lịch sử chăm sóc"
            />
          ) : (
            <Timeline
              items={activities.map((a) => ({
                key: a.id,
                color: a.type === 'VISIT' ? 'green' : a.type === 'CALL' ? 'blue' : 'gray',
                children: (
                  <div>
                    <Space size={6}>
                      <Tag>{LEAD_ACTIVITY_LABELS[a.type] ?? a.type}</Tag>
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        {formatDateTime(a.created_at)}
                        {a.created_by_name ? ` · ${a.created_by_name}` : ''}
                      </Typography.Text>
                    </Space>
                    <div style={{ marginTop: 4 }}>{a.content}</div>
                  </div>
                ),
              }))}
            />
          )}
        </Space>
      )}
    </Drawer>
  );
}

function LeadsCrmTab() {
  const { can } = usePermission();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const qc = useQueryClient();
  const { message } = App.useApp();
  const canManage = can(PERM.LEAD_MANAGE);

  const [viewMode, setViewMode] = useState<'kanban' | 'table'>('kanban');
  const [formModal, setFormModal] = useState<{ open: boolean; record: Lead | null }>({
    open: false,
    record: null,
  });
  const [selectedLead, setSelectedLead] = useState<Lead | null>(null);

  const table = useTableQuery<Lead, LeadListParams>({
    queryKey: [leadsApi.key, 'list'],
    queryFn: (p) => leadsApi.list({ ...p, limit: viewMode === 'kanban' ? 100 : p.limit }),
    filterKeys: ['branch_id', 'status', 'source'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const moveStatus = useMutation({
    mutationFn: ({ id, status }: { id: number; status: LeadStatus }) =>
      leadsApi.update(id, { status }),
    onSuccess: () => {
      message.success('Đã chuyển trạng thái khách tiềm năng');
      void qc.invalidateQueries({ queryKey: [leadsApi.key] });
    },
    onError: (err) => showError(err, 'Không thể chuyển trạng thái'),
  });

  const groupedByStatus = useMemo(() => {
    const map: Record<LeadStatus, Lead[]> = {
      NEW: [],
      CONTACTED: [],
      TRIAL: [],
      CONVERTED: [],
      LOST: [],
    };
    for (const item of table.items) {
      const st = item.status in map ? item.status : 'NEW';
      map[st].push(item);
    }
    return map;
  }, [table.items]);

  const columns = useMemo<ColumnsType<Lead>>(
    () => [
      {
        title: 'Khách tiềm năng',
        dataIndex: 'full_name',
        render: (name: string, r) => (
          <Button
            type="link"
            size="small"
            style={{ padding: 0, fontWeight: 600 }}
            onClick={() => setSelectedLead(r)}
          >
            {name}
          </Button>
        ),
      },
      {
        title: 'Số điện thoại',
        dataIndex: 'phone',
        render: (v: string) => formatPhone(v),
      },
      {
        title: 'Nguồn',
        dataIndex: 'source',
        render: (s: LeadSource) => (
          <Tag color={LEAD_SOURCE_COLORS[s] ?? 'default'}>{LEAD_SOURCE_LABELS[s] ?? s}</Tag>
        ),
      },
      {
        title: 'Chi nhánh',
        dataIndex: 'branch_name',
        render: (v: string | undefined, r) => v || `CN #${r.branch_id}`,
      },
      {
        title: 'Hẹn tập thử',
        dataIndex: 'trial_date',
        render: (v: string | null | undefined) => formatDate(v),
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        render: (s: LeadStatus) => <StatusTag status={s} map={LEAD_STATUS_MAP} />,
      },
      {
        title: 'Ngày tạo',
        dataIndex: 'created_at',
        render: (v: string) => formatDateTime(v),
      },
      {
        title: '',
        key: 'actions',
        width: 110,
        render: (_, r) => (
          <Button size="small" type="link" onClick={() => setSelectedLead(r)}>
            Chăm sóc
          </Button>
        ),
      },
    ],
    [],
  );

  return (
    <>
      <Row justify="space-between" align="middle" gutter={[8, 12]} style={{ marginBottom: 12 }}>
        <Col>
          <Segmented
            value={viewMode}
            onChange={(v) => setViewMode(v as 'kanban' | 'table')}
            options={[
              { label: 'Kanban theo trạng thái', value: 'kanban', icon: <AppstoreOutlined /> },
              { label: 'Danh sách (Bảng)', value: 'table', icon: <TableOutlined /> },
            ]}
          />
        </Col>
        <Col>
          <ActionBar
            onAdd={() => setFormModal({ open: true, record: null })}
            addPerm={PERM.LEAD_MANAGE}
            addText="Thêm khách tiềm năng"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên, số điện thoại, email..."
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
            name: 'status',
            placeholder: 'Trạng thái',
            options: LEAD_STATUS_OPTIONS,
            col: 4,
          },
          {
            name: 'source',
            placeholder: 'Nguồn khách',
            options: LEAD_SOURCE_OPTIONS,
            col: 4,
          },
        ]}
      />

      {viewMode === 'table' ? (
        <DataTable<Lead>
          table={table}
          columns={columns}
          emptyText="Chưa có khách tiềm năng nào"
        />
      ) : (
        <Row gutter={[12, 12]}>
          {LEAD_STATUS_ORDER.map((st) => {
            const items = groupedByStatus[st] ?? [];
            const meta = LEAD_STATUS_MAP[st];
            const nextStatus: Partial<Record<LeadStatus, LeadStatus>> = {
              NEW: 'CONTACTED',
              CONTACTED: 'TRIAL',
              TRIAL: 'CONVERTED',
            };
            const next = nextStatus[st];

            return (
              <Col xs={24} md={12} xl={4} style={{ flex: '1 1 210px', minWidth: 220 }} key={st}>
                <Card
                  size="small"
                  style={{ background: '#f8fafc', borderRadius: 10, height: '100%' }}
                  title={
                    <Space>
                      <StatusTag status={st} map={LEAD_STATUS_MAP} />
                      <Badge count={items.length} showZero color={meta?.color || '#64748b'} />
                    </Space>
                  }
                >
                  <Space direction="vertical" size={8} style={{ width: '100%' }}>
                    {items.length === 0 ? (
                      <Typography.Text
                        type="secondary"
                        style={{ display: 'block', textAlign: 'center', padding: '20px 0', fontSize: 12 }}
                      >
                        Trống
                      </Typography.Text>
                    ) : (
                      items.map((lead) => (
                        <Card
                          key={lead.id}
                          size="small"
                          hoverable
                          onClick={() => setSelectedLead(lead)}
                          style={{ borderRadius: 8 }}
                        >
                          <div style={{ fontWeight: 600 }}>{lead.full_name}</div>
                          <div style={{ fontSize: 12, color: '#475569', marginTop: 2 }}>
                            <PhoneOutlined /> {formatPhone(lead.phone)}
                          </div>
                          <Space size={4} wrap style={{ marginTop: 6 }}>
                            <Tag
                              color={LEAD_SOURCE_COLORS[lead.source] ?? 'default'}
                              style={{ margin: 0, fontSize: 11 }}
                            >
                              {LEAD_SOURCE_LABELS[lead.source] ?? lead.source}
                            </Tag>
                            {lead.trial_date && (
                              <Tag color="gold" style={{ margin: 0, fontSize: 11 }}>
                                Hẹn: {formatDate(lead.trial_date)}
                              </Tag>
                            )}
                          </Space>
                          {canManage && next && (
                            <div
                              style={{ marginTop: 8, textAlign: 'right' }}
                              onClick={(e) => e.stopPropagation()}
                            >
                              <Button
                                size="small"
                                type="link"
                                icon={<SwapRightOutlined />}
                                style={{ padding: 0, fontSize: 12 }}
                                onClick={() =>
                                  moveStatus.mutate({ id: lead.id, status: next })
                                }
                              >
                                Chuyển sang {LEAD_STATUS_MAP[next]?.label}
                              </Button>
                            </div>
                          )}
                        </Card>
                      ))
                    )}
                  </Space>
                </Card>
              </Col>
            );
          })}
        </Row>
      )}

      <LeadFormModal
        open={formModal.open}
        record={formModal.record}
        onClose={() => setFormModal({ open: false, record: null })}
      />

      <LeadDetailDrawer
        lead={selectedLead}
        canManage={canManage}
        onClose={() => setSelectedLead(null)}
        onEdit={(l) => {
          setSelectedLead(null);
          setFormModal({ open: true, record: l });
        }}
      />
    </>
  );
}

function AtRiskMembersTab() {
  const { branchId } = useWorkingBranch();

  const query = useQuery({
    queryKey: [leadsApi.atRiskKey, branchId],
    queryFn: () => leadsApi.atRiskMembers({ branch_id: branchId ?? undefined, days: 14 }),
  });

  const items = query.data?.items ?? [];

  const columns: ColumnsType<AtRiskMember> = [
    {
      title: 'Hội viên',
      key: 'member',
      render: (_, r) => {
        const mid = r.member_id ?? r.id;
        return (
          <Space direction="vertical" size={0}>
            <Link to={`/app/members/${mid}`} style={{ fontWeight: 600 }}>
              {r.full_name}
            </Link>
            <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
              {r.member_code}
            </Typography.Text>
          </Space>
        );
      },
    },
    {
      title: 'Số điện thoại',
      dataIndex: 'phone',
      render: (v: string) => formatPhone(v),
    },
    {
      title: 'Gói hiện tại',
      dataIndex: 'plan_name',
      render: (v: string | undefined, r) => (
        <Space direction="vertical" size={0}>
          <span>{v || 'Gói tập đang hoạt động'}</span>
          {(r.subscription_end_date || r.end_date) && (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              Hết hạn: {formatDate(r.subscription_end_date || r.end_date)}
            </Typography.Text>
          )}
        </Space>
      ),
    },
    {
      title: 'Check-in gần nhất',
      dataIndex: 'last_checkin_at',
      render: (v: string | null | undefined) => (v ? formatDateTime(v) : 'Chưa từng check-in'),
    },
    {
      title: 'Số ngày vắng mặt',
      key: 'absent',
      align: 'right',
      render: (_, r) => {
        const days = r.days_since_last_checkin ?? r.absent_days ?? r.days_inactive ?? 14;
        return <Tag color="red">{days} ngày chưa đi tập</Tag>;
      },
    },
    {
      title: 'Chi nhánh',
      dataIndex: 'branch_name',
      render: (v: string | undefined) => v || '—',
    },
    {
      title: '',
      key: 'action',
      align: 'right',
      render: (_, r) => {
        const mid = r.member_id ?? r.id;
        return (
          <Link to={`/app/members/${mid}`}>
            <Button size="small">Xem hồ sơ</Button>
          </Link>
        );
      },
    },
  ];

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Alert
        type="warning"
        showIcon
        icon={<WarningOutlined />}
        message="Danh sách hội viên có gói đang ACTIVE nhưng từ 14 ngày trở lên chưa đến phòng tập"
        description="Lễ tân / CSKH chủ động gọi điện hỏi thăm để giữ chân hội viên và tăng tỷ lệ gia hạn."
      />
      <Table<AtRiskMember>
        rowKey={(r) => r.member_id ?? r.id}
        loading={query.isLoading}
        dataSource={items}
        columns={columns}
        locale={{ emptyText: 'Tuyệt vời! Không có hội viên nào vắng mặt quá 14 ngày' }}
      />
    </Space>
  );
}

function LeadFunnelReportTab() {
  const { branchId } = useWorkingBranch();

  const query = useQuery({
    queryKey: [leadsApi.key, 'report', branchId],
    queryFn: () => leadsApi.funnelReport({ branch_id: branchId ?? undefined }),
  });

  const data = query.data;
  const byStatus = data?.by_status ?? [];
  const bySource = data?.by_source ?? [];
  const total =
    data?.total ?? byStatus.reduce((acc, x) => acc + (Number(x.count) || 0), 0);
  const converted =
    data?.converted ??
    (byStatus.find((x) => (x.status ?? x.key) === 'CONVERTED')?.count ?? 0);
  const rate =
    data?.conversion_rate ?? (total > 0 ? Number(((converted / total) * 100).toFixed(1)) : 0);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row gutter={[12, 12]}>
        <Col xs={12} md={8}>
          <StatCard
            label="Tổng khách tiềm năng"
            value={total}
            caption="Toàn phễu CRM"
            icon={<TeamOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={12} md={8}>
          <StatCard
            label="Đã chuyển đổi thành Hội viên"
            value={converted}
            caption="Trạng thái CONVERTED"
            icon={<CheckCircleOutlined />}
            loading={query.isLoading}
          />
        </Col>
        <Col xs={24} md={8}>
          <StatCard
            label="Tỷ lệ chuyển đổi"
            value={`${rate}%`}
            caption="Hội viên mới / Tổng Lead"
            icon={<FilterOutlined />}
            loading={query.isLoading}
          />
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card size="small" title="Phễu chuyển đổi theo trạng thái" loading={query.isLoading}>
            {byStatus.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Chưa có dữ liệu" />
            ) : (
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                {byStatus.map((row) => {
                  const key = (row.status ?? row.key ?? 'NEW') as LeadStatus;
                  const pct =
                    row.pct ?? (total > 0 ? Math.round((row.count / total) * 100) : 0);
                  return (
                    <div key={key}>
                      <Row justify="space-between">
                        <Col>
                          <StatusTag status={key} map={LEAD_STATUS_MAP} />
                        </Col>
                        <Col>
                          <b>{formatNumber(row.count)}</b> khách ({pct}%)
                        </Col>
                      </Row>
                      <Progress percent={pct} showInfo={false} size="small" />
                    </div>
                  );
                })}
              </Space>
            )}
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card size="small" title="Hiệu quả theo nguồn khách (Source)" loading={query.isLoading}>
            <Table
              size="small"
              rowKey={(r) => String(r.source ?? r.key)}
              pagination={false}
              dataSource={bySource}
              locale={{ emptyText: 'Chưa có dữ liệu theo nguồn' }}
              columns={[
                {
                  title: 'Nguồn khách',
                  key: 'source',
                  render: (_, r) => {
                    const src = (r.source ?? r.key ?? 'OTHER') as LeadSource;
                    return (
                      <Tag color={LEAD_SOURCE_COLORS[src] ?? 'default'}>
                        {LEAD_SOURCE_LABELS[src] ?? r.label ?? src}
                      </Tag>
                    );
                  },
                },
                {
                  title: 'Số lượng Lead',
                  dataIndex: 'count',
                  align: 'right',
                  render: (v: number) => <b>{formatNumber(v)}</b>,
                },
                {
                  title: 'Đã chuyển đổi',
                  dataIndex: 'converted',
                  align: 'right',
                  render: (v: number | undefined) => formatNumber(v ?? 0),
                },
                {
                  title: 'Tỷ lệ',
                  key: 'rate',
                  align: 'right',
                  render: (_, r) => {
                    const cr =
                      r.conversion_rate ??
                      r.pct ??
                      (r.count > 0 && r.converted != null
                        ? Number(((r.converted / r.count) * 100).toFixed(1))
                        : 0);
                    return `${cr}%`;
                  },
                },
              ]}
            />
          </Card>
        </Col>
      </Row>
    </Space>
  );
}

/** Trang CRM Khách tiềm năng & Chăm sóc hội viên (`/app/leads`). */
export default function LeadsPage() {
  return (
    <Card variant="borderless" className="page-card">
      <div className="page-head">
        <PageTitle
          title="Khách tiềm năng & CSKH"
          subtitle="Quản lý phễu khách tiềm năng (CRM), cấp vé tập thử QR và chăm sóc hội viên có nguy cơ nghỉ tập"
        />
      </div>

      <Tabs
        defaultActiveKey="crm"
        items={[
          {
            key: 'crm',
            label: 'Khách tiềm năng (CRM)',
            children: <LeadsCrmTab />,
          },
          {
            key: 'at-risk',
            label: 'Hội viên cần chăm sóc (At-Risk)',
            children: <AtRiskMembersTab />,
          },
          {
            key: 'funnel',
            label: 'Báo cáo phễu chuyển đổi',
            children: <LeadFunnelReportTab />,
          },
        ]}
      />
    </Card>
  );
}
