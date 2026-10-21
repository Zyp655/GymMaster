import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Descriptions,
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
  CalculatorOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  DollarOutlined,
  DownloadOutlined,
  EditOutlined,
  FileDoneOutlined,
  LockOutlined,
  LoginOutlined,
  ReloadOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { ATTENDANCE_METHOD_LABELS, payrollApi } from '@/api/payroll-finance';
import type {
  AttendanceMethod,
  Payroll,
  PayrollItem,
  PayrollStatus,
  StaffAttendance,
} from '@/api/payroll-finance';

const ATT_METHOD_OPTIONS = (Object.keys(ATTENDANCE_METHOD_LABELS) as AttendanceMethod[]).map(
  (value) => ({ value, label: ATTENDANCE_METHOD_LABELS[value] }),
);
import staffApi from '@/api/staff';
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
import { PAYROLL_STATUS_MAP } from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { showError } from '@/utils/errors';
import { formatDateTime, formatMoney } from '@/utils/format';

function EditPayrollItemModal({
  open,
  payrollId,
  item,
  onClose,
}: {
  open: boolean;
  payrollId: number;
  item: PayrollItem | null;
  onClose: () => void;
}) {
  const [form] = Form.useForm<{ bonus: number; penalty: number; note?: string }>();
  const { message } = App.useApp();
  const qc = useQueryClient();

  useEffect(() => {
    if (!open || !item) return;
    form.setFieldsValue({
      bonus: item.bonus || 0,
      penalty: item.penalty || 0,
      note: item.note || '',
    });
  }, [open, item, form]);

  const updateMut = useMutation({
    mutationFn: (v: { bonus: number; penalty: number; note?: string }) =>
      payrollApi.updateItem(payrollId, item!.id, {
        staff_id: item!.staff_id,
        bonus: Number(v.bonus) || 0,
        penalty: Number(v.penalty) || 0,
        note: v.note?.trim() || '',
      }),
    onSuccess: () => {
      message.success(`Đã cập nhật thưởng/phạt cho ${item?.staff_name || `NV #${item?.staff_id}`}`);
      void qc.invalidateQueries({ queryKey: [payrollApi.key] });
      onClose();
    },
    onError: (err) => showError(err, 'Không thể cập nhật dòng lương'),
  });

  return (
    <Modal
      open={open}
      title={`Điều chỉnh Thưởng / Phạt – ${item?.staff_name || `Nhân viên #${item?.staff_id}`}`}
      okText="Lưu điều chỉnh"
      cancelText="Hủy"
      onOk={() => form.submit()}
      onCancel={onClose}
      confirmLoading={updateMut.isPending}
      width={480}
      forceRender
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={(v) => updateMut.mutate(v)}
        style={{ marginTop: 12 }}
      >
        <Row gutter={12}>
          <Col xs={12}>
            <Form.Item name="bonus" label="Tiền thưởng (VND)">
              <MoneyInput />
            </Form.Item>
          </Col>
          <Col xs={12}>
            <Form.Item name="penalty" label="Khấu trừ / Phạt (VND)">
              <MoneyInput />
            </Form.Item>
          </Col>
        </Row>
        <Form.Item name="note" label="Ghi chú điều chỉnh">
          <Input.TextArea rows={2} placeholder="VD: Thưởng vượt KPI tháng, phạt đi muộn..." />
        </Form.Item>
      </Form>
    </Modal>
  );
}

function MonthlyPayrollTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();

  const [selectedBranch, setSelectedBranch] = useState<number | undefined>(
    branchId ?? branches[0]?.id ?? 1,
  );
  const [periodDayjs, setPeriodDayjs] = useState<Dayjs>(dayjs());
  const [activePayrollId, setActivePayrollId] = useState<number | null>(null);
  const [editingItem, setEditingItem] = useState<PayrollItem | null>(null);

  useEffect(() => {
    if (branchId) setSelectedBranch(branchId);
    else if (!selectedBranch && branches.length > 0) setSelectedBranch(branches[0].id);
  }, [branchId, branches, selectedBranch]);

  const periodStr = periodDayjs.format('YYYY-MM');

  const listQuery = useQuery({
    queryKey: [payrollApi.key, 'list', selectedBranch, periodStr],
    queryFn: () =>
      payrollApi.list({
        branch_id: selectedBranch,
        period: periodStr,
        limit: 50,
      }),
  });

  const matchedPayroll = useMemo(() => {
    const items = listQuery.data?.items ?? [];
    if (activePayrollId) {
      const found = items.find((p) => p.id === activePayrollId);
      if (found) return found;
    }
    return items.find(
      (p) =>
        (!selectedBranch || p.branch_id === selectedBranch) &&
        (!periodStr || p.period === periodStr),
    ) ?? items[0] ?? null;
  }, [listQuery.data, activePayrollId, selectedBranch, periodStr]);

  const detailQuery = useQuery({
    queryKey: [payrollApi.key, 'detail', matchedPayroll?.id],
    queryFn: () => payrollApi.get(matchedPayroll!.id),
    enabled: !!matchedPayroll?.id,
  });

  const currentPayroll: Payroll | null = detailQuery.data ?? matchedPayroll;
  const payrollItems = useMemo<PayrollItem[]>(
    () => currentPayroll?.items ?? [],
    [currentPayroll?.items],
  );
  const isDraft = currentPayroll?.status === 'DRAFT';

  const generateMut = useMutation({
    mutationFn: () => {
      const bid = selectedBranch ?? branchId ?? branches[0]?.id ?? 1;
      return payrollApi.generate({ branch_id: bid, period: periodStr });
    },
    onSuccess: (res) => {
      message.success(`Đã tạo / tính lại bảng lương kỳ ${periodStr}`);
      if (res?.id) setActivePayrollId(res.id);
      void qc.invalidateQueries({ queryKey: [payrollApi.key] });
    },
    onError: (err) => showError(err, 'Không thể tạo bảng lương tháng'),
  });

  const lockMut = useMutation({
    mutationFn: (id: number) => payrollApi.lock(id),
    onSuccess: () => {
      message.success('Đã khóa sổ bảng lương tháng!');
      void qc.invalidateQueries({ queryKey: [payrollApi.key] });
    },
    onError: (err) => showError(err, 'Không thể khóa bảng lương'),
  });

  const payMut = useMutation({
    mutationFn: (id: number) => payrollApi.pay(id),
    onSuccess: () => {
      message.success('Đã xác nhận chi trả lương và tất toán hoa hồng HLV!');
      void qc.invalidateQueries({ queryKey: [payrollApi.key] });
    },
    onError: (err) => showError(err, 'Không thể chi trả bảng lương'),
  });

  const totals = useMemo(() => {
    return payrollItems.reduce(
      (acc, it) => ({
        base: acc.base + (Number(it.base_salary) || 0),
        comm: acc.comm + (Number(it.pt_commission) || 0) + (Number(it.sales_commission) || 0),
        bonusPenalty: acc.bonusPenalty + (Number(it.bonus) || 0) - (Number(it.penalty) || 0),
        net: acc.net + (Number(it.total) || 0),
      }),
      { base: 0, comm: 0, bonusPenalty: 0, net: 0 },
    );
  }, [payrollItems]);

  const itemColumns = useMemo<ColumnsType<PayrollItem>>(
    () => [
      {
        title: 'Nhân viên',
        key: 'staff',
        fixed: 'left',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>{r.staff_name || `Nhân viên #${r.staff_id}`}</Typography.Text>
            {r.staff_code && (
              <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
                {r.staff_code}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Ngày công',
        dataIndex: 'work_days',
        align: 'right',
        width: 100,
        render: (v: number) => <strong>{v ?? 0}</strong>,
      },
      {
        title: 'Giờ OT',
        dataIndex: 'ot_hours',
        align: 'right',
        width: 90,
        render: (v: number) => v || 0,
      },
      {
        title: 'Lương cơ bản',
        dataIndex: 'base_salary',
        align: 'right',
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        title: 'Hoa hồng PT',
        dataIndex: 'pt_commission',
        align: 'right',
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        title: 'HH Bán gói',
        dataIndex: 'sales_commission',
        align: 'right',
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        title: 'Thưởng',
        dataIndex: 'bonus',
        align: 'right',
        render: (v: number) => (
          <Typography.Text type={v > 0 ? 'success' : undefined}>
            {formatMoney(v || 0)}
          </Typography.Text>
        ),
      },
      {
        title: 'Phạt / Trừ',
        dataIndex: 'penalty',
        align: 'right',
        render: (v: number) => (
          <Typography.Text type={v > 0 ? 'danger' : undefined}>
            {formatMoney(v || 0)}
          </Typography.Text>
        ),
      },
      {
        title: 'Thực lĩnh',
        dataIndex: 'total',
        align: 'right',
        render: (v: number) => <MoneyText value={v} strong />,
      },
      {
        title: 'Ghi chú',
        dataIndex: 'note',
        render: (v: string) => v || '—',
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 120,
        render: (_, r) =>
          canManage && isDraft ? (
            <Button
              size="small"
              type="link"
              icon={<EditOutlined />}
              onClick={() => setEditingItem(r)}
            >
              Thưởng/Phạt
            </Button>
          ) : null,
      },
    ],
    [canManage, isDraft],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card size="small">
        <Row justify="space-between" align="middle" gutter={[12, 12]}>
          <Col>
            <Space wrap>
              {!branchId && (
                <Select
                  placeholder="Chọn chi nhánh"
                  style={{ width: 210 }}
                  value={selectedBranch}
                  onChange={(v) => {
                    setSelectedBranch(v);
                    setActivePayrollId(null);
                  }}
                  options={branches.map((b) => ({ value: b.id, label: b.name }))}
                />
              )}
              <DatePicker
                picker="month"
                format="YYYY-MM"
                allowClear={false}
                value={periodDayjs}
                onChange={(d) => {
                  if (d) {
                    setPeriodDayjs(d);
                    setActivePayrollId(null);
                  }
                }}
              />
              {canManage && (
                <Button
                  color="orange"
                  variant="solid"
                  icon={<CalculatorOutlined />}
                  loading={generateMut.isPending}
                  disabled={currentPayroll?.status === 'LOCKED' || currentPayroll?.status === 'PAID'}
                  onClick={() => generateMut.mutate()}
                >
                  Tạo / Tính lại bảng lương
                </Button>
              )}
            </Space>
          </Col>
          <Col>
            <Space wrap>
              {currentPayroll && (
                <>
                  <Button
                    icon={<DownloadOutlined />}
                    onClick={() => void payrollApi.exportCsv(currentPayroll)}
                  >
                    Xuất CSV / Excel
                  </Button>
                  {canManage && currentPayroll.status === 'DRAFT' && (
                    <ConfirmButton
                      perm={PERM.PAYROLL_MANAGE}
                      type="primary"
                      icon={<LockOutlined />}
                      confirmTitle={`Khóa sổ bảng lương kỳ ${currentPayroll.period}? Sau khi khóa sẽ không thể sửa thưởng/phạt.`}
                      onConfirm={() => lockMut.mutateAsync(currentPayroll.id)}
                    >
                      Khóa bảng lương
                    </ConfirmButton>
                  )}
                  {canManage &&
                    (currentPayroll.status === 'LOCKED' || currentPayroll.status === 'DRAFT') && (
                      <ConfirmButton
                        perm={PERM.PAYROLL_MANAGE}
                        type="primary"
                        icon={<CheckCircleOutlined />}
                        confirmTitle={`Xác nhận đã chi trả lương kỳ ${currentPayroll.period} cho toàn bộ nhân sự?`}
                        onConfirm={() => payMut.mutateAsync(currentPayroll.id)}
                      >
                        Chi trả lương
                      </ConfirmButton>
                    )}
                </>
              )}
              <Button
                icon={<ReloadOutlined spin={listQuery.isFetching || detailQuery.isFetching} />}
                onClick={() => {
                  void listQuery.refetch();
                  void detailQuery.refetch();
                }}
              >
                Làm mới
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>

      <Row gutter={[12, 12]}>
        <Col xs={12} md={6}>
          <StatCard
            label="Tổng thực lĩnh kỳ này"
            value={totals.net}
            currency
            icon={<DollarOutlined />}
            loading={listQuery.isLoading || detailQuery.isLoading}
          />
        </Col>
        <Col xs={12} md={6}>
          <StatCard
            label="Tổng lương cơ bản"
            value={totals.base}
            currency
            icon={<TeamOutlined />}
            loading={listQuery.isLoading || detailQuery.isLoading}
          />
        </Col>
        <Col xs={12} md={6}>
          <StatCard
            label="Tổng hoa hồng PT & Bán gói"
            value={totals.comm}
            currency
            icon={<FileDoneOutlined />}
            loading={listQuery.isLoading || detailQuery.isLoading}
          />
        </Col>
        <Col xs={12} md={6}>
          <StatCard
            label="Chênh lệch Thưởng – Phạt"
            value={totals.bonusPenalty}
            currency
            icon={<CalculatorOutlined />}
            loading={listQuery.isLoading || detailQuery.isLoading}
          />
        </Col>
      </Row>

      {listQuery.isLoading || detailQuery.isLoading ? (
        <div style={{ textAlign: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      ) : !currentPayroll ? (
        <Card>
          <Empty
            description={`Chưa có bảng lương cho kỳ ${periodStr}. Hãy bấm "Tạo / Tính lại bảng lương" để tổng hợp công và hoa hồng.`}
          />
        </Card>
      ) : (
        <Card
          size="small"
          title={
            <Space>
              <Typography.Text strong>
                Bảng lương kỳ {currentPayroll.period} –{' '}
                {currentPayroll.branch_name || `Chi nhánh #${currentPayroll.branch_id}`}
              </Typography.Text>
              <StatusTag
                status={currentPayroll.status as PayrollStatus}
                map={PAYROLL_STATUS_MAP}
              />
            </Space>
          }
          extra={
            currentPayroll.paid_at ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                Đã chi lương lúc: {formatDateTime(currentPayroll.paid_at)}
              </Typography.Text>
            ) : currentPayroll.locked_at ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                Đã khóa sổ lúc: {formatDateTime(currentPayroll.locked_at)}
              </Typography.Text>
            ) : null
          }
        >
          {isDraft && canManage && (
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 12 }}
              message="Bảng lương đang ở trạng thái Nháp (DRAFT) – Bạn có thể bấm 'Thưởng/Phạt' ở từng nhân viên để điều chỉnh trước khi Khóa sổ."
            />
          )}
          <Table<PayrollItem>
            size="small"
            rowKey="id"
            scroll={{ x: 'max-content' }}
            columns={itemColumns}
            dataSource={payrollItems}
            pagination={false}
            locale={{ emptyText: <Empty description="Chưa có dòng lương nhân viên" /> }}
          />
        </Card>
      )}

      {currentPayroll && (
        <EditPayrollItemModal
          open={!!editingItem}
          payrollId={currentPayroll.id}
          item={editingItem}
          onClose={() => setEditingItem(null)}
        />
      )}
    </Space>
  );
}

function StaffAttendanceTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const [scanOpen, setScanOpen] = useState(false);
  const [scanForm] = Form.useForm<{
    branch_id: number;
    staff_id?: number;
    qr_code?: string;
    method: 'QR' | 'MANUAL';
    note?: string;
  }>();

  const staffQuery = useQuery({
    queryKey: [staffApi.key, 'options', branchId],
    queryFn: () => staffApi.options(branchId),
    enabled: scanOpen,
  });

  const table = useTableQuery<
    StaffAttendance,
    { branch_id?: number; staff_id?: number; page?: number; limit?: number }
  >({
    queryKey: [payrollApi.attendanceKey, 'list'],
    queryFn: (p) => payrollApi.listAttendances(p),
    filterKeys: ['branch_id'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const scanMut = useMutation({
    mutationFn: (v: {
      branch_id: number;
      staff_id?: number;
      qr_code?: string;
      method: 'QR' | 'MANUAL';
      note?: string;
    }) => payrollApi.scanAttendance(v),
    onSuccess: (res) => {
      message.success(
        res?.check_out_at
          ? 'Đã ghi nhận chấm công RA CA thành công!'
          : 'Đã ghi nhận chấm công VÀO CA thành công!',
      );
      void qc.invalidateQueries({ queryKey: [payrollApi.attendanceKey] });
      setScanOpen(false);
    },
    onError: (err) => showError(err, 'Không thể ghi nhận chấm công'),
  });

  const columns = useMemo<ColumnsType<StaffAttendance>>(
    () => [
      {
        title: 'Nhân viên',
        key: 'staff',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>{r.staff_name || `Nhân viên #${r.staff_id}`}</Typography.Text>
            {r.staff_code && (
              <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
                {r.staff_code}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Chi nhánh',
        dataIndex: 'branch_name',
        render: (v: string | undefined, r) => v || `CN #${r.branch_id}`,
      },
      {
        title: 'Giờ vào ca (Check-in)',
        dataIndex: 'check_in_at',
        render: (v: string) => formatDateTime(v),
      },
      {
        title: 'Giờ tan ca (Check-out)',
        dataIndex: 'check_out_at',
        render: (v: string | null | undefined) =>
          v ? formatDateTime(v) : <Tag color="processing">Đang trong ca</Tag>,
      },
      {
        title: 'Đi muộn / Về sớm',
        key: 'late_early',
        render: (_, r) => (
          <Space size={4}>
            {r.late_minutes > 0 ? (
              <Tag color="red">Muộn {r.late_minutes} phút</Tag>
            ) : (
              <Tag color="green">Đúng giờ</Tag>
            )}
            {r.early_minutes > 0 && <Tag color="orange">Về sớm {r.early_minutes} phút</Tag>}
          </Space>
        ),
      },
      {
        title: 'Hình thức',
        dataIndex: 'method',
        render: (m: string) => (
          <Tag color={m === 'QR' ? 'geekblue' : 'default'}>
            {m === 'QR' ? 'Quét QR' : 'Tại quầy'}
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
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col />
        <Col>
          <Space>
            {canManage && (
              <Button
                color="orange"
                variant="solid"
                icon={<LoginOutlined />}
                onClick={() => {
                  scanForm.resetFields();
                  scanForm.setFieldsValue({
                    branch_id: branchId ?? branches[0]?.id ?? 1,
                    method: 'MANUAL',
                  });
                  setScanOpen(true);
                }}
              >
                Chấm công nhanh (Vào / Ra)
              </Button>
            )}
            <Button
              icon={<ReloadOutlined spin={table.isFetching} />}
              onClick={() => table.refetch()}
            >
              Làm mới
            </Button>
          </Space>
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo tên hoặc mã nhân viên..."
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

      <DataTable<StaffAttendance>
        table={table}
        columns={columns}
        emptyText="Chưa có dữ liệu chấm công nhân viên"
      />

      <Modal
        open={scanOpen}
        title="Chấm công nhanh Nhân viên / HLV (Vào ca / Ra ca)"
        okText="Xác nhận chấm công"
        cancelText="Hủy"
        onOk={() => scanForm.submit()}
        onCancel={() => setScanOpen(false)}
        confirmLoading={scanMut.isPending}
        width={500}
        forceRender
      >
        <Form
          form={scanForm}
          layout="vertical"
          onFinish={(v) => scanMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Form.Item
            name="branch_id"
            label="Chi nhánh làm việc"
            rules={[{ required: true, message: 'Chọn chi nhánh' }]}
          >
            <BranchSelect activeOnly />
          </Form.Item>
          <Form.Item
            name="staff_id"
            label="Chọn nhân viên / HLV"
            rules={[{ required: true, message: 'Vui lòng chọn nhân viên' }]}
          >
            <Select
              showSearch
              optionFilterProp="label"
              placeholder="Tìm theo tên hoặc mã nhân viên"
              options={(staffQuery.data ?? []).map((s) => ({
                value: s.id,
                label: `${s.full_name} (${s.staff_code})`,
              }))}
            />
          </Form.Item>
          <Form.Item name="method" label="Phương thức chấm công">
            <Select options={ATT_METHOD_OPTIONS} />
          </Form.Item>
          <Form.Item name="note" label="Ghi chú ca làm việc">
            <Input placeholder="VD: Làm thay ca sáng, tăng ca..." />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

function MyPayslipsTab() {
  const [selectedSlip, setSelectedSlip] = useState<PayrollItem | null>(null);

  const slipsQuery = useQuery({
    queryKey: [payrollApi.myPayslipKey],
    queryFn: () => payrollApi.myPayslips(),
  });

  const columns = useMemo<ColumnsType<PayrollItem>>(
    () => [
      {
        title: 'Kỳ lương',
        dataIndex: 'period',
        render: (v: string | undefined, r) => (
          <Typography.Text strong className="code-text">
            {v || `Bảng lương #${r.payroll_id}`}
          </Typography.Text>
        ),
      },
      {
        title: 'Ngày công',
        dataIndex: 'work_days',
        align: 'right',
      },
      {
        title: 'Lương cơ bản',
        dataIndex: 'base_salary',
        align: 'right',
        render: (v: number) => <MoneyText value={v} />,
      },
      {
        title: 'Hoa hồng PT & Sales',
        key: 'commission',
        align: 'right',
        render: (_, r) => (
          <MoneyText value={(Number(r.pt_commission) || 0) + (Number(r.sales_commission) || 0)} />
        ),
      },
      {
        title: 'Thưởng / Phạt',
        key: 'bonus_penalty',
        align: 'right',
        render: (_, r) => (
          <MoneyText value={(Number(r.bonus) || 0) - (Number(r.penalty) || 0)} />
        ),
      },
      {
        title: 'Thực lĩnh',
        dataIndex: 'total',
        align: 'right',
        render: (v: number) => <MoneyText value={v} strong />,
      },
      {
        title: '',
        key: 'action',
        width: 110,
        render: (_, r) => (
          <Button size="small" type="link" onClick={() => setSelectedSlip(r)}>
            Xem phiếu
          </Button>
        ),
      },
    ],
    [],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card size="small">
        <Table<PayrollItem>
          size="small"
          rowKey="id"
          loading={slipsQuery.isLoading}
          columns={columns}
          dataSource={slipsQuery.data ?? []}
          locale={{ emptyText: <Empty description="Bạn chưa có phiếu lương cá nhân nào" /> }}
        />
      </Card>

      <Modal
        open={!!selectedSlip}
        title={`Phiếu lương cá nhân – Kỳ ${selectedSlip?.period || `#${selectedSlip?.payroll_id}`}`}
        footer={<Button onClick={() => setSelectedSlip(null)}>Đóng</Button>}
        onCancel={() => setSelectedSlip(null)}
        width={520}
      >
        {selectedSlip && (
          <Descriptions bordered size="small" column={1} style={{ marginTop: 12 }}>
            <Descriptions.Item label="Nhân sự">
              <strong>{selectedSlip.staff_name || 'Cá nhân'}</strong>{' '}
              {selectedSlip.staff_code ? `(${selectedSlip.staff_code})` : ''}
            </Descriptions.Item>
            <Descriptions.Item label="Số ngày công / Giờ OT">
              {selectedSlip.work_days} ngày công · {selectedSlip.ot_hours || 0} giờ OT
            </Descriptions.Item>
            <Descriptions.Item label="Lương cơ bản">
              {formatMoney(selectedSlip.base_salary)}
            </Descriptions.Item>
            <Descriptions.Item label="Hoa hồng dạy PT">
              {formatMoney(selectedSlip.pt_commission)}
            </Descriptions.Item>
            <Descriptions.Item label="Hoa hồng bán gói tập">
              {formatMoney(selectedSlip.sales_commission)}
            </Descriptions.Item>
            <Descriptions.Item label="Tiền thưởng">
              <Typography.Text type="success">{formatMoney(selectedSlip.bonus)}</Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label="Khấu trừ / Phạt">
              <Typography.Text type="danger">{formatMoney(selectedSlip.penalty)}</Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label="Tổng thực lĩnh">
              <Typography.Text strong style={{ fontSize: 16, color: '#0A3150' }}>
                {formatMoney(selectedSlip.total)}
              </Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label="Ghi chú">{selectedSlip.note || '—'}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </Space>
  );
}

export default function PayrollsPage() {
  const { can } = usePermission();
  const canManage = can(PERM.PAYROLL_MANAGE);

  // Bố cục chuẩn: Card.page-card (padding 20px như các trang khác).
  return (
    <Card variant="borderless" className="page-card">
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <PageTitle
          title="Chấm công & Bảng lương"
          subtitle="Tổng hợp ngày công, tính lương cơ bản, hoa hồng PT, hoa hồng bán gói và quản lý chi lương hàng tháng"
        />

        <Tabs
          defaultActiveKey="payrolls"
          items={[
            {
              key: 'payrolls',
              label: (
                <Space size={6}>
                  <DollarOutlined />
                  <span>Bảng lương tháng</span>
                </Space>
              ),
              children: <MonthlyPayrollTab canManage={canManage} />,
            },
            {
              key: 'attendances',
              label: (
                <Space size={6}>
                  <ClockCircleOutlined />
                  <span>Chấm công nhân viên</span>
                </Space>
              ),
              children: <StaffAttendanceTab canManage={canManage} />,
            },
            {
              key: 'my-payslips',
              label: (
                <Space size={6}>
                  <FileDoneOutlined />
                  <span>Phiếu lương của tôi</span>
                </Space>
              ),
              children: <MyPayslipsTab />,
            },
          ]}
        />
      </Space>
    </Card>
  );
}
