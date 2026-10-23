// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from 'vitest';
import { toOrgClaimBody } from './org-claim';

describe('toOrgClaimBody', () => {
  it('returns nothing for an individual donation', () => {
    expect(toOrgClaimBody(undefined)).toEqual({});
  });

  it('maps the org to the backend wire shape', () => {
    expect(toOrgClaimBody({ name: 'Acme', domain: 'acme.com' })).toEqual({
      organization: { name: 'Acme', domain: 'acme.com' },
    });
  });
});
