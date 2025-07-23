import { defineConfig, devices } from '@playwright/test';

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1, // there is only 1 instance of the app open at a time
  reporter: process.env.CI ? 'html' : 'list',

  use: {
    baseURL: 'http://localhost:34115',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],

  webServer: [
    {
      command: 'cd .. && PICKLE_INSECURE_S3=yes wails dev',
      url: 'http://127.0.0.1:34115',
      gracefulShutdown: { signal: 'SIGTERM', timeout: 1000 },
      reuseExistingServer: !process.env.CI,
      stdout: 'pipe',
      stderr: 'pipe',
      timeout: 300000,
    },
  ],
});
