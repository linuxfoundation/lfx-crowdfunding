// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody } from 'h3';
import { recordOfflineDonation } from '../../services/offline-donations.services';
import type {
  OfflineDonationRequest,
  OfflineDonationResult,
} from '#shared/types/offline-donation.types';

// Approver-only (#464): the gateway and the backend both check, so the BFF only forwards.
export default defineEventHandler(async (event): Promise<OfflineDonationResult> => {
  const body = await readBody<Partial<OfflineDonationRequest>>(event);
  return recordOfflineDonation(event, body);
});
