// Scenario B — "Ko'p parallel dars"
// 60 alohida darsga bir vaqtda oqim: 200 → 600 → 1200 join/s, VU'lar 60 slug bo'yicha taqsimlanadi.
// Har iteratsiya unikal IP (X-Forwarded-For). slugs.json shu papkada bo'lishi kerak.
// Server TRUSTED_PROXIES=0.0.0.0/0 bilan.
import http from 'k6/http'
import { check } from 'k6'
import { SharedArray } from 'k6/data'

const BASE = __ENV.BASE || 'http://localhost:8087'
const slugs = new SharedArray('slugs', () => JSON.parse(open('./slugs.json')))

function fakeIP() {
  return `10.${(Math.random() * 255) | 0}.${(Math.random() * 255) | 0}.${(Math.random() * 255) | 0}`
}

export const options = {
  summaryTrendStats: ['avg', 'min', 'med', 'p(50)', 'p(95)', 'p(99)', 'max'],
  scenarios: {
    parallel_lessons: {
      executor: 'ramping-arrival-rate',
      startRate: 50,
      timeUnit: '1s',
      preAllocatedVUs: 200,
      maxVUs: 2000,
      stages: [
        { duration: '20s', target: 200 },
        { duration: '20s', target: 600 },
        { duration: '30s', target: 1200 },
        { duration: '20s', target: 1200 },
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<1500'],
    http_req_failed: ['rate<0.05'],
  },
}

export default function () {
  const slug = slugs[(Math.random() * slugs.length) | 0]
  const res = http.post(
    `${BASE}/api/v1/joinlink/${slug}`,
    JSON.stringify({ guest_name: `k6-${__VU}-${__ITER}` }),
    { headers: { 'Content-Type': 'application/json', 'X-Forwarded-For': fakeIP() } },
  )
  check(res, {
    'status 200': (r) => r.status === 200,
    'got token': (r) => r.json('data.room.token') != null,
  })
}
