import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import {
  AUDIT_ACTION_LABEL,
  AUDIT_ENTITY_LABEL,
  auditActionLabel,
  auditEntityLabel,
  buildAuditOptions,
} from '../src/api/audit.ts';
import { searchTerm } from '../src/layouts/globalSearch.ts';
import { isMfaCodeComplete, mfaInputProps, normalizeMfaCode } from '../src/routes/auth/mfaCode.ts';
import { chatSendState } from '../src/routes/cms-marketing/chatAccess.ts';
import { summarizeBranches } from '../src/routes/home/dashboardStats.ts';
import {
  canManageRole,
  grantablePermissions,
  missingPermissions,
} from '../src/routes/roles/roleAccess.ts';
import { isSelfStaff, isStaffGuardError } from '../src/routes/staff/staffGuards.ts';
import { ApiError, ErrorCode } from '../src/utils/apiClient.ts';

describe('gymmaster-admin: nhãn nhật ký thao tác (api/audit)', () => {
  // Đối chiếu với mọi hằng Audit*/audit*/ptAct* + bulk trong gymmaster-api/internal/service.
  const API_ACTIONS = [
    'ACTIVATE', 'ADJUST', 'ASSIGN_ROLES', 'AUTO_NO_SHOW', 'BRANCH_TRANSFER', 'BULK_ACTIVATE',
    'BULK_DEACTIVATE', 'BULK_NOTIFY', 'BULK_TRANSFER_BRANCH', 'CANCEL', 'CHANGE_PASSWORD',
    'CHANGE_TRAINER', 'COMPLETE', 'COPY_WEEK', 'CREATE', 'DELETE', 'EXPIRE', 'FREEZE_APPROVE',
    'FREEZE_END', 'FREEZE_REJECT', 'FREEZE_REQUEST', 'GRANT_PORTAL', 'IMPORT', 'LOCK',
    'LOCK_PORTAL', 'LOGIN', 'LOGIN_FAILED', 'NO_SHOW', 'PAY', 'PAYMENT', 'PORTAL_UPDATE_PROFILE',
    'PRICE_CHANGE', 'REFUND', 'RENEW', 'RESCHEDULE', 'RESET_PASSWORD', 'RESET_PASSWORD_REQUEST',
    'RESET_QR', 'RETURN', 'SALE', 'SCHEDULE', 'SELL', 'TRANSFER', 'UNLOCK', 'UPDATE', 'UPGRADE',
    'VOID',
  ];
  const API_ENTITIES = [
    'access_device', 'announcement', 'auth', 'badge', 'body_metric', 'branch', 'branch_holiday',
    'cash_session', 'challenge', 'class_schedule', 'class_schedule_series', 'class_type',
    'contract_template', 'data_request', 'day_pass', 'equipment', 'exercise', 'expense', 'food',
    'installment_plan', 'invoice', 'lead', 'leave_request', 'legacy_import', 'locker',
    'loyalty_points', 'loyalty_reward', 'maintenance_log', 'meal_plan', 'member',
    'member_contract', 'membership_plan', 'payroll', 'pos_order', 'product', 'product_category',
    'promotion', 'pt_contract', 'pt_package', 'pt_session', 'review', 'role', 'sales_target',
    'setting', 'shift', 'staff', 'stock_voucher', 'subscription', 'subscription_freeze',
    'supplier', 'survey', 'trainer_availability', 'trainer_commission', 'user_mfa', 'workout_plan',
  ];

  it('có nhãn tiếng Việt cho mọi hành động & đối tượng API ghi nhật ký', () => {
    for (const a of API_ACTIONS) assert.ok(AUDIT_ACTION_LABEL[a], `thiếu nhãn hành động ${a}`);
    for (const e of API_ENTITIES) assert.ok(AUDIT_ENTITY_LABEL[e], `thiếu nhãn đối tượng ${e}`);
    assert.equal(auditActionLabel('LOGIN_FAILED'), 'Đăng nhập thất bại');
    assert.equal(auditActionLabel('RESET_PASSWORD_REQUEST'), 'Yêu cầu đặt lại mật khẩu');
    assert.equal(auditActionLabel('BULK_NOTIFY'), 'Gửi thông báo hàng loạt');
    assert.equal(auditEntityLabel('user_mfa'), 'Xác thực 2 lớp (2FA)');
    assert.equal(auditEntityLabel('legacy_import'), 'Nhập dữ liệu cũ');
  });

  it('nhãn theo ngữ cảnh đối tượng (IMPORT/TRANSFER/PAY)', () => {
    assert.equal(auditActionLabel('IMPORT', 'stock_voucher'), 'Nhập kho');
    assert.equal(auditActionLabel('IMPORT', 'member'), 'Nhập hội viên');
    assert.equal(auditActionLabel('IMPORT', 'legacy_import'), 'Nhập dữ liệu cũ');
    assert.equal(auditActionLabel('IMPORT'), 'Nhập dữ liệu');
    assert.equal(auditActionLabel('TRANSFER', 'stock_voucher'), 'Chuyển kho');
    assert.equal(auditActionLabel('TRANSFER', 'subscription'), 'Chuyển nhượng');
    assert.equal(auditActionLabel('PAY', 'trainer_commission'), 'Chi hoa hồng');
  });

  it('mã chưa biết → hiển thị mã gốc; BULK_* chưa biết → "Hàng loạt: ..."', () => {
    assert.equal(auditActionLabel('SOME_NEW_ACTION'), 'SOME_NEW_ACTION');
    assert.equal(auditActionLabel('SOME_NEW_ACTION', 'member'), 'SOME_NEW_ACTION');
    assert.equal(auditActionLabel('BULK_LOCK'), 'Hàng loạt: khóa');
    assert.equal(auditActionLabel('BULK_XYZ'), 'BULK_XYZ');
    assert.equal(auditActionLabel('BULK_'), 'BULK_');
    assert.equal(auditActionLabel(''), '');
    assert.equal(auditEntityLabel('new_entity'), 'new_entity');
    assert.equal(auditEntityLabel(''), '');
  });

  it('buildAuditOptions gộp mã từ API + mã đã biết, không trùng, sắp theo nhãn', () => {
    const opts = buildAuditOptions(['LOGIN', 'X_CUSTOM', ''], AUDIT_ACTION_LABEL, (a) =>
      auditActionLabel(a),
    );
    const values = opts.map((o) => o.value);
    assert.equal(new Set(values).size, values.length);
    assert.ok(values.includes('X_CUSTOM'));
    assert.ok(values.includes('BULK_NOTIFY'));
    assert.ok(!values.includes(''));
    assert.equal(opts.find((o) => o.value === 'X_CUSTOM')?.label, 'X_CUSTOM');
    const labels = opts.map((o) => o.label);
    assert.deepEqual(labels, [...labels].sort((a, b) => a.localeCompare(b, 'vi')));
    assert.ok(buildAuditOptions(undefined, AUDIT_ENTITY_LABEL, auditEntityLabel).length > 50);
  });
});

describe('gymmaster-admin: dashboard – chi nhánh trong phạm vi', () => {
  const branches = [
    { id: 1, status: 'ACTIVE' },
    { id: 2, status: 'ACTIVE' },
    { id: 3, status: 'INACTIVE' },
  ];
  it('branch_ids rỗng = toàn hệ thống; có branch_ids → chỉ đếm chi nhánh được phân công', () => {
    assert.deepEqual(summarizeBranches(branches, []), { total: 3, active: 2, inactive: 1 });
    assert.deepEqual(summarizeBranches(branches, null), { total: 3, active: 2, inactive: 1 });
    assert.deepEqual(summarizeBranches(branches, [1]), { total: 1, active: 1, inactive: 0 });
    assert.deepEqual(summarizeBranches(branches, [3, 99]), { total: 1, active: 0, inactive: 1 });
    assert.deepEqual(summarizeBranches([], [1]), { total: 0, active: 0, inactive: 0 });
  });
});

describe('gymmaster-admin: CMS – quyền gửi tin nhắn PT', () => {
  it('chỉ HLV phụ trách gửi được khi hợp đồng ACTIVE; quản lý chỉ đọc', () => {
    const conv = { trainer_id: 10, contract_status: 'ACTIVE' };
    assert.deepEqual(chatSendState(conv, 10), { canSend: true, reason: null });
    const supervisor = chatSendState(conv, 1);
    assert.equal(supervisor.canSend, false);
    assert.match(supervisor.reason ?? '', /chỉ đọc/);
    const expired = chatSendState({ trainer_id: 10, contract_status: 'EXPIRED' }, 10);
    assert.equal(expired.canSend, false);
    assert.match(expired.reason ?? '', /ACTIVE/);
    assert.equal(chatSendState(null, 10).canSend, false);
    assert.equal(chatSendState(conv, undefined).canSend, false);
  });
});

describe('gymmaster-admin: vai trò – chống leo thang đặc quyền (khớp role_service.go)', () => {
  const owner = { roles: ['OWNER'], permissions: [] };
  const manager = { roles: ['MANAGER'], permissions: ['member.view', 'role.view', 'role.update'] };

  it('OWNER quản lý mọi vai trò', () => {
    assert.equal(canManageRole(owner, { code: 'OWNER', permissions: ['x'] }).ok, true);
    assert.equal(canManageRole(owner, { code: 'MANAGER', permissions: ['x'] }).ok, true);
  });

  it('không phải OWNER: chặn OWNER, vai trò đang giữ, vai trò có quyền vượt quyền mình', () => {
    const o = canManageRole(manager, { code: 'OWNER', permissions: [] });
    assert.equal(o.ok, false);
    assert.match(o.reason ?? '', /chủ trung tâm/);
    const self = canManageRole(manager, { code: 'MANAGER', permissions: [] });
    assert.equal(self.ok, false);
    assert.match(self.reason ?? '', /chính mình/);
    const higher = canManageRole(manager, {
      code: 'ACCOUNTANT',
      permissions: ['member.view', 'finance.manage'],
    });
    assert.equal(higher.ok, false);
    assert.match(higher.reason ?? '', /cao hơn/);
    assert.deepEqual(
      canManageRole(manager, { code: 'RECEPTIONIST', permissions: ['member.view'] }),
      { ok: true, reason: null },
    );
    assert.equal(canManageRole(manager, { code: 'EMPTY', permissions: null }).ok, true);
    assert.equal(canManageRole(null, { code: 'EMPTY' }).ok, false);
  });

  it('"Chọn tất cả quyền" chỉ gồm quyền người dùng đang có (OWNER: tất cả)', () => {
    const all = ['finance.manage', 'member.view', 'role.update', 'role.view'];
    assert.deepEqual(grantablePermissions(owner, all), all);
    assert.deepEqual(grantablePermissions(manager, all), ['member.view', 'role.update', 'role.view']);
    assert.deepEqual(grantablePermissions({ roles: [], permissions: [] }, all), []);
    assert.deepEqual(missingPermissions(manager, ['member.view', 'finance.manage']), [
      'finance.manage',
    ]);
    assert.deepEqual(missingPermissions(owner, ['finance.manage']), []);
  });
});

describe('gymmaster-admin: ô nhập mã 2FA (TOTP / mã khôi phục)', () => {
  it('TOTP: chỉ số, tối đa 6, bàn phím số', () => {
    assert.equal(normalizeMfaCode(' 12a34 5678', 'totp'), '123456');
    assert.equal(isMfaCodeComplete('123456', 'totp'), true);
    assert.equal(isMfaCodeComplete('12345', 'totp'), false);
    assert.equal(mfaInputProps('totp').inputMode, 'numeric');
    assert.equal(mfaInputProps('totp').maxLength, 6);
  });

  it('mã khôi phục: chữ + số, in hoa, tự chèn "-", bàn phím chữ', () => {
    assert.equal(normalizeMfaCode('abcd1234', 'recovery'), 'ABCD-1234');
    assert.equal(normalizeMfaCode(' ab12-cd34 \n', 'recovery'), 'AB12-CD34');
    assert.equal(normalizeMfaCode('ABCD-', 'recovery'), 'ABCD');
    assert.equal(normalizeMfaCode('ABCD-1', 'recovery'), 'ABCD-1');
    assert.equal(normalizeMfaCode('ABCD-12345678', 'recovery'), 'ABCD-1234');
    assert.equal(isMfaCodeComplete('ABCD-1234', 'recovery'), true);
    assert.equal(isMfaCodeComplete('ABCD1234', 'recovery'), false);
    assert.equal(isMfaCodeComplete('123456', 'recovery'), false);
    const props = mfaInputProps('recovery');
    assert.equal(props.inputMode, 'text');
    assert.ok(props.maxLength >= 9);
  });
});

describe('gymmaster-admin: nhân viên – lỗi chặn thao tác & chính mình', () => {
  it('nhận diện 403 và 400 "chính mình"; lỗi validate thường không tính', () => {
    assert.equal(
      isStaffGuardError(
        new ApiError('Không thể thao tác trên nhân viên có quyền cao hơn', ErrorCode.PERMISSION_DENIED, 403),
      ),
      true,
    );
    assert.equal(
      isStaffGuardError(
        new ApiError('Không thể tự khóa tài khoản của chính mình', ErrorCode.BAD_REQUEST, 400),
      ),
      true,
    );
    assert.equal(
      isStaffGuardError(new ApiError('Email đã tồn tại', ErrorCode.CONFLICT, 409)),
      false,
    );
    assert.equal(
      isStaffGuardError(new ApiError('Mật khẩu tối thiểu 8 ký tự', ErrorCode.BAD_REQUEST, 400)),
      false,
    );
    assert.equal(isStaffGuardError(new Error('x')), false);
    assert.equal(isSelfStaff({ id: 3 }, 3), true);
    assert.equal(isSelfStaff({ id: 3 }, 4), false);
    assert.equal(isSelfStaff(null, 3), false);
    assert.equal(isSelfStaff({ id: 3 }, undefined), false);
  });
});

describe('gymmaster-admin: tìm kiếm nhanh Ctrl+K', () => {
  it('searchTerm bỏ khoảng trắng, rỗng → null', () => {
    assert.equal(searchTerm('  GM0001 '), 'GM0001');
    assert.equal(searchTerm('   '), null);
    assert.equal(searchTerm(''), null);
  });
});
