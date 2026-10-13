// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect } from 'vitest';

vi.mock('../utils/backend-fetch', () => ({ useBackendFetch: vi.fn() }));
vi.mock('../utils/auth-cookies', () => ({ getAuthCookie: vi.fn() }));

import { isAcceptedLogoUrl } from './affiliations.services';

// Must match the backend's Attribution.Validate logo_url rules, or a query-service logo fails
// the whole fundraise instead of being dropped.
describe('isAcceptedLogoUrl', () => {
  it.each([
    ['https://example.com/logo.png', true],
    [undefined, false],
    ['', false],
    ['http://example.com/logo.png', false],
    ['javascript:alert(1)', false],
    ['https:///logo.png', false],
    ['https://cdn.example.com:8443/a%20b/logo.png?v=2', true],
    ['https://[::1/logo.png', false],
    ['https://example.com:bad/logo.png', false],
    ['https://example.com/a%zzb.png', false],
    ['https://example.com/a b.png', false],
    [`https://example.com/${'a'.repeat(2048)}`, false],
  ])('%s → %s', (url, expected) => {
    expect(isAcceptedLogoUrl(url)).toBe(expected);
  });
});
