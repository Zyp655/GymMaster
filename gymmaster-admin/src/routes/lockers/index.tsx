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
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Radio,
  Row,
  Select,
  Space,
  Spin,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  AppstoreOutlined,
  BankOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  InboxOutlined,
  LockOutlined,
  PlusOutlined,
  ReloadOutlined,
  TableOutlined,
  ToolOutlined,
  UnlockOutlined,
  UserOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import {
  LOCKER_RENTAL_TYPE_OPTIONS,
  LOCKER_SIZE_LABELS,
  LOCKER_SIZE_OPTIONS,
  lockersApi,
} from '@/api/operations-ext';
import type {
  Locker,
  LockerAssignInput,
  LockerBoardZone,
  LockerInput,
  LockerListParams,
  LockerRentalType,
  LockerSize,
  LockerStatus,
} from '@/api/operations-ext';
import ActionBar from '@/components/ActionBar';
import BranchSelect from '@/components/BranchSelect';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MemberSelect from '@/components/MemberSelect';
import MoneyInput from '@/components/Money/MoneyInput';
import MoneyText from '@/components/Money/MoneyText';
import PageTitle from '@/components/PageTitle';
import StatCard from '@/components/StatCard';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import {
  LOCKER_STATUS_MAP,
  LOCKER_STATUS_OPTIONS,
  COUNTER_PAYMENT_METHOD_OPTIONS,
} from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import { DATE_FORMAT, formatDate, formatDateTime, formatMoney, toApiDate } from '@/utils/format';
import { groupLockersByBranch } from './board';
import './lockers.css';

const LOCKER_VISUAL_STYLES: Record<
  LockerStatus,
  { bg: string; border: string; text: string; badge: string }
> = {
  // Biến CSS định nghĩa ở lockers.css (có biến thể chế độ tối).
  AVAILABLE: {
    bg: 'var(--locker-available-bg)',
    border: 'var(--locker-available-border)',
    text: 'var(--locker-available-text)',
    badge: '#16a34a',
  },
  OCCUPIED: {
    bg: 'var(--locker-occupied-bg)',
    border: 'var(--locker-occupied-border)',
    text: 'var(--locker-occupied-text)',
    badge: '#ea580c',
  },
  RENTED: {
    bg: 'var(--locker-rented-bg)',
    border: 'var(--locker-rented-border)',
    text: 'var(--locker-rented-text)',
    badge: '#2563eb',
  },
  BROKEN: {
    bg: 'var(--locker-broken-bg)',
    border: 'var(--locker-broken-border)',
    text: 'var(--locker-broken-text)',
    badge: '#dc2626',
  },
};

function LockerFormModal({
  open,
  record,
  readOnly,
  onClose,
}: {
  open: boolean;
  record: Locker | null;
  readOnly: boolean;
  onClose: () => void;
}) {
  const [form] = Form.useForm<LockerInput>();
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const isEdit = !!record;

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    if (record) {
      form.setFieldsValue({
        branch_id: record.branch_id,
        code: record.code,
        zone: record.zone,
        size: record.size,
        monthly_fee: record.monthly_fee,
        status: record.status,
        note: record.note,
      });
    } else {
      form.setFieldsValue({
        branch_id: branchId ?? undefined,
        zone: 'Khu A - Nam',
        size: 'STANDARD',
        monthly_fee: 150000,
        status: 'AVAILABLE',
      });
    }
  }, [open, record, form, branchId]);

  const save = useMutation({
    mutationFn: (v: LockerInput) => {
      const payload: LockerInput = {
        branch_id: v.branch_id,
        code: v.code.trim().toUpperCase(),
        zone: v.zone.trim(),
        size: v.size,
        monthly_fee: Number(v.monthly_fee) || 0,
        status: v.status,
        note: v.note?.trim() || '',
      };
      return record ? lockersApi.update(record.id, payload) : lockersApi.create(payload);
    },
    onSuccess: () => {
      message.success(isEdit ? 'Đã cập nhật thông tin tủ đồ' : 'Đã thêm tủ đồ mới');
      void qc.invalidateQueries({ queryKey: [lockersApi.key] });
      onClose();
    },
    onError: (err) => showError(err, 'Không thể lưu tủ đồ'),
  });

  return (
    <Modal
      open={open}
      title={isEdit ? `Sửa tủ đồ: ${record?.code}` : 'Thêm tủ đồ mới'}
      okText={isEdit ? 'Lưu thay đổi' : 'Thêm tủ'}
      cancelText={readOnly ? 'Đóng' : 'Hủy'}
      okButtonProps={{ style: readOnly ? { display: 'none' } : undefined }}
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={save.isPending}
      width={540}
      forceRender
    >
      <Form<LockerInput>
        form={form}
        layout="vertical"
        disabled={readOnly}
        onFinish={(v) => save.mutate(v)}
        style={{ marginTop: 12 }}
      >
        <Row gutter={12}>
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
            <Form.Item
              name="code"
              label="Mã số tủ"
              rules={[{ required: true, whitespace: true, message: 'Nhập mã tủ (VD: A-01)' }]}
            >
              <Input placeholder="VD: A-01" maxLength={30} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item
              name="zone"
              label="Khu vực đặt tủ"
              rules={[{ required: true, whitespace: true, message: 'Nhập khu vực (VD: Khu A)' }]}
            >
              <Input placeholder="VD: Khu A - Nam" maxLength={50} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="size" label="Kích cỡ tủ" rules={[{ required: true }]}>
              <Select options={LOCKER_SIZE_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="monthly_fee" label="Giá thuê tháng (VND)">
              <MoneyInput />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12}>
            <Form.Item name="status" label="Trạng thái" rules={[{ required: true }]}>
              <Select options={LOCKER_STATUS_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={24}>
            <Form.Item name="note" label="Ghi chú">
              <Input.TextArea rows={2} maxLength={255} placeholder="Ghi chú khóa từ, ổ khóa..." />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}

interface AssignFormValues {
  member_id: number;
  type: LockerRentalType;
  months: number;
  end_at?: Dayjs | null;
  price?: number;
  payment_method?: string;
  note?: string;
}

function AssignLockerModal({
  open,
  locker,
  onClose,
}: {
  open: boolean;
  locker: Locker | null;
  onClose: () => void;
}) {
  const [form] = Form.useForm<AssignFormValues>();
  const { message } = App.useApp();
  const qc = useQueryClient();
  const rentalType = Form.useWatch('type', form) ?? 'DAILY';
  const months = Form.useWatch('months', form) ?? 1;

  useEffect(() => {
    if (!open || !locker) return;
    form.resetFields();
    form.setFieldsValue({
      type: 'DAILY',
      months: 1,
      end_at: dayjs().add(1, 'month'),
      price: 0,
      payment_method: 'CASH',
    });
  }, [open, locker, form]);

  useEffect(() => {
    if (!open || !locker) return;
    if (rentalType === 'MONTHLY') {
      const m = Number(months) || 1;
      form.setFieldsValue({
        price: (locker.monthly_fee ?? 150000) * m,
        end_at: dayjs().add(m, 'month'),
      });
    } else {
      form.setFieldsValue({
        price: 0,
        end_at: dayjs().endOf('day'),
      });
    }
  }, [rentalType, months, locker, open, form]);

  const assignMut = useMutation({
    mutationFn: (v: AssignFormValues) => {
      const payload: LockerAssignInput = {
        member_id: v.member_id,
        type: v.type,
        months: v.type === 'MONTHLY' ? Number(v.months) || 1 : 1,
        end_at: toApiDate(v.end_at) ?? null,
        price: Number(v.price) || 0,
        payment_method: v.payment_method,
        note: v.note?.trim() || '',
      };
      return lockersApi.assign(locker!.id, payload);
    },
    onSuccess: (_, v) => {
      message.success(
        v.type === 'MONTHLY'
          ? `Đã cho thuê tủ ${locker?.code} theo tháng`
          : `Đã gán tủ ${locker?.code} theo lượt cho hội viên`,
      );
      void qc.invalidateQueries({ queryKey: [lockersApi.key] });
      onClose();
    },
    onError: (err) => showError(err, 'Không thể gán / cho thuê tủ đồ'),
  });

  return (
    <Modal
      open={open}
      title={`Gán tủ theo lượt / Cho thuê tủ tháng – Tủ ${locker?.code ?? ''} (${[locker?.zone, locker?.branch_name].filter(Boolean).join(' · ')})`}
      okText={rentalType === 'MONTHLY' ? 'Xác nhận cho thuê tháng' : 'Gán tủ theo lượt'}
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={assignMut.isPending}
      width={560}
      forceRender
    >
      <Form<AssignFormValues>
        form={form}
        layout="vertical"
        onFinish={(v) => assignMut.mutate(v)}
        style={{ marginTop: 12 }}
      >
        <Form.Item name="type" label="Hình thức sử dụng tủ" rules={[{ required: true }]}>
          <Radio.Group
            optionType="button"
            buttonStyle="solid"
            options={LOCKER_RENTAL_TYPE_OPTIONS}
          />
        </Form.Item>

        <Form.Item
          name="member_id"
          label="Hội viên sử dụng"
          rules={[{ required: true, message: 'Vui lòng chọn hội viên' }]}
        >
          <MemberSelect branchId={locker?.branch_id} activeOnly />
        </Form.Item>

        {rentalType === 'MONTHLY' && (
          <Row gutter={12}>
            <Col xs={12}>
              <Form.Item name="months" label="Số tháng thuê" rules={[{ required: true }]}>
                <InputNumber min={1} max={24} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="end_at" label="Ngày kết thúc thuê">
                <DatePicker style={{ width: '100%' }} format={DATE_FORMAT} />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="price" label="Phí thuê tủ (VND)">
                <MoneyInput />
              </Form.Item>
            </Col>
            <Col xs={12}>
              <Form.Item name="payment_method" label="Phương thức thu tiền">
                <Select options={COUNTER_PAYMENT_METHOD_OPTIONS} />
              </Form.Item>
            </Col>
          </Row>
        )}

        <Form.Item name="note" label="Ghi chú">
          <Input placeholder="VD: Đã giao chìa khóa từ số #12..." />
        </Form.Item>
      </Form>
    </Modal>
  );
}

function OccupiedLockerModal({
  open,
  locker,
  canManage,
  onClose,
  onEdit,
}: {
  open: boolean;
  locker: Locker | null;
  canManage: boolean;
  onClose: () => void;
  onEdit: (locker: Locker) => void;
}) {
  const { message } = App.useApp();
  const qc = useQueryClient();

  const releaseMut = useMutation({
    mutationFn: (id: number) => lockersApi.release(id),
    onSuccess: () => {
      message.success(`Đã giải phóng tủ ${locker?.code}, tủ chuyển về trạng thái Trống!`);
      void qc.invalidateQueries({ queryKey: [lockersApi.key] });
      onClose();
    },
    onError: (err) => showError(err, 'Không thể giải phóng tủ đồ'),
  });

  if (!locker) return null;
  const rental = locker.active_rental ?? locker.current_rental;
  const memberName =
    rental?.member_name ??
    locker.member_name ??
    (rental?.member_id ? `Hội viên #${rental.member_id}` : '—');
  const memberCode = rental?.member_code ?? locker.member_code;
  const rType = rental?.type ?? locker.rental_type;
  const endAt = rental?.end_at ?? locker.end_at;

  return (
    <Modal
      open={open}
      title={
        <Space>
          <span>Chi tiết tủ đồ {locker.code}</span>
          <StatusTag status={locker.status} map={LOCKER_STATUS_MAP} />
        </Space>
      }
      footer={
        <Space>
          <Button onClick={onClose}>Đóng</Button>
          {canManage && (
            <Button icon={<EditOutlined />} onClick={() => onEdit(locker)}>
              Sửa thông tin tủ
            </Button>
          )}
          {canManage && (locker.status === 'OCCUPIED' || locker.status === 'RENTED') && (
            <ConfirmButton
              perm={PERM.LOCKER_MANAGE}
              type="primary"
              danger
              icon={<UnlockOutlined />}
              confirmTitle={`Xác nhận giải phóng tủ ${locker.code} và thu hồi chìa khóa?`}
              onConfirm={() => releaseMut.mutateAsync(locker.id)}
            >
              Giải phóng tủ
            </ConfirmButton>
          )}
        </Space>
      }
      onCancel={onClose}
      width={520}
    >
      <Descriptions bordered size="small" column={1} style={{ marginTop: 12 }}>
        <Descriptions.Item label="Mã tủ & Khu vực">
          <strong>{locker.code}</strong> · {locker.zone}
          {locker.branch_name ? ` · ${locker.branch_name}` : ''}
        </Descriptions.Item>
        <Descriptions.Item label="Kích cỡ & Phí thuê tháng">
          {LOCKER_SIZE_LABELS[locker.size] ?? locker.size} · {formatMoney(locker.monthly_fee)}/tháng
        </Descriptions.Item>
        <Descriptions.Item label="Trạng thái hiện tại">
          <StatusTag status={locker.status} map={LOCKER_STATUS_MAP} />
        </Descriptions.Item>
        {(locker.status === 'OCCUPIED' || locker.status === 'RENTED') && (
          <>
            <Descriptions.Item label="Hội viên đang dùng">
              <strong>{memberName}</strong> {memberCode ? `(${memberCode})` : ''}
            </Descriptions.Item>
            <Descriptions.Item label="Loại sử dụng">
              {rType === 'MONTHLY' ? (
                <Tag color="blue">Thuê cố định theo tháng</Tag>
              ) : (
                <Tag color="orange">Mượn theo lượt trong ngày</Tag>
              )}
            </Descriptions.Item>
            {endAt && (
              <Descriptions.Item label="Hạn sử dụng đến">{formatDateTime(endAt)}</Descriptions.Item>
            )}
          </>
        )}
        <Descriptions.Item label="Ghi chú">{locker.note || '—'}</Descriptions.Item>
      </Descriptions>
    </Modal>
  );
}

function LockerBoardTab({
  canManage,
  onAddLocker,
  onEditLocker,
}: {
  canManage: boolean;
  onAddLocker: () => void;
  onEditLocker: (l: Locker) => void;
}) {
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const [selectedBranch, setSelectedBranch] = useState<number | undefined>(branchId ?? undefined);
  const [statusFilter, setStatusFilter] = useState<LockerStatus | 'ALL'>('ALL');
  const [assignTarget, setAssignTarget] = useState<Locker | null>(null);
  const [detailTarget, setDetailTarget] = useState<Locker | null>(null);

  useEffect(() => {
    if (branchId) setSelectedBranch(branchId);
  }, [branchId]);

  const boardQuery = useQuery({
    queryKey: [lockersApi.key, 'board', selectedBranch ?? branchId],
    queryFn: () => lockersApi.board(selectedBranch ?? branchId),
  });

  const data = boardQuery.data;
  const summary = data?.summary ?? { total: 0, available: 0, occupied: 0, rented: 0, broken: 0 };

  // "Tất cả chi nhánh": mỗi chi nhánh đều có tủ A-01, khu "Khu Nam – Tầng 2"... → nhóm chi nhánh → khu vực,
  // thống kê riêng từng chi nhánh để không trộn tủ cùng mã.
  const branchGroups = useMemo(
    () =>
      groupLockersByBranch(data?.lockers ?? [], (id) => branches.find((b) => b.id === id)?.name),
    [data, branches],
  );
  const multiBranch = branchGroups.length > 1;

  const handleTileClick = (locker: Locker) => {
    if (locker.status === 'AVAILABLE' && canManage) {
      setAssignTarget(locker);
    } else {
      setDetailTarget(locker);
    }
  };

  // Ô tủ của một khu vực (lọc theo trạng thái); null khi khu không còn tủ nào sau lọc.
  const renderZone = (z: LockerBoardZone, keyPrefix: string) => {
    const visibleLockers =
      statusFilter === 'ALL' ? z.lockers : z.lockers.filter((l) => l.status === statusFilter);
    if (visibleLockers.length === 0) return null;
    return (
      <Card
        key={`${keyPrefix}-${z.zone}`}
        size="small"
        title={
          <Space>
            <Typography.Text strong style={{ fontSize: 15 }}>
              {z.zone}
            </Typography.Text>
            <Badge count={visibleLockers.length} color="#0A3150" />
          </Space>
        }
      >
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(155px, 1fr))',
            gap: 12,
          }}
        >
          {visibleLockers.map((lk) => {
            const st = LOCKER_VISUAL_STYLES[lk.status] ?? LOCKER_VISUAL_STYLES.AVAILABLE;
            const rental = lk.active_rental ?? lk.current_rental;
            const memberLabel =
              rental?.member_name ?? lk.member_name ?? rental?.member_code ?? lk.member_code;

            return (
              <div
                key={lk.id}
                onClick={() => handleTileClick(lk)}
                className="locker-tile"
                style={{ background: st.bg, border: `1.5px solid ${st.border}` }}
              >
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                  }}
                >
                  <Typography.Text
                    strong
                    className="code-text"
                    style={{ fontSize: 16, color: st.text, whiteSpace: 'nowrap' }}
                  >
                    {lk.code}
                  </Typography.Text>
                  <StatusTag status={lk.status} map={LOCKER_STATUS_MAP} />
                </div>

                <div className="locker-tile__meta">
                  {memberLabel ? (
                    <div className="locker-tile__member">
                      <UserOutlined /> {memberLabel}
                    </div>
                  ) : (
                    <div>Cỡ: {LOCKER_SIZE_LABELS[lk.size] ?? lk.size}</div>
                  )}
                  {lk.status === 'RENTED' && (rental?.end_at || lk.end_at) && (
                    <div
                      className="locker-tile__note"
                      style={{ color: LOCKER_VISUAL_STYLES.RENTED.text }}
                    >
                      Hạn: {formatDate(rental?.end_at ?? lk.end_at)}
                    </div>
                  )}
                  {lk.status === 'AVAILABLE' && (
                    <div
                      className="locker-tile__note"
                      style={{ color: LOCKER_VISUAL_STYLES.AVAILABLE.text }}
                    >
                      {formatMoney(lk.monthly_fee)}/tháng
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </Card>
    );
  };

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row gutter={[12, 12]}>
        <Col xs={12} sm={8} md={4}>
          <StatCard
            label="Tổng số tủ"
            value={summary.total}
            caption={multiBranch ? `Cộng ${branchGroups.length} chi nhánh` : undefined}
            icon={<InboxOutlined />}
            loading={boardQuery.isLoading}
          />
        </Col>
        <Col xs={12} sm={8} md={5}>
          <StatCard
            label="Trống (Sẵn sàng)"
            value={summary.available}
            icon={<CheckCircleOutlined />}
            loading={boardQuery.isLoading}
          />
        </Col>
        <Col xs={12} sm={8} md={5}>
          <StatCard
            label="Đang dùng theo lượt"
            value={summary.occupied}
            icon={<ClockCircleOutlined />}
            loading={boardQuery.isLoading}
          />
        </Col>
        <Col xs={12} sm={12} md={5}>
          <StatCard
            label="Đang thuê tháng"
            value={summary.rented}
            icon={<LockOutlined />}
            loading={boardQuery.isLoading}
          />
        </Col>
        <Col xs={12} sm={12} md={5}>
          <StatCard
            label="Đang hỏng / Bảo trì"
            value={summary.broken}
            icon={<ToolOutlined />}
            loading={boardQuery.isLoading}
          />
        </Col>
      </Row>

      <Card size="small">
        <Row justify="space-between" align="middle" gutter={[12, 12]}>
          <Col>
            <Space wrap>
              {!branchId && (
                <Select
                  allowClear
                  placeholder="Tất cả chi nhánh"
                  style={{ width: 200 }}
                  value={selectedBranch}
                  onChange={(v) => setSelectedBranch(v)}
                  options={branches.map((b) => ({ value: b.id, label: b.name }))}
                />
              )}
              <Radio.Group
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                optionType="button"
                options={[
                  { label: `Tất cả (${summary.total})`, value: 'ALL' },
                  { label: `Trống (${summary.available})`, value: 'AVAILABLE' },
                  { label: `Theo lượt (${summary.occupied})`, value: 'OCCUPIED' },
                  { label: `Thuê tháng (${summary.rented})`, value: 'RENTED' },
                  { label: `Hỏng (${summary.broken})`, value: 'BROKEN' },
                ]}
              />
            </Space>
          </Col>
          <Col>
            <Space>
              {canManage && (
                <Button
                  color="orange"
                  variant="solid"
                  icon={<PlusOutlined />}
                  onClick={onAddLocker}
                >
                  Thêm tủ mới
                </Button>
              )}
              <Button
                icon={<ReloadOutlined spin={boardQuery.isFetching} />}
                onClick={() => boardQuery.refetch()}
              >
                Làm mới sơ đồ
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      {/* Chữ dùng màu văn bản chính (colorText) để đủ tương phản trên nền xanh nhạt ở cả chế độ sáng/tối. */}
      <Alert
        type="info"
        showIcon
        message={<Typography.Text strong>Hướng dẫn thao tác nhanh trên sơ đồ tủ</Typography.Text>}
        description={
          <Typography.Text>
            Bấm vào ô tủ <Typography.Text strong>Trống</Typography.Text> (màu xanh lá) để Gán tủ
            theo lượt hoặc Cho thuê tủ tháng. Bấm vào ô tủ{' '}
            <Typography.Text strong>Đang dùng</Typography.Text> (màu cam / xanh dương) để xem hội
            viên và Giải phóng tủ. Tủ <Typography.Text strong>Hỏng</Typography.Text> giữ nguyên
            trạng thái khi giải phóng lượt thuê.
          </Typography.Text>
        }
      />

      {boardQuery.isLoading ? (
        <div style={{ textAlign: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      ) : !data || data.zones.length === 0 ? (
        <Card>
          <Empty description="Chưa có dữ liệu sơ đồ tủ đồ tại chi nhánh này" />
        </Card>
      ) : (
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          {multiBranch
            ? branchGroups.map((g) => {
                const zones = g.zones.map((z) => renderZone(z, `b${g.branchId}`)).filter(Boolean);
                if (zones.length === 0) return null;
                return (
                  <section key={g.branchId} aria-label={`Tủ đồ ${g.branchName}`}>
                    <Space wrap size={[8, 4]} align="center" style={{ marginBottom: 8 }}>
                      <Typography.Title level={5} style={{ margin: 0 }}>
                        <BankOutlined /> {g.branchName}
                      </Typography.Title>
                      <Tag bordered={false}>Tổng {g.summary.total}</Tag>
                      <Tag bordered={false} color="success">
                        Trống {g.summary.available}
                      </Tag>
                      <Tag bordered={false} color="warning">
                        Theo lượt {g.summary.occupied}
                      </Tag>
                      <Tag bordered={false} color="processing">
                        Thuê tháng {g.summary.rented}
                      </Tag>
                      <Tag bordered={false} color="error">
                        Hỏng {g.summary.broken}
                      </Tag>
                    </Space>
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      {zones}
                    </Space>
                  </section>
                );
              })
            : data.zones.map((z) => renderZone(z, 'z'))}
        </Space>
      )}

      <AssignLockerModal
        open={!!assignTarget}
        locker={assignTarget}
        onClose={() => setAssignTarget(null)}
      />

      <OccupiedLockerModal
        open={!!detailTarget}
        locker={detailTarget}
        canManage={canManage}
        onClose={() => setDetailTarget(null)}
        onEdit={(lk) => {
          setDetailTarget(null);
          onEditLocker(lk);
        }}
      />
    </Space>
  );
}

function LockerListTab({
  canManage,
  onAddLocker,
  onEditLocker,
}: {
  canManage: boolean;
  onAddLocker: () => void;
  onEditLocker: (l: Locker) => void;
}) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const [assignTarget, setAssignTarget] = useState<Locker | null>(null);

  const table = useTableQuery<Locker, LockerListParams>({
    queryKey: [lockersApi.key, 'list'],
    queryFn: (p) => lockersApi.list(p),
    filterKeys: ['branch_id', 'size', 'status'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'code', order_dir: 'ASC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const releaseMut = useMutation({
    mutationFn: (id: number) => lockersApi.release(id),
    onSuccess: () => {
      message.success('Đã giải phóng tủ đồ thành công');
      void qc.invalidateQueries({ queryKey: [lockersApi.key] });
    },
    onError: (err) => showError(err, 'Không thể giải phóng tủ'),
  });

  const removeMut = useMutation({
    mutationFn: (id: number) => lockersApi.remove(id),
    onSuccess: () => {
      message.success('Đã xóa tủ đồ');
      void qc.invalidateQueries({ queryKey: [lockersApi.key] });
    },
    onError: (err) => showError(err, 'Không thể xóa tủ đồ'),
  });

  const columns = useMemo<ColumnsType<Locker>>(
    () => [
      {
        title: 'Mã tủ',
        dataIndex: 'code',
        width: 110,
        sorter: true,
        render: (code: string, r) => (
          <Button
            type="link"
            size="small"
            className="code-text"
            style={{ padding: 0, fontWeight: 600 }}
            onClick={() => onEditLocker(r)}
          >
            {code}
          </Button>
        ),
      },
      {
        title: 'Khu vực',
        dataIndex: 'zone',
      },
      {
        title: 'Kích cỡ',
        dataIndex: 'size',
        render: (s: LockerSize) => LOCKER_SIZE_LABELS[s] ?? s,
      },
      {
        title: 'Chi nhánh',
        dataIndex: 'branch_name',
        render: (v: string | undefined, r) => v || `CN #${r.branch_id}`,
      },
      {
        title: 'Giá thuê tháng',
        dataIndex: 'monthly_fee',
        align: 'right',
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        title: 'Hội viên đang dùng',
        key: 'member',
        render: (_, r) => {
          const rental = r.active_rental ?? r.current_rental;
          const name = rental?.member_name ?? r.member_name;
          if (!name) return '—';
          return (
            <Space direction="vertical" size={0}>
              <Typography.Text strong>{name}</Typography.Text>
              {(rental?.end_at || r.end_at) && (
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  Đến: {formatDate(rental?.end_at ?? r.end_at)}
                </Typography.Text>
              )}
            </Space>
          );
        },
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        render: (s: LockerStatus) => <StatusTag status={s} map={LOCKER_STATUS_MAP} />,
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 210,
        render: (_, r) => (
          <Space size={4}>
            {canManage && r.status === 'AVAILABLE' && (
              <Button size="small" type="primary" ghost onClick={() => setAssignTarget(r)}>
                Gán / Thuê tủ
              </Button>
            )}
            {canManage && (r.status === 'OCCUPIED' || r.status === 'RENTED') && (
              <ConfirmButton
                perm={PERM.LOCKER_MANAGE}
                size="small"
                danger
                icon={<UnlockOutlined />}
                confirmTitle={`Giải phóng tủ ${r.code}?`}
                onConfirm={() => releaseMut.mutateAsync(r.id)}
              >
                Giải phóng
              </ConfirmButton>
            )}
            {canManage && (
              <Tooltip title="Sửa">
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => onEditLocker(r)}
                />
              </Tooltip>
            )}
            <ConfirmButton
              perm={PERM.LOCKER_MANAGE}
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              confirmTitle={`Xóa tủ đồ ${r.code}?`}
              onConfirm={() => removeMut.mutateAsync(r.id)}
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
            onAdd={onAddLocker}
            addPerm={PERM.LOCKER_MANAGE}
            addText="Thêm tủ đồ mới"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo mã tủ, khu vực..."
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
            name: 'size',
            placeholder: 'Kích cỡ',
            options: LOCKER_SIZE_OPTIONS,
            col: 4,
          },
          {
            name: 'status',
            placeholder: 'Trạng thái',
            options: LOCKER_STATUS_OPTIONS,
            col: 4,
          },
        ]}
      />

      <DataTable<Locker> table={table} columns={columns} emptyText="Chưa có tủ đồ nào" />

      <AssignLockerModal
        open={!!assignTarget}
        locker={assignTarget}
        onClose={() => setAssignTarget(null)}
      />
    </>
  );
}

export default function LockersPage() {
  const { can } = usePermission();
  const canManage = can(PERM.LOCKER_MANAGE);
  const [formModal, setFormModal] = useState<{ open: boolean; record: Locker | null }>({
    open: false,
    record: null,
  });

  return (
    <Card variant="borderless" className="page-card">
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <PageTitle
          title="Sơ đồ tủ đồ & Dịch vụ thuê tủ"
          subtitle="Quản lý sơ đồ tủ đồ trực quan theo khu vực, gán tủ theo lượt check-in hoặc cho thuê tủ cố định theo tháng"
        />

        <Tabs
          defaultActiveKey="board"
          items={[
            {
              key: 'board',
              label: (
                <Space size={6}>
                  <AppstoreOutlined />
                  <span>Sơ đồ tủ trực quan (Board)</span>
                </Space>
              ),
              children: (
                <LockerBoardTab
                  canManage={canManage}
                  onAddLocker={() => setFormModal({ open: true, record: null })}
                  onEditLocker={(lk) => setFormModal({ open: true, record: lk })}
                />
              ),
            },
            {
              key: 'list',
              label: (
                <Space size={6}>
                  <TableOutlined />
                  <span>Danh sách & Quản lý tủ đồ</span>
                </Space>
              ),
              children: (
                <LockerListTab
                  canManage={canManage}
                  onAddLocker={() => setFormModal({ open: true, record: null })}
                  onEditLocker={(lk) => setFormModal({ open: true, record: lk })}
                />
              ),
            },
          ]}
        />

        <LockerFormModal
          open={formModal.open}
          record={formModal.record}
          readOnly={!canManage}
          onClose={() => setFormModal({ open: false, record: null })}
        />
      </Space>
    </Card>
  );
}
