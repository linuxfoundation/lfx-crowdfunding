// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody } from 'h3';
import { recordLedgerAdjustment } from '../../services/ledger-adjustments.services';
import type {
  LedgerAdjustmentRequest,
  LedgerAdjustmentResult,
} from '#shared/types/ledger-adjustment.types';

// Approver-only (#489): the gateway and the backend both check, so the BFF only forwards.
export default defineEventHandler(async (event): Promise<LedgerAdjustmentResult> => {
  const body = await readBody<Partial<LedgerAdjustmentRequest>>(event);
  return recordLedgerAdjustment(event, body);
});
