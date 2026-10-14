import { api, http } from '@/utils/api';
import type { ListParams, Page } from './types';
import { parseFilename, saveFile } from './report';

function normalizePage<T>(res: Page<T> | T[] | null | undefined): Page<T> {
  if (!res) {
    return { items: [], total: 0, total_page: 1, current_page: 1, limit: 20 };
  }
  if (Array.isArray(res)) {
    return {
      items: res,
      total: res.length,
      total_page: 1,
      current_page: 1,
      limit: res.length || 20,
    };
  }
  return {
    items: res.items ?? [],
    total: res.total ?? (res.items?.length || 0),
    total_page: res.total_page ?? 1,
    current_page: res.current_page ?? 1,
    limit: res.limit ?? 20,
  };
}

/* ============================================================================
 * SPRINT 20 – Chấm công & Bảng lương (/admin/payrolls, /admin/attendances, /admin/my-payslips)
 * ============================================================================ */

export type AttendanceMethod = 'QR' | 'MANUAL';
export type PayrollStatus = 'DRAFT' | 'LOCKED' | 'PAID';

export const ATTENDANCE_METHOD_LABELS: Record<AttendanceMethod, string> = {
  QR: 'Quét mã QR',
  MANUAL: 'Chấm công tại quầy',
};

export interface StaffAttendance {
  id: number;
  staff_id: number;
  staff_name?: string;
  staff_code?: string;
  branch_id: number;
  branch_name?: string;
  shift_id?: number | null;
  shift_name?: string;
  check_in_at: string;
  check_out_at?: string | null;
  late_minutes: number;
  early_minutes: number;
  method: AttendanceMethod;
  note: string;
  created_at?: string;
}

export interface AttendanceScanInput {
  staff_id?: number;
  branch_id?: number;
  qr_code?: string;
  method?: AttendanceMethod;
  note?: string;
}

export interface PayrollItem {
  id: number;
  payroll_id: number;
  staff_id: number;
  staff_name?: string;
  staff_code?: string;
  role_name?: string;
  period?: string;
  branch_name?: string;
  status?: PayrollStatus;
  base_salary: number;
  work_days: number;
  ot_hours: number;
  pt_commission: number;
  sales_commission: number;
  bonus: number;
  penalty: number;
  total: number;
  note: string;
  paid_at?: string | null;
  created_at?: string;
}

export interface Payroll {
  id: number;
  branch_id: number;
  branch_name?: string;
  period: string;
  status: PayrollStatus;
  locked_by?: number | null;
  locked_at?: string | null;
  paid_by?: number | null;
  paid_at?: string | null;
  total_amount?: number;
  staff_count?: number;
  items?: PayrollItem[];
  created_at?: string;
  updated_at?: string;
}

export interface PayrollItemUpdateInput {
  item_id?: number;
  staff_id?: number;
  bonus: number;
  penalty: number;
  ot_hours?: number;
  work_days?: number;
  note?: string;
}

class PayrollApi {
  readonly key = 'payrolls';
  readonly attendanceKey = 'staff-attendances';
  readonly myPayslipKey = 'my-payslips';

  async list(params: ListParams & { branch_id?: number; period?: string; status?: PayrollStatus } = {}): Promise<Page<Payroll>> {
    const res = await http.get<Page<Payroll> | Payroll[]>('/admin/payrolls', { params });
    return normalizePage(res);
  }

  get(id: number) {
    return http.get<Payroll>(`/admin/payrolls/${id}`);
  }

  generate(data: { branch_id: number; period: string }) {
    return http.post<Payroll>('/admin/payrolls/generate', data);
  }

  async updateItem(payrollId: number, itemId: number, data: PayrollItemUpdateInput): Promise<Payroll | PayrollItem> {
    try {
      return await http.put<Payroll>(`/admin/payrolls/${payrollId}`, {
        item_id: itemId,
        ...data,
        items: [{ id: itemId, ...data }],
      });
    } catch {
      return await http.put<PayrollItem>(`/admin/payroll-items/${itemId}`, data);
    }
  }

  lock(id: number) {
    return http.post<Payroll>(`/admin/payrolls/${id}/lock`);
  }

  pay(id: number) {
    return http.post<Payroll>(`/admin/payrolls/${id}/pay`);
  }

  async exportCsv(payroll: Payroll): Promise<void> {
    try {
      const res = await api.get(`/admin/payrolls/${payroll.id}/export`, {
        responseType: 'arraybuffer',
      });
      const contentType = String(res.headers?.['content-type'] ?? '');
      if (!contentType.includes('application/json')) {
        const blob = new Blob([res.data as ArrayBuffer], {
          type: contentType || 'text/csv;charset=utf-8;',
        });
        const filename = parseFilename(
          String(res.headers?.['content-disposition'] ?? ''),
          `bang-luong-${payroll.period}-CN${payroll.branch_id}.csv`,
        );
        saveFile({ blob, filename });
        return;
      }
    } catch {
      // Fallback: xuất CSV trực tiếp từ dữ liệu bảng lương trên trình duyệt
    }

    const items = payroll.items ?? [];
    const header = [
      'Mã NV',
      'Họ tên nhân viên',
      'Ngày công',
      'Giờ OT',
      'Lương cơ bản',
      'Hoa hồng PT',
      'Hoa hồng bán gói',
      'Thưởng',
      'Phạt',
      'Thực lĩnh',
      'Ghi chú',
    ];
    const rows = items.map((it) => [
      it.staff_code || `NV#${it.staff_id}`,
      `"${(it.staff_name || '').replace(/"/g, '""')}"`,
      it.work_days,
      it.ot_hours,
      it.base_salary,
      it.pt_commission,
      it.sales_commission,
      it.bonus,
      it.penalty,
      it.total,
      `"${(it.note || '').replace(/"/g, '""')}"`,
    ]);
    const csv = '\uFEFF' + [header.join(','), ...rows.map((r) => r.join(','))].join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    saveFile({
      blob,
      filename: `bang-luong-${payroll.period}-CN${payroll.branch_id}.csv`,
    });
  }

  async listAttendances(
    params: ListParams & { branch_id?: number; staff_id?: number; from?: string; to?: string } = {},
  ): Promise<Page<StaffAttendance>> {
    const res = await http.get<Page<StaffAttendance> | StaffAttendance[]>('/admin/attendances', {
      params,
    });
    return normalizePage(res);
  }

  scanAttendance(data: AttendanceScanInput) {
    return http.post<StaffAttendance>('/admin/attendances/scan', data);
  }

  async myPayslips(): Promise<PayrollItem[]> {
    try {
      const res = await http.get<PayrollItem[] | Page<PayrollItem>>('/admin/my-payslips');
      return Array.isArray(res) ? res : (res?.items ?? []);
    } catch {
      const res = await http.get<PayrollItem[] | Page<PayrollItem>>('/staff/me/payslips');
      return Array.isArray(res) ? res : (res?.items ?? []);
    }
  }
}

export const payrollApi = new PayrollApi();

/* ============================================================================
 * SPRINT 21 – Chi phí, Đối soát ca, Trả góp & Báo cáo P&L
 * ============================================================================ */

export type ExpenseCategoryType = 'FIXED' | 'VARIABLE';
export type ExpenseStatus = 'PENDING' | 'APPROVED' | 'REJECTED';
export type CashSessionStatus = 'OPEN' | 'CLOSED';
export type InstallmentStatus = 'PENDING' | 'PAID' | 'OVERDUE';

export const EXPENSE_CATEGORY_TYPE_LABELS: Record<ExpenseCategoryType, string> = {
  FIXED: 'Chi phí cố định (Fixed)',
  VARIABLE: 'Chi phí biến đổi (Variable)',
};

export const EXPENSE_CATEGORY_TYPE_OPTIONS = (
  Object.keys(EXPENSE_CATEGORY_TYPE_LABELS) as ExpenseCategoryType[]
).map((value) => ({ value, label: EXPENSE_CATEGORY_TYPE_LABELS[value] }));

export interface ExpenseCategory {
  id: number;
  name: string;
  type: ExpenseCategoryType;
  created_at?: string;
}

export interface Expense {
  id: number;
  branch_id: number;
  branch_name?: string;
  category_id: number;
  category_name?: string;
  category_type?: ExpenseCategoryType;
  amount: number;
  spent_at: string;
  description: string;
  attachment_url?: string;
  maintenance_log_id?: number | null;
  created_by?: number | null;
  created_by_name?: string;
  approved_by?: number | null;
  approved_by_name?: string;
  status: ExpenseStatus;
  created_at: string;
}

export interface ExpenseInput {
  branch_id: number;
  category_id: number;
  amount: number;
  spent_at: string;
  description: string;
  attachment_url?: string;
}

export interface CashSession {
  id: number;
  branch_id: number;
  branch_name?: string;
  staff_id: number;
  staff_name?: string;
  opened_at: string;
  closed_at?: string | null;
  opening_cash: number;
  expected_cash: number;
  counted_cash: number;
  difference: number;
  note: string;
  status: CashSessionStatus;
  created_at?: string;
}

export interface InstallmentSchedule {
  id: number;
  plan_id: number;
  invoice_id?: number;
  invoice_code?: string;
  member_id?: number;
  member_name?: string;
  member_code?: string;
  member_phone?: string;
  seq: number;
  total_installments?: number;
  due_date: string;
  amount: number;
  paid_amount: number;
  status: InstallmentStatus;
  paid_at?: string | null;
  created_at?: string;
}

export interface InstallmentPlan {
  id: number;
  invoice_id: number;
  invoice_code?: string;
  member_id: number;
  member_name?: string;
  total: number;
  installments: number;
  schedules?: InstallmentSchedule[];
  created_at?: string;
}

export interface ProfitLossReport {
  from?: string;
  to?: string;
  branch_id?: number | null;
  revenue: {
    total: number;
    membership?: number;
    pt?: number;
    pos?: number;
    locker?: number;
    day_pass?: number;
    other?: number;
  };
  total_revenue?: number;
  pos_cogs: number;
  cogs?: number;
  gross_profit?: number;
  operating_expenses: number;
  expenses_total?: number;
  payroll_cost: number;
  payroll_total?: number;
  net_profit: number;
  margin_pct?: number;
  expenses_by_category?: {
    category_id?: number;
    category_name: string;
    type?: ExpenseCategoryType;
    amount: number;
  }[];
  monthly?: {
    period: string;
    revenue: number;
    cogs: number;
    expenses: number;
    payroll: number;
    net_profit: number;
  }[];
}

export interface EInvoiceResult {
  invoice_id?: number;
  invoice_code?: string;
  einvoice_no?: string;
  lookup_code?: string;
  provider?: string;
  pdf_url?: string;
  issued_at?: string;
  status?: string;
}

class FinanceExtApi {
  readonly pnlKey = 'profit-loss-report';
  readonly expenseKey = 'expenses';
  readonly categoryKey = 'expense-categories';
  readonly cashSessionKey = 'cash-sessions';
  readonly installmentKey = 'installments';

  async profitLoss(
    params: { branch_id?: number; from?: string; to?: string; period?: string } = {},
  ): Promise<ProfitLossReport> {
    const raw = await http.get<Record<string, unknown>>('/admin/reports/profit-loss', { params });
    const revObj = (raw?.revenue && typeof raw.revenue === 'object' ? raw.revenue : {}) as Record<
      string,
      number
    >;
    const totalRevenue = Number(
      revObj.total ?? raw?.total_revenue ?? raw?.revenue ?? 0,
    );
    const posCogs = Number(raw?.pos_cogs ?? raw?.cogs ?? 0);
    const operatingExpenses = Number(
      raw?.operating_expenses ?? raw?.expenses_total ?? raw?.expenses ?? 0,
    );
    const payrollCost = Number(raw?.payroll_cost ?? raw?.payroll_total ?? raw?.payroll ?? 0);
    const netProfit = Number(
      raw?.net_profit ?? totalRevenue - posCogs - operatingExpenses - payrollCost,
    );

    return {
      ...(raw as unknown as ProfitLossReport),
      revenue: {
        total: totalRevenue,
        membership: Number(revObj.membership ?? raw?.membership_revenue ?? 0),
        pt: Number(revObj.pt ?? raw?.pt_revenue ?? 0),
        pos: Number(revObj.pos ?? raw?.pos_revenue ?? 0),
        locker: Number(revObj.locker ?? raw?.locker_revenue ?? 0),
        day_pass: Number(revObj.day_pass ?? 0),
        other: Number(revObj.other ?? 0),
      },
      total_revenue: totalRevenue,
      pos_cogs: posCogs,
      gross_profit: Number(raw?.gross_profit ?? totalRevenue - posCogs),
      operating_expenses: operatingExpenses,
      payroll_cost: payrollCost,
      net_profit: netProfit,
      margin_pct:
        totalRevenue > 0 ? Math.round((netProfit / totalRevenue) * 1000) / 10 : 0,
    };
  }

  async listCategories(): Promise<ExpenseCategory[]> {
    const res = await http.get<ExpenseCategory[] | Page<ExpenseCategory>>(
      '/admin/expense-categories',
    );
    return Array.isArray(res) ? res : (res?.items ?? []);
  }

  createCategory(data: { name: string; type: ExpenseCategoryType }) {
    return http.post<ExpenseCategory>('/admin/expense-categories', data);
  }

  async listExpenses(
    params: ListParams & {
      branch_id?: number;
      category_id?: number;
      status?: ExpenseStatus;
      from?: string;
      to?: string;
    } = {},
  ): Promise<Page<Expense>> {
    const res = await http.get<Page<Expense> | Expense[]>('/admin/expenses', { params });
    return normalizePage(res);
  }

  createExpense(data: ExpenseInput) {
    return http.post<Expense>('/admin/expenses', data);
  }

  updateExpense(id: number, data: Partial<ExpenseInput>) {
    return http.put<Expense>(`/admin/expenses/${id}`, data);
  }

  deleteExpense(id: number) {
    return http.delete(`/admin/expenses/${id}`);
  }

  approveExpense(id: number) {
    return http.post<Expense>(`/admin/expenses/${id}/approve`);
  }

  rejectExpense(id: number) {
    return http.post<Expense>(`/admin/expenses/${id}/reject`);
  }

  async listCashSessions(
    params: ListParams & { branch_id?: number; status?: CashSessionStatus } = {},
  ): Promise<Page<CashSession>> {
    const res = await http.get<Page<CashSession> | CashSession[]>('/admin/cash-sessions', {
      params,
    });
    return normalizePage(res);
  }

  openCashSession(data: { branch_id: number; opening_cash: number; note?: string }) {
    return http.post<CashSession>('/admin/cash-sessions/open', data);
  }

  closeCashSession(id: number, data: { counted_cash: number; note?: string }) {
    return http.post<CashSession>(`/admin/cash-sessions/${id}/close`, data);
  }

  async listInstallments(
    params: ListParams & { branch_id?: number; status?: InstallmentStatus; invoice_id?: number } = {},
  ): Promise<Page<InstallmentSchedule>> {
    const res = await http.get<Page<InstallmentSchedule> | InstallmentSchedule[]>(
      '/admin/installments',
      { params },
    );
    return normalizePage(res);
  }

  createInstallmentPlan(
    invoiceId: number,
    data: { installments: number; first_due_date?: string },
  ) {
    return http.post<InstallmentPlan>(`/admin/invoices/${invoiceId}/installments`, data);
  }

  payInstallment(scheduleId: number, data: { amount?: number; payment_method?: string; note?: string } = {}) {
    return http.post<InstallmentSchedule>(`/admin/installments/${scheduleId}/pay`, {
      payment_method: data.payment_method ?? 'CASH',
      amount: data.amount,
      note: data.note ?? '',
    });
  }

  issueEInvoice(invoiceId: number) {
    return http.post<EInvoiceResult>(`/admin/invoices/${invoiceId}/einvoice`);
  }
}

export const financeExtApi = new FinanceExtApi();
export default payrollApi;
