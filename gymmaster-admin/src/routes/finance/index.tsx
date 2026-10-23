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
  InputNumber,
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
  AuditOutlined,
  BarChartOutlined,
  CalculatorOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  CreditCardOutlined,
  DeleteOutlined,
  DollarOutlined,
  FileProtectOutlined,
  FolderAddOutlined,
  LockOutlined,
  PlusOutlined,
  ReloadOutlined,
  UnlockOutlined,
  WalletOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import {
  EXPENSE_CATEGORY_TYPE_LABELS,
  EXPENSE_CATEGORY_TYPE_OPTIONS,
  financeExtApi,
} from '@/api/payroll-finance';
import type {
  CashSession,
  CashSessionStatus,
  EInvoiceResult,
  Expense,
  ExpenseCategoryType,
  ExpenseInput,
  ExpenseStatus,
  InstallmentSchedule,
  InstallmentStatus,
} from '@/api/payroll-finance';
import invoiceApi from '@/api/invoice';
import ActionBar from '@/components/ActionBar';
import BranchSelect from '@/components/BranchSelect';
import ConfirmButton from '@/components/ConfirmButton';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import HelpLabel from '@/components/HelpLabel';
import MoneyInput from '@/components/Money/MoneyInput';
import MoneyText from '@/components/Money/MoneyText';
import PageTitle from '@/components/PageTitle';
import StatCard from '@/components/StatCard';
import StatusTag from '@/components/StatusTag';
import { PERM } from '@/constants/permissions';
import { REVENUE_HELP } from '@/constants/revenueBasis';
import {
  CASH_SESSION_STATUS_MAP,
  CASH_SESSION_STATUS_OPTIONS,
  EXPENSE_STATUS_MAP,
  EXPENSE_STATUS_OPTIONS,
  INSTALLMENT_STATUS_MAP,
  INSTALLMENT_STATUS_OPTIONS,
  COUNTER_PAYMENT_METHOD_OPTIONS,
} from '@/constants/status';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import InstallmentPlanModal from '@/routes/invoices/InstallmentPlanModal';
import { showError } from '@/utils/errors';
import { DATE_FORMAT, formatDate, formatDateTime, formatMoney, toApiDate } from '@/utils/format';

/* --------------------------------------------------------------------------
 * TAB 1: Báo cáo Lãi / Lỗ (P&L)
 * -------------------------------------------------------------------------- */
function ProfitLossTab() {
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const [selectedBranch, setSelectedBranch] = useState<number | undefined>(branchId ?? undefined);
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([
    dayjs().startOf('month'),
    dayjs().endOf('month'),
  ]);

  useEffect(() => {
    if (branchId) setSelectedBranch(branchId);
  }, [branchId]);

  const fromStr = dateRange[0].format('YYYY-MM-DD');
  const toStr = dateRange[1].format('YYYY-MM-DD');

  const pnlQuery = useQuery({
    queryKey: [financeExtApi.pnlKey, selectedBranch ?? branchId, fromStr, toStr],
    queryFn: () =>
      financeExtApi.profitLoss({
        branch_id: selectedBranch ?? branchId ?? undefined,
        from: fromStr,
        to: toStr,
      }),
  });

  const pnl = pnlQuery.data;

  const breakdownRows = useMemo(() => {
    if (!pnl) return [];
    return [
      {
        key: 'rev_gross',
        item: '1. Tổng doanh thu',
        amount: pnl.gross_revenue,
        isHeader: true,
        color: '#16a34a',
      },
      { key: 'rev_mem', item: '   • Gói tập hội viên', amount: pnl.membership_revenue },
      { key: 'rev_pt', item: '   • Hợp đồng PT 1-1', amount: pnl.pt_revenue },
      { key: 'rev_pos', item: '   • Bán lẻ POS', amount: pnl.pos_revenue },
      { key: 'rev_locker', item: '   • Cho thuê tủ đồ', amount: pnl.locker_revenue },
      { key: 'rev_other', item: '   • Phí & dịch vụ khác', amount: pnl.other_revenue },
      { key: 'refunds', item: '2. Hoàn tiền', amount: -Math.abs(pnl.refunds), color: '#ea580c' },
      {
        key: 'rev_net',
        item: '3. Doanh thu thuần (A = 1 − 2)',
        amount: pnl.net_revenue,
        isHeader: true,
        color: '#16a34a',
      },
      {
        key: 'cogs',
        item: '4. Giá vốn hàng bán POS (B)',
        amount: -Math.abs(pnl.pos_cost),
        isHeader: true,
        color: '#ea580c',
      },
      {
        key: 'gross',
        item: '5. Lợi nhuận gộp (C = A − B)',
        amount: pnl.gross_profit,
        isHeader: true,
      },
      {
        key: 'opex',
        item: '6. Chi phí vận hành đã duyệt (D)',
        amount: -Math.abs(pnl.operating_expenses),
        isHeader: true,
        color: '#dc2626',
      },
      {
        key: 'payroll',
        item: '7. Quỹ lương nhân sự (E)',
        amount: -Math.abs(pnl.payroll_expenses),
        isHeader: true,
        color: '#dc2626',
      },
      {
        key: 'net',
        item: '8. LỢI NHUẬN RÒNG (C − D − E)',
        amount: pnl.net_profit,
        isHeader: true,
        color: pnl.net_profit >= 0 ? '#15803d' : '#b91c1c',
      },
    ];
  }, [pnl]);

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card size="small">
        <Row justify="space-between" align="middle" gutter={[12, 12]}>
          <Col>
            <Space wrap>
              {!branchId && (
                <Select
                  allowClear
                  placeholder="Toàn hệ thống"
                  style={{ width: 200 }}
                  value={selectedBranch}
                  onChange={(v) => setSelectedBranch(v)}
                  options={branches.map((b) => ({ value: b.id, label: b.name }))}
                />
              )}
              <DatePicker.RangePicker
                format={DATE_FORMAT}
                allowClear={false}
                value={dateRange}
                onChange={(vals) => {
                  if (vals && vals[0] && vals[1]) {
                    setDateRange([vals[0], vals[1]]);
                  }
                }}
              />
            </Space>
          </Col>
          <Col>
            <Button
              icon={<ReloadOutlined spin={pnlQuery.isFetching} />}
              onClick={() => pnlQuery.refetch()}
            >
              Làm mới báo cáo
            </Button>
          </Col>
        </Row>
      </Card>

      <Row gutter={[12, 12]}>
        <Col xs={12} sm={8} lg={4} style={{ flex: '1 1 180px' }}>
          <StatCard
            label={<HelpLabel help={REVENUE_HELP.pnl}>Doanh thu thuần</HelpLabel>}
            value={pnl?.net_revenue ?? 0}
            currency
            icon={<DollarOutlined />}
            loading={pnlQuery.isLoading}
          />
        </Col>
        <Col xs={12} sm={8} lg={4} style={{ flex: '1 1 180px' }}>
          <StatCard
            label="Giá vốn POS"
            value={pnl?.pos_cost ?? 0}
            currency
            icon={<CalculatorOutlined />}
            loading={pnlQuery.isLoading}
          />
        </Col>
        <Col xs={12} sm={8} lg={4} style={{ flex: '1 1 180px' }}>
          <StatCard
            label="Chi phí vận hành"
            value={pnl?.operating_expenses ?? 0}
            currency
            icon={<WalletOutlined />}
            loading={pnlQuery.isLoading}
          />
        </Col>
        <Col xs={12} sm={12} lg={4} style={{ flex: '1 1 180px' }}>
          <StatCard
            label="Quỹ lương nhân sự"
            value={pnl?.payroll_expenses ?? 0}
            currency
            icon={<AuditOutlined />}
            loading={pnlQuery.isLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={6} style={{ flex: '1 1 220px' }}>
          <StatCard
            label="Lợi nhuận ròng (P&L)"
            value={pnl?.net_profit ?? 0}
            currency
            caption={pnl ? `Biên lợi nhuận ròng: ${pnl.margin_pct ?? 0}%` : undefined}
            icon={<BarChartOutlined />}
            loading={pnlQuery.isLoading}
          />
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={14}>
          <Card
            size="small"
            title={
              <HelpLabel help={REVENUE_HELP.pnl}>
                Bảng cân đối Lãi / Lỗ (Profit & Loss Statement)
              </HelpLabel>
            }
          >
            <Table
              size="small"
              rowKey="key"
              pagination={false}
              scroll={{ x: 'max-content' }}
              loading={pnlQuery.isLoading}
              dataSource={breakdownRows}
              columns={[
                {
                  title: 'Chỉ tiêu tài chính',
                  dataIndex: 'item',
                  render: (v: string, r) => (
                    <span
                      style={{
                        fontWeight: r.isHeader ? 600 : 400,
                        // pre-wrap: giữ thụt lề nhưng cho xuống dòng ở màn hẹp (D8 – 390px).
                        whiteSpace: 'pre-wrap',
                        display: 'inline-block',
                        maxWidth: 'min(420px, 44vw)',
                        color: r.color,
                      }}
                    >
                      {v}
                    </span>
                  ),
                },
                {
                  title: 'Số tiền',
                  dataIndex: 'amount',
                  align: 'right',
                  render: (v: number, r) => (
                    <span
                      style={{ fontWeight: r.isHeader ? 700 : 400, color: r.color, whiteSpace: 'nowrap' }}
                    >
                      {formatMoney(v || 0)}
                    </span>
                  ),
                },
              ]}
            />
          </Card>
        </Col>
        <Col xs={24} lg={10}>
          <Card size="small" title="Cơ cấu chi phí vận hành theo danh mục">
            {pnlQuery.isLoading ? (
              <div style={{ textAlign: 'center', padding: 32 }}>
                <Spin />
              </div>
            ) : !pnl?.expenses_by_category || pnl.expenses_by_category.length === 0 ? (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="Chưa phát sinh chi phí đã duyệt trong kỳ"
              />
            ) : (
              <Table
                size="small"
                rowKey="category_id"
                pagination={false}
                scroll={{ x: 'max-content' }}
                dataSource={pnl.expenses_by_category}
                columns={[
                  {
                    title: 'Danh mục chi phí',
                    dataIndex: 'category_name',
                    render: (v: string, r) => (
                      <Space size={4} wrap style={{ maxWidth: 'min(200px, 48vw)' }}>
                        <span>{v || `Danh mục #${r.category_id}`}</span>
                        {r.category_type && (
                          <Tag color={r.category_type === 'FIXED' ? 'geekblue' : 'cyan'}>
                            {EXPENSE_CATEGORY_TYPE_LABELS[r.category_type] ?? r.category_type}
                          </Tag>
                        )}
                      </Space>
                    ),
                  },
                  {
                    title: 'Tổng chi',
                    dataIndex: 'amount',
                    align: 'right',
                    render: (v: number) => <MoneyText value={v} strong />,
                  },
                  {
                    title: 'Tỷ trọng',
                    key: 'share',
                    align: 'right',
                    width: 80,
                    // Ẩn trên điện thoại để bảng vừa 390px.
                    responsive: ['sm'],
                    render: (_, r) =>
                      pnl.operating_expenses > 0
                        ? `${Math.round((r.amount / pnl.operating_expenses) * 1000) / 10}%`
                        : '—',
                  },
                ]}
              />
            )}
          </Card>
        </Col>
      </Row>
    </Space>
  );
}

/* --------------------------------------------------------------------------
 * TAB 2: Chi phí vận hành (/admin/expenses, /admin/expense-categories)
 * -------------------------------------------------------------------------- */
function OperatingExpensesTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();

  const [expenseModal, setExpenseModal] = useState(false);
  const [catModal, setCatModal] = useState(false);
  const [expenseForm] = Form.useForm<{
    branch_id: number;
    category_id: number;
    amount: number;
    spent_at: Dayjs;
    description: string;
    attachment_url?: string;
  }>();
  const [catForm] = Form.useForm<{ name: string; type: ExpenseCategoryType }>();

  const categoriesQuery = useQuery({
    queryKey: [financeExtApi.categoryKey],
    queryFn: () => financeExtApi.listCategories(),
  });

  const table = useTableQuery<
    Expense,
    { branch_id?: number; category_id?: number; status?: ExpenseStatus; page?: number; limit?: number }
  >({
    queryKey: [financeExtApi.expenseKey, 'list'],
    queryFn: (p) => financeExtApi.listExpenses(p),
    filterKeys: ['branch_id', 'category_id', 'status'],
    numberKeys: ['branch_id', 'category_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const createExpenseMut = useMutation({
    mutationFn: (v: {
      branch_id: number;
      category_id: number;
      amount: number;
      spent_at: Dayjs;
      description: string;
      attachment_url?: string;
    }) => {
      const payload: ExpenseInput = {
        branch_id: v.branch_id,
        category_id: v.category_id,
        amount: Number(v.amount) || 0,
        spent_at: toApiDate(v.spent_at) ?? dayjs().format('YYYY-MM-DD'),
        description: v.description.trim(),
        attachment_url: v.attachment_url?.trim() || '',
      };
      return financeExtApi.createExpense(payload);
    },
    onSuccess: () => {
      message.success('Đã tạo phiếu chi phí vận hành');
      void qc.invalidateQueries({ queryKey: [financeExtApi.expenseKey] });
      void qc.invalidateQueries({ queryKey: [financeExtApi.pnlKey] });
      setExpenseModal(false);
    },
    onError: (err) => showError(err, 'Không thể lưu khoản chi phí'),
  });

  const createCategoryMut = useMutation({
    mutationFn: (v: { name: string; type: ExpenseCategoryType }) =>
      financeExtApi.createCategory({ name: v.name.trim(), type: v.type }),
    onSuccess: () => {
      message.success('Đã thêm danh mục chi phí');
      void qc.invalidateQueries({ queryKey: [financeExtApi.categoryKey] });
      catForm.resetFields();
      setCatModal(false);
    },
    onError: (err) => showError(err, 'Không thể tạo danh mục chi phí'),
  });

  const approveMut = useMutation({
    mutationFn: (id: number) => financeExtApi.approveExpense(id),
    onSuccess: () => {
      message.success('Đã duyệt khoản chi phí');
      void qc.invalidateQueries({ queryKey: [financeExtApi.expenseKey] });
      void qc.invalidateQueries({ queryKey: [financeExtApi.pnlKey] });
    },
    onError: (err) => showError(err, 'Không thể duyệt khoản chi'),
  });

  const rejectMut = useMutation({
    mutationFn: (id: number) => financeExtApi.rejectExpense(id),
    onSuccess: () => {
      message.success('Đã từ chối khoản chi phí');
      void qc.invalidateQueries({ queryKey: [financeExtApi.expenseKey] });
    },
    onError: (err) => showError(err, 'Không thể từ chối khoản chi'),
  });

  const deleteMut = useMutation({
    mutationFn: (id: number) => financeExtApi.deleteExpense(id),
    onSuccess: () => {
      message.success('Đã xóa khoản chi phí');
      void qc.invalidateQueries({ queryKey: [financeExtApi.expenseKey] });
    },
    onError: (err) => showError(err, 'Không thể xóa khoản chi'),
  });

  const columns = useMemo<ColumnsType<Expense>>(
    () => [
      {
        title: 'Ngày chi',
        dataIndex: 'spent_at',
        width: 105,
        render: (v: string) => formatDate(v),
      },
      {
        title: 'Danh mục chi phí',
        dataIndex: 'category_name',
        width: 200,
        render: (v: string | undefined, r) => (
          <Space direction="vertical" size={2}>
            <Typography.Text strong>{v || `Danh mục #${r.category_id}`}</Typography.Text>
            {r.category_type && (
              <Tag color={r.category_type === 'FIXED' ? 'geekblue' : 'cyan'}>
                {EXPENSE_CATEGORY_TYPE_LABELS[r.category_type] ?? r.category_type}
              </Tag>
            )}
          </Space>
        ),
      },
      {
        title: 'Nội dung / Chi nhánh',
        dataIndex: 'description',
        width: 280,
        render: (v: string, r) => (
          <Space direction="vertical" size={0} style={{ maxWidth: 264 }}>
            <Typography.Text ellipsis={{ tooltip: v }} style={{ maxWidth: 264 }}>
              {v || '—'}
            </Typography.Text>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {r.branch_name || `CN #${r.branch_id}`}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Số tiền chi',
        dataIndex: 'amount',
        align: 'right',
        width: 140,
        render: (v: number) => (
          <span style={{ whiteSpace: 'nowrap' }}>
            <MoneyText value={v} strong />
          </span>
        ),
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        width: 140,
        render: (s: ExpenseStatus, r) => (
          <Space direction="vertical" size={0}>
            <StatusTag status={s} map={EXPENSE_STATUS_MAP} />
            {r.approver_name && s !== 'PENDING' && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                bởi {r.approver_name}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 150,
        render: (_, r) => (
          <Space size={4}>
            {canManage && r.status === 'PENDING' && (
              <>
                <ConfirmButton
                  perm={PERM.EXPENSE_MANAGE}
                  size="small"
                  type="primary"
                  icon={<CheckCircleOutlined />}
                  confirmTitle={`Duyệt khoản chi ${formatMoney(r.amount)} – "${r.description}"?`}
                  onConfirm={() => approveMut.mutateAsync(r.id)}
                >
                  Duyệt
                </ConfirmButton>
                <ConfirmButton
                  perm={PERM.EXPENSE_MANAGE}
                  size="small"
                  danger
                  icon={<CloseCircleOutlined />}
                  confirmTitle="Từ chối khoản chi phí này?"
                  okText="Từ chối"
                  onConfirm={() => rejectMut.mutateAsync(r.id)}
                  aria-label="Từ chối"
                />
              </>
            )}
            {/* Khoản đã duyệt đã vào P&L → API chặn xóa (409); chỉ xóa khi Chờ duyệt / Từ chối. */}
            {r.status !== 'APPROVED' && (
              <ConfirmButton
                perm={PERM.EXPENSE_MANAGE}
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
                confirmTitle="Xóa phiếu chi này?"
                onConfirm={() => deleteMut.mutateAsync(r.id)}
                aria-label="Xóa"
              />
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
      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col />
        <Col>
          <ActionBar
            onAdd={() => {
              expenseForm.resetFields();
              expenseForm.setFieldsValue({
                branch_id: branchId ?? branches[0]?.id ?? 1,
                spent_at: dayjs(),
                amount: 0,
              });
              setExpenseModal(true);
            }}
            addPerm={PERM.EXPENSE_MANAGE}
            addText="Thêm khoản chi"
            onRefresh={() => table.refetch()}
            refreshing={table.isFetching}
          >
            {canManage && (
              <Button icon={<FolderAddOutlined />} onClick={() => setCatModal(true)}>
                Danh mục chi phí
              </Button>
            )}
          </ActionBar>
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tìm theo mô tả khoản chi..."
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
            name: 'category_id',
            placeholder: 'Danh mục chi phí',
            options: (categoriesQuery.data ?? []).map((c) => ({ label: c.name, value: c.id })),
            col: 5,
          },
          {
            name: 'status',
            placeholder: 'Trạng thái duyệt',
            options: EXPENSE_STATUS_OPTIONS,
            col: 4,
          },
        ]}
      />

      <DataTable<Expense>
        table={table}
        columns={columns}
        emptyText="Chưa có khoản chi phí vận hành nào"
      />

      <Modal
        open={expenseModal}
        title="Ghi nhận khoản chi phí vận hành"
        okText="Lưu khoản chi"
        cancelText="Hủy"
        onOk={() => expenseForm.submit()}
        onCancel={() => setExpenseModal(false)}
        confirmLoading={createExpenseMut.isPending}
        width={540}
        forceRender
      >
        <Form
          form={expenseForm}
          layout="vertical"
          onFinish={(v) => createExpenseMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Row gutter={12}>
            <Col xs={24} sm={12}>
              <Form.Item
                name="branch_id"
                label="Chi nhánh"
                rules={[{ required: true, message: 'Chọn chi nhánh' }]}
              >
                <BranchSelect activeOnly />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="category_id"
                label="Danh mục chi phí"
                rules={[{ required: true, message: 'Chọn danh mục' }]}
              >
                <Select
                  placeholder="Chọn danh mục"
                  options={(categoriesQuery.data ?? []).map((c) => ({
                    value: c.id,
                    label: c.name,
                  }))}
                />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="amount"
                label="Số tiền chi (VND)"
                rules={[{ required: true, message: 'Nhập số tiền' }]}
              >
                <MoneyInput />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item
                name="spent_at"
                label="Ngày chi"
                tooltip="Ngày đã chi thực tế – không được sau hôm nay (khoản trả trước ghi theo ngày chi tiền)"
                rules={[{ required: true, message: 'Chọn ngày chi' }]}
              >
                <DatePicker
                  style={{ width: '100%' }}
                  format={DATE_FORMAT}
                  disabledDate={(d) => d.isAfter(dayjs(), 'day')}
                />
              </Form.Item>
            </Col>
            <Col xs={24}>
              <Form.Item
                name="description"
                label="Nội dung chi tiết"
                rules={[{ required: true, whitespace: true, message: 'Nhập nội dung chi' }]}
              >
                <Input.TextArea rows={3} placeholder="VD: Thanh toán tiền điện tháng 10/2026..." />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      <Modal
        open={catModal}
        title="Quản lý Danh mục chi phí"
        okText="Thêm danh mục"
        cancelText="Đóng"
        onOk={() => catForm.submit()}
        onCancel={() => setCatModal(false)}
        confirmLoading={createCategoryMut.isPending}
        width={520}
        forceRender
      >
        <Form
          form={catForm}
          layout="vertical"
          initialValues={{ type: 'VARIABLE' }}
          onFinish={(v) => createCategoryMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Row gutter={8}>
            <Col xs={14}>
              <Form.Item
                name="name"
                label="Tên danh mục mới"
                rules={[{ required: true, whitespace: true, message: 'Nhập tên danh mục' }]}
              >
                <Input placeholder="VD: Chi phí nước uống, khăn tập" />
              </Form.Item>
            </Col>
            <Col xs={10}>
              <Form.Item name="type" label="Phân loại" rules={[{ required: true }]}>
                <Select options={EXPENSE_CATEGORY_TYPE_OPTIONS} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
        <Table
          size="small"
          rowKey="id"
          pagination={false}
          dataSource={categoriesQuery.data ?? []}
          columns={[
            { title: 'Tên danh mục', dataIndex: 'name' },
            {
              title: 'Loại',
              dataIndex: 'type',
              render: (t: ExpenseCategoryType) => (
                <Tag color={t === 'FIXED' ? 'geekblue' : 'cyan'}>
                  {EXPENSE_CATEGORY_TYPE_LABELS[t] ?? t}
                </Tag>
              ),
            },
          ]}
        />
      </Modal>
    </>
  );
}

/* --------------------------------------------------------------------------
 * TAB 3: Đối soát ca thu ngân (/admin/cash-sessions)
 * -------------------------------------------------------------------------- */
function CashSessionsTab() {
  const { message } = App.useApp();
  const qc = useQueryClient();
  const { can } = usePermission();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const canCash = can([PERM.CASH_SESSION_MANAGE, PERM.EXPENSE_MANAGE]);

  const [openModal, setOpenModal] = useState(false);
  const [closeTarget, setCloseTarget] = useState<CashSession | null>(null);
  const [closedResult, setClosedResult] = useState<CashSession | null>(null);
  const [openForm] = Form.useForm<{ branch_id: number; opening_cash: number; note?: string }>();
  const [closeForm] = Form.useForm<{ counted_cash: number; note?: string }>();

  const table = useTableQuery<
    CashSession,
    { branch_id?: number; status?: CashSessionStatus; page?: number; limit?: number }
  >({
    queryKey: [financeExtApi.cashSessionKey, 'list'],
    queryFn: (p) => financeExtApi.listCashSessions(p),
    filterKeys: ['branch_id', 'status'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'id', order_dir: 'DESC' },
    extraParams: branchId ? { branch_id: branchId } : {},
  });

  const openSessionMut = useMutation({
    mutationFn: (v: { branch_id: number; opening_cash: number; note?: string }) =>
      financeExtApi.openCashSession({
        branch_id: v.branch_id,
        opening_cash: Number(v.opening_cash) || 0,
        note: v.note?.trim() || '',
      }),
    onSuccess: () => {
      message.success('Đã mở ca thu ngân mới!');
      void qc.invalidateQueries({ queryKey: [financeExtApi.cashSessionKey] });
      setOpenModal(false);
    },
    onError: (err) => showError(err, 'Không thể mở ca thu ngân'),
  });

  const closeSessionMut = useMutation({
    mutationFn: (v: { counted_cash: number; note?: string }) =>
      financeExtApi.closeCashSession(closeTarget!.id, {
        counted_cash: Number(v.counted_cash) || 0,
        note: v.note?.trim() || '',
      }),
    onSuccess: (res) => {
      message.success('Đã đóng ca thu ngân và đối soát tiền mặt!');
      void qc.invalidateQueries({ queryKey: [financeExtApi.cashSessionKey] });
      setCloseTarget(null);
      setClosedResult(res);
    },
    onError: (err) => showError(err, 'Không thể chốt ca thu ngân'),
  });

  const columns = useMemo<ColumnsType<CashSession>>(
    () => [
      {
        title: 'Ca',
        dataIndex: 'id',
        width: 110,
        render: (id: number, r) => (
          <Space direction="vertical" size={2}>
            <span className="code-text">CA-#{id}</span>
            <StatusTag status={r.status} map={CASH_SESSION_STATUS_MAP} />
          </Space>
        ),
      },
      {
        title: 'Chi nhánh / Thu ngân',
        dataIndex: 'branch_name',
        width: 200,
        render: (v: string | undefined, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>{v || `CN #${r.branch_id}`}</Typography.Text>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {r.staff_name || `NV #${r.staff_id}`}
            </Typography.Text>
          </Space>
        ),
      },
      {
        title: 'Thời gian ca',
        dataIndex: 'opened_at',
        width: 160,
        render: (v: string, r) => (
          <Space direction="vertical" size={0} style={{ whiteSpace: 'nowrap' }}>
            <span>Mở: {formatDateTime(v)}</span>
            {r.closed_at && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                Đóng: {formatDateTime(r.closed_at)}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Tiền đầu ca',
        dataIndex: 'opening_cash',
        align: 'right',
        width: 125,
        render: (v: number) => (
          <span style={{ whiteSpace: 'nowrap' }}>
            <MoneyText value={v} />
          </span>
        ),
      },
      {
        title: 'Tiền mặt kỳ vọng',
        dataIndex: 'expected_cash',
        align: 'right',
        width: 135,
        render: (v: number, r) =>
          r.status === 'CLOSED' ? (
            <span style={{ whiteSpace: 'nowrap' }}>
              <MoneyText value={v} strong />
            </span>
          ) : (
            '—'
          ),
      },
      {
        title: 'Tiền thực đếm',
        dataIndex: 'counted_cash',
        align: 'right',
        width: 125,
        render: (v: number, r) =>
          r.status === 'CLOSED' ? (
            <span style={{ whiteSpace: 'nowrap' }}>
              <MoneyText value={v} strong />
            </span>
          ) : (
            '—'
          ),
      },
      {
        title: 'Chênh lệch',
        dataIndex: 'difference',
        align: 'right',
        width: 120,
        render: (diff: number, r) => {
          if (r.status !== 'CLOSED') return '—';
          const num = Number(diff) || 0;
          if (num === 0) return <Tag color="green">Khớp</Tag>;
          return (
            <Typography.Text strong type={num > 0 ? 'success' : 'danger'} style={{ whiteSpace: 'nowrap' }}>
              {num > 0 ? `+${formatMoney(num)}` : formatMoney(num)}
            </Typography.Text>
          );
        },
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 150,
        render: (_, r) =>
          canCash && r.status === 'OPEN' ? (
            <Button
              size="small"
              type="primary"
              danger
              icon={<LockOutlined />}
              onClick={() => {
                closeForm.resetFields();
                closeForm.setFieldsValue({ counted_cash: r.opening_cash || 0 });
                setCloseTarget(r);
              }}
            >
              Đóng & Đối soát
            </Button>
          ) : null,
      },
    ],
    [canCash, closeForm],
  );

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col />
        <Col>
          <Space>
            {canCash && (
              <Button
                color="orange"
                variant="solid"
                icon={<UnlockOutlined />}
                onClick={() => {
                  openForm.resetFields();
                  openForm.setFieldsValue({
                    branch_id: branchId ?? branches[0]?.id ?? 1,
                    opening_cash: 2000000,
                  });
                  setOpenModal(true);
                }}
              >
                Mở ca thu ngân
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
        textSearch={false}
        filters={[
          ...(branchId
            ? []
            : [
                {
                  name: 'branch_id',
                  placeholder: 'Chi nhánh',
                  options: branches.map((b) => ({ label: b.name, value: b.id })),
                  col: 5,
                },
              ]),
          {
            name: 'status',
            placeholder: 'Trạng thái ca',
            options: CASH_SESSION_STATUS_OPTIONS,
            col: 5,
          },
        ]}
      />

      <DataTable<CashSession>
        table={table}
        columns={columns}
        emptyText="Chưa có phiên đối soát ca thu ngân nào"
      />

      <Modal
        open={openModal}
        title="Mở ca thu ngân mới"
        okText="Mở ca"
        cancelText="Hủy"
        onOk={() => openForm.submit()}
        onCancel={() => setOpenModal(false)}
        confirmLoading={openSessionMut.isPending}
        width={480}
        forceRender
      >
        <Form
          form={openForm}
          layout="vertical"
          onFinish={(v) => openSessionMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Form.Item
            name="branch_id"
            label="Chi nhánh"
            rules={[{ required: true, message: 'Chọn chi nhánh' }]}
          >
            <BranchSelect activeOnly />
          </Form.Item>
          <Form.Item
            name="opening_cash"
            label="Tiền mặt đầu ca trong két"
            rules={[{ required: true, message: 'Nhập tiền mặt đầu ca' }]}
          >
            <MoneyInput />
          </Form.Item>
          <Form.Item
            name="note"
            label="Ghi chú đầu ca"
            rules={[{ max: 500, message: 'Ghi chú tối đa 500 ký tự' }]}
          >
            <Input placeholder="VD: Bàn giao ca sáng..." maxLength={500} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={!!closeTarget}
        title={`Đóng ca & Đối soát tiền mặt – CA-#${closeTarget?.id}`}
        okText="Xác nhận chốt ca"
        cancelText="Hủy"
        onOk={() => closeForm.submit()}
        onCancel={() => setCloseTarget(null)}
        confirmLoading={closeSessionMut.isPending}
        width={500}
        forceRender
      >
        {closeTarget && (
          <Space direction="vertical" size={12} style={{ width: '100%', marginTop: 8 }}>
            <Alert
              type="info"
              showIcon
              message={`Tiền đầu ca: ${formatMoney(closeTarget.opening_cash)}`}
              description="Tiền mặt kỳ vọng = tiền đầu ca + các phiếu thu tiền mặt − các phiếu hoàn tiền mặt phát sinh trong thời gian mở ca; chênh lệch = tiền thực đếm − tiền kỳ vọng."
            />
            <Form
              form={closeForm}
              layout="vertical"
              onFinish={(v) => closeSessionMut.mutate(v)}
            >
              <Form.Item
                name="counted_cash"
                label="Tiền mặt thực đếm tại két cuối ca"
                rules={[{ required: true, message: 'Nhập số tiền thực đếm' }]}
              >
                <MoneyInput />
              </Form.Item>
              <Form.Item
                name="note"
                label="Ghi chú bàn giao / Giải trình chênh lệch"
                rules={[{ max: 500, message: 'Ghi chú tối đa 500 ký tự' }]}
              >
                <Input.TextArea
                  rows={2}
                  placeholder="Nhập ghi chú đối soát..."
                  maxLength={500}
                  showCount
                />
              </Form.Item>
            </Form>
          </Space>
        )}
      </Modal>

      <Modal
        open={!!closedResult}
        title={`Kết quả đối soát ca thu ngân CA-#${closedResult?.id}`}
        okText="Hoàn tất"
        cancelButtonProps={{ style: { display: 'none' } }}
        onOk={() => setClosedResult(null)}
        onCancel={() => setClosedResult(null)}
        width={500}
      >
        {closedResult && (
          <Descriptions bordered size="small" column={1} style={{ marginTop: 12 }}>
            <Descriptions.Item label="Tiền đầu ca">
              {formatMoney(closedResult.opening_cash)}
            </Descriptions.Item>
            <Descriptions.Item label="Tiền mặt kỳ vọng">
              <Typography.Text strong>{formatMoney(closedResult.expected_cash)}</Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label="Tiền thực đếm">
              <Typography.Text strong>{formatMoney(closedResult.counted_cash)}</Typography.Text>
            </Descriptions.Item>
            <Descriptions.Item label="Chênh lệch">
              <Typography.Text
                strong
                type={
                  closedResult.difference === 0
                    ? 'success'
                    : closedResult.difference > 0
                      ? 'warning'
                      : 'danger'
                }
              >
                {closedResult.difference > 0
                  ? `+${formatMoney(closedResult.difference)}`
                  : formatMoney(closedResult.difference)}
              </Typography.Text>
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </>
  );
}

/* --------------------------------------------------------------------------
 * TAB 4: Quản lý trả góp & Hóa đơn điện tử (/admin/installments, /admin/invoices/:id/installments, /admin/invoices/:id/einvoice)
 * -------------------------------------------------------------------------- */
/** Kỳ còn mở (PENDING/OVERDUE) đã qua ngày đến hạn. */
const installmentIsOverdue = (r: InstallmentSchedule) =>
  r.status === 'OVERDUE' || (r.status === 'PENDING' && dayjs(r.due_date).isBefore(dayjs(), 'day'));

function InstallmentsAndEInvoiceTab({ canManage }: { canManage: boolean }) {
  const { message } = App.useApp();
  const qc = useQueryClient();

  const [planModal, setPlanModal] = useState(false);
  const [einvoiceModal, setEinvoiceModal] = useState(false);
  const [einvoiceResult, setEinvoiceResult] = useState<EInvoiceResult | null>(null);
  const [payTarget, setPayTarget] = useState<InstallmentSchedule | null>(null);

  const [einvoiceForm] = Form.useForm<{ invoice_id: number }>();
  const [payForm] = Form.useForm<{ method: string; note?: string }>();

  const table = useTableQuery<
    InstallmentSchedule,
    { status?: InstallmentStatus; page?: number; limit?: number }
  >({
    queryKey: [financeExtApi.installmentKey, 'list'],
    queryFn: (p) => financeExtApi.listInstallments(p),
    filterKeys: ['status'],
    defaults: { order_by: 'due_date', order_dir: 'ASC' },
  });

  const payScheduleMut = useMutation({
    mutationFn: (v: { method: string; note?: string }) =>
      financeExtApi.payInstallment(payTarget!.id, {
        method: v.method,
        note: v.note,
      }),
    onSuccess: () => {
      message.success(`Đã thu tiền kỳ trả góp #${payTarget?.seq} thành công!`);
      void qc.invalidateQueries({ queryKey: [financeExtApi.installmentKey] });
      void qc.invalidateQueries({ queryKey: [invoiceApi.key] });
      setPayTarget(null);
    },
    onError: (err) => showError(err, 'Không thể thu tiền kỳ trả góp'),
  });

  const issueEInvoiceMut = useMutation({
    mutationFn: (v: { invoice_id: number }) => financeExtApi.issueEInvoice(Number(v.invoice_id)),
    onSuccess: (res, v) => {
      message.success(`Đã phát hành Hóa đơn điện tử cho Hóa đơn #${v.invoice_id}`);
      setEinvoiceModal(false);
      setEinvoiceResult({
        invoice_id: v.invoice_id,
        einvoice_no: res?.einvoice_no || `EINV-${dayjs().format('YYYYMM')}-${v.invoice_id}`,
        lookup_code: res?.lookup_code || `GM-${v.invoice_id}-${Date.now().toString().slice(-6)}`,
        provider: res?.provider || 'GymMaster Mock E-Invoice',
        issued_at: res?.issued_at || new Date().toISOString(),
        ...res,
      });
    },
    onError: (err) => showError(err, 'Không thể phát hành hóa đơn điện tử'),
  });

  const columns = useMemo<ColumnsType<InstallmentSchedule>>(
    () => [
      {
        title: 'Kỳ thứ',
        dataIndex: 'seq',
        width: 90,
        render: (seq: number) => (
          <Tag color="blue">Kỳ #{seq}</Tag>
        ),
      },
      {
        title: 'Hóa đơn',
        key: 'invoice',
        render: (_, r) => (
          <span className="code-text">
            {r.invoice_no || (r.invoice_id ? `HĐ #${r.invoice_id}` : `Kế hoạch #${r.plan_id}`)}
          </span>
        ),
      },
      {
        title: 'Hội viên',
        key: 'member',
        render: (_, r) => (
          <Space direction="vertical" size={0}>
            <Typography.Text strong>
              {r.member_name || (r.member_id ? `Hội viên #${r.member_id}` : '—')}
            </Typography.Text>
            {r.member_code && (
              <Typography.Text type="secondary" className="code-text" style={{ fontSize: 12 }}>
                {r.member_code}
              </Typography.Text>
            )}
          </Space>
        ),
      },
      {
        title: 'Ngày đến hạn',
        dataIndex: 'due_date',
        render: (v: string, r) => {
          const overdue = installmentIsOverdue(r);
          return (
            <Typography.Text type={overdue ? 'danger' : undefined}>
              {formatDate(v)} {overdue ? '(Quá hạn)' : ''}
            </Typography.Text>
          );
        },
      },
      {
        title: 'Số tiền kỳ này',
        dataIndex: 'amount',
        align: 'right',
        render: (v: number) => <MoneyText value={v} strong />,
      },
      {
        title: 'Đã thu',
        dataIndex: 'paid_amount',
        align: 'right',
        render: (v: number) => <MoneyText value={v || 0} />,
      },
      {
        title: 'Trạng thái',
        dataIndex: 'status',
        // PENDING đã qua ngày đến hạn hiển thị "Quá hạn" (trạng thái suy ra, API vẫn trả PENDING).
        render: (s: InstallmentStatus, r) => (
          <StatusTag status={installmentIsOverdue(r) ? 'OVERDUE' : s} map={INSTALLMENT_STATUS_MAP} />
        ),
      },
      {
        title: '',
        key: 'actions',
        fixed: 'right',
        width: 150,
        render: (_, r) =>
          // Chỉ kỳ còn mở mới thu được – kỳ đã thu/tất toán/hủy (hóa đơn đã thu đủ hoặc đã hủy) ẩn nút.
          canManage && (r.status === 'PENDING' || r.status === 'OVERDUE') ? (
            <Button
              size="small"
              type="primary"
              icon={<CreditCardOutlined />}
              onClick={() => {
                payForm.resetFields();
                payForm.setFieldsValue({ method: 'CASH' });
                setPayTarget(r);
              }}
            >
              Thu tiền kỳ này
            </Button>
          ) : null,
      },
    ],
    [canManage, payForm],
  );

  return (
    <>
      <Row justify="space-between" align="middle" style={{ marginBottom: 12 }}>
        <Col />
        <Col>
          <Space wrap>
            {canManage && (
              <>
                <Button
                  color="orange"
                  variant="solid"
                  icon={<PlusOutlined />}
                  onClick={() => {
                    setPlanModal(true);
                  }}
                >
                  Tạo kế hoạch trả góp
                </Button>
                <Button
                  type="primary"
                  ghost
                  icon={<FileProtectOutlined />}
                  onClick={() => {
                    einvoiceForm.resetFields();
                    setEinvoiceModal(true);
                  }}
                >
                  Xuất Hóa đơn điện tử (E-Invoice)
                </Button>
              </>
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
        searchPlaceholder="Tìm theo mã hóa đơn, hội viên..."
        colTextSearch={6}
        filters={[
          {
            name: 'status',
            placeholder: 'Trạng thái kỳ trả góp',
            options: INSTALLMENT_STATUS_OPTIONS,
            col: 5,
          },
        ]}
      />

      <DataTable<InstallmentSchedule>
        table={table}
        columns={columns}
        emptyText="Chưa có lịch trả góp nào"
      />

      <InstallmentPlanModal open={planModal} onClose={() => setPlanModal(false)} />

      <Modal
        open={!!payTarget}
        title={`Thu tiền kỳ trả góp #${payTarget?.seq} – Số tiền: ${formatMoney(
          (payTarget?.amount ?? 0) - (payTarget?.paid_amount ?? 0),
        )}`}
        okText="Xác nhận thu tiền"
        cancelText="Hủy"
        onOk={() => payForm.submit()}
        onCancel={() => setPayTarget(null)}
        confirmLoading={payScheduleMut.isPending}
        width={460}
        forceRender
      >
        <Form
          form={payForm}
          layout="vertical"
          onFinish={(v) => payScheduleMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Form.Item
            name="method"
            label="Phương thức thanh toán"
            rules={[{ required: true, message: 'Chọn phương thức thanh toán' }]}
          >
            <Select options={COUNTER_PAYMENT_METHOD_OPTIONS} />
          </Form.Item>
          <Form.Item name="note" label="Ghi chú phiếu thu">
            <Input placeholder="VD: Thu kỳ trả góp thứ 2..." />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={einvoiceModal}
        title="Phát hành Hóa đơn điện tử (E-Invoice)"
        okText="Phát hành E-Invoice"
        cancelText="Hủy"
        onOk={() => einvoiceForm.submit()}
        onCancel={() => setEinvoiceModal(false)}
        confirmLoading={issueEInvoiceMut.isPending}
        width={460}
        forceRender
      >
        <Form
          form={einvoiceForm}
          layout="vertical"
          onFinish={(v) => issueEInvoiceMut.mutate(v)}
          style={{ marginTop: 12 }}
        >
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 12 }}
            message="Tích hợp cổng Hóa đơn điện tử (EInvoiceProvider)"
            description="Nhập ID Hóa đơn đã thanh toán để ký số và cấp mã tra cứu hóa đơn điện tử."
          />
          <Form.Item
            name="invoice_id"
            label="ID Hóa đơn"
            rules={[{ required: true, message: 'Vui lòng nhập ID Hóa đơn' }]}
          >
            <InputNumber min={1} style={{ width: '100%' }} placeholder="VD: 1" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={!!einvoiceResult}
        title="Hóa đơn điện tử đã phát hành"
        okText="Đóng"
        cancelButtonProps={{ style: { display: 'none' } }}
        onOk={() => setEinvoiceResult(null)}
        onCancel={() => setEinvoiceResult(null)}
        width={500}
      >
        {einvoiceResult && (
          <Descriptions bordered size="small" column={1} style={{ marginTop: 12 }}>
            <Descriptions.Item label="Số hóa đơn điện tử">
              <Tag color="green" className="code-text">
                {einvoiceResult.einvoice_no}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Mã tra cứu thuế">
              <span className="code-text">{einvoiceResult.lookup_code}</span>
            </Descriptions.Item>
            <Descriptions.Item label="Nhà cung cấp">
              {einvoiceResult.provider || 'VNPT / Viettel Sandbox'}
            </Descriptions.Item>
            <Descriptions.Item label="Thời điểm ký số">
              {formatDateTime(einvoiceResult.issued_at)}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </>
  );
}

export default function FinancePage() {
  const { can } = usePermission();

  // Chỉ hiện tab user có quyền gọi API tương ứng (lễ tân chỉ có cash_session.manage → chỉ thấy Đối soát ca).
  const tabs = [
    {
      key: 'pnl',
      allowed: can([PERM.EXPENSE_VIEW, PERM.REPORT_VIEW]),
      label: (
        <Space size={6}>
          <BarChartOutlined />
          <span>Báo cáo Lãi / Lỗ (P&L)</span>
        </Space>
      ),
      children: <ProfitLossTab />,
    },
    {
      key: 'expenses',
      allowed: can([PERM.EXPENSE_VIEW, PERM.EXPENSE_MANAGE]),
      label: (
        <Space size={6}>
          <WalletOutlined />
          <span>Chi phí vận hành</span>
        </Space>
      ),
      children: <OperatingExpensesTab canManage={can(PERM.EXPENSE_MANAGE)} />,
    },
    {
      key: 'cash-sessions',
      allowed: can([PERM.CASH_SESSION_MANAGE, PERM.EXPENSE_VIEW]),
      label: (
        <Space size={6}>
          <CalculatorOutlined />
          <span>Đối soát ca thu ngân</span>
        </Space>
      ),
      children: <CashSessionsTab />,
    },
    {
      key: 'installments',
      allowed: can([PERM.INVOICE_VIEW, PERM.INVOICE_COLLECT]),
      label: (
        <Space size={6}>
          <CreditCardOutlined />
          <span>Quản lý trả góp & Hóa đơn điện tử</span>
        </Space>
      ),
      children: <InstallmentsAndEInvoiceTab canManage={can(PERM.INVOICE_COLLECT)} />,
    },
  ]
    .filter((t) => t.allowed)
    .map(({ allowed: _allowed, ...t }) => t);

  // Bố cục chuẩn: Card.page-card (padding 20px như các trang khác).
  return (
    <Card variant="borderless" className="page-card">
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <PageTitle
          title="Chi phí, Đối soát ca, Trả góp & Báo cáo P&L"
          subtitle="Quản trị báo cáo Lãi/Lỗ (P&L), duyệt chi phí vận hành, đối soát tiền mặt cuối ca và theo dõi công nợ trả góp"
        />

        {tabs.length === 0 ? (
          <Empty description="Bạn chưa được cấp quyền xem mục tài chính nào" />
        ) : (
          <Tabs defaultActiveKey={tabs[0].key} items={tabs} />
        )}
      </Space>
    </Card>
  );
}
