// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, getQuery } from 'h3';
import { listOfflineDonations } from '../../services/offline-donations.services';
import type { OfflineDonationList } from '#shared/types/offline-donation.types';

export default defineEventHandler((event): Promise<OfflineDonationList> => {
  const { initiativeId } = getQuery(event);
  return listOfflineDonations(event, typeof initiativeId === 'string' ? initiativeId : '');
});
