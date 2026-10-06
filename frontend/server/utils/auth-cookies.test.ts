// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect, vi } from 'vitest';

vi.stubGlobal('useRuntimeConfig', () => ({ auth0ClientSecret: 'test-secret' }));

import { encryptValue, decryptValue, authCookieOptions } from './auth-cookies';

describe('auth cookies', () => {
  it('round-trips and does not expose the plaintext', () => {
    const enc = encryptValue('raw-access-token');
    expect(enc).not.toContain('raw-access-token');
    expect(decryptValue(enc)).toBe('raw-access-token');
  });

  it('rejects tampered, legacy plaintext and missing values', () => {
    const enc = encryptValue('t');
    const tampered = enc.slice(0, -2) + (enc.endsWith('AA') ? 'BB' : 'AA');
    expect(decryptValue(tampered)).toBeUndefined();
    expect(decryptValue('eyJhbGciOi.legacy.jwt')).toBeUndefined();
    expect(decryptValue(undefined)).toBeUndefined();
  });

  it('is host-only', () => {
    expect(authCookieOptions(60)).not.toHaveProperty('domain');
  });
});
