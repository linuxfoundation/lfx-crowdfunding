// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler } from 'h3';
import { useBackendFetch } from '../../utils/backend-fetch';
import type { CardDetailsWire } from '../../types/payment.types';
import type { CardDetails } from '#shared/types/payment.types';

export default defineEventHandler(async (event): Promise<CardDetails> => {
  const raw = await useBackendFetch<CardDetailsWire>(event, '/v1/me/payment-account', {
    method: 'GET',
  });
  return {
    paymentMethodId: raw.payment_method_id,
    lastFour: raw.last_four,
    brand: raw.brand,
    expiryMonth: raw.expiry_month,
    expiryYear: raw.expiry_year,
  };
});
