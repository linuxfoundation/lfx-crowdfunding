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
  webServer: [
    {
      command: 'pnpm dev',
      url: 'http://localhost:3000/api/health',
      timeout: 180_000,
      reuseExistingServer: !process.env.CI,
      stdout: 'pipe',
      stderr: 'pipe',
    },
    // Production build for the csp project: the CSP plugins skip in dev (Vite needs
    // inline scripts), so a CSP test against `pnpm dev` would pass even with the bug back.
    {
      command:
        'NUXT_BUILD_DIR=node_modules/.cache/nuxt-csp pnpm build && PORT=3001 node --env-file-if-exists=.env .output/server/index.mjs',
      url: 'http://localhost:3001/api/health',
      timeout: 300_000,
      reuseExistingServer: !process.env.CI,
      stdout: 'pipe',
      stderr: 'pipe',
    },
  ],
  projects: [
    {
      name: 'chromium',
      testIgnore: /csp\.spec\.ts/,
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'csp',
      testMatch: /csp\.spec\.ts/,
      use: { ...devices['Desktop Chrome'], baseURL: 'http://localhost:3001' },
    },
  ],
});
