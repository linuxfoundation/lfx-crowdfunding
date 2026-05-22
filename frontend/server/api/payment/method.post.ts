// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody } from 'h3';
import { useBackendFetch } from '../../utils/backend-fetch';
import type { CardDetails } from '#shared/types/payment.types';

interface CardDetailsWire {
  payment_method_id: string;
  last_four: string;
  brand: string;
  expiry_month: number;
  expiry_year: number;
}

export default defineEventHandler(async (event): Promise<CardDetails> => {
  const { paymentMethodId } = await readBody<{ paymentMethodId: string }>(event);
  const raw = await useBackendFetch<CardDetailsWire>(event, '/v1/me/payment-method', {
    method: 'POST',
    body: { payment_method_id: paymentMethodId },
  });
  return {
    paymentMethodId: raw.payment_method_id,
    lastFour: raw.last_four,
    brand: raw.brand,
    expiryMonth: raw.expiry_month,
    expiryYear: raw.expiry_year,
  };
});
