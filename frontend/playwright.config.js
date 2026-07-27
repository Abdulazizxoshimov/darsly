import { defineConfig, devices } from '@playwright/test'

// Smoke test MSW mock bilan ishlaydi (backend shart emas) — deterministik.
export default defineConfig({
  testDir: './e2e',
  timeout: 30000,
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    env: { VITE_USE_MOCK: 'true' },
    timeout: 60000,
  },
})
