// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody, createError } from 'h3';
import type { FundraiseSubmission, FundraiseRecord } from '../../types/fundraise.types';

const submissions: FundraiseRecord[] = [];

export default defineEventHandler(async (event) => {
  const body = await readBody<FundraiseSubmission>(event);

  if (!body.initiativeType || !body.details?.name || !body.details?.description) {
    throw createError({
      statusCode: 400,
      statusMessage: 'initiativeType, details.name, and details.description are required',
    });
  }

  const record: FundraiseRecord = {
    id: crypto.randomUUID(),
    initiativeType: body.initiativeType,
    details: body.details,
    goals: body.goals ?? null,
    status: 'pending_review',
    createdAt: new Date().toISOString(),
  };

  submissions.push(record);

  return { success: true, submission: record };
});
