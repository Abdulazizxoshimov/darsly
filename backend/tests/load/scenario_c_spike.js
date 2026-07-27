// Scenario C — "Spike (keskin yuk)"
// Dars 20:00da, hamma 19:59da: 30 soniyada 0 → 1000 join/s keskin ko'tarilish, keyin ushlab turish.
// Har iteratsiya unikal IP. Server TRUSTED_PROXIES=0.0.0.0/0 bilan.
import http from 'k6/http'
import { check } from 'k6'

const BASE = __ENV.BASE || 'http://localhost:8087'
const SLUG = __ENV.SLUG

function fakeIP() {
  return `10.${(Math.random() * 255) | 0}.${(Math.random() * 255) | 0}.${(Math.random() * 255) | 0}`
}

export const options = {
  summaryTrendStats: ['avg', 'min', 'med', 'p(50)', 'p(95)', 'p(99)', 'max'],
  scenarios: {
    spike: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      preAllocatedVUs: 300,
      maxVUs: 3000,
      stages: [
        { duration: '30s', target: 1000 }, // keskin: 0→1000 join/s
        { duration: '30s', target: 1000 }, // ushlab turish
        { duration: '10s', target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.1'],
  },
}

export default function () {
  const res = http.post(
    `${BASE}/api/v1/joinlink/${SLUG}`,
    JSON.stringify({ guest_name: `k6-${__VU}-${__ITER}` }),
    { headers: { 'Content-Type': 'application/json', 'X-Forwarded-For': fakeIP() } },
  )
  check(res, {
    'status 200': (r) => r.status === 200,
    'got token': (r) => r.json('data.room.token') != null,
  })
}
