// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, expect, it } from 'vitest';
import {
  adjustmentErrorMessage,
  adjustmentOutcome,
  adjustmentSummary,
  entryKindOption,
  isOfflineEntryFormComplete,
  toLedgerAdjustmentRequest,
} from './ledger-adjustment.config';
import { emptyOfflineDonationForm } from './offline-donation.config';
import type { OfflineDonationForm } from '~/types/offline-donation.types';
import type { LedgerAdjustmentResult } from '#shared/types/ledger-adjustment.types';

const cncf = { name: 'CNCF', domain: 'cncf.io' };

const filledForm = (overrides: Partial<OfflineDonationForm>): OfflineDonationForm => ({
  ...emptyOfflineDonationForm('mentorship'),
  amount: '$1,500',
  receivedOn: new Date(2026, 10, 2),
  sourceTxnId: ' TEST Row 15 return ',
  ...overrides,
});

describe('isOfflineEntryFormComplete', () => {
  it('needs an org for a donation, and no reason', () => {
    expect(isOfflineEntryFormComplete(filledForm({ kind: 'donation', organization: cncf }))).toBe(
      true,
    );
    expect(isOfflineEntryFormComplete(filledForm({ kind: 'donation' }))).toBe(false);
  });

  it('needs an org and a reason for a return', () => {
    expect(
      isOfflineEntryFormComplete(
        filledForm({ kind: 'return', organization: cncf, reason: 'Mentee failed' }),
      ),
    ).toBe(true);
    expect(
      isOfflineEntryFormComplete(filledForm({ kind: 'return', organization: cncf, reason: '  ' })),
    ).toBe(false);
    expect(
      isOfflineEntryFormComplete(filledForm({ kind: 'return', reason: 'Mentee failed' })),
    ).toBe(false);
  });

  it('needs a payee and a reason for a payout, but no org', () => {
    expect(
      isOfflineEntryFormComplete(filledForm({ kind: 'payout', paidTo: 'Jane', reason: 'Reward' })),
    ).toBe(true);
    expect(
      isOfflineEntryFormComplete(filledForm({ kind: 'payout', paidTo: ' ', reason: 'Reward' })),
    ).toBe(false);
    expect(
      isOfflineEntryFormComplete(
        filledForm({ kind: 'payout', paidTo: 'Jane', reason: 'x'.repeat(501) }),
      ),
    ).toBe(false);
  });
});

describe('toLedgerAdjustmentRequest', () => {
  it('sends a positive amount, the picked date and trimmed text', () => {
    const req = toLedgerAdjustmentRequest(
      'i1',
      filledForm({
        kind: 'return',
        organization: cncf,
        reason: ' Mentee failed ',
        paidTo: 'leftover',
      }),
    );
    expect(req).toEqual({
      kind: 'return',
      initiativeId: 'i1',
      organization: cncf,
      paidTo: '',
      amount: '1500',
      adjustedOn: '2026-11-02',
      category: 'mentorship',
      reason: 'Mentee failed',
      sourceTxnId: 'TEST Row 15 return',
    });
  });

  it('keeps the payee of a payout', () => {
    const req = toLedgerAdjustmentRequest(
      'i1',
      filledForm({ kind: 'payout', paidTo: ' Jane ', reason: 'Reward' }),
    );
    expect(req.paidTo).toBe('Jane');
    expect(req.organization).toBeNull();
  });

  it('refuses a donation', () => {
    expect(() => toLedgerAdjustmentRequest('i1', filledForm({ kind: 'donation' }))).toThrow();
  });
});

describe('adjustment display', () => {
  const result: LedgerAdjustmentResult = {
    id: 'a1',
    kind: 'return',
    initiativeId: 'i1',
    initiativeName: 'Kmesh',
    initiativeSlug: 'kmesh',
    organizationName: 'CNCF',
    organizationDomain: 'cncf.io',
    paidTo: '',
    amountCents: 150000,
    category: 'mentorship',
    reason: 'Mentee failed',
    adjustedOn: '2026-11-02',
    sourceTxnId: 'TEST',
    status: 'succeeded',
    recordedBy: 'approver',
    createdOn: '2026-11-13T00:00:00Z',
    created: true,
  };

  it('summarises a return and a payout', () => {
    expect(adjustmentSummary(result)).toBe('Returned to CNCF');
    expect(adjustmentSummary({ ...result, kind: 'payout', paidTo: 'Jane' })).toBe('Paid to Jane');
  });

  it('describes a new and a resubmitted adjustment', () => {
    expect(adjustmentOutcome(result, 'Kmesh')).toMatchObject({
      title: 'Return recorded',
      message: expect.stringContaining('Returned to CNCF: $1,500.00 out of Kmesh'),
    });
    expect(adjustmentOutcome({ ...result, created: false }, 'Kmesh').title).toBe(
      'Already recorded',
    );
  });

  it('labels the form per kind', () => {
    expect(entryKindOption('return').submitLabel).toBe('Record return');
    expect(entryKindOption('payout').orgRequired).toBe(false);
  });

  it('explains a reused reference', () => {
    expect(adjustmentErrorMessage(409)).toContain('already recorded');
    expect(adjustmentErrorMessage(400, 'reason must be 1-500 characters')).toContain(
      'reason must be',
    );
  });
});
