// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody, getHeader, createError } from 'h3';
import { useBackendFetch } from '../../../utils/backend-fetch';
import type { BackendInitiative } from '../../../types/initiatives.types';
import type { DonationResultWire } from '../../../types/payment.types';
import type { DonationRequest, DonationResult } from '#shared/types/payment.types';

export default defineEventHandler(async (event): Promise<DonationResult> => {
  const slug = getRouterParam(event, 'slug');
  const idempotencyKey = getHeader(event, 'idempotency-key');
  if (!idempotencyKey) {
    throw createError({ statusCode: 400, statusMessage: 'Idempotency-Key header required' });
  }

  const { apiBaseUrl } = useRuntimeConfig();
  const initiative = await $fetch<BackendInitiative>(`${apiBaseUrl}/v1/initiatives/${slug}`).catch(
    () => {
      throw createError({ statusCode: 404, statusMessage: 'Initiative not found' });
    },
  );

  const body = await readBody<DonationRequest>(event);
  const raw = await useBackendFetch<DonationResultWire>(
    event,
    `/v1/initiatives/${initiative.id}/donations`,
    {
      method: 'POST',
      body: {
        amount_cents: body.amountInCents,
        stripe_payment_method_id: body.stripePaymentMethodId,
        ...(body.category ? { category: body.category } : {}),
        ...(body.organizationId ? { organization_id: body.organizationId } : {}),
      },
      headers: { 'Idempotency-Key': idempotencyKey },
    },
  );
  return {
    id: raw.id,
    status: raw.status,
    ...(raw.client_secret ? { clientSecret: raw.client_secret } : {}),
  };
});
