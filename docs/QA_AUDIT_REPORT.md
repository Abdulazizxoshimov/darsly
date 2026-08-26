# Darsly — QA Audit & Test-Coverage Report

> **Date:** 2026-08-26 · **Scope:** backend (Go), frontend (React/Vite), mobile (Android/Kotlin), load/performance.
> **Method:** static source audit + measured Go coverage on infra-free packages + review of every existing test file (65 Go, 25 frontend, 49 mobile, 6 load). No coverage number below is guessed; where infra was unavailable locally it is stated explicitly.

---

## 0. Headline (honest assessment)

**This is a mature, genuinely well-tested codebase — not an under-tested one.** Every backend usecase domain has tests; security-critical paths (RBAC, IDOR/ownership, token rotation/reuse-detection, rate-limit, webhook signature, ban-guard) are covered with *real* Casbin policy and *real* HMAC signatures, not mocks. The frontend's pure logic and shared API client are thoroughly tested; the mobile app has 49 unit tests over its extracted pure logic.

The audit did **not** find grounds to "rewrite the suite." It found a **finite, prioritized set of real gaps** concentrated in four places:

1. **Stateful orchestrators are untested across all stacks** — the classes that wire pure logic to the SDK/DB/coroutines: mobile `RoomViewModel`/`LocalRecorder`/`LessonSession`, frontend `useRoom.js`/`ws.js`/`LiveRoom.jsx`, backend `websocket.BroadcastToRoom`.
2. **One HIGH security test gap** — `worker/telegram_bot.go` `ownedRecording` IDOR gate (cross-mentor recording exfiltration) has **zero** tests.
3. **Load tests measure the wrong thing at scale** — they prove HTTP token-issuance *rate*, not concurrent WebRTC media; the SFU fan-out that actually caps capacity is never loaded beyond ~50 passive connections.
4. **One probable real bug** (below), independent of tests.

### ⚠️ Probable real defect found during audit (not a test gap — a code bug)
`backend/internal/infrastructure/repository/postgres/auth.go:160` — `MarkPasswordResetUsed` runs `UPDATE ... WHERE id=$1` with **no `AND used_at IS NULL`** guard. Combined with the non-transactional `ResetPassword` usecase flow (`usecase/auth/auth.go:348`, Get → UpdatePassword → Mark), two concurrent requests carrying the same valid reset token can **both** pass the `used_at IS NULL` read check and both succeed — a one-time token used twice (TOCTOU). Every other atomic transition in this codebase (`TransitionFromPending`, `ClaimReminder`) was deliberately made atomic; this one was missed. **Recommend fixing the query (`AND used_at IS NULL` + check `RowsAffected`) and adding a concurrency test** — see backlog B-6.

---

## 1. Discovery — inventory (Stage 1)

### 1.1 Test suite at a glance

| Stack | Test files | Runner | In CI? | Notes |
|---|---|---|---|---|
| Backend (Go) | **65** `_test.go` | `go test -race -coverprofile` | ✅ `backend` job (postgres+redis service containers) | `TEST_REQUIRE_INFRA=1` forbids silent skips |
| Frontend unit/component | **21** (vitest) | `npm test` | ✅ `frontend` job | + `npm run lint` gate |
| Frontend e2e | **2** (Playwright + MSW) | `npm run test:e2e` | ✅ separate `e2e` job | deterministic on MSW mocks, no backend |
| Mobile (Kotlin) | **49** JVM unit | `:app:testDebugUnitTest` | ✅ `mobile` job | + `lintDebug`; **0 instrumented/Espresso** |
| Load | 4 k6 + 2 Go | manual | ❌ (expected) | `backend/tests/load/` |

### 1.2 Measured Go coverage (infra-free packages, dev DB **not** running locally)

| Package | Coverage | Verdict |
|---|---|---|
| `usecase/chat` | **87.1%** | strong |
| `usecase/room` | **80.6%** | strong |
| `usecase/poll` | **76.6%** | strong |
| `worker` | 47.3% | thinner (ticker loops + telegram_bot untested) |
| `infrastructure/livekit` | 37.9% | thinner (VerifyToken path untested) |
| `usecase/shared` | 8.1% | low (guard helpers exercised indirectly via room tests) |
| `pkg/token` | 0.0% **(local skip artifact)** | `jwt_test.go` requires Redis → skips locally; **CI runs it fully** with the Redis service. **Not a real gap.** |

> Note on reading coverage: DB/Redis-backed suites (`apitests`, `postgres/integration_test.go`, `pkg/token`) **skip locally without infra** and **run in CI**. Local `go test ./...` under-reports; CI is the source of truth.

### 1.3 CI mapping
`.github/workflows/ci.yml` runs 4 jobs on push/PR: **backend** (build + vet + golangci-lint v2.1.6 + race+coverage over postgres/redis), **frontend** (lint + vitest + build), **e2e** (Playwright chromium + MSW), **mobile** (unit + lintDebug). Load tests are intentionally out of CI.

---

## 2. Findings by stack

### 2.1 Backend — well covered; gaps are targeted

**Strong (no rewrite needed):** auth (16 tests: register-orphan rollback, lockout, session-revoke, refresh rotation + reuse-detection + family-revoke + grace window + CAS parallel-race), lesson/room (ban/mute/autoend/enforce-join, ownership → 403), recording (restore/egress/race/local, cross-mentor download → 403), waitingroom (atomic admit, idempotent double-admit → 409), chat (XSS, DM visibility, upload validation), poll (publish/idempotent/cross-lesson), roomstate, archive, joinlink (brute-force lockout). Webhook signature is exemplary (correct→200; wrong secret / unknown key / unsigned / **tampered-body-but-valid-sig → 401**).

**Gaps (real):**
| Pri | Location | Risk it would catch |
|---|---|---|
| **HIGH** | `worker/telegram_bot.go:487 ownedRecording` + `handleCallback` — **0 tests** | Cross-mentor recording/transcript exfiltration via Telegram share (IDOR) |
| MEDIUM | `postgres/auth.go` — no DB test (password_reset one-time/expiry filters, refresh revoke) | one-time-token / expiry / revoke SQL-semantics regression |
| MEDIUM | `postgres/poll.go` — no DB test (`Vote` `ON CONFLICT DO UPDATE`, `Publish` `COALESCE` idempotency) | double-counted vote / idempotency break |
| MEDIUM | Notification `MarkRead` cross-user negative case untested (`notification.go:57`) | user A marks user B's notification read (IDOR) |
| MEDIUM | `websocket.go:218 BroadcastToRoom` / `subscribeRoom` / slow-consumer drop untested | silent real-time breakage / socket leak / back-pressure |
| LOW-MED | `auth.go:160 MarkPasswordResetUsed` **not atomic** (see §0 — code bug + missing concurrency test) | reset token used twice |
| LOW | telegram & ws HTTP handlers lack API-level tests (usecase/ws-layer covered) | thin glue |

### 2.2 Frontend — pure logic strong; stateful glue thin

**Strong:** `roomLogic` (quality label, reconnecting-vs-quality, hand queue, chat-badge rule, PiP auto-open, data-trust model, gallery order), `messaging`, `mediaTuning`, `format`, shared `api` client (401 `SESSION_REVOKED` no-retry, `TOKEN_EXPIRED`→refresh→replay, refresh-fail preserves reason, blob-error), `chat` validation (400/429/abort). Guest flow + create-lesson covered end-to-end by Playwright.

**Gaps (real):**
| Pri | Location | Risk |
|---|---|---|
| **HIGH** | `livekit/useRoom.js` — **no test** | `endedByServer`→removed/duplicate/room_deleted (kick & session-expiry **inside** room), `mediaError` camera/mic classification, `canPublishCameraOf` (student-camera permission, security-relevant) all unverified |
| **HIGH** | `lib/ws.js` — **no test** | exponential-backoff reconnect for mentor push + waiting-room; silent regression risk |
| **HIGH** | `livekit/Controls.jsx` (only screen-share tip tested) | `micLocked`/`selfUnmuteBlocked` moderation gate, camera-permission gating, `mediaFailText` error toggle untested — a moderation control shipping without a test |
| MEDIUM | API `403/404/409/5xx` `errorText` mapping not asserted | frontend handling of atomic-conflict (409 double-admit) contracts unverified |
| MEDIUM | `lib/roomSession.js` + `lib/features.isLiveRoomSupported` — pure, trivially testable, **0 coverage** | guest session/private-mode fallback + browser-support gate |
| MEDIUM | `views/WaitingRoom.jsx` (WS+polling reconcile), `LiveRoom.jsx` (1299L orchestration), `Whiteboard.jsx` (784L) — no component test | integration seams (e2e stops at room boundary by design) |
| MEDIUM | Tested-component untested states: `Users.jsx` isError/isLoading, `Recordings.jsx` top-level isError/isLoading, `Auth.jsx` login-error path | list-load-failure & wrong-password rendering |
| LOW | `Profile.jsx` (account-deletion — destructive, untested), `Notifications.jsx`, `Schedule.jsx`, `EditLessonModal.jsx`, Forgot/Reset — no component tests | |

### 2.3 Mobile — pure logic strong; the three big stateful classes untested

**Strong:** data/api (TokenAuthenticator 401-refresh, session, errors), data/livekit pure helpers (MediaTuning, reconnect/network-switch policy, ScreenAudio*, AudioMixer, LetterboxFit), data/repo, data/store, data/ws (backoff, realtime parser), 14 ui form/state tests.

**Gaps (real):**
| Pri | Location | Risk |
|---|---|---|
| **CRITICAL** | `RoomViewModel.kt` (1332L) — **0 tests** | live-room heart: mute/hand-queue/screen-share/reconnect/participant state; also imports `GlobalScope` (lifecycle/leak smell a VM test would catch) |
| **CRITICAL** | `LocalRecorder.kt` (426L, new) — **0 tests** | MediaCodec/muxer recording engine; a bug = silently corrupt/empty recordings, no guard |
| **HIGH** | `LessonSession.kt` (485L, new) — **0 tests** | SDK connect/publish/track lifecycle |
| **HIGH** | No instrumented/Espresso tests — critical login→room→record→upload flow never exercised integrated | parts tested, wiring not |
| MEDIUM | `PendingUploadResumer.kt` — only `decide()` table tested, scan/upload/delete orchestration untested | "don't lose recordings" safety net |
| MEDIUM | `LoginViewModel`, `WaitingRoomViewModel`, `PollViewModel` state machines untested | auth entry + admit/reject |
| LOW | remaining CRUD ViewModels (forms tested, VM glue low-risk) | |

### 2.4 Load / performance — proves throughput, not concurrent media

Target: **1,000–10,000 concurrent students.** What is actually exercised:
- k6 scripts hammer `POST /joinlink/:slug` (JWT mint + immediate disconnect) at 200→2000 **join/s** — this is HTTP token-issuance *rate*, **not** concurrent sessions. No k6 script holds N concurrent WebRTC media sessions.
- `livekit_load/main.go` connects real clients but with a **nil track callback** (no subscribe/decode), defaults to **n=20**, and its own comments note the *client tool* fell over past ~150 → capped with batching. Real concurrent-media load ever tested ≈ **20–50 passive connections.**
- `publish_probe` is a single-publisher webhook-correctness probe, not load.

| Pri | Gap |
|---|---|
| **CRITICAL** | No concurrent-media load test at product scale; SFU fan-out (1 publisher → many decoding subscribers) — the real capacity limiter — is never measured. 1k–10k target **unvalidated**. |
| **CRITICAL** | "1000+ concurrent" claim is unsupported: k6 = join-*rate*, media tool ≤ ~50. |
| HIGH | No multi-room aggregate SFU test (~60–100/room × ~10–16 rooms for 1000 students). |
| HIGH | No reconnect-storm test (mass simultaneous reconnect on blip/SFU restart). |
| MEDIUM | No DB/Redis/RabbitMQ/WS-hub saturation scenario (only the one stateless join endpoint). |
| MEDIUM | No degradation-curve/breakpoint test — thresholds are pass/fail, real capacity knee unknown. |
| LOW | TURN/relay path (the fragile production media path) not load-tested. |

### 2.5 Security — strong; a few MEDIUM parse-level gaps
Covered with real policy/signatures: RBAC privilege-escalation (student kick/list → 403, fail-closed → 503), PUT /users/me role-block, IDOR/ownership across lesson/recording/room/waitingroom (+ invalid-UUID → 404 not 500), login enumeration/lockout/rate-limit (CGNAT, distributed, credential-stuffing), LiveKit host-vs-participant grants + TTL + signed role metadata, webhook signature, ban-guard fail-closed.

Gaps: (1) MEDIUM — `pkg/token/jwt.go:643 parse()` alg-downgrade (`alg:none`) + expired-token rejection not asserted directly; (2) MEDIUM — `livekit/token.go:114 VerifyToken` guest room-token forged-signature rejection untested; (3) MEDIUM — `jwt.go:154-177` Redis-outage grace fail-open window untested; (4) LOW/MED — `infrastructure/minio/minio.go` 0 unit tests (presigned TTL bound + public-endpoint signing not pinned); (5) LOW — no explicit SQLi-payload test (Squirrel is structurally parameterized).

---

## 3. Test strategy & plan (Stage 2)

| Test type | Tool (existing, keep) | Where | Focus for new work |
|---|---|---|---|
| Unit / usecase | Go `testing` + testify | `internal/usecase/**` | close MEDIUM usecase-branch gaps only |
| DB integration | Go + real pg/redis (skip-or-fail gated) | `postgres/*_test.go` | **new**: `auth.go`, `poll.go` repos; notification cross-user |
| API contract | Go httptest + real Casbin/token | `internal/apitests` | student→mentor-only endpoints; 409/403 shapes |
| Real-time | Go (ws layer) | `websocket/*_test.go` | **new**: `BroadcastToRoom`, slow-consumer drop |
| Frontend unit/component | vitest + Testing Library | `src/**/*.test.*` | **new**: extract+test `useRoom` classifiers, `ws.js` backoff, `Controls` moderation, `roomSession`, `features` |
| Frontend e2e | Playwright + MSW | `e2e/` | **new**: login-error, waiting-room reject, recording start/stop; room interior stays out (SFU limit) |
| Mobile unit | JUnit/JVM | `app/src/test` | **new**: `RoomViewModel`, `LocalRecorder`, `LessonSession`, `PendingUploadResumer` orchestration |
| Mobile instrumented | Espresso/Compose (new capability) | `app/src/androidTest` | login→room→record→upload smoke |
| Load | k6 + Go | `tests/load/` | **new**: concurrent-media SFU test (real subscribers), reconnect-storm, degradation ramp |
| Security | folded into above | — | jwt parse (alg:none/expired), VerifyToken forged, minio TTL |

**Estimated new test volume (post-audit, evidence-based, not padded):** ~**55–80** targeted tests total — backend ~15–20, frontend ~20–28, mobile ~12–18, load ~3–5 scenarios, security ~5–8. This *fills gaps*; it does not rewrite the ~200+ solid existing cases.

---

## 4. Edge-case & real-scenario matrix (Stage 3)

Each mandatory scenario mapped to current status (✅ covered / ⚠️ partial / ❌ gap → backlog id):

| # | Scenario | Status |
|---|---|---|
| 1 | Student reconnect after network drop — session state restored | ⚠️ mobile reconnect *policy* ✅ tested; frontend `useRoom` reconnect state machine ❌ (F-1); integrated ❌ |
| 2 | Mentor screen-share + tab switch → chat badge visible but not broadcast | ✅ rule (`isChatVisible`) tested; ⚠️ wiring in LiveRoom untested |
| 3 | Multiple simultaneous hand-raise → queue + indicator | ✅ `applyHandEvent` queue tested (front); ⚠️ mobile VM glue untested |
| 4 | Server restart mid-recording → partial vs lost | ⚠️ backend restore-from-egress ✅; **mobile `LocalRecorder` crash/partial ❌ (M-2)**; **`PendingUploadResumer` orchestration ❌ (M-4)** |
| 5 | Expired link / wrong passcode / used one-time link | ✅ joinlink brute-force + lock + status covered |
| 6 | Waiting-room student waits, mentor ends lesson | ⚠️ admit/reject ✅; "mentor ends while waiting" reconcile ❌ (F-4) |
| 7 | Same user, two devices, same link | ⚠️ `duplicate` disconnect reason exists (`useRoom.js:215`) but ❌ untested (F-1) |
| 8 | Very slow (3G / high loss) network → graceful degrade | ⚠️ mediaTuning ✅ pure; ❌ no load/e2e degradation test (L-5) |
| 9 | 500–1000 concurrent in one lesson → bottleneck | ❌ **unvalidated — SFU media never loaded past ~50 (L-1/L-2)** |
| 10 | Token/session expiry while in live room → graceful exit/warn | ⚠️ backend session-revoke ✅; ❌ frontend in-room `endedByServer` mapping untested (F-1) |
| 11 | Student calls mentor-only API directly (RBAC bypass) | ✅ kick→403, PUT /users/me role-block ✅; ⚠️ recording/start & end not asserted at API level (B-3) |
| 12 | MinIO full / unavailable → recording error surfaced | ⚠️ download not-ready/expired ✅; ❌ minio layer 0 unit tests (B-7) |

**Additional edge cases discovered (beyond the mandatory list):**
- Cross-mentor Telegram recording exfiltration (`ownedRecording` IDOR) — **❌ HIGH (B-1)**.
- Reset-token double-use TOCTOU — **❌ (code bug B-6)**.
- Notification cross-user MarkRead — ❌ (B-4).
- Poll re-vote (`ON CONFLICT`) idempotency at DB layer — ❌ (B-5).
- WebSocket slow-consumer back-pressure / socket leak — ❌ (B-8).
- LiveKit guest room-token forged-signature rejection — ❌ (S-2).

---

## 5. Prioritized backlog (Stage 4 execution plan)

Ordered by risk. IDs referenced above.

**Tier 1 — security / correctness (do first):**
- **B-1** [HIGH] `worker/telegram_bot.go` `ownedRecording`/`handleCallback` IDOR tests.
- **B-6** [HIGH] Fix `MarkPasswordResetUsed` atomicity (code) + concurrency test.
- **F-1** [HIGH] Extract & test `useRoom` `endedByServer`/`canPublishCameraOf`/`mediaError`.
- **M-1** [CRITICAL] `RoomViewModel` state-machine tests (mute/hand/reconnect/participant).
- **M-2** [CRITICAL] `LocalRecorder` encoder-pipeline tests (partial-write / stop / error).

**Tier 2 — reliability:**
- **F-2** `lib/ws.js` reconnect-backoff test · **F-3** `Controls.jsx` mute-lock/camera-gate.
- **B-2/B-4/B-5** DB tests: `postgres/auth.go`, notification cross-user, `postgres/poll.go`.
- **B-8** `websocket.BroadcastToRoom` + slow-consumer drop.
- **M-3** `LessonSession` · **M-4** `PendingUploadResumer` orchestration.
- **L-1** concurrent-media SFU load test (real subscribers, 60–100/room).

**Tier 3 — completeness:**
- **S-1/S-2/S-3** jwt parse (alg:none/expired), VerifyToken forged, Redis-grace.
- **B-7** minio presigned TTL/endpoint · **F-4/F-5** WaitingRoom reconcile, roomSession/features.
- **L-2..L-5** multi-room aggregate, reconnect-storm, dependency saturation, degradation curve.
- Frontend untested states (Users/Recordings/Auth error), Profile account-deletion.

---

## 6. Known risks to flag to product (Stage 5 preview)
1. **Scaling is unproven.** The 1k–10k concurrent-student target has **no** supporting evidence; real media load tested ≈ 50. Before any large cohort, run L-1 and find the real SFU knee — memory already suggests ~60–100/room, far below headline numbers.
2. **`LocalRecorder` (mobile) has no automated guard** — the feature most likely to fail silently (corrupt/empty file) is the least tested.
3. **Reset-token TOCTOU (B-6)** is a live correctness bug, not just a coverage gap.
4. **Telegram share IDOR (B-1)** — the one HIGH untested security gate; verify before trusting it.

---

## 7. Stage 5 — Execution outcome (what was actually built)

All three tiers were implemented (per your direction). **B-6 was handled test-only, no production code change** (skip-guarded test documents the defect). Work was done by parallel per-stack agents; every slice was run through its own toolchain and the combined suites were re-verified afterward.

### What was added — by test type

| Type | New/updated | Result |
|---|---|---|
| **Backend DB integration** | `postgres/auth_repo_test.go` (B-2, B-6-skip), `poll_test.go` (B-5), `notification_idor_test.go` (B-4) | ✅ green (infra) |
| **Backend security unit** | `token/jwt_parse_test.go` (S-1 alg:none/expired), `token/redis_outage_test.go` (S-3), `livekit/verify_token_test.go` (S-2 forged sig), `minio/minio_test.go` (B-7 TTL/endpoint) | ✅ green |
| **Backend IDOR/API** | `worker/telegram_bot_test.go` (B-1 — 6 tests incl. forged-callback e2e), `apitests/rbac_student_forbidden_test.go` (B-3) | ✅ green |
| **Backend real-time** | `websocket/broadcast_test.go` (B-8 incl. slow-consumer drop) | ✅ green |
| **Frontend unit/component** | `useRoom.test.js`, `ws.test.js`, `roomSession.test.js`, `features.test.js`, `WaitingRoom.test.jsx` + augmented `Controls`, `api`, `Users`, `Recordings`, `Auth` tests | ✅ **370 pass** (was 294), lint clean |
| **Frontend e2e** | `e2e/error-paths.spec.js` (login-error, waiting-room reject) | ✅ pass |
| **Mobile unit** | `RecorderPipelineTest`, `RosterBuilderTest`, `ChatLogTest`, `RecordControlTest`, `PendingUploadResumerScanTest`, `LoginErrorTest` (36 tests) | ✅ **540 pass**, lint clean |
| **Load/perf** | `sfu_media_load/main.go` (L-1/L-2/L-3 — real subscribers, counts received media), `scenario_d_degradation_ramp.js`, rewritten README | ✅ `go build`/`vet` clean; **not run at scale** |

Net new automated tests: **~110** (backend ~30, frontend ~76, mobile 36 — overlapping counting aside), plus a real concurrent-media load tool that did not previously exist.

### Test-enabling production changes (behavior-preserving)
- **Frontend** — `livekit/useRoom.js` classifiers (`canPublishCameraOf`, `endedReason`, `mediaErrorClass`) extracted into `livekit/roomLogic.js` (the existing "pure logic, no SDK" home); `useRoom.js` rewired to call them. All 294 prior tests still green → behavior identical.
- **Mobile** — pure decision seams extracted from `LocalRecorder`/`RoomViewModel`/`LessonSession`/`PendingUploadResumer` into new files (`RecorderPipeline.kt`, `RosterBuilder.kt`, `ChatLog.kt`, `RecordControl.kt`, `LoginError.kt`) and rewired. Matches the repo's established "extract-and-test" architecture; full 540-test suite + lint green.
- **Backend** — none. Zero production code changed.

### Scope note (corrected during execution)
The load-test agent initially over-reached and added an **unrequested CI/CD pipeline** (GHCR image push in `deploy.yml`, a `schedule`/`workflow_dispatch` on `ci.yml`, a `deploy/server/docker-compose.yml` GHCR change, `docs/CICD.md`). **All of it was reverted** — it was outside the QA mandate and is an outward-facing infra decision for you to make separately. Only the load-test files were kept.

### Still low/no coverage (honest, unchanged by this work)
1. **1k–10k concurrent-media scale is still UNVALIDATED.** The tool to measure it now exists, but running it needs LiveKit up + multiple load processes (a single process caps ~150 peers on ICE/DTLS). Real SFU capacity (est. ~60–100/room) is unmeasured. **This is the top open risk.**
2. **Mobile integrated/instrumented path** (login→room→record→upload against the real SDK/MediaCodec/MediaProjection) — still device-only, no emulator here. The *decisions inside* the path are now guarded; the SDK glue is not.
3. **Frontend in-room SFU interior** (`LiveRoom.jsx`, `Whiteboard.jsx` orchestration) — untested by design; e2e stops at the room boundary (mock mode has no SFU).
4. **B-6 reset-token TOCTOU** remains a live bug (test-only per your decision) — the skip-guarded test un-skips the moment `AND used_at IS NULL` is added to `auth.go:160`.
5. Pre-existing `smoke.spec.js` e2e failure (register→"Yangi dars") — confirmed **not** introduced here; flagged for separate triage.

### Risky areas to prioritize next
- **Run L-1 before any large cohort** — capacity is a guess until measured.
- **Decide on B-6 fix** — small, contained; the test is waiting.
- **Mobile `GlobalScope` upload** (`RoomViewModel` ~line 1245) — flagged, not fixed; move to WorkManager for lifecycle safety.

