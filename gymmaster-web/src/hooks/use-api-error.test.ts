import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { afterEach, beforeEach, describe, it } from 'node:test';

import { type ApiErrorMessageKey, resolveApiErrorMessage } from '@/hooks/use-api-error';
import { sendRequest } from '@/lib/http.core';
import { ApiCode, ApiError } from '@/types/api';

const msgDir = path.join(process.cwd(), 'messages', 'vi');
const common = JSON.parse(fs.readFileSync(path.join(msgDir, 'common.json'), 'utf8'));
const auth = JSON.parse(fs.readFileSync(path.join(msgDir, 'auth.json'), 'utf8'));

/** Giống useApiErrorMessage: common.errors.* + auth.errors.tooManyRequests. */
const t = (key: ApiErrorMessageKey): string =>
  key === 'tooManyRequests' ? auth.errors.tooManyRequests : common.errors[key];

const resolve = (error: unknown, overrides?: Partial<Record<number, string>>) =>
  resolveApiErrorMessage(error, t, overrides);

const SESSION_EXPIRED = 'Phiên đăng nhập đã hết hạn. Vui lòng đăng nhập lại.';
const THROTTLE = 'Bạn thao tác quá nhanh, vui lòng thử lại sau ít phút.';

describe('hooks/use-api-error: resolveApiErrorMessage', () => {
  it('khóa mặc định tồn tại trong messages/vi', () => {
    assert.equal(common.errors.unauthorized, SESSION_EXPIRED);
    assert.equal(auth.errors.tooManyRequests, THROTTLE);
    assert.equal(common.errors.forbidden, 'Bạn không có quyền thực hiện thao tác này.');
  });

  it('401 phiên đăng nhập (40100/40101/40103) → "Phiên đăng nhập đã hết hạn"', () => {
    for (const code of [
      ApiCode.UNAUTHORIZED,
      ApiCode.TOKEN_EXPIRED,
      ApiCode.REFRESH_TOKEN_INVALID,
    ]) {
      assert.equal(resolve(new ApiError(code, 401, 'Token không hợp lệ')), SESSION_EXPIRED);
    }
  });

  it('401 nghiệp vụ (40102/40104/40105/40107/40108) → câu của server, KHÔNG phải hết phiên', () => {
    const cases: Array<[number, string]> = [
      [ApiCode.INVALID_CREDENTIALS, 'Email hoặc mật khẩu không đúng'],
      [ApiCode.RESET_TOKEN_INVALID, 'Mã OTP không hợp lệ hoặc đã hết hạn'],
      [
        ApiCode.TOO_MANY_ATTEMPTS,
        'Nhập sai OTP quá 5 lần, mã đã bị vô hiệu. Vui lòng yêu cầu mã mới',
      ],
      [ApiCode.MFA_REQUIRED, 'Vui lòng nhập mã xác thực 2 lớp'],
      [ApiCode.MFA_INVALID_CODE, 'Mã xác thực 2 lớp không đúng'],
    ];
    for (const [code, message] of cases) {
      assert.equal(resolve(new ApiError(code, 401, message)), message);
    }
  });

  it('401 nghiệp vụ không có câu dùng được → câu chung, không báo hết phiên', () => {
    assert.equal(
      resolve(new ApiError(ApiCode.RESET_TOKEN_INVALID, 401, '')),
      common.errors.unknown,
    );
    assert.equal(
      resolve(new ApiError(ApiCode.INVALID_CREDENTIALS, 401, 'INVALID_CREDENTIALS')),
      common.errors.unknown,
    );
  });

  it('403 → câu của server, thiếu thì "Bạn không có quyền..."', () => {
    assert.equal(
      resolve(
        new ApiError(
          ApiCode.PERMISSION_DENIED,
          403,
          'Bạn chỉ được đánh giá buổi tập PT của chính mình',
        ),
      ),
      'Bạn chỉ được đánh giá buổi tập PT của chính mình',
    );
    assert.equal(
      resolve(new ApiError(ApiCode.ACCOUNT_LOCKED, 403, 'Tài khoản đã bị khóa')),
      'Tài khoản đã bị khóa',
    );
    assert.equal(
      resolve(new ApiError(ApiCode.PERMISSION_DENIED, 403, '')),
      common.errors.forbidden,
    );
    // Response không có envelope (proxy) mang statusText tiếng Anh.
    assert.equal(resolve(new ApiError(40300, 403, 'Forbidden')), common.errors.forbidden);
  });

  it('429 (42900) → câu của server, thiếu thì câu throttle', () => {
    const wait = 'Mã OTP vừa được gửi, vui lòng đợi 60 giây trước khi yêu cầu mã mới';
    assert.equal(resolve(new ApiError(ApiCode.TOO_MANY_REQUESTS, 429, wait)), wait);
    assert.equal(resolve(new ApiError(ApiCode.TOO_MANY_REQUESTS, 429, '')), THROTTLE);
    assert.equal(
      resolve(new ApiError(ApiCode.TOO_MANY_REQUESTS, 429, 'TOO_MANY_REQUESTS')),
      THROTTLE,
    );
    assert.equal(resolve(new ApiError(42900, 429, 'Too Many Requests')), THROTTLE);
  });

  it('giữ nguyên hành vi cũ: override, validate/400, 404, 409, 5xx, mạng, lỗi lạ', () => {
    assert.equal(
      resolve(new ApiError(ApiCode.RESET_TOKEN_INVALID, 401, 'x'), {
        [ApiCode.RESET_TOKEN_INVALID]: 'Câu riêng',
      }),
      'Câu riêng',
    );
    assert.equal(
      resolve(new ApiError(ApiCode.VALIDATION, 400, 'Email không hợp lệ')),
      'Email không hợp lệ',
    );
    assert.equal(resolve(new ApiError(ApiCode.BAD_REQUEST, 400, '')), common.errors.badRequest);
    assert.equal(resolve(new ApiError(ApiCode.NOT_FOUND, 404, '')), common.errors.notFound);
    assert.equal(resolve(new ApiError(ApiCode.CONFLICT, 409, 'Đã tồn tại')), 'Đã tồn tại');
    assert.equal(resolve(new ApiError(ApiCode.INTERNAL, 500, 'boom')), common.errors.internal);
    assert.equal(resolve(new ApiError(ApiCode.NETWORK, 0, 'NETWORK_ERROR')), common.errors.network);
    assert.equal(resolve(new Error('x')), common.errors.unknown);
  });
});

describe('hooks/use-api-error + lib/http.core: response giả lập từ API', () => {
  const originalFetch = globalThis.fetch;
  let next: { status: number; body: unknown };

  beforeEach(() => {
    globalThis.fetch = (async () =>
      new Response(typeof next.body === 'string' ? next.body : JSON.stringify(next.body), {
        status: next.status,
        statusText: next.status === 429 ? 'Too Many Requests' : 'Error',
        headers: { 'Content-Type': 'application/json' },
      })) as typeof fetch;
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  async function messageFor(status: number, body: unknown) {
    next = { status, body };
    try {
      await sendRequest('http://api.test/v1', 'POST', '/auth/verify-member-otp', { body: {} });
    } catch (error) {
      return resolve(error);
    }
    assert.fail('sendRequest phải ném lỗi');
  }

  it('OTP sai (401/40104) hiển thị câu OTP, không báo hết phiên', async () => {
    assert.equal(
      await messageFor(401, {
        code: 40104,
        message: 'Mã OTP không hợp lệ hoặc đã hết hạn',
        data: null,
      }),
      'Mã OTP không hợp lệ hoặc đã hết hạn',
    );
  });

  it('token hết hạn (401/40101) báo hết phiên', async () => {
    assert.equal(
      await messageFor(401, { code: 40101, message: 'Token đã hết hạn', data: null }),
      SESSION_EXPIRED,
    );
  });

  it('429 có envelope → câu server; 429 không có envelope (gateway) → câu throttle', async () => {
    assert.equal(
      await messageFor(429, {
        code: 42900,
        message: 'Bạn đã yêu cầu quá nhiều mã OTP, vui lòng thử lại sau 1 giờ',
        data: null,
      }),
      'Bạn đã yêu cầu quá nhiều mã OTP, vui lòng thử lại sau 1 giờ',
    );
    assert.equal(await messageFor(429, '<html>Too Many Requests</html>'), THROTTLE);
  });

  it('403 có envelope → câu server', async () => {
    assert.equal(
      await messageFor(403, {
        code: 40300,
        message: 'Bạn không có quyền thực hiện thao tác này',
        data: null,
      }),
      'Bạn không có quyền thực hiện thao tác này',
    );
  });
});
