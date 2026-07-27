# Darsly — Orkestratsiya jurnali

> Orchestrator har siklni shu yerda raqamlab kuzatadi. Cheksiz aylanishning oldini olish uchun.

## Holat xulosasi
- Backend: QURILGAN (build=0), audit rejimida.
- Frontend: noldan quriladi (vanilla CSS + .jsx).
- Reja real holatga moslashtirilgan (backend audit ≠ noldan yozish).

## Bosqichlar
| # | Bosqich | Holat |
|---|---|---|
| 0 | Poydevor (docs + agent ta'riflari) | ✅ tayyor (10 agent, 6 doc) |
| 1a | Backend tuzatish (backend-agent-1: 3 muammo) | ✅ tayyor (HIGH+IDOR+CORS; build/vet/test yashil) |
| 1b | Frontend build (orchestrator, vanilla CSS/.jsx) | ✅ tayyor (barcha view+panel+livekit; build toza, dev HTTP 200) |
| 2 | Integration kontrakt tekshiruvi | ✅ KONTRAKT TOZA |
| 3 | Security review + backend hardening | ✅ 3 tasdiq + 4 hardening |
| 4 | Devops verify | ✅ compose tozalandi, CI, LiveKit prod |
| 5 | Frontend QA fix sikli (1-aylanish) | ✅ 9 topilma → 8 tuzatildi |
| 6 | Frontend build + Playwright smoke | ✅ build toza, 3/3 test |
| 7 | PM yakuniy | ✅ "Inson testiga TAYYOR" (100% uchun 5 tirik stsenariy kerak) |
| 8 | Hand-off (final-report.md) | ✅ tayyor |

## Sikl jurnali
- **Backend fix (1-aylanish)**: 3 muammo (HIGH parol-reset sessiya, IDOR, CORS) → tuzatildi, build/test yashil.
- **Integration (1-aylanish)**: KONTRAKT TOZA — frontend↔backend nomuvofiqlik yo'q.
- **Security (1-aylanish)**: 3 tuzatish tasdiqlandi + 4 yangi topilma → hammasi tuzatildi (refresh-reuse family revoke, CSP qattiqlash, rate-limit in-memory fallback, admin-reset :id). Fail-open Redis = ataylab availability tanlovi (final-report'da hujjatlashtirildi).
- **Frontend QA (1-aylanish)**: 9 topilma (1 kritik: LiveRoom disconnect infinite-loader; 4 error-state yo'q; 5 hardcode rgba; 2 dead-code) → orchestrator 8 tasini tuzatdi (+ forgot/reset-password oqimi qo'shildi). 1 (test-mock hex) — past ustuvorlik, qoldirildi. Build toza, Playwright 3/3.
- **Devops (1-aylanish)**: docker-compose jiraflow/collab tozalandi, portlar mos, LiveKit prod compose + CI + DEPLOYMENT.md + Makefile.

## API + Yuklama testlash bosqichi (2026-07-24)
- **API funksional testlar** (backend-agent-1 + backend-agent-2): `internal/apitests/` da 9 domen success+bad — auth/user/lesson/joinlink/waitingroom/room/recording/notification + E2E zanjir (variantlar bilan). **To'liq `go test ./...` YASHIL** (~46 test, 2 skip LiveKit). Hujjat: `docs/api-test-coverage.md`.
- **Test paytida 2 HAQIQIY bug topildi+tuzatildi**: room.HostToken + recording.StartRecording da LiveKit-disabled 404/403ni 500 bilan niqoblardi (tekshiruv tartibi to'g'irlandi).
- **Yuklama testi (k6)**: 3 stsenariy (A bitta guruh, B 60 parallel dars, C spike) + ceiling. Metodologiya: unikal XFF IP (rate-limit chetlab, sof sig'im). **Natija: join issiq yo'li 1000-2000 join/s @ p99<4ms, 0% xato, to'yinmadi; DB pool tugamadi.** Bottleneck backend emas — LiveKit media + DB write-pool + rate-limit policy (DEPLOYMENT.md). Hujjat: `docs/load-test-report.md`.
- **Cleanup**: barcha ko'tarilgan Docker (infra + LiveKit) va k6 image o'chirildi.
