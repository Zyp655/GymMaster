import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { describe, it } from 'node:test';

import { normalizeReferralCode } from '@/features/landing/referral';
import { isValidPostSlug, postHref } from '@/features/landing/post-slug';
import { normalizePublicSchedules, scheduleAvailableSlots } from '@/features/landing/schedule';
import {
  defaultInput,
  fromServings,
  gramsPerServing,
  maxInput,
  qtyMode,
  toServings,
} from '@/features/portal/food-units';
import {
  mapPortalNotificationLink,
  NOTIFICATION_FALLBACK_ROUTE,
} from '@/features/portal/notification-links';
import { createFoodLogSchema, createWorkoutLogSchema } from '@/features/portal/schema';
import { describeUserAgent } from '@/features/portal/user-agent';
import { pickDefaultExerciseId } from '@/features/portal/workout';
import { clientInfoHeaders } from '@/lib/client-info';
import { isSafeRelativePath, resolveAuthRedirect, safeReturnUrl } from '@/lib/safe-return-url';
import { safeUrl } from '@/lib/sanitize-html';
import { buildContentSecurityPolicy, buildSecurityHeaders } from '@/lib/security-headers';

const tr = (key: string) => key;

describe('QA2 web: chống open-redirect (safeReturnUrl / NextAuth redirect)', () => {
  it('chỉ nhận đường dẫn nội bộ bắt đầu bằng MỘT dấu "/"', () => {
    for (const ok of ['/me', '/me/invoices?invoice=3', '/blog/abc#x', '/']) {
      assert.equal(isSafeRelativePath(ok), true, ok);
      assert.equal(safeReturnUrl(ok), ok);
    }
    for (const bad of [
      '/\\example.org',
      '//example.org',
      '/\\/example.org',
      '/\t/example.org',
      '/\n/example.org',
      'https://example.org',
      'javascript:alert(1)',
      'example.org',
      '',
      null,
      undefined,
    ]) {
      assert.equal(isSafeRelativePath(bad), false, String(bad));
      assert.equal(safeReturnUrl(bad), '/me');
    }
    assert.equal(safeReturnUrl('//evil', '/login'), '/login');
  });

  it('resolveAuthRedirect: tương đối/same-origin → giữ; khác origin hoặc "/\\" → baseUrl', () => {
    const base = 'http://localhost:3000';
    assert.equal(resolveAuthRedirect('/me/account', base), 'http://localhost:3000/me/account');
    assert.equal(
      resolveAuthRedirect('http://localhost:3000/login?expired=1', base),
      'http://localhost:3000/login?expired=1',
    );
    assert.equal(resolveAuthRedirect('https://example.org/x', base), base);
    assert.equal(resolveAuthRedirect('/\\example.org', base), base);
    assert.equal(resolveAuthRedirect('//example.org', base), base);
    assert.equal(resolveAuthRedirect('http://localhost:3000//example.org', base), base);
  });
});

describe('QA2 web: sanitize-html chặn URL nguy hiểm', () => {
  it('safeUrl từ chối "/\\evil", "//evil", javascript:, ký tự điều khiển', () => {
    assert.equal(safeUrl('/blog/abc'), '/blog/abc');
    assert.equal(safeUrl('https://gymmaster.vn'), 'https://gymmaster.vn');
    assert.equal(safeUrl('mailto:a@b.vn'), 'mailto:a@b.vn');
    assert.equal(safeUrl('/\\evil.com'), null);
    assert.equal(safeUrl('//evil.com'), null);
    assert.equal(safeUrl('/\t/evil.com'), null);
    assert.equal(safeUrl('javascript:alert(1)'), null);
    assert.equal(safeUrl(' java\nscript:alert(1)'), null);
    assert.equal(safeUrl('/uploads/a.png', 'src'), '/uploads/a.png');
    assert.equal(safeUrl('//cdn.evil/a.png', 'src'), null);
    assert.equal(safeUrl('data:image/png;base64,AAAA', 'src'), 'data:image/png;base64,AAAA');
    assert.equal(safeUrl('data:text/html;base64,AAAA', 'src'), null);
  });
});

describe('QA2 web: link thông báo', () => {
  it('map link hóa đơn sang ?invoice= và link lạ về /me/notifications', () => {
    assert.equal(mapPortalNotificationLink('/me/invoices/12'), '/me/invoices?invoice=12');
    assert.equal(mapPortalNotificationLink('/member/invoices/7'), '/me/invoices?invoice=7');
    assert.equal(mapPortalNotificationLink('/member/bookings'), '/me/bookings');
    assert.equal(mapPortalNotificationLink('/member/pt-sessions'), '/me/pt');
    assert.equal(mapPortalNotificationLink('/member/subscriptions'), '/me/subscriptions');
    assert.equal(mapPortalNotificationLink('/me/messages'), '/me/messages');
    assert.equal(mapPortalNotificationLink('/me/subscriptions/buy'), '/me/subscriptions/buy');
    assert.equal(mapPortalNotificationLink('/me/invoices?invoice=5'), '/me/invoices?invoice=5');
    for (const unknown of [
      '/me/khong-ton-tai',
      '/admin/pt-sessions/3',
      '/member/xyz',
      '//evil.com',
      'https://evil.com',
    ]) {
      assert.equal(mapPortalNotificationLink(unknown), NOTIFICATION_FALLBACK_ROUTE, unknown);
    }
    assert.equal(mapPortalNotificationLink(''), null);
    assert.equal(mapPortalNotificationLink(null), null);
  });
});

describe('QA2 web: quy đổi khẩu phần dinh dưỡng', () => {
  it('món theo gam nhập gam, quy đổi chính xác (12,5 g không thành 13 g)', () => {
    assert.equal(gramsPerServing('100g'), 100);
    assert.equal(gramsPerServing('50 g'), 50);
    assert.equal(gramsPerServing('1 bát'), null);
    assert.equal(qtyMode('100g'), 'gram');
    assert.equal(qtyMode('1 quả'), 'serving');
    assert.equal(toServings('100g', 12.5), 0.125);
    assert.equal(toServings('100g', 5), 0.05);
    assert.equal(toServings('100g', 150), 1.5);
    assert.equal(toServings('50g', 25), 0.5);
    assert.equal(toServings('1 bát', 1.5), 1.5);
    assert.equal(fromServings('100g', 0.125), 12.5);
    assert.equal(fromServings('1 bát', 2), 2);
    assert.equal(maxInput('100g'), 10000);
    assert.equal(maxInput('1 tô'), 100);
    assert.equal(defaultInput('100g'), 100);
    assert.equal(defaultInput('1 quả'), 1);
  });

  it('schema bữa ăn nhận số dương (kể cả < 10 g), từ chối 0/âm/NaN', () => {
    const schema = createFoodLogSchema(tr);
    assert.equal(schema.safeParse({ food_id: 1, meal_type: 'LUNCH', qty: 5 }).success, true);
    assert.equal(schema.safeParse({ food_id: 1, meal_type: 'LUNCH', qty: 0.5 }).success, true);
    assert.equal(schema.safeParse({ food_id: 1, meal_type: 'LUNCH', qty: 0 }).success, false);
    assert.equal(schema.safeParse({ food_id: 1, meal_type: 'LUNCH', qty: -1 }).success, false);
    assert.equal(
      schema.safeParse({ food_id: 1, meal_type: 'LUNCH', qty: Number.NaN }).success,
      false,
    );
  });
});

describe('QA2 web: nhật ký tập luyện khớp API', () => {
  it('schema dùng trường weight (API), không phải weight_kg', () => {
    const schema = createWorkoutLogSchema(tr);
    const parsed = schema.safeParse({ exercise_id: 1, sets: 4, reps: 8, weight: 82.5, note: '' });
    assert.equal(parsed.success, true);
    assert.equal(parsed.success && parsed.data.weight, 82.5);
  });

  it('chọn mặc định bài tập có nhật ký mới nhất', () => {
    const logs = [
      { exercise_id: 7, performed_at: '2026-10-18T18:20:00+07:00' },
      { exercise_id: 19, performed_at: '2026-10-20T18:10:00+07:00' },
      { exercise_id: 1, performed_at: '2026-10-16T18:15:00+07:00' },
    ];
    assert.equal(pickDefaultExerciseId(logs, [{ id: 99 }]), 19);
    assert.equal(pickDefaultExerciseId([], [{ id: 99 }]), 99);
    assert.equal(pickDefaultExerciseId([], []), undefined);
  });
});

describe('QA2 web: lịch lớp công khai', () => {
  it('available_slots ưu tiên API, thiếu thì tính capacity − booked_count (≥ 0)', () => {
    assert.equal(scheduleAvailableSlots({ capacity: 20, booked_count: 5 }), 15);
    assert.equal(scheduleAvailableSlots({ capacity: 20, booked_count: 25 }), 0);
    assert.equal(scheduleAvailableSlots({ capacity: 20, booked_count: 5, available_slots: 3 }), 3);
    const list = normalizePublicSchedules([
      { capacity: 10, booked_count: 4 } as never,
      { capacity: 10, booked_count: 10 } as never,
    ]);
    assert.deepEqual(
      list.map(s => s.available_slots),
      [6, 0],
    );
    assert.deepEqual(normalizePublicSchedules({ items: [] }), []);
    assert.deepEqual(normalizePublicSchedules(null), []);
  });
});

describe('QA2 web: blog / giới thiệu / thiết bị', () => {
  it('slug bài viết & mã giới thiệu được kiểm tra', () => {
    assert.equal(isValidPostSlug('lo-trinh-tang-co'), true);
    assert.equal(isValidPostSlug('Lo-Trinh'), false);
    assert.equal(isValidPostSlug('../etc'), false);
    assert.equal(postHref('abc-1'), '/blog/abc-1');
    assert.equal(normalizeReferralCode('REF-GM000056'), 'REF-GM000056');
    assert.equal(normalizeReferralCode(['ABC', 'x']), 'ABC');
    assert.equal(normalizeReferralCode('<script>'), null);
    assert.equal(normalizeReferralCode(undefined), null);
  });

  it('mô tả user-agent dễ đọc', () => {
    assert.equal(
      describeUserAgent(
        'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Safari/537.36',
      ),
      'Chrome · macOS',
    );
    assert.equal(
      describeUserAgent(
        'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1',
      ),
      'Safari · iOS',
    );
    assert.equal(describeUserAgent('node'), null);
  });

  it('chuyển tiếp UA/IP thật của trình duyệt khi đăng nhập qua server Next', () => {
    assert.deepEqual(
      clientInfoHeaders({
        'user-agent': 'Mozilla/5.0 Chrome/141',
        'x-forwarded-for': '203.0.113.7, 10.0.0.1',
      }),
      {
        'User-Agent': 'Mozilla/5.0 Chrome/141',
        'X-Forwarded-For': '203.0.113.7',
        'X-Real-IP': '203.0.113.7',
      },
    );
    assert.deepEqual(clientInfoHeaders({ 'x-forwarded-for': '<script>' }), {});
    assert.deepEqual(clientInfoHeaders(undefined), {});
    assert.deepEqual(clientInfoHeaders({ 'user-agent': 'UA\nX', 'x-real-ip': '::1' }), {
      'User-Agent': 'UAX',
      'X-Forwarded-For': '::1',
      'X-Real-IP': '::1',
    });
    const h = new Headers({ 'User-Agent': 'Firefox/130', 'X-Forwarded-For': '198.51.100.2' });
    assert.equal(clientInfoHeaders(h)['X-Forwarded-For'], '198.51.100.2');
  });
});

describe('QA2 web: header bảo mật', () => {
  it('CSP cho phép API origin, chặn nhúng iframe; prod không có unsafe-eval', () => {
    const prod = buildContentSecurityPolicy({
      apiUrl: 'https://api.gymmaster.vn/v1',
      isDev: false,
    });
    assert.match(prod, /connect-src 'self' https:\/\/api\.gymmaster\.vn/);
    assert.match(prod, /frame-ancestors 'none'/);
    assert.match(prod, /form-action 'self' https:\/\/sandbox\.vnpayment\.vn/);
    assert.doesNotMatch(prod, /unsafe-eval/);
    assert.match(prod, /upgrade-insecure-requests/);
    const dev = buildContentSecurityPolicy({ apiUrl: 'http://localhost:8000/v1', isDev: true });
    assert.match(dev, /'unsafe-eval'/);
    assert.match(dev, /connect-src 'self' http:\/\/localhost:8000 ws: wss:/);
    assert.doesNotMatch(dev, /upgrade-insecure-requests/);
    const keys = buildSecurityHeaders({ isDev: false }).map(h => h.key);
    for (const k of [
      'Content-Security-Policy',
      'X-Frame-Options',
      'X-Content-Type-Options',
      'Referrer-Policy',
      'Permissions-Policy',
    ]) {
      assert.ok(keys.includes(k), k);
    }
  });
});

describe('QA2 web: token màu chữ', () => {
  it('không dùng `text-muted` (màu nền #f3f4f6) hay `text-small` (không tồn tại) làm màu/cỡ chữ', () => {
    const root = path.resolve(import.meta.dirname, '..');
    const offenders: string[] = [];
    const walk = (dir: string) => {
      for (const name of fs.readdirSync(dir)) {
        const p = path.join(dir, name);
        if (fs.statSync(p).isDirectory()) {
          if (name !== '__tests__') walk(p);
        } else if (/\.tsx?$/.test(name)) {
          const src = fs.readFileSync(p, 'utf8');
          if (/\btext-muted(?!-foreground)\b|\btext-small\b/.test(src))
            offenders.push(path.relative(root, p));
        }
      }
    };
    walk(root);
    assert.deepEqual(offenders, []);
  });
});
