// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineConfig, devices } from '@playwright/test';

// Required env vars for authenticated e2e tests:
//   NUXT_E2E_TEST_MODE=true         — enables /api/e2e-auth endpoint in Nuxt
//   NUXT_E2E_TEST_USERNAME=<user>   — username injected into the mock session
//   DISABLED_MOCK_LOCAL_PRINCIPAL=<user>  — backend bypasses JWT for this username
//   ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS=true — required by backend config validation

export default defineConfig({
  testDir: './e2e/tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:3000',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
