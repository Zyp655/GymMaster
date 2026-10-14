import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
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
  CrownOutlined,
  DeleteOutlined,
  EditOutlined,
  FireOutlined,
  GiftOutlined,
  PlusCircleOutlined,
  SafetyCertificateOutlined,
  StarOutlined,
  TrophyOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import {
  CHALLENGE_METRIC_LABELS,
  CHALLENGE_METRIC_OPTIONS,
  LOYALTY_REASON_LABELS,
  loyaltyApi,
  REWARD_TYPE_LABELS,
  REWARD_TYPE_OPTIONS,
} from '@/api/loyalty-engagement';
import type {
  BadgeInput,
  BadgeItem,
  Challenge,
  ChallengeInput,
  ChallengeMetric,
  ChallengeStatus,
  LoyaltyLedger,
  LoyaltyReason,
  LoyaltyReward,
  LoyaltyRewardInput,
  RewardType,
} from '@/api/loyalty-engagement';
import ActionBar from '@/components/ActionBar';
import BranchSelect from '@/components/BranchSelect';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MemberSelect from '@/components/MemberSelect';
import PageTitle from '@/components/PageTitle';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { ACTIVE_STATUS_OPTIONS } from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import { DATE_FORMAT, formatDate, formatDateTime, formatNumber, toApiDate } from '@/utils/format';

/* --------------------------------------------------------------------------
 * TAB 1: Quà đổi điểm (/admin/loyalty-rewards)
 * -------------------------------------------------------------------------- */
function RewardsTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const [modal, setModal] = useState<{ open: boolean; record: LoyaltyReward | null }>({
    open: false,
    record: null,
  });
  const [form] = Form.useForm<LoyaltyRewardInput>();

  const table = useTableQuery<
    LoyaltyReward,
    { status?: string; reward_type?: RewardType; page?: number; limit?: number }
  >({
    queryKey: [loyaltyApi.rewardKey, 'list'],
    queryFn: (p) => loyaltyApi.listRewards(p),
    filterKeys: ['reward_type', 'status'],
    defaults: { order_by: 'cost_points', order_dir: 'ASC' },
  });

  useEffect(() => {
    if (!modal.open) return;
    form.resetFields();
    if (modal.record) {
      form.setFieldsValue(modal.record);
    } else {
      form.setFieldsValue({
        reward_type: 'VOUCHER',
        cost_points: 200,
        discount_pct: 10,
        stock: 100,
        status: 'ACTIVE',
      });
    }
  }, [modal, form]);

  const saveMut = useMutation({
    mutationFn: (v: LoyaltyRewardInput) => {
      const payload: LoyaltyRewardInput = {
        name: v.name.trim(),
        description: v.description?.trim() || '',
        cost_points: Number(v.cost_points) || 100,
        reward_type: v.reward_type,
        discount_pct: v.reward_type === 'VOUCHER' ? Number(v.discount_pct) || 10 : 0,
        stock: Number(v.stock) || 0,
        status: v.status ?? 'ACTIVE',
      };
      return modal.record
        ? loyaltyApi.updateReward(modal.record.id, payload)
        : loyaltyApi.createReward(payload);
    },
    onSuccess: () => {
      message.success(modal.record ? 'Đã cập nhật quà đổi điểm' : 'Đã thêm quà đổi điểm mới');
      void qc.invalidateQueries({ queryKey: [loyaltyApi.rewardKey] });
      setModal({ open: false, record: null });
    },
    onError: (err) => showError(err, 'Không thể lưu quà đổi điểm'),
  });

  const deleteMut = useMutation({
    mutationFn: (id: number) => loyaltyApi.deleteReward(id),
    onSuccess: () => {
      message.success('Đã xóa quà đổi điểm');
      void qc.invalidateQueries({ queryKey: [loyaltyApi.rewardKey] });
    },
    onError: (err) => showError(err, 'Không thể xóa quà đổi điểm'),
  });

  const columns = useMemo<ColumnsType<LoyaltyReward>>(
    () => [
      {
        title: 'Tên phần quà',
        dataIndex: 'name',
        render: (name: string, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>{name}</Typography.Text>
            {r.description && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {r.description}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Loại quà',
        dataIndex: 'reward_type',
        render: (t: RewardType, r) => (
          <Space>
            <Tag color={t === 'VOUCHER' ? 'purple' : 'cyan'}>
              {REWARD_TYPE_LABELS[t] ?? t}
            </Tag>
            {t === 'VOUCHER' && r.discount_pct > 0 && <Tag color="gold">-{r.discount_pct}%</Tag>}
          </Space>
        ),
      },
      {
        title: 'Điểm quy đổi',
        dataIndex: 'cost_points',
        align: 'right',
        render: (pts: number) => (
          <Tag color="orange" style={{ fontWeight: 700, fontSize: 13 }}>
            {formatNumber(pts)} điểm
          </Tag>
        ),
      },
      {
        title: 'Số lượng còn lại',
        dataIndex: 'stock',
        align: 'right',
        render: (v: number) => formatNumber(v),
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        render: (s: string) => <StatusTag status={s} />,
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 100,
        render: (_, r) =>
          canManage ? (
            <Space size={4}>
              <Tooltip title="Sửa">
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => setModal({ open: true, record: r })}
                />
              </Tooltip>
              <ConfirmButton
                perm={PERM.LOYALTY_MANAGE}
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
                confirmTitle={`Xóa quà "${r.name}"?`}
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
            addPerm={PERM.LOYALTY_MANAGE}
            addText="Thêm quà đổi điểm"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên quà đổi điểm..."
        colTextSearch={6}
        filters={[
          {
            name: 'reward_type',
            placeholder: 'Loại quà thưởng',
            options: REWARD_TYPE_OPTIONS,
            col: 5,
          },
          {
            name: 'status',
            placeholder: 'Trạng thái',
            options: ACTIVE_STATUS_OPTIONS,
            col: 4,
          },
        ]}
      />

      <DataTable<LoyaltyReward>
        table={table}
        columns={columns}
        emptyText="Chưa có phần quà đổi điểm nào"
      />

      <Modal
        open={modal.open}
        title={modal.record ? `Sửa quà: ${modal.record.name}` : 'Thêm quà đổi điểm mới'}
        okText="Lưu phần quà"
        cancelText="Hủy"
        onOk={() => form.submit()}
        onCancel={() => setModal({ open: false, record: null })}
        confirmLoading={saveMut.isPending}
        width={540}
        forceRender
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(v) => saveMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Form.Item
            name="name"
            label="Tên phần quà đổi điểm"
            rules={[{ required: true, whitespace: true, message: 'Vui lòng nhập tên quà' }]}
          >
            <Input placeholder="VD: Voucher giảm 15% khi gia hạn gói tập" />
          </Form.Item>
          <Row gutter={12}>
            <Col xs={12}>
              <Form.Item name="reward_type" label="Loại phần thưởng" rules={[{ required: true }]}>
                <Select options={REWARD_TYPE_OPTIONS} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item
                name="cost_points"
                label="Số điểm cần đổi"
                rules={[{ required: true, message: 'Nhập điểm quy đổi' }]}
              >
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="discount_pct" label="Mức giảm giá Voucher (%)">
                <InputNumber min={0} max={100} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="stock" label="Số lượng phát hành" rules={[{ required: true }]}>
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="status" label="Trạng thái">
                <Select options={ACTIVE_STATUS_OPTIONS} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="description" label="Mô tả điều kiện áp dụng">
            <Input.TextArea rows={2} placeholder="Mô tả chi tiết ưu đãi..." />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

/* --------------------------------------------------------------------------
 * TAB 2: Thử thách phòng tập (/admin/challenges)
 * -------------------------------------------------------------------------- */
interface ChallengeFormValues {
  branch_id?: number | null;
  name: string;
  description?: string;
  metric: ChallengeMetric;
  target: number;
  date_range: [Dayjs, Dayjs];
  reward_points: number;
  status: ChallengeStatus;
}

function ChallengesTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();

  const [modal, setModal] = useState<{ open: boolean; record: Challenge | null }>({
    open: false,
    record: null,
  });
  const [form] = Form.useForm<ChallengeFormValues>();

  const table = useTableQuery<
    Challenge,
    { branch_id?: number; metric?: ChallengeMetric; status?: ChallengeStatus; page?: number; limit?: number }
  >({
    queryKey: [loyaltyApi.challengeKey, 'list'],
    queryFn: (p) => loyaltyApi.listChallenges(p),
    filterKeys: ['branch_id', 'metric', 'status'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  useEffect(() => {
    if (!modal.open) return;
    form.resetFields();
    if (modal.record) {
      form.setFieldsValue({
        branch_id: modal.record.branch_id ?? undefined,
        name: modal.record.name,
        description: modal.record.description,
        metric: modal.record.metric,
        target: modal.record.target,
        date_range: [dayjs(modal.record.start_date), dayjs(modal.record.end_date)],
        reward_points: modal.record.reward_points,
        status: modal.record.status,
      });
    } else {
      form.setFieldsValue({
        branch_id: branchId ?? undefined,
        metric: 'CHECKINS',
        target: 20,
        date_range: [dayjs(), dayjs().add(30, 'day')],
        reward_points: 300,
        status: 'ACTIVE',
      });
    }
  }, [modal, form, branchId]);

  const saveMut = useMutation({
    mutationFn: (v: ChallengeFormValues) => {
      const payload: ChallengeInput = {
        branch_id: v.branch_id ?? null,
        name: v.name.trim(),
        description: v.description?.trim() || '',
        metric: v.metric,
        target: Number(v.target) || 10,
        start_date: toApiDate(v.date_range?.[0]) ?? dayjs().format('YYYY-MM-DD'),
        end_date: toApiDate(v.date_range?.[1]) ?? dayjs().add(30, 'day').format('YYYY-MM-DD'),
        reward_points: Number(v.reward_points) || 100,
        status: v.status,
      };
      return modal.record
        ? loyaltyApi.updateChallenge(modal.record.id, payload)
        : loyaltyApi.createChallenge(payload);
    },
    onSuccess: () => {
      message.success(modal.record ? 'Đã cập nhật thử thách' : 'Đã tạo thử thách phòng tập mới');
      void qc.invalidateQueries({ queryKey: [loyaltyApi.challengeKey] });
      setModal({ open: false, record: null });
    },
    onError: (err) => showError(err, 'Không thể lưu thử thách'),
  });

  const deleteMut = useMutation({
    mutationFn: (id: number) => loyaltyApi.deleteChallenge(id),
    onSuccess: () => {
      message.success('Đã xóa thử thách');
      void qc.invalidateQueries({ queryKey: [loyaltyApi.challengeKey] });
    },
    onError: (err) => showError(err, 'Không thể xóa thử thách'),
  });

  const columns = useMemo<ColumnsType<Challenge>>(
    () => [
      {
        title: 'Tên thử thách',
        dataIndex: 'name',
        render: (name: string, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>{name}</Typography.Text>
            {r.description && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {r.description}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Tiêu chí & Mục tiêu',
        key: 'metric',
        render: (_, r) => (
          <Space>
            <Tag color="blue">{CHALLENGE_METRIC_LABELS[r.metric] ?? r.metric}</Tag>
            <Typography.Text strong>Mục tiêu: {r.target}</Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Thời gian diễn ra',
        key: 'dates',
        render: (_, r) => `${formatDate(r.start_date)} → ${formatDate(r.end_date)}`,
      },
      {
        title: 'Điểm thưởng',
        dataIndex: 'reward_points',
        align: 'right',
        render: (pts: number) => <Tag color="gold">+{formatNumber(pts)} điểm</Tag>,
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        render: (s: string) => <StatusTag status={s} />,
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
                perm={PERM.LOYALTY_MANAGE}
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
                confirmTitle={`Xóa thử thách "${r.name}"?`}
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
            addPerm={PERM.LOYALTY_MANAGE}
            addText="Tạo thử thách mới"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên thử thách..."
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
            name: 'metric',
            placeholder: 'Tiêu chí thử thách',
            options: CHALLENGE_METRIC_OPTIONS,
            col: 5,
          },
        ]}
      />

      <DataTable<Challenge>
        table={table}
        columns={columns}
        emptyText="Chưa có chương trình thử thách phòng tập nào"
      />

      <Modal
        open={modal.open}
        title={modal.record ? `Sửa thử thách: ${modal.record.name}` : 'Tạo thử thách phòng tập'}
        okText="Lưu thử thách"
        cancelText="Hủy"
        onOk={() => form.submit()}
        onCancel={() => setModal({ open: false, record: null })}
        confirmLoading={saveMut.isPending}
        width={560}
        forceRender
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(v) => saveMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Form.Item
            name="name"
            label="Tên chương trình thử thách"
            rules={[{ required: true, whitespace: true, message: 'Nhập tên thử thách' }]}
          >
            <Input placeholder="VD: Thử thách 20 buổi tập tháng 10" />
          </Form.Item>
          <Row gutter={12}>
            <Col xs={12}>
              <Form.Item name="branch_id" label="Chi nhánh áp dụng (Để trống = Toàn hệ thống)">
                <BranchSelect activeOnly allowClear />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="metric" label="Tiêu chí đo lường" rules={[{ required: true }]}>
                <Select options={CHALLENGE_METRIC_OPTIONS} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="target" label="Mục tiêu cần đạt (buổi/lượt)" rules={[{ required: true }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="reward_points" label="Điểm thưởng hoàn thành" rules={[{ required: true }]}>
                <InputNumber min={10} step={50} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={16}>
              <Form.Item
                name="date_range"
                label="Thời gian diễn ra"
                rules={[{ required: true, message: 'Chọn khoảng thời gian' }]}
              >
                <DatePicker.RangePicker style={{ width: '100%' }} format={DATE_FORMAT} />
              </Form.Item>
            </Col>
            <Col xs={8}>
              <Form.Item name="status" label="Trạng thái">
                <Select
                  options={[
                    { value: 'ACTIVE', label: 'Đang diễn ra' },
                    { value: 'DRAFT', label: 'Nháp' },
                    { value: 'ENDED', label: 'Đã kết thúc' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="description" label="Mô tả thể lệ">
            <Input.TextArea rows={2} placeholder="Mô tả chi tiết thể lệ và giải thưởng..." />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

/* --------------------------------------------------------------------------
 * TAB 3: Huy hiệu thành tích (/admin/badges)
 * -------------------------------------------------------------------------- */
function BadgesTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const [modal, setModal] = useState<{ open: boolean; record: BadgeItem | null }>({
    open: false,
    record: null,
  });
  const [form] = Form.useForm<{
    code: string;
    name: string;
    description?: string;
    icon: string;
    metric: ChallengeMetric;
    target: number;
  }>();

  const table = useTableQuery<BadgeItem, { page?: number; limit?: number }>({
    queryKey: [loyaltyApi.badgeKey, 'list'],
    queryFn: (p) => loyaltyApi.listBadges(p),
    defaults: { order_by: 'id', order_dir: 'ASC' },
  });

  useEffect(() => {
    if (!modal.open) return;
    form.resetFields();
    if (modal.record) {
      form.setFieldsValue({
        code: modal.record.code,
        name: modal.record.name,
        description: modal.record.description,
        icon: modal.record.icon || 'trophy',
        metric: (modal.record.rule?.metric as ChallengeMetric) || 'CHECKINS',
        target: Number(modal.record.rule?.target) || 10,
      });
    } else {
      form.setFieldsValue({
        icon: 'trophy',
        metric: 'CHECKINS',
        target: 30,
      });
    }
  }, [modal, form]);

  const saveMut = useMutation({
    mutationFn: (v: {
      code: string;
      name: string;
      description?: string;
      icon: string;
      metric: ChallengeMetric;
      target: number;
    }) => {
      const payload: BadgeInput = {
        code: v.code.trim().toUpperCase(),
        name: v.name.trim(),
        description: v.description?.trim() || '',
        icon: v.icon,
        rule: {
          metric: v.metric,
          target: Number(v.target) || 10,
        },
      };
      return modal.record
        ? loyaltyApi.updateBadge(modal.record.id, payload)
        : loyaltyApi.createBadge(payload);
    },
    onSuccess: () => {
      message.success(modal.record ? 'Đã cập nhật huy hiệu' : 'Đã tạo huy hiệu mới');
      void qc.invalidateQueries({ queryKey: [loyaltyApi.badgeKey] });
      setModal({ open: false, record: null });
    },
    onError: (err) => showError(err, 'Không thể lưu huy hiệu'),
  });

  const deleteMut = useMutation({
    mutationFn: (id: number) => loyaltyApi.deleteBadge(id),
    onSuccess: () => {
      message.success('Đã xóa huy hiệu');
      void qc.invalidateQueries({ queryKey: [loyaltyApi.badgeKey] });
    },
    onError: (err) => showError(err, 'Không thể xóa huy hiệu'),
  });

  const renderBadgeIcon = (icon: string) => {
    if (icon === 'fire') return <FireOutlined style={{ color: '#ea580c', fontSize: 18 }} />;
    if (icon === 'crown') return <CrownOutlined style={{ color: '#eab308', fontSize: 18 }} />;
    if (icon === 'star') return <StarOutlined style={{ color: '#2563eb', fontSize: 18 }} />;
    return <TrophyOutlined style={{ color: '#d97706', fontSize: 18 }} />;
  };

  const columns = useMemo<ColumnsType<BadgeItem>>(
    () => [
      {
        title: 'Mã huy hiệu',
        dataIndex: 'code',
        width: 150,
        render: (c: string) => <Tag className="code-text">{c}</Tag>,
      },
      {
        title: 'Tên huy hiệu',
        dataIndex: 'name',
        render: (name: string, r) => (
          <Space>
            {renderBadgeIcon(r.icon)}
            <Typography.Text strong>{name}</Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Mô tả thành tích',
        dataIndex: 'description',
      },
      {
        title: 'Điều kiện tự động xét duyệt',
        key: 'rule',
        render: (_, r) => {
          const m = (r.rule?.metric as ChallengeMetric) || 'CHECKINS';
          const t = r.rule?.target ?? 0;
          return (
            <Tag color="geekblue">
              {CHALLENGE_METRIC_LABELS[m] ?? m} ≥ {String(t)}
            </Tag>
          );
        },
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
                perm={PERM.LOYALTY_MANAGE}
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
                confirmTitle={`Xóa huy hiệu "${r.name}"?`}
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
            addPerm={PERM.LOYALTY_MANAGE}
            addText="Thêm huy hiệu"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <DataTable<BadgeItem>
        table={table}
        columns={columns}
        emptyText="Chưa có huy hiệu thành tích nào"
      />

      <Modal
        open={modal.open}
        title={modal.record ? `Sửa huy hiệu: ${modal.record.name}` : 'Thêm huy hiệu thành tích'}
        okText="Lưu huy hiệu"
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
            <Col xs={10}>
              <Form.Item
                name="code"
                label="Mã huy hiệu"
                rules={[{ required: true, whitespace: true, message: 'Nhập mã' }]}
              >
                <Input placeholder="VD: STREAK_60" disabled={!!modal.record} />
              </Form.Item>
            </Col>
            <Col xs={14}>
              <Form.Item
                name="name"
                label="Tên huy hiệu"
                rules={[{ required: true, whitespace: true, message: 'Nhập tên huy hiệu' }]}
              >
                <Input placeholder="VD: Chiến binh 60 buổi" />
              </Form.Item>
            </Col>
            <Col xs={8}>
              <Form.Item name="icon" label="Biểu tượng">
                <Select
                  options={[
                    { value: 'trophy', label: 'Cúp vàng (Trophy)' },
                    { value: 'fire', label: 'Ngọn lửa (Fire)' },
                    { value: 'crown', label: 'Vương miện (Crown)' },
                    { value: 'star', label: 'Ngôi sao (Star)' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col xs={10}>
              <Form.Item name="metric" label="Tiêu chí xét" rules={[{ required: true }]}>
                <Select options={CHALLENGE_METRIC_OPTIONS} />
              </Form.Item>
            </Col>
            <Col xs={6}>
              <Form.Item name="target" label="Cột mốc" rules={[{ required: true }]}>
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="description" label="Mô tả huy hiệu">
            <Input.TextArea rows={2} placeholder="Mô tả ý nghĩa của huy hiệu..." />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

/* --------------------------------------------------------------------------
 * TAB 4: Điều chỉnh điểm hội viên (POST /admin/members/:id/points/adjust)
 * -------------------------------------------------------------------------- */
function AdjustPointsTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const [form] = Form.useForm<{ member_id: number; points: number; note: string }>();

  const ledgerTable = useTableQuery<LoyaltyLedger, { member_id?: number; page?: number; limit?: number }>({
    queryKey: [loyaltyApi.ledgerKey, 'list'],
    queryFn: (p) => loyaltyApi.listLedgers(p),
    defaults: { order_by: 'id', order_dir: 'DESC' },
  });

  const adjustMut = useMutation({
    mutationFn: (v: { member_id: number; points: number; note: string }) =>
      loyaltyApi.adjustPoints(v.member_id, {
        points: Number(v.points),
        note: v.note.trim(),
      }),
    onSuccess: (res, v) => {
      message.success(
        `Đã ${v.points >= 0 ? 'cộng' : 'trừ'} ${Math.abs(v.points)} điểm thưởng cho hội viên!${
          res?.loyalty_points !== undefined ? ` Số dư mới: ${res.loyalty_points} điểm.` : ''
        }`,
      );
      form.resetFields();
      void qc.invalidateQueries({ queryKey: [loyaltyApi.ledgerKey] });
    },
    onError: (err) => showError(err, 'Không thể điều chỉnh điểm thưởng hội viên'),
  });

  const ledgerColumns = useMemo<ColumnsType<LoyaltyLedger>>(
    () => [
      {
        title: 'Thời gian',
        dataIndex: 'created_at',
        width: 150,
        render: (v: string) => formatDateTime(v),
      },
      {
        title: 'Hội viên',
        key: 'member',
        render: (_, r) => (
          <Space>
            <Typography.Text strong>{r.member_name || `Hội viên #${r.member_id}`}</Typography.Text>
            {r.member_code && <Tag className="code-text">{r.member_code}</Tag>}
          </Space>
        ),
      },
      {
        title: 'Biến động điểm',
        dataIndex: 'points',
        align: 'right',
        render: (pts: number) => (
          <Typography.Text strong type={pts >= 0 ? 'success' : 'danger'}>
            {pts >= 0 ? `+${formatNumber(pts)}` : formatNumber(pts)} điểm
          </Typography.Text>
        ),
      },
      {
        title: 'Nguyên nhân',
        dataIndex: 'reason',
        render: (reason: LoyaltyReason) => (
          <Tag color={reason === 'ADJUST' ? 'gold' : 'blue'}>
            {LOYALTY_REASON_LABELS[reason] ?? reason}
          </Tag>
        ),
      },
      {
        title: 'Ghi chú',
        dataIndex: 'note',
        render: (v: string) => v || '—',
      },
    ],
    [],
  );

  return (
    <Row gutter={[16, 16]}>
      <Col xs={24} lg={9}>
        <Card
          size="small"
          title="Điều chỉnh điểm thưởng hội viên (Cộng / Trừ điểm)"
        >
          {!canManage ? (
            <Alert
              type="warning"
              showIcon
              message="Bạn cần quyền Quản lý điểm thưởng (loyalty.manage) để điều chỉnh điểm."
            />
          ) : (
            <Form
              form={form}
              layout="vertical"
              initialValues={{ points: 100 }}
              onFinish={(v) => adjustMut.mutate(v)}
            >
              <Form.Item
                name="member_id"
                label="Chọn hội viên"
                rules={[{ required: true, message: 'Vui lòng chọn hội viên' }]}
              >
                <MemberSelect />
              </Form.Item>
              <Form.Item
                name="points"
                label="Số điểm điều chỉnh (Nhập số dương để Cộng, số âm để Trừ)"
                rules={[
                  { required: true, message: 'Nhập số điểm' },
                  {
                    validator: (_, val) =>
                      Number(val) !== 0
                        ? Promise.resolve()
                        : Promise.reject(new Error('Số điểm phải khác 0')),
                  },
                ]}
              >
                <InputNumber style={{ width: '100%' }} step={50} placeholder="VD: 100 hoặc -50" />
              </Form.Item>
              <Form.Item
                name="note"
                label="Lý do điều chỉnh điểm"
                rules={[{ required: true, whitespace: true, message: 'Vui lòng nhập lý do' }]}
              >
                <Input.TextArea
                  rows={3}
                  placeholder="VD: Tặng điểm tri ân sinh nhật hội viên / Hoàn điểm đổi quà..."
                />
              </Form.Item>
              <Button
                type="primary"
                htmlType="submit"
                icon={<PlusCircleOutlined />}
                loading={adjustMut.isPending}
                block
              >
                Xác nhận điều chỉnh điểm
              </Button>
            </Form>
          )}
        </Card>
      </Col>
      <Col xs={24} lg={15}>
        <Card size="small" title="Nhật ký biến động điểm thưởng gần đây">
          <DataTable<LoyaltyLedger>
            table={ledgerTable}
            columns={ledgerColumns}
            emptyText="Chưa có nhật ký biến động điểm thưởng"
          />
        </Card>
      </Col>
    </Row>
  );
}

export default function LoyaltyPage() {
  const { can } = usePermission();
  const canManage = can(PERM.LOYALTY_MANAGE);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <PageTitle
        title="Điểm thưởng, Huy hiệu & Thử thách"
        subtitle="Thiết lập danh mục quà đổi điểm, thử thách vận động theo tháng, huy hiệu thành tích và điều chỉnh điểm hội viên"
      />

      <Tabs
        defaultActiveKey="rewards"
        items={[
          {
            key: 'rewards',
            label: (
              <Space size={6}>
                <GiftOutlined />
                <span>Quà đổi điểm</span>
              </Space>
            ),
            children: <RewardsTab canManage={canManage} />,
          },
          {
            key: 'challenges',
            label: (
              <Space size={6}>
                <FireOutlined />
                <span>Thử thách phòng tập</span>
              </Space>
            ),
            children: <ChallengesTab canManage={canManage} />,
          },
          {
            key: 'badges',
            label: (
              <Space size={6}>
                <SafetyCertificateOutlined />
                <span>Huy hiệu</span>
              </Space>
            ),
            children: <BadgesTab canManage={canManage} />,
          },
          {
            key: 'adjust',
            label: (
              <Space size={6}>
                <PlusCircleOutlined />
                <span>Điều chỉnh điểm hội viên</span>
              </Space>
            ),
            children: <AdjustPointsTab canManage={canManage} />,
          },
        ]}
      />
    </Space>
  );
}
