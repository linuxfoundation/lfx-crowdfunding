// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import type { H3Event } from 'h3';

vi.mock('../utils/backend-fetch', () => ({ useBackendFetch: vi.fn() }));

import { useBackendFetch } from '../utils/backend-fetch';
import {
  getApproverStatus,
  listOfflineDonations,
  recordOfflineDonation,
  toBackendInput,
} from './offline-donations.services';

const fetchMock = vi.mocked(useBackendFetch);
const event = {} as H3Event;

const backendRow = {
  id: 'd1',
  initiative_id: 'i1',
  initiative_name: 'SPDK',
  initiative_slug: 'spdk',
  organization_name: 'Dell Corporation',
  organization_domain: 'dell.com',
  amount_cents: 800000,
  category: 'Donation',
  received_on: '2026-10-20',
  source_txn_id: 'SPDK Dell 2026-10',
  status: 'succeeded',
  recorded_by: 'approver',
  created_on: '2026-11-13T00:00:00Z',
};

const request = {
  initiativeId: 'i1',
  organization: { name: 'Dell Corporation', domain: 'dell.com' },
  amount: '8000',
  receivedOn: '2026-10-20',
  category: 'Donation',
  sourceTxnId: 'SPDK Dell 2026-10',
};

beforeEach(() => fetchMock.mockReset());

describe('recordOfflineDonation', () => {
  it('posts the snake_case body and maps the reply', async () => {
    fetchMock.mockResolvedValueOnce(backendRow);

    const got = await recordOfflineDonation(event, request);

    expect(fetchMock).toHaveBeenCalledWith(event, '/crowdfunding/offline-donations', {
      method: 'POST',
      body: {
        initiative_id: 'i1',
        organization: { name: 'Dell Corporation', domain: 'dell.com' },
        amount: '8000',
        received_on: '2026-10-20',
        category: 'Donation',
        source_txn_id: 'SPDK Dell 2026-10',
      },
    });
    expect(got).toMatchObject({
      id: 'd1',
      amountCents: 800000,
      sourceTxnId: 'SPDK Dell 2026-10',
      recordedBy: 'approver',
    });
  });

  it.each([
    ['no initiative', { ...request, initiativeId: '' }],
    ['no organization', { ...request, organization: undefined }],
    ['no body', null],
  ])('rejects %s without calling the backend', async (_, body) => {
    await expect(recordOfflineDonation(event, body)).rejects.toMatchObject({ statusCode: 400 });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('drops non-string fields', () => {
    const input = toBackendInput({ ...request, amount: 8000 as unknown as string });
    expect(input.amount).toBe('');
  });
});

describe('listOfflineDonations', () => {
  it('filters by initiative', async () => {
    fetchMock.mockResolvedValueOnce({
      data: [backendRow],
      meta: { total: 1, limit: 50, offset: 0 },
    });

    const got = await listOfflineDonations(event, 'i1');

    expect(fetchMock).toHaveBeenCalledWith(
      event,
      '/crowdfunding/offline-donations?limit=50&initiative_id=i1',
    );
    expect(got).toEqual({ data: [expect.objectContaining({ id: 'd1' })], total: 1 });
  });
});

describe('getApproverStatus', () => {
  it('is true only when the backend says so', async () => {
    fetchMock.mockResolvedValueOnce({ approver: true });
    expect(await getApproverStatus(event)).toBe(true);
    fetchMock.mockResolvedValueOnce({ approver: false });
    expect(await getApproverStatus(event)).toBe(false);
  });

  it('is false when the call fails (e.g. signed out)', async () => {
    fetchMock.mockRejectedValueOnce(Object.assign(new Error('unauthorized'), { statusCode: 401 }));
    expect(await getApproverStatus(event)).toBe(false);
  });
});
