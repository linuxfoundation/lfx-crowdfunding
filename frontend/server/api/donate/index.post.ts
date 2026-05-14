// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody, createError } from 'h3';
import type { DonateSubmission, DonationRecord } from '#shared/types/donate.types';

const donations: DonationRecord[] = [];

export default defineEventHandler(async (event) => {
  const body = await readBody<DonateSubmission>(event);

  if (!body.initiativeId || !body.amountCents || body.amountCents <= 0) {
    throw createError({
      statusCode: 400,
      statusMessage: 'initiativeId and a positive amountCents are required',
    });
  }

  const record: DonationRecord = {
    id: crypto.randomUUID(),
    initiativeId: body.initiativeId,
    tierId: body.tierId ?? null,
    tierName: body.tierName ?? null,
    amountCents: body.amountCents,
    contact: body.contact,
    createdAt: new Date().toISOString(),
  };

  donations.push(record);

  return { success: true, donation: record };
});
