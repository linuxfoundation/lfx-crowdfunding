// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import type { H3Event } from 'h3';

vi.mock('../utils/backend-fetch', () => ({
  useBackendFetch: vi.fn(),
  useBackendFetchWithStatus: vi.fn(),
}));

import { useBackendFetch, useBackendFetchWithStatus } from '../utils/backend-fetch';
import {
  listLedgerAdjustments,
  recordLedgerAdjustment,
  toBackendAdjustmentInput,
} from './ledger-adjustments.services';

const fetchMock = vi.mocked(useBackendFetch);
const fetchWithStatusMock = vi.mocked(useBackendFetchWithStatus);
const event = {} as H3Event;

const backendRow = {
  id: 'a1',
  kind: 'return',
  initiative_id: 'i1',
  initiative_name: 'Example Initiative',
  initiative_slug: 'example-initiative',
  organization_name: 'CNCF',
  organization_domain: 'cncf.io',
  paid_to: '',
  amount_cents: 150000,
  category: 'mentorship',
  reason: 'Mentee failed',
  adjusted_on: '2026-11-02',
  source_txn_id: 'TEST-RETURN-1',
  status: 'succeeded',
  recorded_by: 'approver',
  created_on: '2026-11-13T00:00:00Z',
};

const request = {
  kind: 'return' as const,
  initiativeId: 'i1',
  organization: { name: 'CNCF', domain: 'cncf.io' },
  paidTo: '',
  amount: '1500',
  adjustedOn: '2026-11-02',
  category: 'mentorship',
  reason: 'Mentee failed',
  sourceTxnId: 'TEST-RETURN-1',
};

describe('ledger-adjustments.services', () => {
  beforeEach(() => {
    fetchMock.mockReset();
    fetchWithStatusMock.mockReset();
  });

  it('sends the snake_case body and maps a 201 to created', async () => {
    fetchWithStatusMock.mockResolvedValueOnce({ status: 201, data: backendRow });
    const res = await recordLedgerAdjustment(event, request);
    expect(fetchWithStatusMock).toHaveBeenCalledWith(event, '/crowdfunding/ledger-adjustments', {
      method: 'POST',
      body: {
        kind: 'return',
        initiative_id: 'i1',
        organization: { name: 'CNCF', domain: 'cncf.io' },
        paid_to: '',
        amount: '1500',
        adjusted_on: '2026-11-02',
        category: 'mentorship',
        reason: 'Mentee failed',
        source_txn_id: 'TEST-RETURN-1',
      },
    });
    expect(res).toMatchObject({
      id: 'a1',
      kind: 'return',
      amountCents: 150000,
      reason: 'Mentee failed',
      created: true,
    });
  });

  it('maps a 200 resubmit to created=false', async () => {
    fetchWithStatusMock.mockResolvedValueOnce({ status: 200, data: backendRow });
    expect((await recordLedgerAdjustment(event, request)).created).toBe(false);
  });

  it('sends a payout without an organization as null', () => {
    const input = toBackendAdjustmentInput({
      ...request,
      kind: 'payout',
      organization: null,
      paidTo: 'Jane',
    });
    expect(input.organization).toBeNull();
    expect(input.paid_to).toBe('Jane');
  });

  it('rejects an unknown kind or a missing initiative before calling the backend', () => {
    expect(() => toBackendAdjustmentInput({ ...request, kind: 'refund' as never })).toThrow();
    expect(() => toBackendAdjustmentInput({ ...request, initiativeId: '' })).toThrow();
    expect(() => toBackendAdjustmentInput(null)).toThrow();
  });

  it('lists with clamped paging and the initiative filter', async () => {
    fetchMock.mockResolvedValueOnce({
      data: [backendRow],
      meta: { total: 1, limit: 50, offset: 0 },
    });
    const res = await listLedgerAdjustments(event, 'i1', { limit: '500', offset: 'x' });
    expect(fetchMock).toHaveBeenCalledWith(
      event,
      '/crowdfunding/ledger-adjustments?limit=50&offset=0&initiative_id=i1',
    );
    expect(res).toEqual({ data: [expect.objectContaining({ id: 'a1', paidTo: '' })], total: 1 });
  });
});
