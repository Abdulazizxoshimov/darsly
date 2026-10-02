import { defineConfig, devices } from '@playwright/test'

// ⭐ LIVE E2E — MSW mock EMAS. Brauzer → Vite dev (proxy /api → :8087) → HAQIQIY
// backend → Postgres/Redis (DBgacha to'liq flow). Mavjud `e2e/` (MSW smoke) bilan
// yonma-yon: u deterministik, bu esa real uchdan-uchga.
//
// Talab: backend ishlab turishi (`cd backend && go run ./cmd`), seed admin
// (admin@darsly.uz) mavjud bo'lishi — fixtures test hisoblarini admin API orqali yaratadi.
export default defineConfig({
  testDir: './e2e-live',
  testMatch: /.*\.spec\.js/,
  timeout: 45000,
  expect: { timeout: 15000 },
  fullyParallel: false, // bitta backend/DB — ketma-ket xavfsizroq (seed/cleanup toza)
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: [['list'], ['html', { outputFolder: 'playwright-report-live', open: 'never' }]],
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    // Xona testlari uchun kamera/mikrofon ruxsati (headless'да soxta qurilma).
    permissions: ['camera', 'microphone'],
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          args: ['--use-fake-ui-for-media-stream', '--use-fake-device-for-media-stream'],
        },
      },
    },
  ],
  webServer: {
    // VITE_USE_MOCK BERILMAYDI → MSW o'chiq → real backendга proxy.
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    timeout: 60000,
  },
})
