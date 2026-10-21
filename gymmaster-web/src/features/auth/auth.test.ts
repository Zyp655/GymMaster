import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { describe, it } from 'node:test';

import {
  buildVerifyMemberOtpPayload,
  classifyVerifyOtpError,
  formatCountdown,
  isSamePhone,
  isThrottleError,
  normalizePhone,
  OTP_RESEND_COOLDOWN_SECONDS,
  parseRetryAfterSeconds,
} from '@/features/auth/register-otp';
import {
  createChangePasswordSchema,
  createRegisterMemberSchema,
  createResetPasswordSchema,
  isStrongPassword,
  isValidMfaCode,
  MFA_RECOVERY_MAX_LENGTH,
  normalizeMfaCode,
} from '@/features/auth/schema';
import { ApiCode, ApiError } from '@/types/api';

const auth = JSON.parse(
  fs.readFileSync(path.join(process.cwd(), 'messages', 'vi', 'auth.json'), 'utf8'),
);
const POLICY = 'Mật khẩu phải có ít nhất 8 ký tự, gồm cả chữ và số';

/** Translator giả: trả đúng câu trong auth.json cho khóa passwordPolicy, còn lại trả khóa. */
const ta = (key: string) => (key === 'passwordPolicy' ? auth.passwordPolicy : key);
const tv = (key: string) => key;

function firstIssue(
  result: { success: boolean; error?: { issues: { path: unknown[]; message: string }[] } },
  field: string,
) {
  return result.error?.issues.find(i => i.path[0] === field)?.message;
}

describe('features/auth/schema: chính sách mật khẩu mới', () => {
  it('câu thông báo khớp backend', () => {
    assert.equal(auth.passwordPolicy, POLICY);
  });

  it('isStrongPassword: ≥ 8 ký tự, có chữ và số (chữ Unicode tính là chữ cái)', () => {
    assert.equal(isStrongPassword('abc12345'), true);
    assert.equal(isStrongPassword('Matkhau1'), true);
    assert.equal(isStrongPassword('mậtkhẩu1'), true);
    assert.equal(isStrongPassword('abcdefgh'), false); // không có số
    assert.equal(isStrongPassword('12345678'), false); // không có chữ
    assert.equal(isStrongPassword('abc1234'), false); // 7 ký tự
    assert.equal(isStrongPassword('!!!!!!!!'), false);
  });

  it('register / reset / change password đều áp dụng chính sách với câu tiếng Việt', () => {
    const register = createRegisterMemberSchema(tv, ta);
    const base = {
      full_name: 'Trần Minh Anh',
      email: 'minhanh@gymmaster.vn',
      phone: '0912345678',
      branch_id: 1,
    };
    const weak = register.safeParse({
      ...base,
      password: 'abcdefgh',
      confirm_password: 'abcdefgh',
    });
    assert.equal(weak.success, false);
    assert.equal(firstIssue(weak, 'password'), POLICY);
    assert.equal(
      register.safeParse({ ...base, password: 'abcd1234', confirm_password: 'abcd1234' }).success,
      true,
    );

    const reset = createResetPasswordSchema(tv, ta);
    const weakReset = reset.safeParse({ new_password: '12345678', confirm_password: '12345678' });
    assert.equal(firstIssue(weakReset, 'new_password'), POLICY);
    assert.equal(
      reset.safeParse({ new_password: 'abcd1234', confirm_password: 'abcd1234' }).success,
      true,
    );

    const change = createChangePasswordSchema(tv, ta);
    const weakChange = change.safeParse({
      old_password: 'old', // mật khẩu cũ không bị kiểm tra chính sách
      new_password: 'short1',
      confirm_password: 'short1',
    });
    assert.equal(firstIssue(weakChange, 'new_password'), POLICY);
    assert.equal(firstIssue(weakChange, 'old_password'), undefined);
    assert.equal(
      change.safeParse({
        old_password: 'old',
        new_password: 'abcd1234',
        confirm_password: 'abcd1234',
      }).success,
      true,
    );
  });

  it('mật khẩu rỗng → "bắt buộc"; quá 72 byte → maxLength', () => {
    const reset = createResetPasswordSchema(tv, ta);
    assert.equal(
      firstIssue(reset.safeParse({ new_password: '', confirm_password: '' }), 'new_password'),
      'required',
    );
    const long = `a1${'ậ'.repeat(30)}`; // 2 + 30×3 = 92 byte, 32 ký tự
    assert.equal(
      firstIssue(reset.safeParse({ new_password: long, confirm_password: long }), 'new_password'),
      'maxLength',
    );
  });
});

describe('features/auth/schema: mã 2FA (TOTP / mã khôi phục)', () => {
  it('TOTP: đúng 6 chữ số', () => {
    assert.equal(isValidMfaCode(normalizeMfaCode(' 123456 ', 'totp'), 'totp'), true);
    assert.equal(isValidMfaCode(normalizeMfaCode('12345', 'totp'), 'totp'), false);
    assert.equal(isValidMfaCode(normalizeMfaCode('ABCD-1234', 'totp'), 'totp'), false);
  });

  it('mã khôi phục: nhận chữ + số, in hoa, tự thêm gạch khi gõ liền', () => {
    assert.equal(normalizeMfaCode('ABCD-1234', 'recovery'), 'ABCD-1234');
    assert.equal(normalizeMfaCode('abcd-1234', 'recovery'), 'ABCD-1234');
    assert.equal(normalizeMfaCode('abcd1234', 'recovery'), 'ABCD-1234');
    assert.equal(normalizeMfaCode(' 9f3a-0b7c ', 'recovery'), '9F3A-0B7C');
    assert.equal(isValidMfaCode('ABCD-1234', 'recovery'), true);
    assert.equal(isValidMfaCode('9F3A-0B7C', 'recovery'), true);
    assert.equal(isValidMfaCode('123456', 'recovery'), false);
    assert.equal(isValidMfaCode('ABC-1234', 'recovery'), false);
    assert.equal('ABCD-1234'.length, MFA_RECOVERY_MAX_LENGTH);
  });
});

describe('features/auth/register-otp: luồng liên kết hồ sơ hội viên', () => {
  it('verify gửi phone + otp + password (mật khẩu đã nhập ở form), không gửi email', () => {
    const payload = buildVerifyMemberOtpPayload(
      { phone: '0912345678', password: 'abcd1234' },
      ' 123456 ',
    );
    assert.deepEqual(payload, { phone: '0912345678', otp: '123456', password: 'abcd1234' });
    assert.equal('email' in payload, false);
  });

  it('phân loại lỗi verify: 40104 sai/hết hạn, 40105 khóa, 429 throttle, còn lại other', () => {
    assert.equal(
      classifyVerifyOtpError(new ApiError(ApiCode.RESET_TOKEN_INVALID, 401, 'Mã OTP không hợp lệ')),
      'invalid',
    );
    assert.equal(
      classifyVerifyOtpError(new ApiError(ApiCode.TOO_MANY_ATTEMPTS, 401, 'x')),
      'locked',
    );
    assert.equal(
      classifyVerifyOtpError(new ApiError(ApiCode.TOO_MANY_REQUESTS, 429, 'x')),
      'throttled',
    );
    assert.equal(classifyVerifyOtpError(new ApiError(ApiCode.CONFLICT, 409, 'x')), 'other');
    assert.equal(classifyVerifyOtpError(new ApiError(ApiCode.BAD_REQUEST, 400, POLICY)), 'other');
    assert.equal(classifyVerifyOtpError(new Error('x')), 'other');
    assert.equal(isThrottleError(new ApiError(42900, 429, 'Too Many Requests')), true);
    assert.equal(isThrottleError(new ApiError(ApiCode.TOO_MANY_ATTEMPTS, 401, 'x')), false);
  });

  it('đọc thời gian chờ từ câu 429 của server', () => {
    assert.equal(
      parseRetryAfterSeconds('Mã OTP vừa được gửi, vui lòng đợi 60 giây trước khi yêu cầu mã mới'),
      60,
    );
    assert.equal(parseRetryAfterSeconds('Vui lòng đợi 45 giây'), 45);
    assert.equal(parseRetryAfterSeconds('Thử lại sau 2 phút'), 120);
    assert.equal(
      parseRetryAfterSeconds('Bạn đã yêu cầu quá nhiều mã OTP, vui lòng thử lại sau 1 giờ'),
      3600,
    );
    assert.equal(parseRetryAfterSeconds('Bạn thao tác quá nhanh'), OTP_RESEND_COOLDOWN_SECONDS);
    assert.equal(parseRetryAfterSeconds(undefined), OTP_RESEND_COOLDOWN_SECONDS);
  });

  it('formatCountdown và so sánh SĐT', () => {
    assert.equal(formatCountdown(59), '0:59');
    assert.equal(formatCountdown(60), '1:00');
    assert.equal(formatCountdown(3600), '60:00');
    assert.equal(formatCountdown(-3), '0:00');
    assert.equal(normalizePhone('+84 912 345 678'), '0912345678');
    assert.equal(isSamePhone('0912.345.678', '+84912345678'), true);
    assert.equal(isSamePhone('0912345678', '0912345679'), false);
  });

  it('messages: otpSubtitle dùng {email} (email che bớt từ API) và đủ khóa cho màn OTP', () => {
    const r = auth.register;
    assert.match(r.otpSubtitle, /\{email\}/);
    assert.match(r.otpSubtitle, /\{phone\}/);
    for (const key of [
      'otpSubtitleNoEmail',
      'otpReturn',
      'otpResend',
      'otpResending',
      'otpResendIn',
      'otpResent',
      'otpResentNoEmail',
      'otpInvalid',
      'otpLocked',
    ]) {
      assert.ok(r[key], `thiếu auth.register.${key}`);
    }
    assert.equal(r.otpInvalid, 'Mã OTP không đúng hoặc đã hết hạn');
    assert.match(r.otpResendIn, /\{time\}/);
    for (const key of [
      'mfaUseRecovery',
      'mfaUseTotp',
      'mfaRecoveryCode',
      'mfaRecoveryPlaceholder',
    ]) {
      assert.ok(auth.login[key], `thiếu auth.login.${key}`);
    }
    assert.equal(auth.login.mfaUseRecovery, 'Dùng mã khôi phục');
  });
});
