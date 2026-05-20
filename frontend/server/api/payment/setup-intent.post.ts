// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler } from 'h3';
import { useBackendFetch } from '../../utils/backend-fetch';
import type { SetupIntentResult } from '#shared/types/payment.types';

export default defineEventHandler(async (event): Promise<SetupIntentResult> => {
  return useBackendFetch<SetupIntentResult>(event, '/v1/me/setup-intent', { method: 'POST' });
});
