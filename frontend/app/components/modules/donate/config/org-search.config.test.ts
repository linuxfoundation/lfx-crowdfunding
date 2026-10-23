// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from 'vitest';
import { ORG_WEBSITE_PATTERN } from './org-search.config';

describe('ORG_WEBSITE_PATTERN', () => {
  it.each(['acme.com', 'www.acme.co.uk', 'https://Acme.com/about', 'http://my-org.io'])(
    'accepts %s',
    (website) => {
      expect(ORG_WEBSITE_PATTERN.test(website)).toBe(true);
    },
  );

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
