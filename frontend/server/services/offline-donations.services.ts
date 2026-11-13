// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { H3Event } from 'h3';
import { createError } from 'h3';
import { useBackendFetch } from '../utils/backend-fetch';
import type {
  BackendOfflineDonation,
  BackendOfflineDonationInput,
  BackendOfflineDonationList,
} from '../types/offline-donations.types';
import type {
  OfflineDonation,
  OfflineDonationList,
  OfflineDonationRequest,
} from '#shared/types/offline-donation.types';

const LIST_LIMIT = 50;

export const mapOfflineDonation = (d: BackendOfflineDonation): OfflineDonation => ({
  id: d.id,
  initiativeId: d.initiative_id,
  initiativeName: d.initiative_name ?? '',
  initiativeSlug: d.initiative_slug ?? '',
  organizationName: d.organization_name,
  organizationDomain: d.organization_domain,
  amountCents: d.amount_cents,
  category: d.category,
  receivedOn: d.received_on,
  sourceTxnId: d.source_txn_id,
  status: d.status,
  recordedBy: d.recorded_by ?? '',
  createdOn: d.created_on,
});

const str = (v: unknown) => (typeof v === 'string' ? v : '');

/** Converts the browser's body to the backend's, rejecting a malformed one before it leaves the BFF. */
export const toBackendInput = (
  body: Partial<OfflineDonationRequest> | null | undefined,
): BackendOfflineDonationInput => {
  const org = body?.organization;
  const input: BackendOfflineDonationInput = {
    initiative_id: str(body?.initiativeId),
    organization: { name: str(org?.name), domain: str(org?.domain) },
    amount: str(body?.amount),
    received_on: str(body?.receivedOn),
    category: str(body?.category),
    source_txn_id: str(body?.sourceTxnId),
  };
  if (!input.initiative_id || !input.organization.name || !input.organization.domain) {
    throw createError({ statusCode: 400, message: 'Initiative and organization are required' });
  }
  return input;
};

export const recordOfflineDonation = async (
  event: H3Event,
  body: Partial<OfflineDonationRequest> | null | undefined,
): Promise<OfflineDonation> => {
  const created = await useBackendFetch<BackendOfflineDonation>(
    event,
    '/crowdfunding/offline-donations',
    {
      method: 'POST',
      body: toBackendInput(body),
    },
  );
  return mapOfflineDonation(created);
};

export const listOfflineDonations = async (
  event: H3Event,
  initiativeId: string,
): Promise<OfflineDonationList> => {
  const params = new URLSearchParams({ limit: String(LIST_LIMIT) });
  if (initiativeId) params.set('initiative_id', initiativeId);
  const res = await useBackendFetch<BackendOfflineDonationList>(
    event,
    `/crowdfunding/offline-donations?${params}`,
  );
  return { data: (res.data ?? []).map(mapOfflineDonation), total: res.meta?.total ?? 0 };
};

/** Whether the caller is an approver; any failure (signed out, backend down) hides approver UI. */
export const getApproverStatus = async (event: H3Event): Promise<boolean> => {
  try {
    const res = await useBackendFetch<{ approver?: boolean }>(event, '/crowdfunding/me/approver');
    return res.approver === true;
  } catch {
    return false;
  }
};
