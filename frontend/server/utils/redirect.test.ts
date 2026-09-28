// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from 'vitest';
import { getSafeRedirectUrl, isValidRedirectUrl } from './redirect';

describe('isValidRedirectUrl', () => {
  it.each([
    '/fundraise',
    '/initiatives/my-project?tab=updates',
    '/fundraise/new?step=2#repos',
    'https://crowdfunding.linuxfoundation.org/initiatives',
    '  /fundraise  ',
  ])('accepts %j', (url) => {
    expect(isValidRedirectUrl(url)).toBe(true);
  });

  it.each([
    '//evil.com',
    '/\\evil.com',
    '/%2F/evil.com',
    'https://evil.com',
    'javascript:alert(1)',
    // Browsers strip tabs and newlines, turning these into //evil.com.
    '/\t/evil.com',
    '/\n/evil.com',
    '/\r/evil.com',
    '/%09/evil.com',
    '/%0A/evil.com',
  ])('rejects %j', (url) => {
    expect(isValidRedirectUrl(url)).toBe(false);
  });
});

describe('getSafeRedirectUrl', () => {
  it('returns the fallback for a tab-obfuscated protocol-relative path', () => {
    expect(getSafeRedirectUrl('/\t/evil.com', '/fundraise')).toBe('/fundraise');
  });

  it('returns a valid relative path trimmed', () => {
    expect(getSafeRedirectUrl(' /fundraise ', '/')).toBe('/fundraise');
  });
});
