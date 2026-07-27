// k6 API load test — Darsly.
// Ishga tushirish:  k6 run -e BASE=http://localhost:8087 -e SLUG=<join_slug> tests/load/api_load.js
//
// Guruh darsiga ko'p guest'ning bir vaqtda kirishini simulyatsiya qiladi:
//   POST /joinlink/:slug  (parolsiz, waiting_room OFF dars uchun → participant token)
import http from "k6/http";
import { check, sleep } from "k6";

const BASE = __ENV.BASE || "http://localhost:8087";
const SLUG = __ENV.SLUG; // parolsiz, waiting-room OFF, jonli dars slug'i

export const options = {
  scenarios: {
    join_burst: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "10s", target: 25 }, // 25 ta bir vaqtda
        { duration: "20s", target: 25 },
        { duration: "5s", target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<800"],
    http_req_failed: ["rate<0.01"],
  },
};

export default function () {
  const res = http.post(
    `${BASE}/api/v1/joinlink/${SLUG}`,
    JSON.stringify({ guest_name: `k6-${__VU}-${__ITER}` }),
    { headers: { "Content-Type": "application/json" } },
  );
  check(res, {
    "status 200": (r) => r.status === 200,
    "got token": (r) => r.json("data.room.token") !== undefined,
  });
  sleep(1);
}
