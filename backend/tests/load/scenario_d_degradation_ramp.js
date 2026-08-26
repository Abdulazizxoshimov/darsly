// Scenario D — "Degradatsiya rampasi" (L-4/L-5)
//
// MAQSAD: pass/fail chegara emas, TIZIM TIZZASINI (knee) topish. Join yukini
// past→yuqori uzluksiz oshiramiz va p95/p99 latency qayeRDA keskin ko'tarilishini,
// xato NISBATI qayerda o'sa boshlashini ko'ramiz. Bu — sig'imning haqiqiy chegarasi.
//
// Boshqa scenario'lardan farqi: bu yerda thresholds ATAYIN yumshoq (test
// "muvaffaqiyatsiz" bo'lib to'xtamasin) — biz grafik/xulosadan tizzani O'QIYMIZ,
// abortOnFail QILMAYMIZ. Har bosqichdagi latency'ni taqqoslash uchun
// summaryTrendStats keng.
//
// ⚠️ Bu HTTP token-berish sig'imini o'lchaydi (SFU media EMAS — buni
// sfu_media_load Go vositasi qiladi). To'liq rasm uchun ikkalasini birga
// kuzating: bu k6 ishlar ekan, DB/Redis/RabbitMQ/WS-hub to'yinishini ham
// kuzating (README "Nimani o'lchash" bo'limi).
//
// Ishga tushirish:
//   k6 run -e BASE=http://localhost:8087 -e SLUG=<slug> tests/load/scenario_d_degradation_ramp.js
// Server TRUSTED_PROXIES=0.0.0.0/0 bilan (unikal XFF per-IP rate-limitni chetlab o'tsin).
import http from 'k6/http'
import { check } from 'k6'
import { Trend, Rate } from 'k6/metrics'

const BASE = __ENV.BASE || 'http://localhost:8087'
const SLUG = __ENV.SLUG

// Har bosqichda alohida o'qiladigan maxsus metrikalar — tizzani ko'rsatadi.
const joinLatency = new Trend('join_latency', true)
const joinErrors = new Rate('join_errors')

function fakeIP() {
  return `10.${(Math.random() * 255) | 0}.${(Math.random() * 255) | 0}.${(Math.random() * 255) | 0}`
}

export const options = {
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
  scenarios: {
    // Uzluksiz, sekin ko'tariluvchi rampa: 50 → 3000 join/s.
    // Uzun bosqichlar — har sathda tizim BARQARORLASHSIN, shunda o'lchov toza.
    ramp: {
      executor: 'ramping-arrival-rate',
      startRate: 50,
      timeUnit: '1s',
      preAllocatedVUs: 300,
      maxVUs: 4000,
      stages: [
        { duration: '30s', target: 100 },
        { duration: '30s', target: 250 },
        { duration: '30s', target: 500 },
        { duration: '30s', target: 800 },
        { duration: '30s', target: 1200 },
        { duration: '30s', target: 1800 },
        { duration: '30s', target: 2500 },
        { duration: '30s', target: 3000 },
        { duration: '20s', target: 3000 },
      ],
    },
  },
  // ATAYIN yumshoq — test tizzadan oldin abort bo'lmasin. Tizzani chiqishdan o'qiymiz.
  thresholds: {
    // Faqat "butunlay sindi" holatini belgilaydi, knee'ni EMAS.
    http_req_failed: ['rate<0.5'],
  },
}

export default function () {
  const res = http.post(
    `${BASE}/api/v1/joinlink/${SLUG}`,
    JSON.stringify({ guest_name: `k6-${__VU}-${__ITER}` }),
    { headers: { 'Content-Type': 'application/json', 'X-Forwarded-For': fakeIP() }, tags: { name: 'joinlink' } },
  )
  const ok = check(res, {
    'status 200': (r) => r.status === 200,
    'got token': (r) => r.json('data.room.token') != null,
  })
  joinLatency.add(res.timings.duration)
  joinErrors.add(!ok)
}

// Chiqishni o'qish: `join_latency` p95/p99 va `join_errors` qaysi bosqichda
// (qaysi join/s target'da) sakraydi — o'sha "tizza". Aniqroq ko'rish uchun:
//   k6 run --out json=ramp.json ...   so'ng bosqich vaqtlariga qarab kesing,
// yoki Grafana/Prometheus'ga ulang (k6 experimental prometheus output).
