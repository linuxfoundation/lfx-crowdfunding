// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, getQuery } from 'h3';
import { listLedgerAdjustments } from '../../services/ledger-adjustments.services';
import type { LedgerAdjustmentList } from '#shared/types/ledger-adjustment.types';

export default defineEventHandler((event): Promise<LedgerAdjustmentList> => {
  const { initiativeId, limit, offset } = getQuery(event);
  return listLedgerAdjustments(event, typeof initiativeId === 'string' ? initiativeId : '', {
    limit,
    offset,
  });
});
