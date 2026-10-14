import http from 'k6/http';
import { check } from 'k6';

/**
 * Sprint 28 – Kiểm thử đặt lớp nhóm đồng thời (kiểm chứng SELECT ... FOR UPDATE chống vượt sĩ số).
 * Chạy: k6 run -e BASE_URL=http://localhost:8000 -e SCHEDULE_ID=1 -e ACCESS_TOKEN=... perf/class-booking-race.js
 */
export const options = {
  scenarios: {
    concurrent_booking: {
      executor: 'per-vu-iterations',
      vus: 40,
      iterations: 1,
      maxDuration: '15s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<300'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const SCHEDULE_ID = Number(__ENV.SCHEDULE_ID || 1);
const ACCESS_TOKEN = __ENV.ACCESS_TOKEN || '';

export default function () {
  const memberId = (__VU % 30) + 1;
  const res = http.post(
    `${BASE_URL}/v1/admin/class-schedules/${SCHEDULE_ID}/bookings`,
    JSON.stringify({ member_id: memberId }),
    {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${ACCESS_TOKEN}`,
      },
    },
  );

  check(res, {
    'Xử lý an toàn không lỗi 500': r => r.status === 200 || r.status === 409 || r.status === 400,
  });
}
