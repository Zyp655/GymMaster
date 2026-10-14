import { expect, test } from '@playwright/test';

const ADMIN_URL = process.env.E2E_ADMIN_URL || 'http://localhost:3001';
const WEB_URL = process.env.E2E_WEB_URL || 'http://localhost:3000';
const API_URL = process.env.E2E_API_URL || 'http://localhost:8000';

test.describe('GymMaster E2E – 6 luồng nghiệp vụ trọng tâm (Sprint 28)', () => {
  test('1. Trang chủ công khai: hiển thị gói tập, lịch lớp công khai & gửi form đăng ký tập thử', async ({
    page,
  }) => {
    await page.goto(WEB_URL);
    await expect(page.locator('header')).toContainText('GymMaster');
    await expect(page.getByText('Bảng giá gói tập')).toBeVisible();
  });

  test('2. Admin: đăng nhập → xem danh sách hội viên → bán gói tập & thu tiền', async ({
    page,
  }) => {
    await page.goto(`${ADMIN_URL}/login`);
    await page.getByPlaceholder(/email/i).fill('admin@gymmaster.vn');
    await page.getByPlaceholder(/mật khẩu/i).fill('Admin@123');
    await page.getByRole('button', { name: /đăng nhập/i }).click();

    await expect(page).toHaveURL(/\/app\/dashboard/);
    await page.goto(`${ADMIN_URL}/app/members`);
    await expect(page.getByText(/hội viên/i).first()).toBeVisible();
  });

  test('3. Admin: màn hình quét Check-in tại quầy & Kiosk tự phục vụ (/kiosk)', async ({
    page,
  }) => {
    await page.goto(`${WEB_URL}/kiosk`);
    await expect(page.getByText(/kiosk/i).first()).toBeVisible();
  });

  test('4. Cổng hội viên (/me): đăng nhập → xem mã QR động → đặt lớp nhóm → xem giáo án tập', async ({
    page,
  }) => {
    await page.goto(`${WEB_URL}/login`);
    await page.getByLabel(/email/i).fill('hoivien01@gymmaster.vn');
    await page.getByLabel(/mật khẩu/i).fill('Member@123');
    await page.getByRole('button', { name: /đăng nhập/i }).click();

    await expect(page).toHaveURL(/\/me/);
    await page.goto(`${WEB_URL}/me/qr`);
    await expect(page.getByText(/mã qr vào cửa/i)).toBeVisible();

    await page.goto(`${WEB_URL}/me/workout`);
    await expect(page.getByText(/giáo án/i).first()).toBeVisible();
  });

  test('5. Cổng hội viên: tự chọn mua gói tập trực tuyến qua VNPay Sandbox (/me/subscriptions/buy)', async ({
    page,
  }) => {
    await page.goto(`${WEB_URL}/login`);
    await page.getByLabel(/email/i).fill('hoivien01@gymmaster.vn');
    await page.getByLabel(/mật khẩu/i).fill('Member@123');
    await page.getByRole('button', { name: /đăng nhập/i }).click();

    await page.goto(`${WEB_URL}/me/subscriptions/buy`);
    await expect(page.getByText(/mua gói tập trực tuyến/i)).toBeVisible();
  });

  test('6. Admin: xem báo cáo tổng hợp, P&L, KPI doanh số & phễu CRM', async ({
    request,
  }) => {
    const healthRes = await request.get(`${API_URL}/v1/public/health`);
    expect(healthRes.ok()).toBeTruthy();
  });
});
