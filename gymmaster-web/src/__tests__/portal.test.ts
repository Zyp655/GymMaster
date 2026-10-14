import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { describe, it } from 'node:test';

import {
  createChangePasswordSchema,
  createForgotPasswordSchema,
  createLoginSchema,
  createRegisterMemberSchema,
  createResetPasswordSchema,
  createVerifyOtpSchema,
} from '@/features/auth/schema';
import {
  createBookPTSessionSchema,
  createCancelPTSessionSchema,
  createFoodLogSchema,
  createFreezeSchema,
  createGuestPassSchema,
  createMemberReviewSchema,
  createMfaEnableSchema,
  createProfileSchema,
  createSurveyResponseSchema,
  createWorkoutLogSchema,
} from '@/features/portal/schema';
import { formatCurrency, formatDate, formatDateTime, initials } from '@/lib/format';
import { buildUrl, sendRequest } from '@/lib/http.core';
import { ApiCode, ApiError } from '@/types/api';

const mockTranslate = (key: string) => key;

describe('gymmaster-web: lib/format', () => {
  it('formatCurrency, formatDate, formatDateTime, initials hoạt động chuẩn xác', () => {
    assert.match(formatCurrency(1_350_000), /1\.350\.000/);
    assert.match(formatCurrency(null), /0/);

    assert.equal(formatDate('2026-10-14'), '14/10/2026');
    assert.equal(formatDate(null), '');
    assert.equal(formatDateTime('2026-10-14T18:30:00'), '14/10/2026 18:30');

    assert.equal(initials('Nguyễn Minh Khôi'), 'NK');
    assert.equal(initials('Hoàng'), 'H');
    assert.equal(initials(''), '?');
    assert.equal(initials(null), '?');
  });
});

describe('gymmaster-web: lib/http.core', () => {
  it('buildUrl gắn query params và bỏ qua giá trị rỗng/null/undefined', () => {
    const url = buildUrl('http://localhost:8000/v1/', '/member/classes', {
      branch_id: 1,
      from: '2026-10-14',
      empty: '',
      nil: null,
      undef: undefined,
    });
    assert.equal(url, 'http://localhost:8000/v1/member/classes?branch_id=1&from=2026-10-14');
  });

  it('sendRequest unwrap envelope thành công và ném ApiError khi lỗi nghiệp vụ hoặc mạng', async () => {
    const originalFetch = globalThis.fetch;
    try {
      globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        const headers = init?.headers as Record<string, string> | undefined;
        if (url.endsWith('/member/me')) {
          assert.equal(headers?.Authorization, 'Bearer member-jwt');
          return new Response(
            JSON.stringify({ code: 0, message: 'OK', data: { member_code: 'GM000001' } }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          );
        }
        if (url.endsWith('/member/conflict')) {
          return new Response(
            JSON.stringify({ code: ApiCode.CONFLICT, message: 'Bạn đã đặt lớp này rồi', data: null }),
            { status: 409, headers: { 'Content-Type': 'application/json' } },
          );
        }
        throw new TypeError('Failed to fetch');
      }) as typeof fetch;

      const me = await sendRequest<{ member_code: string }>(
        'http://localhost:8000/v1',
        'GET',
        '/member/me',
        { accessToken: 'member-jwt' },
      );
      assert.equal(me.member_code, 'GM000001');

      await assert.rejects(
        () => sendRequest('http://localhost:8000/v1', 'POST', '/member/conflict'),
        (err: unknown) =>
          err instanceof ApiError &&
          err.code === ApiCode.CONFLICT &&
          err.message === 'Bạn đã đặt lớp này rồi',
      );

      await assert.rejects(
        () => sendRequest('http://localhost:8000/v1', 'GET', '/member/net-fail'),
        (err: unknown) => err instanceof ApiError && err.code === ApiCode.NETWORK,
      );
    } finally {
      globalThis.fetch = originalFetch;
    }
  });
});

describe('gymmaster-web: features/portal/schema & features/auth/schema', () => {
  it('createProfileSchema kiểm tra email và số điện thoại liên hệ khẩn cấp', () => {
    const schema = createProfileSchema(mockTranslate);
    assert.equal(
      schema.safeParse({
        email: 'hoivien@gymmaster.vn',
        address: '120 Cầu Giấy, Hà Nội',
        emergency_contact_name: 'Nguyễn Văn A',
        emergency_contact_phone: '0901 234 567',
        avatar_url: '',
      }).success,
      true,
    );

    assert.equal(
      schema.safeParse({
        email: 'not-an-email',
        address: '',
        emergency_contact_name: '',
        emergency_contact_phone: '',
        avatar_url: '',
      }).success,
      false,
    );

    assert.equal(
      schema.safeParse({
        email: '',
        address: '',
        emergency_contact_name: '',
        emergency_contact_phone: '12345',
        avatar_url: '',
      }).success,
      false,
    );
  });

  it('createFreezeSchema bắt buộc end_date >= start_date và có lý do', () => {
    const schema = createFreezeSchema(mockTranslate, mockTranslate);
    assert.equal(
      schema.safeParse({
        start_date: '2026-10-15',
        end_date: '2026-10-20',
        reason: 'Đi công tác dài ngày',
      }).success,
      true,
    );

    const badDate = schema.safeParse({
      start_date: '2026-10-20',
      end_date: '2026-10-15',
      reason: 'Ngược ngày',
    });
    assert.equal(badDate.success, false);
  });

  it('createBookPTSessionSchema & createCancelPTSessionSchema kiểm tra dữ liệu đặt/hủy buổi PT', () => {
    const bookSchema = createBookPTSessionSchema(mockTranslate);
    assert.equal(
      bookSchema.safeParse({ contract_id: 1, start_at: '2026-10-15T10:00:00+07:00' }).success,
      true,
    );
    assert.equal(bookSchema.safeParse({ contract_id: 0, start_at: '' }).success, false);

    const cancelSchema = createCancelPTSessionSchema(mockTranslate);
    assert.equal(cancelSchema.safeParse({ reason: 'Bận đột xuất' }).success, true);
  });

  it('createLoginSchema & createChangePasswordSchema kiểm tra mật khẩu mới và xác nhận', () => {
    const loginSchema = createLoginSchema(mockTranslate);
    assert.equal(
      loginSchema.safeParse({ email: 'hoivien@gymmaster.vn', password: 'Password123' }).success,
      true,
    );
    assert.equal(loginSchema.safeParse({ email: 'bad', password: '' }).success, false);

    const pwdSchema = createChangePasswordSchema(mockTranslate, mockTranslate);
    assert.equal(
      pwdSchema.safeParse({
        old_password: 'OldPassword123',
        new_password: 'NewPassword456',
        confirm_password: 'NewPassword456',
      }).success,
      true,
    );
    // Trùng mật khẩu cũ -> lỗi
    assert.equal(
      pwdSchema.safeParse({
        old_password: 'SamePassword123',
        new_password: 'SamePassword123',
        confirm_password: 'SamePassword123',
      }).success,
      false,
    );
    // Không khớp confirm -> lỗi
    assert.equal(
      pwdSchema.safeParse({
        old_password: 'OldPassword123',
        new_password: 'NewPassword456',
        confirm_password: 'MismatchPassword',
      }).success,
      false,
    );
  });

  it('Sprint 12–17 auth & portal schemas (forgot, reset, register, workout, review) hoạt động chuẩn xác', () => {
    const forgotSchema = createForgotPasswordSchema(mockTranslate);
    assert.equal(forgotSchema.safeParse({ email: 'hoivien@gymmaster.vn' }).success, true);
    assert.equal(forgotSchema.safeParse({ email: 'invalid' }).success, false);

    const resetSchema = createResetPasswordSchema(mockTranslate, mockTranslate);
    assert.equal(
      resetSchema.safeParse({
        new_password: 'NewPassword123',
        confirm_password: 'NewPassword123',
      }).success,
      true,
    );
    assert.equal(
      resetSchema.safeParse({
        new_password: 'NewPassword123',
        confirm_password: 'WrongPassword',
      }).success,
      false,
    );

    const registerSchema = createRegisterMemberSchema(mockTranslate, mockTranslate);
    assert.equal(
      registerSchema.safeParse({
        full_name: 'Trần Minh Anh',
        email: 'minhanh@gymmaster.vn',
        phone: '0912345678',
        password: 'Password123',
        confirm_password: 'Password123',
        branch_id: 1,
      }).success,
      true,
    );

    const otpSchema = createVerifyOtpSchema(mockTranslate);
    assert.equal(otpSchema.safeParse({ otp: '123456' }).success, true);

    const workoutSchema = createWorkoutLogSchema(mockTranslate);
    assert.equal(
      workoutSchema.safeParse({
        exercise_id: 1,
        sets: 4,
        reps: 10,
        weight_kg: 60,
        note: 'Tốt',
      }).success,
      true,
    );
    assert.equal(
      workoutSchema.safeParse({
        exercise_id: 0,
        sets: 0,
        reps: 0,
        weight_kg: -5,
        note: '',
      }).success,
      false,
    );

    const reviewSchema = createMemberReviewSchema(mockTranslate);
    assert.equal(reviewSchema.safeParse({ rating: 5, comment: 'HLV nhiệt tình' }).success, true);
    assert.equal(reviewSchema.safeParse({ rating: 6, comment: '' }).success, false);
  });

  it('Sprint 18–27 Stage 3 & Stage 4 schemas (survey, food log, guest pass, 2FA MFA) hoạt động chuẩn xác', () => {
    const surveySchema = createSurveyResponseSchema(mockTranslate);
    assert.equal(surveySchema.safeParse({ score: 9, comment: 'Phòng tập sạch đẹp' }).success, true);
    assert.equal(surveySchema.safeParse({ score: 11, comment: '' }).success, false);

    const foodSchema = createFoodLogSchema(mockTranslate);
    assert.equal(
      foodSchema.safeParse({ food_id: 2, meal_type: 'LUNCH', qty: 150 }).success,
      true,
    );
    assert.equal(
      foodSchema.safeParse({ food_id: 0, meal_type: 'LUNCH', qty: 0 }).success,
      false,
    );

    const guestPassSchema = createGuestPassSchema(mockTranslate);
    assert.equal(
      guestPassSchema.safeParse({
        guest_name: 'Lê Hoàng Nam',
        guest_phone: '0987654321',
        branch_id: 1,
        valid_date: '2026-10-20',
      }).success,
      true,
    );
    assert.equal(
      guestPassSchema.safeParse({
        guest_name: '',
        guest_phone: '123',
        branch_id: 0,
        valid_date: '',
      }).success,
      false,
    );

    const mfaSchema = createMfaEnableSchema(mockTranslate);
    assert.equal(mfaSchema.safeParse({ code: '123456' }).success, true);
    assert.equal(mfaSchema.safeParse({ code: '12a456' }).success, false);
  });
});

describe('gymmaster-web: messages/vi translations completeness', () => {
  it('messages/vi/{common,auth,portal}.json hợp lệ và đầy đủ các nhóm trạng thái cổng hội viên', () => {
    const msgDir = path.resolve(process.cwd(), 'messages/vi');
    const common = JSON.parse(fs.readFileSync(path.join(msgDir, 'common.json'), 'utf8'));
    const auth = JSON.parse(fs.readFileSync(path.join(msgDir, 'auth.json'), 'utf8'));
    const portal = JSON.parse(fs.readFileSync(path.join(msgDir, 'portal.json'), 'utf8'));

    assert.ok(common.actions);
    assert.ok(common.validation);
    assert.ok(common.landing.schedules);
    assert.ok(common.landing.trial);
    assert.ok(auth.login);
    assert.ok(auth.changePassword);
    assert.ok(auth.forgotPassword);
    assert.ok(auth.resetPassword);
    assert.ok(auth.register);

    for (const section of [
      'nav',
      'overview',
      'qr',
      'subscriptions',
      'buySubscription',
      'paymentResult',
      'workout',
      'reviews',
      'classes',
      'pt',
      'progress',
      'invoices',
      'notifications',
      'account',
      'rewards',
      'challenges',
      'nutrition',
      'contracts',
      'kiosk',
      'status',
    ]) {
      assert.ok(portal[section], `portal.json phải có mục ${section}`);
    }

    for (const statusGroup of [
      'member',
      'subscription',
      'freeze',
      'booking',
      'ptContract',
      'ptSession',
      'invoice',
      'checkinResult',
      'checkinMethod',
      'occupancy',
      'lockerType',
      'installment',
      'contract',
      'dataRequest',
    ]) {
      assert.ok(portal.status[statusGroup], `portal.status phải có nhóm ${statusGroup}`);
    }
  });
});
