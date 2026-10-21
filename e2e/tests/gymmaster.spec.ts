import { expect, test, type APIRequestContext, type Page } from '@playwright/test';

/**
 * E2E 6 luồng nghiệp vụ trọng tâm. Mỗi test làm ĐÚNG việc tên của nó mô tả (thao tác UI thật),
 * dùng API chỉ để chuẩn bị dữ liệu / đối chiếu kết quả / dọn dẹp.
 * Dữ liệu tạo ra có tiền tố "E2E-<run>" để không trùng giữa các lần chạy.
 */
const ADMIN_URL = process.env.E2E_ADMIN_URL || 'http://localhost:3001';
const WEB_URL = process.env.E2E_WEB_URL || 'http://localhost:3000';
const API_URL = (process.env.E2E_API_URL || 'http://localhost:8000').replace(/\/$/, '');
const V1 = `${API_URL}/v1`;

const ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL || 'admin@gymmaster.vn';
const MEMBER_EMAIL = process.env.E2E_MEMBER_EMAIL || 'hoivien@gymmaster.vn';
const PASSWORD = process.env.E2E_PASSWORD || 'Admin@123';
const BRANCH_ID = Number(process.env.E2E_BRANCH_ID || 1);

const RUN = `${Date.now().toString(36)}${Math.floor(Math.random() * 1e4)}`;
const TAG = `E2E-${RUN}`;

/** SĐT Việt Nam ngẫu nhiên, duy nhất mỗi lần gọi. */
function uniquePhone(): string {
  return `09${String(Date.now() + Math.floor(Math.random() * 1e6)).slice(-8)}`;
}

function vnd(n: number): string {
  return new Intl.NumberFormat('vi-VN').format(Math.round(n));
}

// ---- API helpers ---------------------------------------------------------------------

async function apiLogin(request: APIRequestContext, email: string): Promise<string> {
  const res = await request.post(`${V1}/auth/login`, { data: { email, password: PASSWORD } });
  expect(res.ok(), `đăng nhập API ${email}`).toBeTruthy();
  const body = await res.json();
  return body.data.access_token as string;
}

async function api<T = unknown>(
  request: APIRequestContext,
  token: string,
  method: 'GET' | 'POST' | 'PUT' | 'DELETE',
  path: string,
  data?: unknown,
): Promise<T> {
  const res = await request.fetch(`${V1}${path}`, {
    method,
    headers: { Authorization: `Bearer ${token}` },
    data,
  });
  const body = await res.json().catch(() => null);
  expect(body?.code, `${method} ${path} → ${res.status()} ${JSON.stringify(body)?.slice(0, 300)}`).toBe(0);
  return body.data as T;
}

interface PublicPlan {
  id: number;
  name: string;
  type: string;
  price: number;
  access_scope: string;
}

/** Gói tập theo thời gian rẻ nhất đang bán công khai (để bán thử / mua thử). */
async function cheapestTimePlan(request: APIRequestContext): Promise<PublicPlan> {
  const res = await request.get(`${V1}/public/plans`);
  const plans = ((await res.json()).data.membership_plans as PublicPlan[]).filter(
    p => p.type === 'TIME' && p.price > 0,
  );
  expect(plans.length, 'cần ít nhất 1 gói TIME đang bán').toBeGreaterThan(0);
  return plans.sort((a, b) => a.price - b.price)[0];
}

/** Gói TIME rẻ nhất vào tập mọi giờ (không phải gói Off-Peak giới hạn khung giờ) – cho check-in. */
async function anytimePlan(
  request: APIRequestContext,
  token: string,
  opts: { classes?: boolean } = {},
): Promise<PublicPlan> {
  const page = await api<{
    items: (PublicPlan & { status: string; access_hours: unknown[] | null; includes_classes: boolean })[];
  }>(
    request,
    token,
    'GET',
    '/admin/membership-plans?status=ACTIVE&limit=100',
  );
  const plans = page.items.filter(
    p =>
      p.status === 'ACTIVE' &&
      p.type === 'TIME' &&
      p.price > 0 &&
      !(p.access_hours ?? []).length &&
      (!opts.classes || p.includes_classes),
  );
  expect(plans.length, 'cần ít nhất 1 gói TIME không giới hạn khung giờ').toBeGreaterThan(0);
  return plans.sort((a, b) => a.price - b.price)[0];
}

/** Tạo hội viên mới + bán gói + thu đủ tiền → hội viên có gói hợp lệ để check-in. */
async function createPaidMember(
  request: APIRequestContext,
  token: string,
  label: string,
  opts: { classes?: boolean } = {},
) {
  const member = await api<{ id: number; member_code: string; full_name: string }>(
    request,
    token,
    'POST',
    '/admin/members',
    { full_name: `${TAG} ${label}`, phone: uniquePhone(), home_branch_id: BRANCH_ID, gender: 'OTHER' },
  );
  const plan = await anytimePlan(request, token, opts);
  const sub = await api<{ invoice?: { id: number; total: number } ; invoice_id?: number }>(
    request,
    token,
    'POST',
    '/admin/subscriptions',
    { member_id: member.id, plan_id: plan.id, branch_id: BRANCH_ID, note: TAG },
  );
  const invoiceId = sub.invoice?.id ?? sub.invoice_id;
  expect(invoiceId, 'bán gói phải sinh hóa đơn').toBeTruthy();
  const inv = await api<{ due: number }>(request, token, 'GET', `/admin/invoices/${invoiceId}`);
  if (inv.due > 0) {
    await api(request, token, 'POST', `/admin/invoices/${invoiceId}/payments`, {
      amount: inv.due,
      method: 'CASH',
      note: TAG,
    });
  }
  return member;
}

// ---- UI helpers ----------------------------------------------------------------------

async function loginMember(page: Page, email = MEMBER_EMAIL) {
  await page.goto(`${WEB_URL}/login`);
  await page.getByLabel(/email/i).fill(email);
  // exact: tránh trùng nút "Hiện mật khẩu"
  await page.getByRole('textbox', { name: 'Mật khẩu', exact: true }).fill(PASSWORD);
  await page.getByRole('button', { name: /đăng nhập/i }).click();
  await expect(page).toHaveURL(/\/me/, { timeout: 20_000 });
}

async function loginAdmin(page: Page) {
  await page.goto(`${ADMIN_URL}/login`);
  await page.getByPlaceholder(/email/i).fill(ADMIN_EMAIL);
  await page.getByPlaceholder(/mật khẩu/i).first().fill(PASSWORD);
  await page.getByRole('button', { name: /đăng nhập/i }).click();
  await expect(page).toHaveURL(/\/app\//, { timeout: 20_000 });
}

test.describe('GymMaster E2E – 6 luồng nghiệp vụ trọng tâm', () => {
  test('1. Trang chủ: gói tập, lịch lớp (số chỗ khớp API), gửi form tập thử → lead hiện ở admin', async ({
    page,
    request,
  }) => {
    const adminToken = await apiLogin(request, ADMIN_EMAIL);
    await page.goto(WEB_URL);

    // Bảng giá có ít nhất một gói.
    const pricing = page.locator('#bang-gia');
    await expect(pricing.getByRole('heading', { level: 3 }).first()).toBeVisible();

    // Lịch lớp: huy hiệu "Còn x/y chỗ" / "Hết chỗ" phải khớp capacity − booked_count từ API.
    const schedRes = await request.get(`${V1}/public/class-schedules`);
    const schedules = (await schedRes.json()).data as {
      capacity: number;
      booked_count: number;
    }[];
    if (schedules.length > 0) {
      const first = schedules[0];
      const free = Math.max(0, first.capacity - first.booked_count);
      const expected =
        free > 0 ? `Còn ${free}/${first.capacity} chỗ` : `Hết chỗ (${first.capacity}/${first.capacity})`;
      await expect(page.locator('#lich-lop').getByText(expected).first()).toBeVisible();
      // Không được hiển thị "Hết chỗ" cho mọi lớp khi API còn chỗ (lỗi available_slots cũ).
      if (schedules.some(s => s.capacity > s.booked_count)) {
        await expect(page.locator('#lich-lop').getByText(/^Còn \d+\/\d+ chỗ$/).first()).toBeVisible();
      }
    }

    // Gửi form đăng ký tập thử với dữ liệu duy nhất.
    const fullName = `${TAG} Khách tập thử`;
    const phone = uniquePhone();
    const trial = page.locator('#tap-thu');
    await trial.getByLabel('Họ và tên').fill(fullName);
    await trial.getByLabel('Số điện thoại').fill(phone);
    await trial.getByRole('button', { name: 'Gửi đăng ký tập thử' }).click();
    await expect(trial.getByText('Đăng ký tập thử thành công!')).toBeVisible({ timeout: 15_000 });

    // Lead xuất hiện trong API và màn hình CRM của admin.
    const leads = await api<{ items: { id: number; full_name: string; phone: string; source: string }[] }>(
      request,
      adminToken,
      'GET',
      `/admin/leads?keyword=${phone}`,
    );
    const lead = leads.items.find(l => l.phone === phone);
    expect(lead, 'lead vừa gửi phải có trong CRM').toBeTruthy();
    expect(lead!.full_name).toBe(fullName);
    expect(lead!.source).toBe('WEBSITE');

    await loginAdmin(page);
    await page.goto(`${ADMIN_URL}/app/leads?keyword=${phone}`);
    await expect(page.getByText(fullName).first()).toBeVisible({ timeout: 15_000 });

    await api(request, adminToken, 'DELETE', `/admin/leads/${lead!.id}`);
  });

  test('2. Admin: bán gói tập cho hội viên mới & thu tiền trên màn hình hóa đơn', async ({
    page,
    request,
  }) => {
    const token = await apiLogin(request, ADMIN_EMAIL);
    const member = await api<{ id: number; full_name: string }>(request, token, 'POST', '/admin/members', {
      full_name: `${TAG} Hội viên mua gói`,
      phone: uniquePhone(),
      home_branch_id: BRANCH_ID,
      gender: 'OTHER',
    });
    const plan = await cheapestTimePlan(request);
    const sub = await api<{ id: number; invoice?: { id: number }; invoice_id?: number }>(
      request,
      token,
      'POST',
      '/admin/subscriptions',
      { member_id: member.id, plan_id: plan.id, branch_id: BRANCH_ID, note: TAG },
    );
    const invoiceId = sub.invoice?.id ?? sub.invoice_id;
    const before = await api<{ invoice_no: string; status: string; due: number; total: number }>(
      request,
      token,
      'GET',
      `/admin/invoices/${invoiceId}`,
    );
    expect(before.status).toBe('UNPAID');
    expect(before.due).toBeGreaterThan(0);

    // Thu tiền qua giao diện admin.
    await loginAdmin(page);
    await page.goto(`${ADMIN_URL}/app/invoices/${invoiceId}`);
    await expect(page.getByText(before.invoice_no).first()).toBeVisible({ timeout: 15_000 });
    await page.getByRole('button', { name: 'Thu tiền' }).click();
    const modal = page.getByRole('dialog', { name: new RegExp(`Thu tiền hóa đơn ${before.invoice_no}`) });
    await expect(modal).toBeVisible();
    await modal.getByRole('button', { name: 'Thu tiền' }).click();
    await expect(page.getByText(`Đã thu đủ hóa đơn ${before.invoice_no}`)).toBeVisible({ timeout: 15_000 });

    const after = await api<{ status: string; due: number; paid_amount: number }>(
      request,
      token,
      'GET',
      `/admin/invoices/${invoiceId}`,
    );
    expect(after.status).toBe('PAID');
    expect(after.due).toBe(0);
    expect(after.paid_amount).toBe(before.total);
    // Số tiền hiển thị trên chi tiết hóa đơn khớp tổng tiền.
    await expect(page.getByText(vnd(before.total)).first()).toBeVisible();
  });

  test('3. Check-in tại quầy (bộ đếm tăng) & Kiosk tự phục vụ (/kiosk) với mã QR động', async ({
    page,
    request,
  }) => {
    const token = await apiLogin(request, ADMIN_EMAIL);
    const counterMember = await createPaidMember(request, token, 'Check-in quầy');
    const kioskMember = await createPaidMember(request, token, 'Check-in kiosk');

    // --- Quầy lễ tân: dán mã QR → MỜI VÀO, thống kê "allowed" hôm nay tăng 1.
    const statsBefore = await api<{ allowed: number }>(
      request,
      token,
      'GET',
      `/admin/checkins/stats?branch_id=${BRANCH_ID}`,
    );
    const qr1 = await api<{ token: string }>(request, token, 'GET', `/admin/members/${counterMember.id}/qr`);
    await loginAdmin(page);
    await page.goto(`${ADMIN_URL}/app/checkin`);
    const qrInput = page.getByPlaceholder('Quét hoặc dán mã QR rồi nhấn Enter');
    await page.waitForLoadState('networkidle');
    // Gõ như máy quét cầm tay (ký tự liên tiếp + Enter).
    await qrInput.click();
    await qrInput.pressSequentially(qr1.token, { delay: 5 });
    await qrInput.press('Enter');
    await expect(page.getByText('MỜI VÀO')).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText(counterMember.full_name).first()).toBeVisible();
    await expect
      .poll(
        async () =>
          (
            await api<{ allowed: number }>(
              request,
              token,
              'GET',
              `/admin/checkins/stats?branch_id=${BRANCH_ID}`,
            )
          ).allowed,
      )
      .toBe(statsBefore.allowed + 1);

    // --- Kiosk: thiết bị mới (khóa riêng) + QR hội viên → "MỜI VÀO CỬA".
    const device = await api<{ device: { id: number }; raw_api_key: string }>(
      request,
      token,
      'POST',
      '/admin/access-devices',
      { branch_id: BRANCH_ID, code: `E2E${RUN}`.slice(0, 20).toUpperCase(), name: `${TAG} Kiosk`, type: 'KIOSK' },
    );
    try {
      await page.goto(`${WEB_URL}/kiosk`);
      await page.evaluate(key => localStorage.setItem('gymmaster:kiosk_device_key', key), device.raw_api_key);
      await page.reload();
      // Heartbeat thành công → header hiển thị tên thiết bị vừa tạo.
      await expect(page.getByText(`${TAG} Kiosk`, { exact: false })).toBeVisible({ timeout: 15_000 });
      const qr2 = await api<{ token: string }>(request, token, 'GET', `/admin/members/${kioskMember.id}/qr`);
      const kioskInput = page.getByPlaceholder('Quét hoặc nhập mã tại đây rồi nhấn Enter...');
      await kioskInput.fill(qr2.token);
      await page.getByRole('button', { name: 'Xác nhận Check-in' }).click();
      await expect(page.getByText('MỜI VÀO CỬA')).toBeVisible({ timeout: 15_000 });
    } finally {
      await api(request, token, 'DELETE', `/admin/access-devices/${device.device.id}`);
    }
  });

  test('4. Cổng hội viên: QR động hiển thị → đặt & hủy lớp nhóm → ghi nhật ký tập (mức tạ đúng)', async ({
    page,
    request,
  }) => {
    // Hội viên mới (gói có lớp nhóm) + tài khoản cổng riêng → không phụ thuộc trạng thái hội viên demo.
    const adminToken = await apiLogin(request, ADMIN_EMAIL);
    const member = await createPaidMember(request, adminToken, 'Cổng hội viên', { classes: true });
    const email = `${RUN}.portal@e2e.gymmaster.vn`.toLowerCase();
    await api(request, adminToken, 'POST', `/admin/members/${member.id}/portal-account`, {
      email,
      password: PASSWORD,
    });
    await loginMember(page, email);
    const memberToken = await apiLogin(request, email);

    // QR động: có mã SVG và đếm ngược làm mới.
    await page.goto(`${WEB_URL}/me/qr`);
    await expect(page.getByRole('heading', { name: /mã qr vào cửa/i })).toBeVisible();
    await expect(page.locator('svg[aria-label]').first()).toBeVisible();
    await expect(page.getByText(/làm mới|giây/i).first()).toBeVisible();

    // Đặt lớp nhóm: chọn buổi đầu tiên còn đặt được, rồi hủy để trả chỗ.
    await page.goto(`${WEB_URL}/me/classes`);
    const bookBtn = page.locator('button:visible', { hasText: /^\s*Đặt chỗ\s*$/ }).first();
    // API cho biết có buổi đặt được không → khi có, nút "Đặt chỗ" BẮT BUỘC phải hiện trên UI.
    const classes = await api<{ items: { bookable: boolean; my_booking: unknown }[] }>(
      request,
      memberToken,
      'GET',
      '/member/classes',
    );
    const canBook = classes.items.some(c => c.bookable && !c.my_booking);
    if (canBook) {
      await expect(bookBtn).toBeVisible({ timeout: 15_000 });
      await bookBtn.click();
      await expect(page.getByText('Đặt chỗ lớp học thành công')).toBeVisible({ timeout: 15_000 });
      await page.goto(`${WEB_URL}/me/bookings`);
      const cancel = page.getByRole('button', { name: 'Hủy đặt chỗ' }).first();
      await expect(cancel).toBeVisible();
      await cancel.click();
      await page.getByRole('alertdialog').getByRole('button', { name: 'Xác nhận hủy' }).click();
      await expect(page.getByText('Đã hủy lượt đặt chỗ')).toBeVisible({ timeout: 15_000 });
    } else {
      test.info().annotations.push({
        type: 'note',
        description: 'Không có buổi lớp nào còn đặt được trong phạm vi gói của hội viên demo – bỏ qua bước đặt lớp.',
      });
    }

    // Nhật ký tập: mức tạ nhập trên form phải được lưu đúng (API trường `weight`).
    const note = `${TAG} nhật ký`;
    await page.goto(`${WEB_URL}/me/workout`);
    await page.getByRole('button', { name: 'Ghi nhận bài tập tự chọn' }).click();
    const dialog = page.getByRole('dialog');
    await dialog.getByLabel('Mức tạ thực tế (kg)').fill('62.5');
    await dialog.getByLabel('Ghi chú cảm nhận / kỹ thuật').fill(note);
    await dialog.getByRole('button', { name: 'Lưu nhật ký tập' }).click();
    await expect(page.getByText('Đã lưu nhật ký tập luyện')).toBeVisible({ timeout: 15_000 });
    const row = page.locator('tr', { hasText: note });
    await expect(row).toContainText('62.5 kg');

    const logs = await api<{ logs: { note: string; weight: number; performed_at: string }[] }>(
      request,
      memberToken,
      'GET',
      '/member/workout-logs',
    );
    const saved = logs.logs.find(l => l.note === note);
    expect(saved, 'nhật ký vừa ghi phải có trong API').toBeTruthy();
    expect(saved!.weight).toBe(62.5);
  });

  test('5. Cổng hội viên: mua gói tập trực tuyến → chuyển sang cổng VNPay Sandbox', async ({
    page,
    request,
  }) => {
    // Không gọi ra Internet: chặn trang VNPay, chỉ kiểm tra URL chuyển hướng hợp lệ.
    await page.route(/vnpayment\.vn/, route =>
      route.fulfill({ status: 200, contentType: 'text/html', body: '<h1>VNPay sandbox (giả lập E2E)</h1>' }),
    );
    await loginMember(page);
    const plan = await cheapestTimePlan(request);

    await page.goto(`${WEB_URL}/me/subscriptions/buy`);
    await expect(page.getByRole('heading', { name: /mua gói tập trực tuyến/i })).toBeVisible();
    await page.getByRole('button', { name: new RegExp(plan.name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')) }).first().click();

    // Đọc kết quả checkout trước khi trang điều hướng sang VNPay (body không còn đọc được sau đó).
    let body: { code: number; data?: { invoice_id?: number; payment_url?: string } } | null = null;
    await page.route(/\/member\/subscriptions\/checkout/, async route => {
      const response = await route.fetch();
      body = await response.json();
      await route.fulfill({ response, json: body });
    });
    await page.getByRole('button', { name: 'Thanh toán qua VNPay' }).click();
    await expect.poll(() => body?.code, { timeout: 20_000 }).toBe(0);

    await page.waitForURL(/vnpayment\.vn/, { timeout: 20_000 });
    const url = new URL(page.url());
    expect(url.searchParams.get('vnp_TxnRef')).toBeTruthy();
    expect(Number(url.searchParams.get('vnp_Amount'))).toBe(Math.round(plan.price) * 100);
    expect(url.searchParams.get('vnp_SecureHash')).toBeTruthy();

    // Dọn dẹp: hủy hóa đơn chưa thanh toán vừa tạo.
    const invoiceId = (body as { data?: { invoice_id?: number } } | null)?.data?.invoice_id;
    if (invoiceId) {
      const adminToken = await apiLogin(request, ADMIN_EMAIL);
      await request.post(`${V1}/admin/invoices/${invoiceId}/void`, {
        headers: { Authorization: `Bearer ${adminToken}` },
        data: { reason: `${TAG} dọn dẹp E2E` },
      });
    }
  });

  test('6. Admin: báo cáo tổng hợp, P&L, KPI hiển thị số liệu thật từ API', async ({
    page,
    request,
  }) => {
    const token = await apiLogin(request, ADMIN_EMAIL);
    const overview = await api<{ members: { active: number }; revenue: { month: number } }>(
      request,
      token,
      'GET',
      '/admin/reports/overview',
    );

    const failed: string[] = [];
    const okReports = new Set<string>();
    page.on('response', r => {
      const m = /\/v1\/admin\/reports\/([a-z-]+)/.exec(r.url());
      if (!m) return;
      if (r.status() >= 400) failed.push(`${r.status()} ${r.url()}`);
      else okReports.add(m[1]);
    });

    await loginAdmin(page);

    // Dashboard: số hội viên đang hoạt động khớp API.
    await page.goto(`${ADMIN_URL}/app/dashboard`);
    await expect(page.getByText(vnd(overview.members.active)).first()).toBeVisible({ timeout: 20_000 });

    // Báo cáo tổng hợp & Tài chính (P&L): tải xong, có số tiền ≠ 0, không có thẻ lỗi.
    for (const path of ['/app/reports', '/app/finance']) {
      await page.goto(`${ADMIN_URL}${path}`);
      await page.waitForLoadState('networkidle');
      await expect(page.locator('.ant-result-error, .ant-result-500')).toHaveCount(0);
      await expect(page.getByText(/[1-9][\d.]*\s?₫/).first()).toBeVisible({ timeout: 20_000 });
    }
    expect(okReports.has('profit-loss'), `P&L phải được tải: ${[...okReports]}`).toBeTruthy();

    // KPI: tab chỉ tiêu doanh số gọi /reports/kpi thành công và hiển thị nội dung.
    await page.goto(`${ADMIN_URL}/app/kpi-analytics`);
    const kpiResp = page.waitForResponse(r => r.url().includes('/v1/admin/reports/kpi'));
    await page.getByRole('tab', { name: /Chỉ tiêu doanh số KPI/ }).click();
    expect((await kpiResp).status()).toBe(200);
    await expect(page.locator('.ant-result-error, .ant-result-500')).toHaveCount(0);
    expect(failed, 'API báo cáo không được lỗi').toEqual([]);
  });
});
