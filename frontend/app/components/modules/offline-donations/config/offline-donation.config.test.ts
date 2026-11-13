// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, expect, it } from 'vitest';
import {
  emptyOfflineDonationForm,
  isOfflineDonationFormComplete,
  isValidDollarAmount,
  normalizeDollarAmount,
  offlineCategoryOptions,
  offlineDonationErrorMessage,
  offlineDonationLink,
  toIsoDate,
  toOfflineDonationRequest,
} from './offline-donation.config';

describe('dollar amounts', () => {
  it.each([
    ['1300', true],
    ['1300.5', true],
    ['$1,300.50', true],
    [' 0.01 ', true],
    ['0', false],
    ['0.00', false],
    ['10.005', false],
    ['-5', false],
    ['abc', false],
    ['', false],
    ['99999999', false],
  ])('%s → %s', (raw, valid) => {
    expect(isValidDollarAmount(raw)).toBe(valid);
  });

  it('strips $, commas and spaces before sending', () => {
    expect(normalizeDollarAmount('$ 1,300.50')).toBe('1300.50');
  });
});

describe('offlineCategoryOptions', () => {
  it('puts the default first and drops duplicates', () => {
    expect(offlineCategoryOptions('project', ['Development', 'Donation', ''])).toEqual([
      'Donation',
      'Development',
    ]);
  });

  it('defaults mentorship initiatives to mentorship', () => {
    expect(offlineCategoryOptions('mentorship', [])).toEqual(['mentorship']);
  });
});

describe('toIsoDate', () => {
  it('uses the picked calendar day', () => {
    expect(toIsoDate(new Date(2026, 9, 5))).toBe('2026-10-05');
  });
});

describe('form → request', () => {
  const complete = {
    ...emptyOfflineDonationForm(),
    organization: { name: 'Dell Corporation', domain: 'dell.com' },
    amount: '$8,000',
    receivedOn: new Date(2026, 9, 20),
    sourceTxnId: '  SPDK Dell 2026-10 ',
  };

  it('needs every field', () => {
    expect(isOfflineDonationFormComplete(complete)).toBe(true);
    expect(isOfflineDonationFormComplete({ ...complete, organization: null })).toBe(false);
    expect(isOfflineDonationFormComplete({ ...complete, amount: '8,000.123' })).toBe(false);
    expect(isOfflineDonationFormComplete({ ...complete, receivedOn: null })).toBe(false);
    expect(isOfflineDonationFormComplete({ ...complete, sourceTxnId: ' ' })).toBe(false);
  });

  it('normalizes the values sent', () => {
    expect(toOfflineDonationRequest('i1', complete)).toEqual({
      initiativeId: 'i1',
      organization: { name: 'Dell Corporation', domain: 'dell.com' },
      amount: '8000',
      receivedOn: '2026-10-20',
      category: 'Donation',
      sourceTxnId: 'SPDK Dell 2026-10',
    });
  });
});

describe('offlineDonationErrorMessage', () => {
  it('explains a reused reference', () => {
    expect(offlineDonationErrorMessage(409)).toMatch(/already recorded in the Ledger/);
  });

  it('passes the backend reason through on 400', () => {
    expect(offlineDonationErrorMessage(400, 'amount must be positive')).toContain(
      'amount must be positive',
    );
  });

  it('says a retry is safe on an outage', () => {
    expect(offlineDonationErrorMessage(503)).toMatch(/won't create a second entry/);
  });
});

it('links to the page with the initiative filled in', () => {
  expect(offlineDonationLink('a b')).toBe('/approvals/offline-donations?initiative=a%20b');
});
