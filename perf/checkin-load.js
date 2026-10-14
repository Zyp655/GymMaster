import http from 'k6/http';
import { check, sleep } from 'k6';

/**
 * Sprint 28 – Kiểm thử tải Check-in (mục tiêu 100 req/s, p95 < 200ms, p99 < 400ms).
 * Chạy: k6 run -e BASE_URL=http://localhost:8000 -e DEVICE_KEY=GMD-demo-kiosk-cg01 perf/checkin-load.js
 */
export const options = {
  scenarios: {
    checkin_throughput: {
      executor: 'constant-arrival-rate',
      rate: 100,
      timeUnit: '1s',
      duration: '30s',
      preAllocatedVUs: 30,
      maxVUs: 100,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<200', 'p(99)<400'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';
const DEVICE_KEY = __ENV.DEVICE_KEY || 'GMD-demo-kiosk-cg01';

export default function () {
  const payload = JSON.stringify({
    qr_token: `GM00000${(__VU % 9) + 1}`,
  });

  const res = http.post(`${BASE_URL}/v1/public/device/checkin`, payload, {
    headers: {
      'Content-Type': 'application/json',
      'X-Device-Key': DEVICE_KEY,
    },
  });

  check(res, {
    'HTTP 200': r => r.status === 200,
    'Envelope code == 0': r => {
      try {
        return JSON.parse(r.body).code === 0;
      } catch {
        return false;
      }
    },
  });

  sleep(0.05);
}
