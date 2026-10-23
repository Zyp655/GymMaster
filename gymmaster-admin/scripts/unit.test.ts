import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { AxiosError } from 'axios';

// QA vòng 2 (audit labels, dashboard, roles, 2FA, staff, CMS chat, Ctrl+K) – tách file riêng.
import './admin-qa2.test.ts';

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
      // UIA-076: hết phiên đã được onSessionExpired báo → handled để showError không toast lần 2.
      (err: unknown) =>
        err instanceof ApiError && err.code === ErrorCode.UNAUTHORIZED && err.handled === true,
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

// ---- QA vòng 2 (ops): trang Cấu hình & chú thích lịch PT ----
import { normalizeTagValues, rangeHint } from '../src/routes/settings/helpers.ts';
import { legendStatusKeys } from '../src/routes/pt/constants.ts';

describe('gymmaster-admin: settings helpers & PT legend', () => {
  it('rangeHint: Switch không hiện khoảng, khóa thường có đơn vị, khóa không giới hạn', () => {
    const base = { key: 'k', label: 'l', help: '', unit: 'ngày', min: 0, max: 90 };
    assert.equal(rangeHint(base), 'Khoảng hợp lệ: 0 – 90 ngày');
    assert.equal(rangeHint({ ...base, unit: '' }), 'Khoảng hợp lệ: 0 – 90');
    assert.equal(rangeHint({ ...base, max: 1, boolean: true }), '');
    assert.equal(rangeHint({ ...base, max: Number.MAX_SAFE_INTEGER }), '');
    assert.equal(rangeHint({ ...base, min: 1, max: Number.MAX_SAFE_INTEGER }), 'Tối thiểu 1');
  });

  it('normalizeTagValues: bỏ trùng, bỏ chữ, giảm dần', () => {
    assert.deepEqual(normalizeTagValues(['1', ' 7', 'abc', '3', '7']), [7, 3, 1]);
    assert.deepEqual(normalizeTagValues([]), []);
  });

  it('legendStatusKeys: luôn có Vắng mặt; lịch HLV bỏ NO_SHOW_PENDING', () => {
    assert.ok(legendStatusKeys('pt').includes('NO_SHOW'));
    assert.ok(legendStatusKeys('pt').includes('NO_SHOW_PENDING'));
    assert.ok(legendStatusKeys('trainer').includes('NO_SHOW'));
    assert.ok(!legendStatusKeys('trainer').includes('NO_SHOW_PENDING'));
  });
});

import {
  ALL_BRANCHES,
  fromSelectValue,
  toSelectValue,
} from '../src/components/BranchSelect/value.ts';

describe('gymmaster-admin: BranchSelect value mapping', () => {
  it('không có "Tất cả": null/undefined → undefined (hiện placeholder, không hiện "0")', () => {
    assert.equal(toSelectValue(null, false), undefined);
    assert.equal(toSelectValue(undefined, false), undefined);
    assert.equal(toSelectValue(3, false), 3);
    assert.deepEqual(toSelectValue([1, 2], false), [1, 2]);
  });
  it('có "Tất cả" (allowAll + toàn hệ thống): null ↔ 0 bên trong', () => {
    assert.equal(toSelectValue(null, true), ALL_BRANCHES);
    assert.equal(toSelectValue(undefined, true), undefined);
    assert.equal(fromSelectValue(ALL_BRANCHES, true), null);
    assert.equal(fromSelectValue(5, true), 5);
  });
  it('bỏ chọn / giá trị 0 khi không có "Tất cả" → undefined', () => {
    assert.equal(fromSelectValue(undefined, false), undefined);
    assert.equal(fromSelectValue(null, false), undefined);
    assert.equal(fromSelectValue(ALL_BRANCHES, false), undefined);
    assert.equal(fromSelectValue(2, false), 2);
  });
});

import { resolveOkText } from '../src/components/ConfirmButton/okText.ts';

describe('gymmaster-admin: ConfirmButton okText', () => {
  it('ưu tiên okText → chữ trên nút → Xóa/Đồng ý', () => {
    assert.equal(resolveOkText('Hủy đặt chỗ', 'Hủy', true), 'Hủy đặt chỗ');
    assert.equal(resolveOkText(undefined, 'Từ chối', true), 'Từ chối');
    assert.equal(resolveOkText(undefined, 'Giải phóng tủ', true), 'Giải phóng tủ');
    assert.equal(resolveOkText(undefined, 'Duyệt', false), 'Duyệt');
    assert.equal(
      resolveOkText(undefined, undefined, true),
      'Xóa',
      'nút chỉ có icon (xóa) vẫn là "Xóa"',
    );
    assert.equal(resolveOkText(undefined, undefined, false), 'Đồng ý');
    assert.equal(
      resolveOkText(undefined, undefined, true, 'Từ chối'),
      'Từ chối',
      'nút icon có aria-label',
    );
    assert.equal(resolveOkText(undefined, undefined, true, 'Xóa'), 'Xóa');
  });
});

import { holidayImpactSummary, normalizeHHMM } from '../src/routes/trainer-schedules/time.ts';

describe('gymmaster-admin: lịch rảnh / ngày lễ – giờ HH:mm', () => {
  it('chuẩn hóa giờ giống backend', () => {
    assert.equal(normalizeHHMM('7:00'), '07:00');
    assert.equal(normalizeHHMM(' 23:59 '), '23:59');
    for (const bad of ['7h sáng', '24:00', '7:5', '', null, undefined, '07:60']) {
      assert.equal(normalizeHHMM(bad), null, String(bad));
    }
    assert.ok(
      (normalizeHHMM('9:00') ?? '') < (normalizeHHMM('10:00') ?? ''),
      'so sánh chuỗi đúng sau chuẩn hóa',
    );
  });
  it('tóm tắt lịch bị ảnh hưởng', () => {
    assert.equal(holidayImpactSummary({}), null);
    assert.deepEqual(
      holidayImpactSummary({
        affected_class_schedules: [{ id: 1, start_at: '', end_at: '' }],
        affected_pt_sessions: [],
      }),
      { classes: 1, pt: 0 },
    );
  });
});

import { readdirSync, readFileSync } from 'node:fs';
import { AUDIT_ACTION_LABEL, AUDIT_ENTITY_LABEL } from '../src/api/audit.ts';

/**
 * Quét tĩnh `gymmaster-api/internal/service/*.go`: mọi action/entity (chuỗi hoặc hằng chuỗi) truyền
 * vào `<x>audit.Log(` và các hàm bọc phải có nhãn tiếng Việt trong `src/api/audit.ts` (UIA-077 / A7).
 */
function collectGoAuditCodes() {
  const dir = new URL('../../gymmaster-api/internal/service/', import.meta.url);
  const files = readdirSync(dir)
    .filter((f) => f.endsWith('.go') && !f.endsWith('_test.go'))
    .map((f) => readFileSync(new URL(f, dir), 'utf8'));
  const all = files.join('\n');
  const consts = new Map<string, string>();
  for (const m of all.matchAll(/(?:^|\bconst)\s*(\w+)\s+(?:string\s+)?=\s*"([^"]*)"/gm)) {
    consts.set(m[1], m[2]);
  }
  const resolve = (t: string | undefined): string | undefined => {
    if (t === undefined) return undefined;
    const lit = /^"([^"]*)"$/.exec(t);
    return lit ? lit[1] : consts.get(t);
  };
  /** Tách đối số của lời gọi bắt đầu ngay sau "(" tại `from`. */
  const splitArgs = (src: string, from: number): string[] => {
    const args: string[] = [];
    let depth = 0;
    let cur = '';
    let inStr = false;
    for (let i = from; i < src.length; i++) {
      const c = src[i];
      if (inStr) {
        cur += c;
        if (c === '\\') cur += src[++i];
        else if (c === '"') inStr = false;
        continue;
      }
      if (c === '"') inStr = true;
      if ('([{'.includes(c)) depth++;
      if (')]}'.includes(c)) {
        if (depth === 0) break;
        depth--;
      }
      if (c === ',' && depth === 0) {
        args.push(cur.trim());
        cur = '';
        continue;
      }
      cur += c;
    }
    args.push(cur.trim());
    return args;
  };
  /** Danh sách tên tham số Go: "ctx context.Context, action, entityName string, id uint" → [ctx, action, entityName, id]. */
  const paramNames = (list: string) =>
    list
      .split(',')
      .map((p) => p.trim().split(/\s+/)[0])
      .filter(Boolean);

  // Hàm bọc: có tham số `action`; vị trí entity theo tham số `entityName`, hoặc cố định theo lời gọi bên trong.
  type Wrapper = { action: number; entity: number | string };
  const DEF = /\bfunc\s+(\([^)]*\)\s*)?(\w+)\(([^)]*)\)[^{]*\{/g;
  const collectWrappers = (src: string, methods: boolean) => {
    const out = new Map<string, Wrapper>();
    for (const m of src.matchAll(DEF)) {
      if (!!m[1] !== methods) continue;
      const names = paramNames(m[3]);
      const ai = names.indexOf('action');
      if (ai < 0 || !names.includes('ctx')) continue;
      const ei = names.indexOf('entityName');
      if (ei >= 0) {
        out.set(m[2], { action: ai, entity: ei });
        continue;
      }
      // Entity cố định: lấy từ lời gọi audit trong thân hàm (vài dòng đầu).
      const body = src.slice((m.index ?? 0) + m[0].length, (m.index ?? 0) + m[0].length + 600);
      const inner = /(?:\.Log|auditLogSafe)\(([^)]*)/.exec(body);
      const innerArgs = inner ? inner[1].split(',').map((x) => x.trim()) : [];
      const ent = innerArgs.find((x, i) => i > 0 && x !== 'action' && x !== 'ctx' && resolve(x));
      if (ent) out.set(m[2], { action: ai, entity: resolve(ent)! });
    }
    return out;
  };
  const globals = collectWrappers(all, false);
  globals.set('Log', { action: 1, entity: 2 });

  const actions = new Set<string>();
  const entities = new Set<string>();
  const unresolved = new Set<string>();
  let calls = 0;
  for (const src of files) {
    const wrappers = new Map([...globals, ...collectWrappers(src, true)]);
    const names = [...wrappers.keys()].join('|');
    const re = new RegExp(
      `(\\bfunc\\s+(?:\\([^)]*\\)\\s*)?)?(?:\\b([\\w.]*?)\\.)?\\b(${names})\\(`,
      'g',
    );
    for (const m of src.matchAll(re)) {
      if (m[1]) continue; // định nghĩa hàm
      const name = m[3];
      if (name === 'Log' && !/[Aa]udit$/.test(m[2] ?? '')) continue; // g.Log(...) – không phải audit
      const w = wrappers.get(name)!;
      const args = splitArgs(src, (m.index ?? 0) + m[0].length);
      if (args[0] !== 'ctx' && !args.includes('ctx')) continue;
      calls++;
      const a = resolve(args[w.action]);
      const e = typeof w.entity === 'string' ? w.entity : resolve(args[w.entity]);
      if (a === undefined) unresolved.add(String(args[w.action]));
      else actions.add(a);
      if (e === undefined) unresolved.add(String(args[w.entity as number]));
      else entities.add(e);
    }
  }
  return { calls, actions, entities, unresolved };
}

describe('gymmaster-admin: nhãn nhật ký kiểm toán phủ đủ mã backend (quét tĩnh)', () => {
  const { calls, actions, entities, unresolved } = collectGoAuditCodes();
  it('tìm thấy các lời gọi audit (bộ quét còn đúng)', () => {
    assert.ok(calls > 150, `chỉ tìm thấy ${calls} lời gọi audit`);
    assert.ok(actions.has('REASSIGN_TRAINER') && entities.has('online_payment_txn'));
    // Tham số biến chỉ được phép trong các hàm bọc / tiền tố BULK_ (nhãn tự sinh).
    for (const u of unresolved) {
      assert.match(u, /^(action|entityName|entity|auditBulk\w*\+.*)$/, `không giải được: ${u}`);
    }
  });
  it('mọi action có nhãn tiếng Việt', () => {
    const missing = [...actions].filter((a) => !AUDIT_ACTION_LABEL[a]);
    assert.deepEqual(missing, []);
  });
  it('mọi entity có nhãn tiếng Việt', () => {
    const missing = [...entities].filter((e) => !AUDIT_ENTITY_LABEL[e]);
    assert.deepEqual(missing, []);
  });
});

import {
  checkNewPassword,
  passwordErrorField,
} from '../src/components/ChangePasswordModal/passwordPolicy.ts';
import { scheduleTrainer, scheduleTrainerText } from '../src/routes/classes/constants.ts';
import { humanizeJobText } from '../src/api/job.ts';
import { notificationTypeLabel } from '../src/api/notification.ts';
import { isFieldValidationError, validateForm } from '../src/utils/errors.ts';
import { connectSse, sseBackoffDelay } from '../src/layouts/sseReconnect.ts';
import type { SseLike } from '../src/layouts/sseReconnect.ts';

describe('QA vòng 4 (fix-ua)', () => {
  it('UIA-072: chính sách mật khẩu mới + lỗi server hiện dưới đúng ô', () => {
    assert.ok(checkNewPassword('abcdefghij'), 'thiếu số');
    assert.ok(checkNewPassword('12345678'), 'thiếu chữ');
    assert.ok(checkNewPassword('abc123'), 'ngắn');
    assert.equal(checkNewPassword('Matkhau123'), null);
    assert.equal(checkNewPassword('mậtkhẩu12'), null);
    assert.equal(
      passwordErrorField(40000, 'Mật khẩu phải có ít nhất 8 ký tự, gồm cả chữ và số'),
      'new_password',
    );
    assert.equal(passwordErrorField(40102, 'Mật khẩu không đúng'), 'old_password');
    assert.equal(passwordErrorField(40000, 'Mật khẩu hiện tại không đúng'), 'old_password');
  });

  it('UIA-065: HLV dạy thay – an toàn khi API chưa có trường substitute_*', () => {
    assert.deepEqual(scheduleTrainer({ trainer_name: 'A' }), {
      name: 'A',
      substituted: false,
      original: 'A',
    });
    const sub = { trainer_name: 'A', substitute_trainer_id: 9, substitute_trainer_name: 'B' };
    assert.equal(scheduleTrainer(sub).name, 'B');
    assert.equal(scheduleTrainerText(sub), 'B (dạy thay A)');
    assert.equal(
      scheduleTrainer({
        trainer_name: 'A',
        substitute_trainer_id: null,
        substitute_trainer_name: '',
      }).substituted,
      false,
    );
  });

  it('UIA-074: không hiển thị mã thô', () => {
    assert.equal(notificationTypeLabel('DEVICE_OFFLINE'), 'Thiết bị mất kết nối');
    assert.equal(notificationTypeLabel('SOMETHING_NEW'), 'Thông báo');
    assert.equal(
      humanizeJobText('Chuyển gói tập quá hạn sang EXPIRED, quét gói ACTIVE'),
      'Chuyển gói tập quá hạn sang Hết hạn, quét gói Đang hiệu lực',
    );
    assert.equal(humanizeJobText('Đánh dấu NO_SHOW lượt đặt'), 'Đánh dấu Vắng mặt lượt đặt');
  });

  it('UIA-075: validateForm trả null khi lỗi validate client, ném lại lỗi khác', async () => {
    const bad = { validateFields: () => Promise.reject({ errorFields: [{ name: ['reason'] }] }) };
    const ok = { validateFields: () => Promise.resolve({ reason: 'x' }) };
    const boom = { validateFields: () => Promise.reject(new Error('boom')) };
    assert.equal(await validateForm(bad as never), null);
    assert.deepEqual(await validateForm(ok as never), { reason: 'x' });
    await assert.rejects(() => validateForm(boom as never), /boom/);
    assert.equal(isFieldValidationError(new Error('x')), false);
  });

  it('A3: SSE nối lại có backoff, auth_expired → refresh rồi mở lại, refresh lỗi → dừng', async () => {
    assert.equal(sseBackoffDelay(0), 1000);
    assert.equal(sseBackoffDelay(3), 8000);
    assert.equal(sseBackoffDelay(20), 60000);

    class FakeEs implements SseLike {
      listeners: Record<string, ((e: MessageEvent) => void)[]> = {};
      onmessage: SseLike['onmessage'] = null;
      onerror: SseLike['onerror'] = null;
      onopen: SseLike['onopen'] = null;
      closed = false;
      url: string;
      constructor(url: string) {
        this.url = url;
      }
      addEventListener(t: string, l: (e: MessageEvent) => void) {
        (this.listeners[t] ??= []).push(l);
      }
      emit(t: string) {
        this.listeners[t]?.forEach((l) => l({ data: '' } as MessageEvent));
      }
      close() {
        this.closed = true;
      }
    }
    const sources: FakeEs[] = [];
    const timers: { fn: () => void; ms: number }[] = [];
    let token = 't1';
    let refreshOk = true;
    let refreshes = 0;
    const stop = connectSse({
      buildUrl: () => `/stream?token=${token}`,
      refresh: async () => {
        refreshes++;
        if (!refreshOk) throw new Error('revoked');
        token = `t${refreshes + 1}`;
      },
      onEvent: () => undefined,
      createSource: (url) => {
        const es = new FakeEs(url);
        sources.push(es);
        return es;
      },
      setTimer: (fn, ms) => {
        timers.push({ fn, ms });
        return timers.length;
      },
      clearTimer: () => undefined,
    });
    const flush = async () => {
      const t = timers.shift();
      t?.fn();
      await new Promise((r) => setImmediate(r));
      return t;
    };
    assert.equal(sources.length, 1);

    // auth_expired → đóng, refresh ngay, mở lại bằng token mới (không mở trùng khi onerror theo sau).
    sources[0].emit('auth_expired');
    sources[0].onerror?.(new Event('error'));
    assert.equal(sources[0].closed, true);
    assert.equal(timers.length, 1);
    assert.equal((await flush())?.ms, 0);
    assert.equal(refreshes, 1);
    assert.equal(sources.length, 2);
    assert.equal(sources[1].url, '/stream?token=t2');

    // Mất mạng: lần 1 nối lại sau 1s không refresh; lần 2 (2s) có refresh.
    sources[1].onerror?.(new Event('error'));
    assert.equal((await flush())?.ms, 1000);
    assert.equal(refreshes, 1);
    sources[2].onerror?.(new Event('error'));
    assert.equal((await flush())?.ms, 2000);
    assert.equal(refreshes, 2);
    assert.equal(sources.length, 4);

    // Bị thu hồi + refresh thất bại → dừng hẳn.
    refreshOk = false;
    sources[3].emit('auth_revoked');
    await flush();
    assert.equal(sources.length, 4);
    assert.equal(timers.length, 0);
    stop();
  });
});

import { nextActiveIndex } from '../src/layouts/globalSearch.ts';
import { groupLockersByBranch, summarizeLockers } from '../src/routes/lockers/board.ts';
import type { Locker } from '../src/api/operations-ext.ts';

describe('QA vòng 5 (fix-ra)', () => {
  it('Ctrl+K: ↑/↓ chọn kết quả vòng tròn, danh sách rỗng → -1', () => {
    assert.equal(nextActiveIndex(-1, 'ArrowDown', 3), 0);
    assert.equal(nextActiveIndex(0, 'ArrowDown', 3), 1);
    assert.equal(nextActiveIndex(2, 'ArrowDown', 3), 0);
    assert.equal(nextActiveIndex(0, 'ArrowUp', 3), 2);
    assert.equal(nextActiveIndex(2, 'ArrowUp', 3), 1);
    assert.equal(nextActiveIndex(-1, 'ArrowUp', 3), 2);
    assert.equal(nextActiveIndex(1, 'Home', 3), 0);
    assert.equal(nextActiveIndex(0, 'End', 3), 2);
    assert.equal(nextActiveIndex(5, 'x', 3), 2);
    assert.equal(nextActiveIndex(0, 'ArrowDown', 0), -1);
  });

  it('Sơ đồ tủ "Tất cả chi nhánh": tủ cùng mã A-01 không bị trộn, thống kê theo chi nhánh', () => {
    const lk = (id: number, branch: number, zone: string, status: Locker['status']): Locker =>
      ({
        id,
        branch_id: branch,
        branch_name: branch === 1 ? 'Cầu Giấy' : undefined,
        code: 'A-01',
        zone,
        size: 'STANDARD',
        monthly_fee: 0,
        status,
        note: '',
      }) as Locker;
    const groups = groupLockersByBranch(
      [
        lk(1, 1, 'Khu Nam', 'AVAILABLE'),
        lk(2, 2, 'Khu Nam', 'RENTED'),
        lk(3, 1, 'Khu Nữ', 'BROKEN'),
        lk(4, 1, 'Khu Nam', 'OCCUPIED'),
      ],
      (id) => (id === 2 ? 'Hai Bà Trưng' : undefined),
    );
    assert.deepEqual(
      groups.map((g) => [
        g.branchId,
        g.branchName,
        g.zones.map((z) => [z.zone, z.lockers.map((l) => l.id)]),
      ]),
      [
        [
          1,
          'Cầu Giấy',
          [
            ['Khu Nam', [1, 4]],
            ['Khu Nữ', [3]],
          ],
        ],
        [2, 'Hai Bà Trưng', [['Khu Nam', [2]]]],
      ],
    );
    assert.deepEqual(groups[0].summary, {
      total: 3,
      available: 1,
      occupied: 1,
      rented: 0,
      broken: 1,
    });
    assert.deepEqual(summarizeLockers([]), {
      total: 0,
      available: 0,
      occupied: 0,
      rented: 0,
      broken: 0,
    });
  });
});
