// Scenario A — "Bitta katta guruh darsi"
// Bitta darsga sekundiga yangi guest'lar oqimi: 200 → 500 → 1000 → 2000 join/s.
// Har iteratsiya = bitta yangi foydalanuvchi, UNIKAL IP (X-Forwarded-For) —
// shunda per-IP rate-limit (600/min) sun'iy to'siq bo'lmaydi, sof server sig'imi o'lchanadi.
// Server TRUSTED_PROXIES=0.0.0.0/0 bilan ishga tushirilishi shart (XFF'ga ishonadi).
// Ishga tushirish: k6 run -e BASE=http://localhost:8087 -e SLUG=<slug> scenario_a_big_group.js
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
    big_group: {
      executor: 'ramping-arrival-rate',
      startRate: 50,
      timeUnit: '1s',
      preAllocatedVUs: 200,
      maxVUs: 2000,
      stages: [
        { duration: '20s', target: 200 },
        { duration: '20s', target: 500 },
        { duration: '20s', target: 1000 },
        { duration: '30s', target: 2000 },
        { duration: '20s', target: 2000 },
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<1500'],
    http_req_failed: ['rate<0.05'],
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
