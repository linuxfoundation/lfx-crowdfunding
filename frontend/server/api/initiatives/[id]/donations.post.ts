// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody, getHeader } from 'h3';
import { useBackendFetch } from '../../../utils/backend-fetch';
import type { DonationRequest, DonationResult } from '#shared/types/payment.types';

export default defineEventHandler(async (event): Promise<DonationResult> => {
  const id = event.context.params!.id;
  const idempotencyKey = getHeader(event, 'idempotency-key') ?? '';
  const body = await readBody<DonationRequest>(event);
  return useBackendFetch<DonationResult>(event, `/v1/initiatives/${id}/donations`, {
    method: 'POST',
    body,
    headers: { 'Idempotency-Key': idempotencyKey },
  });
});
