// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody } from 'h3';
import { useBackendFetch } from '../../utils/backend-fetch';
import type { CardDetails } from '#shared/types/payment.types';

export default defineEventHandler(async (event): Promise<CardDetails> => {
  const body = await readBody<{ payment_method_id: string }>(event);
  return useBackendFetch<CardDetails>(event, '/v1/me/payment-method', { method: 'POST', body });
});
