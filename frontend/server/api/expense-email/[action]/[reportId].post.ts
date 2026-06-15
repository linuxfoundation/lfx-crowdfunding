// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, getRouterParam, createError } from 'h3';
import { useBackendFetch } from '../../../utils/backend-fetch';

// Valid actions the Reimbursement Service accepts.
const VALID_ACTIONS = new Set(['approve', 'reject']);

// POST /api/expense-email/:action/:reportId
// BFF proxy — forwards the expense action to the Go backend which in turn
// calls the Reimbursement Service with X-API-KEY authentication.
// Auth is enforced by server/middleware/require-auth.ts.
export default defineEventHandler(async (event): Promise<void> => {
  const action = getRouterParam(event, 'action')!;
  const reportId = getRouterParam(event, 'reportId')!;

  if (!VALID_ACTIONS.has(action)) {
    throw createError({
      statusCode: 400,
      statusMessage: `Invalid action: must be one of ${[...VALID_ACTIONS].join(', ')}`,
    });
  }

  await useBackendFetch(
    event,
    `/v1/expense/${encodeURIComponent(action)}/${encodeURIComponent(reportId)}`,
    {
      method: 'POST',
    },
  );
});
