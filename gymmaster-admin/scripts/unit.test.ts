import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { AxiosError } from 'axios';

import { auditActionLabel, auditEntityLabel } from '../src/api/audit.ts';
import { NOTIFICATION_TYPE_LABEL, toAppLink } from '../src/api/notification.ts';
import { PERM, PERMISSION_MODULE_LABELS, ROLE, ROLE_LABELS } from '../src/constants/permissions.ts';
import {
  ACCESS_DEVICE_STATUS_MAP,
  CASH_SESSION_STATUS_MAP,
  CONTRACT_STATUS_MAP,
  DATA_REQUEST_STATUS_MAP,
  EQUIPMENT_STATUS_MAP,
  EXPENSE_STATUS_MAP,
  FREEZE_STATUS_MAP,
  INSTALLMENT_STATUS_MAP,
  INVOICE_STATUS_MAP,
  LEAD_STATUS_MAP,
  LEAVE_STATUS_MAP,
  LOCKER_STATUS_MAP,
  MAINTENANCE_STATUS_MAP,
  PAYMENT_METHOD_MAP,
  PAYROLL_STATUS_MAP,
  STATUS_PRESETS,
  STOCK_TX_TYPE_MAP,
  SUBSCRIPTION_STATUS_MAP,
  WORKOUT_PLAN_STATUS_MAP,
} from '../src/constants/status.ts';
import { EXPORT_LABEL, heatColor, rangeParams, shortMoney } from '../src/routes/reports/types.ts';
import { ApiError, createApiClient, ErrorCode } from '../src/utils/apiClient.ts';
import { getErrorMessage, isFormError } from '../src/utils/errors.ts';
import {
  formatDate,
  formatDateTime,
  formatMoney,
  formatNumber,
  formatPhone,
  normalizeText,
  toApiDate,
} from '../src/utils/format.ts';

describe('gymmaster-admin: utils/format', () => {
  it('formatNumber & formatMoney định dạng tiền tệ Việt Nam chuẩn xác', () => {
    assert.equal(formatNumber(1_500_000), '1.500.000');
    assert.equal(formatNumber(0), '0');
    assert.equal(formatNumber(null), '—');
    assert.equal(formatNumber(undefined), '—');
    assert.equal(formatNumber(''), '—');
    assert.equal(formatNumber('abc'), '—');

    assert.equal(formatMoney(1_500_000), '1.500.000 ₫');
    assert.equal(formatMoney(4800000, ' VND'), '4.800.000 VND');
    assert.equal(formatMoney(null), '—');
  });

  it('formatDate, formatDateTime, toApiDate xử lý ngày giờ và giá trị rỗng', () => {
    assert.equal(formatDate('2026-10-14'), '14/10/2026');
    assert.equal(formatDate(null), '—');
    assert.equal(formatDate('invalid-date'), '—');
    assert.equal(formatDateTime('2026-10-14T08:30:00'), '14/10/2026 08:30');
    assert.equal(toApiDate('2026-10-14T08:30:00'), '2026-10-14');
    assert.equal(toApiDate(null), null);
    assert.equal(toApiDate('not-a-date'), null);
  });

  it('formatPhone & normalizeText chuẩn hóa SĐT và tiếng Việt không dấu', () => {
    assert.equal(formatPhone('0901234567'), '0901 234 567');
    assert.equal(formatPhone('02838221111'), '028 3822 1111');
    assert.equal(formatPhone(null), '—');
    assert.equal(normalizeText('Đặng Hoàng Phượng'), 'dang hoang phuong');
  });
});

describe('gymmaster-admin: utils/errors', () => {
  it('getErrorMessage ưu tiên thông điệp tiếng Việt từ backend và ẩn thông điệp kỹ thuật', () => {
    const vnErr = new ApiError(
      'Khung giờ này trùng với lịch dạy lớp của HLV',
      ErrorCode.CONFLICT,
      409,
    );
    assert.equal(getErrorMessage(vnErr), 'Khung giờ này trùng với lịch dạy lớp của HLV');

    const techErr = new ApiError('CONFLICT', ErrorCode.CONFLICT, 409);
    assert.equal(getErrorMessage(techErr), 'Dữ liệu bị trùng hoặc đang được sử dụng');

    const internalErr = new ApiError('Internal Error', ErrorCode.INTERNAL, 500);
    assert.equal(getErrorMessage(internalErr), 'Máy chủ gặp sự cố, vui lòng thử lại sau');

    assert.equal(getErrorMessage(new Error('Lỗi cục bộ')), 'Lỗi cục bộ');
    assert.equal(getErrorMessage(null, 'Fallback mặc định'), 'Fallback mặc định');
  });

  it('isFormError phân biệt đúng lỗi form (422, 400, 409) và lỗi hệ thống', () => {
    assert.equal(isFormError(new ApiError('Sai', ErrorCode.VALIDATION, 422)), true);
    assert.equal(isFormError(new ApiError('Trùng', ErrorCode.CONFLICT, 409)), true);
    assert.equal(isFormError(new ApiError('500', ErrorCode.INTERNAL, 500)), false);
  });
});

describe('gymmaster-admin: utils/apiClient interceptors (in-memory adapter)', () => {
  it('gắn Bearer token + X-Branch-Id, unwrap envelope và xử lý lỗi 40100 / 50000', async () => {
    let accessToken: string | null = 'tok-123';
    let refreshToken: string | null = 'ref-123';
    let expiredCount = 0;
    let serverErrCount = 0;

    const client = createApiClient({
      baseURL: 'http://gymmaster.local/v1',
      tokens: {
        getAccessToken: () => accessToken,
        getRefreshToken: () => refreshToken,
        setTokens: (a, r) => {
          accessToken = a;
          refreshToken = r;
        },
        clear: () => {
          accessToken = null;
          refreshToken = null;
        },
      },
      getBranchId: () => 2,
      onSessionExpired: () => {
        expiredCount += 1;
      },
      onServerError: () => {
        serverErrCount += 1;
      },
    });

    client.instance.defaults.adapter = async (config) => {
      const respond = (status: number, data: unknown) => {
        const res = {
          data,
          status,
          statusText: status === 200 ? 'OK' : 'Error',
          headers: {},
          config,
        };
        if (status >= 400) {
          throw new AxiosError('Request failed', 'ERR_BAD_RESPONSE', config, null, res);
        }
        return res;
      };

      if (config.url === '/ok') {
        assert.equal(config.headers.get('Authorization'), 'Bearer tok-123');
        assert.equal(config.headers.get('X-Branch-Id'), '2');
        return respond(200, { code: 0, message: 'OK', data: { hello: 'world' } });
      }
      if (config.url === '/biz-err') {
        return respond(409, { code: ErrorCode.CONFLICT, message: 'Mã đã tồn tại', data: null });
      }
      if (config.url === '/server-err') {
        return respond(500, { code: ErrorCode.INTERNAL, message: 'Lỗi DB', data: null });
      }
      return respond(401, { code: ErrorCode.UNAUTHORIZED, message: 'Chưa đăng nhập', data: null });
    };

    const okRes = await client.http.get<{ hello: string }>('/ok');
    assert.deepEqual(okRes, { hello: 'world' });

    await assert.rejects(
      () => client.http.get('/biz-err'),
      (err: unknown) => err instanceof ApiError && err.code === ErrorCode.CONFLICT,
    );

    await assert.rejects(
      () => client.http.get('/server-err'),
      (err: unknown) => err instanceof ApiError && err.handled === true,
    );
    assert.equal(serverErrCount, 1);

    await assert.rejects(
      () => client.http.get('/unauth'),
      (err: unknown) => err instanceof ApiError && err.code === ErrorCode.UNAUTHORIZED,
    );
    assert.equal(expiredCount, 1);
    assert.equal(accessToken, null);
  });
});

describe('gymmaster-admin: api/notification & api/audit', () => {
  it('toAppLink chuyển đổi đúng các đường dẫn thông báo từ backend sang route SPA', () => {
    assert.equal(toAppLink('/admin/members/15'), '/app/members/15');
    assert.equal(toAppLink('/app/reports'), '/app/reports');
    assert.equal(toAppLink('/inventory/stock'), '/app/inventory');
    assert.equal(
      toAppLink('/admin/subscriptions?status=expiring'),
      '/app/subscriptions?expiring_in_days=7',
    );
    assert.equal(toAppLink('/admin/pt-contracts/42'), '/app/pt-contracts?cid=42');
    assert.equal(toAppLink('/admin/pt-sessions/99'), '/app/pt-sessions');
    assert.equal(toAppLink('/member/subscriptions'), null);
    assert.equal(toAppLink('https://gymmaster.vn'), null);
    assert.equal(toAppLink(''), null);
    assert.equal(toAppLink(null), null);
  });

  it('auditActionLabel & auditEntityLabel trả nhãn tiếng Việt hoặc fallback mã gốc', () => {
    assert.equal(auditActionLabel('SELL'), 'Bán');
    assert.equal(auditActionLabel('FREEZE_APPROVE'), 'Duyệt bảo lưu');
    assert.equal(auditActionLabel('CUSTOM_ACTION'), 'CUSTOM_ACTION');

    assert.equal(auditEntityLabel('subscription'), 'Đăng ký gói');
    assert.equal(auditEntityLabel('trainer_commission'), 'Hoa hồng HLV');
    assert.equal(auditEntityLabel('custom_entity'), 'custom_entity');
  });
});

describe('gymmaster-admin: reports/types & constants', () => {
  it('heatColor, shortMoney, rangeParams hoạt động đúng trên các biên giá trị', () => {
    assert.equal(heatColor(0, 100), '#F1F5F9');
    assert.equal(heatColor(10, 0), '#F1F5F9');
    assert.equal(heatColor(100, 100), '#0A3150');
    assert.notEqual(heatColor(50, 100), '#F1F5F9');

    assert.match(shortMoney(1_500_000_000), /tỷ$/);
    assert.match(shortMoney(2_500_000), /tr$/);
    assert.equal(shortMoney(50_000), '50k');
    assert.equal(shortMoney(500), '500');

    assert.deepEqual(
      rangeParams({ branchId: 2, from: '2026-10-01', to: '2026-10-14', groupBy: 'day' }),
      { branch_id: 2, from: '2026-10-01', to: '2026-10-14' },
    );
    assert.equal(EXPORT_LABEL.revenue, 'Doanh thu');
    assert.equal(EXPORT_LABEL.invoices, 'Hóa đơn');
  });

  it('constants/permissions & constants/status đầy đủ mã quyền và trạng thái nghiệp vụ (Sprint 1–27)', () => {
    assert.equal(PERM.POS_SELL, 'pos.sell');
    assert.equal(PERM.REPORT_EXPORT, 'report.export');
    assert.equal(PERM.WORKOUT_VIEW, 'workout.view');
    assert.equal(PERM.WORKOUT_MANAGE, 'workout.manage');
    assert.equal(PERM.REVIEW_VIEW, 'review.view');
    assert.equal(PERM.EQUIPMENT_VIEW, 'equipment.view');
    assert.equal(PERM.EQUIPMENT_MANAGE, 'equipment.manage');
    assert.equal(PERM.LEAD_VIEW, 'lead.view');
    assert.equal(PERM.LEAD_MANAGE, 'lead.manage');
    assert.equal(PERM.DEVICE_VIEW, 'device.view');
    assert.equal(PERM.DEVICE_MANAGE, 'device.manage');
    assert.equal(PERM.LOCKER_VIEW, 'locker.view');
    assert.equal(PERM.LOCKER_MANAGE, 'locker.manage');
    assert.equal(PERM.PAYROLL_VIEW, 'payroll.view');
    assert.equal(PERM.PAYROLL_MANAGE, 'payroll.manage');
    assert.equal(PERM.EXPENSE_VIEW, 'expense.view');
    assert.equal(PERM.EXPENSE_MANAGE, 'expense.manage');
    assert.equal(PERM.CASH_SESSION_MANAGE, 'cash_session.manage');
    assert.equal(PERM.LOYALTY_VIEW, 'loyalty.view');
    assert.equal(PERM.LOYALTY_MANAGE, 'loyalty.manage');
    assert.equal(PERM.ENGAGEMENT_VIEW, 'engagement.view');
    assert.equal(PERM.ENGAGEMENT_MANAGE, 'engagement.manage');
    assert.equal(PERM.CONTRACT_MANAGE, 'contract.manage');
    assert.equal(PERM.LEAVE_MANAGE, 'leave.manage');
    assert.equal(PERM.HOLIDAY_MANAGE, 'holiday.manage');
    assert.equal(PERM.KPI_MANAGE, 'kpi.manage');
    assert.equal(PERM.PRIVACY_MANAGE, 'privacy.manage');
    assert.equal(PERM.MEMBER_VIEW_SENSITIVE, 'member.view_sensitive');

    assert.equal(PERMISSION_MODULE_LABELS.workout, 'Giáo án & bài tập');
    assert.equal(PERMISSION_MODULE_LABELS.review, 'Đánh giá');
    assert.equal(PERMISSION_MODULE_LABELS.equipment, 'Thiết bị & bảo trì');
    assert.equal(PERMISSION_MODULE_LABELS.lead, 'Khách tiềm năng (CRM)');
    assert.equal(PERMISSION_MODULE_LABELS.device, 'Thiết bị ra vào & Kiosk');
    assert.equal(PERMISSION_MODULE_LABELS.locker, 'Tủ đồ & dịch vụ');
    assert.equal(PERMISSION_MODULE_LABELS.payroll, 'Chấm công & bảng lương');
    assert.equal(PERMISSION_MODULE_LABELS.expense, 'Chi phí & P&L');
    assert.equal(PERMISSION_MODULE_LABELS.loyalty, 'Điểm thưởng & thử thách');
    assert.equal(PERMISSION_MODULE_LABELS.privacy, 'Bảo mật & dữ liệu cá nhân');
    assert.equal(ROLE_LABELS[ROLE.OWNER], 'Chủ trung tâm');

    assert.ok(SUBSCRIPTION_STATUS_MAP.TRANSFERRED);
    assert.ok(FREEZE_STATUS_MAP.APPROVED);
    assert.ok(INVOICE_STATUS_MAP.PARTIAL);
    assert.ok(STOCK_TX_TYPE_MAP.SALE_RETURN);
    assert.equal(STATUS_PRESETS.VNPAY.label, 'VNPay Online');
    assert.equal(PAYMENT_METHOD_MAP.VNPAY.label, 'VNPay Online');
    assert.ok(WORKOUT_PLAN_STATUS_MAP.ACTIVE);
    assert.ok(EQUIPMENT_STATUS_MAP.BROKEN);
    assert.ok(MAINTENANCE_STATUS_MAP.IN_PROGRESS);
    assert.ok(LEAD_STATUS_MAP.CONVERTED);
    assert.ok(ACCESS_DEVICE_STATUS_MAP.ACTIVE);
    assert.ok(LOCKER_STATUS_MAP.RENTED);
    assert.ok(PAYROLL_STATUS_MAP.LOCKED);
    assert.ok(EXPENSE_STATUS_MAP.APPROVED);
    assert.ok(CASH_SESSION_STATUS_MAP.CLOSED);
    assert.ok(INSTALLMENT_STATUS_MAP.OVERDUE);
    assert.ok(CONTRACT_STATUS_MAP.SIGNED);
    assert.ok(LEAVE_STATUS_MAP.APPROVED);
    assert.ok(DATA_REQUEST_STATUS_MAP.DONE);

    assert.equal(NOTIFICATION_TYPE_LABEL.PAYMENT_SUCCESS, 'Thanh toán thành công');
    assert.equal(NOTIFICATION_TYPE_LABEL.REVIEW_PROMPT, 'Đánh giá buổi tập');
    assert.equal(NOTIFICATION_TYPE_LABEL.MAINTENANCE_DUE, 'Đến hạn bảo trì');
    assert.equal(NOTIFICATION_TYPE_LABEL.BIRTHDAY, 'Sinh nhật hội viên');
    assert.equal(NOTIFICATION_TYPE_LABEL.LEAD_NEW, 'Khách tiềm năng mới');
    assert.equal(NOTIFICATION_TYPE_LABEL.CHECKIN, 'Check-in mới');
  });
});
