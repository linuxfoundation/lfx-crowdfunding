// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody, getHeader, createError } from 'h3';
import { useBackendFetch } from '../../../utils/backend-fetch';
import type { DonationResultWire } from '../../../types/payment.types';
import type { DonationRequest, DonationResult } from '#shared/types/payment.types';

export default defineEventHandler(async (event): Promise<DonationResult> => {
  const id = event.context.params!.id;
  const idempotencyKey = getHeader(event, 'idempotency-key');
  if (!idempotencyKey) {
    throw createError({ statusCode: 400, statusMessage: 'Idempotency-Key header required' });
  }
  const body = await readBody<DonationRequest>(event);
  const raw = await useBackendFetch<DonationResultWire>(event, `/v1/initiatives/${id}/donations`, {
    method: 'POST',
    body: {
      amount_cents: body.amountInCents,
      stripe_payment_method_id: body.stripePaymentMethodId,
      ...(body.category ? { category: body.category } : {}),
      ...(body.organizationId ? { organization_id: body.organizationId } : {}),
    },
    headers: { 'Idempotency-Key': idempotencyKey },
  });
  return {
    id: raw.id,
    status: raw.status,
    ...(raw.client_secret ? { clientSecret: raw.client_secret } : {}),
  };
});
