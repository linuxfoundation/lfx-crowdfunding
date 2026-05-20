// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler } from 'h3';
import { useBackendFetch } from '../../utils/backend-fetch';
import type { CardDetails } from '#shared/types/payment.types';

export default defineEventHandler(async (event): Promise<CardDetails> => {
  return useBackendFetch<CardDetails>(event, '/v1/me/payment-account', { method: 'GET' });
});
