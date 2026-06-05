// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, getRouterParam, createError } from 'h3';
import { useBackendFetch } from '../../../../utils/backend-fetch';
import type { BackendInitiative } from '../../../../types/initiatives.types';
import type { ApprovalResult } from '~/types/approval.types';

const VALID_ACTIONS = new Set(['approve', 'decline']);

export default defineEventHandler(async (event): Promise<ApprovalResult> => {
  const slug = getRouterParam(event, 'slug')!;
  const action = getRouterParam(event, 'action')!;

  if (!VALID_ACTIONS.has(action)) {
    throw createError({
      statusCode: 400,
      statusMessage: `Invalid action: must be "approve" or "decline"`,
    });
  }

  const initiative = await useBackendFetch<BackendInitiative>(
    event,
    `/v1/initiatives/${slug}`,
  ).catch((err) => {
    if (err?.status === 404)
      throw createError({ statusCode: 404, statusMessage: 'Initiative not found' });
    throw createError({ statusCode: 502, statusMessage: 'Failed to resolve initiative' });
  });

  // process-approval is not under /me — the caller is an approver, not the resource owner
  const updated = await useBackendFetch<BackendInitiative>(
    event,
    `/v1/initiatives/${initiative.id}/process-approval/${action}`,
    { method: 'POST' },
  );

  return {
    id: updated.id,
    name: updated.name,
    slug: updated.slug,
    status: updated.status,
  };
});
