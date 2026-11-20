// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { H3Event } from 'h3';
import { createError } from 'h3';
import { useBackendFetch, useBackendFetchWithStatus } from '../utils/backend-fetch';
import type {
  BackendLedgerAdjustment,
  BackendLedgerAdjustmentInput,
  BackendLedgerAdjustmentList,
} from '../types/ledger-adjustments.types';
import { LIST_LIMIT, MAX_LIST_LIMIT, intParam } from './offline-donations.services';
import type {
  LedgerAdjustment,
  LedgerAdjustmentKind,
  LedgerAdjustmentList,
  LedgerAdjustmentRequest,
  LedgerAdjustmentResult,
} from '#shared/types/ledger-adjustment.types';

// Returns and payouts approvers record (#489); gated like offline donations.

const KINDS: readonly string[] = ['return', 'payout'] satisfies LedgerAdjustmentKind[];

export const mapLedgerAdjustment = (a: BackendLedgerAdjustment): LedgerAdjustment => ({
  id: a.id,
  kind: a.kind as LedgerAdjustmentKind,
  initiativeId: a.initiative_id,
  initiativeName: a.initiative_name ?? '',
  initiativeSlug: a.initiative_slug ?? '',
  organizationName: a.organization_name,
  organizationDomain: a.organization_domain,
  paidTo: a.paid_to,
  amountCents: a.amount_cents,
  category: a.category,
  reason: a.reason,
  adjustedOn: a.adjusted_on,
  sourceTxnId: a.source_txn_id,
  status: a.status,
  recordedBy: a.recorded_by ?? '',
  createdOn: a.created_on,
});

const str = (v: unknown) => (typeof v === 'string' ? v : '');

/** Converts the browser's body to the backend's, rejecting a malformed one before it leaves the BFF. */
export const toBackendAdjustmentInput = (
  body: Partial<LedgerAdjustmentRequest> | null | undefined,
): BackendLedgerAdjustmentInput => {
  const org = body?.organization;
  const input: BackendLedgerAdjustmentInput = {
    kind: str(body?.kind),
    initiative_id: str(body?.initiativeId),
    organization: org ? { name: str(org.name), domain: str(org.domain) } : null,
    paid_to: str(body?.paidTo),
    amount: str(body?.amount),
    adjusted_on: str(body?.adjustedOn),
    category: str(body?.category),
    reason: str(body?.reason),
    source_txn_id: str(body?.sourceTxnId),
  };
  if (!KINDS.includes(input.kind) || !input.initiative_id) {
    throw createError({ statusCode: 400, message: 'Kind and initiative are required' });
  }
  return input;
};

/** Records the adjustment. The backend answers 201 for a new entry and 200 with the stored one for a resubmit. */
export const recordLedgerAdjustment = async (
  event: H3Event,
  body: Partial<LedgerAdjustmentRequest> | null | undefined,
): Promise<LedgerAdjustmentResult> => {
  const { status, data } = await useBackendFetchWithStatus<BackendLedgerAdjustment>(
    event,
    '/crowdfunding/ledger-adjustments',
    { method: 'POST', body: toBackendAdjustmentInput(body) },
  );
  return { ...mapLedgerAdjustment(data), created: status === 201 };
};

export const listLedgerAdjustments = async (
  event: H3Event,
  initiativeId: string,
  page: { limit?: unknown; offset?: unknown } = {},
): Promise<LedgerAdjustmentList> => {
  const limit = intParam(page.limit, LIST_LIMIT, 1, MAX_LIST_LIMIT);
  const offset = intParam(page.offset, 0, 0, Number.MAX_SAFE_INTEGER);
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  if (initiativeId) params.set('initiative_id', initiativeId);
  const res = await useBackendFetch<BackendLedgerAdjustmentList>(
    event,
    `/crowdfunding/ledger-adjustments?${params}`,
  );
  return { data: (res.data ?? []).map(mapLedgerAdjustment), total: res.meta?.total ?? 0 };
};
