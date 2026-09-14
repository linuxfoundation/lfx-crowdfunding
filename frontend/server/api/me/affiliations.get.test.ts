// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, afterEach } from 'vitest';
import type { H3Event } from 'h3';

// Auth guards are tested in require-auth.test.ts. This file tests handler logic only.

vi.mock('h3', async (importOriginal) => {
  const actual = await importOriginal<typeof import('h3')>();
  return {
    ...actual,
    defineEventHandler: (fn: unknown) => fn,
  };
});

const mockEvent = {} as H3Event;

// isProduction in affiliations.services.ts is a module-level const read at import time, so each
// case reloads the module fresh with the env var it needs to observe.
describe('GET /api/me/affiliations BFF handler', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.resetModules();
  });

  it('returns sample organizations and projects outside production', async () => {
    vi.stubEnv('NUXT_PUBLIC_APP_ENV', 'development');
    const handler = (await import('./affiliations.get')).default;

    const result = await (handler as (e: unknown) => Promise<unknown>)(mockEvent);

    expect(result).toEqual({
      organizations: [
        { id: '8b1e2c3d-4f56-4a78-9b0c-1d2e3f4a5b6c', name: 'Sample Org — Acme Corp' },
        { id: '3f4a5b6c-7d8e-4f90-a1b2-c3d4e5f60718', name: 'Sample Org — Globex' },
      ],
      projects: [
        { id: '0c1d2e3f-4a5b-4c6d-8e9f-0a1b2c3d4e5f', name: 'Sample Project — Kubernetes' },
        { id: '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d', name: 'Sample Foundation — CNCF' },
      ],
    });
  });

  it('returns empty organizations and projects in production', async () => {
    vi.stubEnv('NUXT_PUBLIC_APP_ENV', 'production');
    const handler = (await import('./affiliations.get')).default;

    const result = await (handler as (e: unknown) => Promise<unknown>)(mockEvent);

    expect(result).toEqual({ organizations: [], projects: [] });
  });
});
