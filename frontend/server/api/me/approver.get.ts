// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, setResponseHeader } from 'h3';
import { getApproverStatus } from '../../services/offline-donations.services';
import type { ApproverStatus } from '#shared/types/offline-donation.types';

export default defineEventHandler(async (event): Promise<ApproverStatus> => {
  setResponseHeader(event, 'Cache-Control', 'private, no-store');
  return { approver: await getApproverStatus(event) };
});
