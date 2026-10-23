import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { classifyHeartbeatError, DEFAULT_DEMO_KEY } from '@/features/kiosk/kiosk-utils';
import { formatAccessDays, formatAccessHours, guestPassesPerMonth } from '@/features/landing/utils';
import { normalizeCalorieGoals } from '@/features/portal/calorie-goals';
import { notBookableReason, waitlistRank } from '@/features/portal/classes';
import { normalizeOccupancy, toOccupancyList } from '@/features/portal/occupancy';
import { createWorkoutLogSchema } from '@/features/portal/schema';
import { SSE_RETRY_MAX_MS, sseRetryDelay } from '@/features/portal/sse';
import { isNotFutureDate, localDateISO } from '@/features/portal/workout';
import { hasVisibleText, stripInvisible } from '@/lib/text';
import { ApiError } from '@/types/api';

const tr = (key: string) => key;

describe('QA5 web: mật độ phòng tập đọc present_count', () => {
  it('API trả present_count → hiển thị đúng số người (không còn 0/120)', () => {
    const list = toOccupancyList([
      { branch_id: 1, present_count: 37, capacity: 120, occupancy_pct: 30.8, level: 'NORMAL' },
    ]);
    assert.equal(list[0].present_count, 37);
  });
  it('tên trường cũ vẫn được dùng làm dự phòng', () => {
    assert.equal(
      normalizeOccupancy({ branch_id: 1, capacity: 50, current_count: 5 }).present_count,
      5,
    );
    assert.equal(normalizeOccupancy({ branch_id: 1, capacity: 50 }).present_count, 0);
    assert.deepEqual(toOccupancyList(null), []);
    assert.equal(
      toOccupancyList({ items: [{ branch_id: 2, capacity: 9, present: 3 }] })[0].present_count,
      3,
    );
  });
});

describe('QA5 web: tin nhắn chỉ gồm ký tự vô hình', () => {
  it('zero-width / BOM / khoảng trắng → không có nội dung', () => {
    for (const v of ['​', ' ​‌‍ ', '﻿', '⁠­', '   ', '']) {
      assert.equal(hasVisibleText(v), false, JSON.stringify(v));
    }
  });
  it('văn bản thật (kể cả emoji có ZWJ) vẫn hợp lệ', () => {
    assert.equal(hasVisibleText('Chào HLV'), true);
    assert.equal(hasVisibleText('👨‍👩‍👧'), true);
    assert.equal(stripInvisible(' a​b '), 'ab');
  });
});

describe('QA5 web: nhật ký tập – giới hạn mức tạ & ngày', () => {
  const schema = createWorkoutLogSchema(tr);
  const base = { exercise_id: 1, sets: 3, reps: 10, weight: 60, note: '' };
  it('mức tạ ≤ 500 kg', () => {
    assert.equal(schema.safeParse({ ...base, weight: 500 }).success, true);
    assert.equal(schema.safeParse({ ...base, weight: 62.5 }).success, true);
    assert.equal(schema.safeParse({ ...base, weight: 1000 }).success, false);
    assert.equal(schema.safeParse({ ...base, weight: 500.5 }).success, false);
  });
  it('không nhận ngày tương lai', () => {
    const now = new Date(2026, 9, 22, 10, 0, 0);
    assert.equal(localDateISO(now), '2026-10-22');
    assert.equal(isNotFutureDate('2026-10-22', now), true);
    assert.equal(isNotFutureDate('2026-10-01', now), true);
    assert.equal(isNotFutureDate('2026-10-23', now), false);
    assert.equal(isNotFutureDate('abc', now), false);
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    assert.equal(
      schema.safeParse({ ...base, performed_date: localDateISO(tomorrow) }).success,
      false,
    );
    assert.equal(schema.safeParse({ ...base, performed_date: localDateISO() }).success, true);
  });
});

describe('QA5 web: lớp học – lý do không đặt được & thứ hạng chờ', () => {
  const now = new Date('2026-10-22T03:00:00Z'); // 10:00 giờ VN
  const res = { advance_days: 7, booking_blocked_until: null };
  it('lớp trong hạn đặt trước nhưng bookable=false (HLV nghỉ phép) → không báo "ngoài hạn"', () => {
    const item = { bookable: false, start_at: '2026-10-26T18:00:00+07:00' };
    assert.equal(notBookableReason(item, res, now), 'UNAVAILABLE');
  });
  it('vượt advance_days → OUT_OF_RANGE; bị khóa → BLOCKED; API có lý do → dùng lý do API', () => {
    assert.equal(
      notBookableReason({ bookable: false, start_at: '2026-10-30T06:00:00+07:00' }, res, now),
      'OUT_OF_RANGE',
    );
    assert.equal(
      notBookableReason({ bookable: false, start_at: '2026-10-29T23:00:00+07:00' }, res, now),
      'UNAVAILABLE',
    );
    assert.equal(
      notBookableReason(
        { bookable: false, start_at: '2026-10-26T18:00:00+07:00' },
        { advance_days: 7, booking_blocked_until: '2026-11-01T00:00:00+07:00' },
        now,
      ),
      'BLOCKED',
    );
    assert.equal(
      notBookableReason(
        {
          bookable: false,
          start_at: '2026-10-26T18:00:00+07:00',
          not_bookable_reason: 'trainer_leave',
        },
        res,
        now,
      ),
      'TRAINER_LEAVE',
    );
    assert.equal(
      notBookableReason({ bookable: true, start_at: '2026-10-26T18:00:00+07:00' }, res, now),
      null,
    );
  });
  it('waitlist_rank ưu tiên hơn waitlist_position', () => {
    assert.equal(waitlistRank({ waitlist_position: 8, waitlist_rank: 2 }), 2);
    assert.equal(waitlistRank({ waitlist_position: 8 }), 8);
    assert.equal(waitlistRank({ waitlist_position: 8, waitlist_rank: null }), 8);
    assert.equal(waitlistRank({ waitlist_position: null }), null);
  });
});

describe('QA5 web: kiosk', () => {
  it('khóa thiết bị sai (40106/401) ≠ mất kết nối máy chủ', () => {
    assert.equal(
      classifyHeartbeatError(new ApiError(40106, 401, 'Device key không hợp lệ')),
      'invalid_key',
    );
    assert.equal(classifyHeartbeatError(new ApiError(40300, 403, 'x')), 'invalid_key');
    assert.equal(classifyHeartbeatError(new ApiError(50000, 500, 'x')), 'offline');
    assert.equal(classifyHeartbeatError(new TypeError('Failed to fetch')), 'offline');
  });
  it('khóa demo chỉ tồn tại ngoài production', () => {
    assert.equal(
      DEFAULT_DEMO_KEY,
      process.env.NODE_ENV === 'production' ? '' : 'GMD-demo-kiosk-cg01',
    );
  });
});

describe('QA5 web: SSE kết nối lại có backoff', () => {
  it('tăng dần và chặn trần 60s (+ jitter ≤ 20%)', () => {
    assert.equal(
      sseRetryDelay(0, () => 0),
      2_000,
    );
    assert.equal(
      sseRetryDelay(1, () => 0),
      4_000,
    );
    assert.equal(
      sseRetryDelay(3, () => 0),
      16_000,
    );
    assert.equal(
      sseRetryDelay(50, () => 0),
      SSE_RETRY_MAX_MS,
    );
    assert.equal(
      sseRetryDelay(50, () => 1),
      SSE_RETRY_MAX_MS * 1.2,
    );
  });
});

describe('QA5 web: bảng giá – khung giờ & vé khách mời', () => {
  const day = (d: number) => ['', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7', 'CN'][d];
  it('định dạng khung giờ theo đúng cấu trúc API (dow mảng, from/to)', () => {
    assert.deepEqual(
      formatAccessHours([{ dow: [1, 2, 3, 4, 5], from: '06:00', to: '16:00' }], day, 'Hằng ngày'),
      ['T2–T6 · 06:00–16:00'],
    );
    assert.deepEqual(
      formatAccessHours([{ dow: [6, 0], start: '08:00:00', end: '12:00:00' }], day, 'Hằng ngày'),
      ['T7, CN · 08:00–12:00'],
    );
    assert.deepEqual(formatAccessHours([{ from: '05:00', to: '22:00' }], day, 'Hằng ngày'), [
      'Hằng ngày · 05:00–22:00',
    ]);
    assert.deepEqual(formatAccessHours(undefined, day, 'x'), []);
    assert.deepEqual(formatAccessHours([{ dow: 1 } as never], day, 'x'), []);
    assert.equal(formatAccessDays([1, 3, 4, 5, 7], day, 'x'), 'T2, T4–T6, CN');
  });
  it('vé khách mời: thiếu/0/âm → 0', () => {
    assert.equal(guestPassesPerMonth({}), 0);
    assert.equal(guestPassesPerMonth({ guest_passes_per_month: null }), 0);
    assert.equal(guestPassesPerMonth({ guest_passes_per_month: 2 }), 2);
  });
});

describe('R-W: gợi ý calo theo mục tiêu (/me/nutrition)', () => {
  it('API chưa có trường → null (UI ẩn khối, không lỗi)', () => {
    assert.equal(normalizeCalorieGoals({ recommended_calories: 2400, plans: [] }), null);
    assert.equal(normalizeCalorieGoals(undefined), null);
  });
  it('dạng mảng có macro → sắp xếp giảm mỡ/duy trì/tăng cơ', () => {
    const r = normalizeCalorieGoals({
      bmr: 1650.4,
      tdee: 2558,
      calorie_goals: [
        { goal: 'GAIN_MUSCLE', calories: 2858, protein_g: 150, carbs_g: 360, fat_g: 80 },
        { goal: 'lose_fat', calories: 2058 },
        { goal: 'MAINTAIN', calories: 2558 },
      ],
    });
    assert.equal(r?.bmr, 1650);
    assert.deepEqual(
      r?.goals.map(g => g.goal),
      ['LOSE_FAT', 'MAINTAIN', 'GAIN_MUSCLE'],
    );
    assert.equal(r?.goals[2].proteinG, 150);
    assert.equal(r?.goals[0].proteinG, undefined);
  });
  it('hợp đồng API thật (goal_targets CUT/MAINTAIN/BULK, goal_source)', () => {
    const r = normalizeCalorieGoals({
      recommended_calories: 2569,
      bmr: 1657,
      tdee: 2569,
      has_body_metric: true,
      goal: 'CUT',
      goal_source: 'MEAL_PLAN',
      goal_targets: [
        { goal: 'CUT', label: 'Giảm mỡ', calories: 2055, protein_g: 150, carbs_g: 154, fat_g: 57 },
        { goal: 'MAINTAIN', calories: 2569, protein_g: 120, carbs_g: 0, fat_g: 86 },
        { goal: 'BULK', calories: 2826, protein_g: 135 },
      ],
    });
    assert.equal(r?.currentGoal, 'LOSE_FAT');
    assert.equal(r?.hasBodyMetric, true);
    assert.deepEqual(
      r?.goals.map(g => [g.goal, g.calories]),
      [
        ['LOSE_FAT', 2055],
        ['MAINTAIN', 2569],
        ['GAIN_MUSCLE', 2826],
      ],
    );
    // goal_source = DEFAULT → không gắn nhãn "Mục tiêu của bạn"
    const d = normalizeCalorieGoals({ goal: 'MAINTAIN', goal_source: 'DEFAULT', tdee: 2000 });
    assert.equal(d?.currentGoal, undefined);
  });
  it('dạng map số / object lồng', () => {
    const r = normalizeCalorieGoals({
      calorie_recommendation: { tdee: 2500, goals: { cut: 2000, bulk: { calories: 2800 } } },
    });
    assert.equal(r?.tdee, 2500);
    assert.deepEqual(
      r?.goals.map(g => [g.goal, g.calories]),
      [
        ['LOSE_FAT', 2000],
        ['GAIN_MUSCLE', 2800],
      ],
    );
  });
});
