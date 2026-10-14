import http from 'k6/http';
import { check, sleep } from 'k6';

/**
 * Sprint 28 – Kiểm thử truy vấn danh sách hội viên lớn & tìm kiếm phân trang.
 * Chạy: k6 run -e BASE_URL=http://localhost:8000 -e ACCESS_TOKEN=... perf/members-list.js
 */
export const options = {
  vus: 25,
  duration: '30s',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<250', 'p(99)<500'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const ACCESS_TOKEN = __ENV.ACCESS_TOKEN || '';

export default function () {
  const page = (__ITER % 20) + 1;
  const res = http.get(
    `${BASE_URL}/v1/admin/members?page=${page}&limit=50&status=ACTIVE`,
    {
      headers: {
        Authorization: `Bearer ${ACCESS_TOKEN}`,
      },
    },
  );

  check(res, {
    'HTTP 200': r => r.status === 200,
  });

  sleep(0.1);
}
