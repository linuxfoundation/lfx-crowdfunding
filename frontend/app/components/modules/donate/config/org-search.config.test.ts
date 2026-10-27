// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from 'vitest';
import { ORG_WEBSITE_PATTERN, manualOrgClaim } from './org-search.config';

describe('ORG_WEBSITE_PATTERN', () => {
  it.each([
    'acme.com',
    'www.acme.co.uk',
    'https://Acme.com/about',
    'http://my-org.io',
    'acme.com.',
    'https://acme.com?ref=donate',
    'https://acme.com#about',
  ])('accepts %s', (website) => {
    expect(ORG_WEBSITE_PATTERN.test(website)).toBe(true);
  });

  it.each([
    'acme',
    'acme.',
    'acme .com',
    'ftp://acme.com',
    'acme.c',
    'https://',
    'acme.com:8080',
    '-acme.com',
    'acme-.com',
  ])('rejects %s', (website) => {
    expect(ORG_WEBSITE_PATTERN.test(website)).toBe(false);
  });
});

describe('manualOrgClaim', () => {
  it('is null until both fields are valid, so Continue stays disabled', () => {
    expect(manualOrgClaim('', '')).toBeNull();
    expect(manualOrgClaim('Acme', '')).toBeNull();
    expect(manualOrgClaim('  ', 'acme.com')).toBeNull();
    expect(manualOrgClaim('Acme', 'acme')).toBeNull();
  });

  it('trims the name and reduces the website to its host', () => {
    expect(manualOrgClaim(' Acme ', 'https://www.Acme.com/about?x=1')).toEqual({
      name: 'Acme',
      domain: 'acme.com',
    });
    expect(manualOrgClaim('Acme', 'acme.com.')).toEqual({ name: 'Acme', domain: 'acme.com' });
  });
});
