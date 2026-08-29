import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E Test Configuration
 * For filter and UI integration tests (setup ready for new tests)
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:8080',
    trace: 'on-first-retry',
    screenshot: 'on',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'cd .. && PORT=8081 ./report_server',
    url: 'http://localhost:8081',
    reuseExistingServer: false,
    timeout: 30000,
  },
});
