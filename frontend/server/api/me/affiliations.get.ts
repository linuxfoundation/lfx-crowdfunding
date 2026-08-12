// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler } from 'h3';
import { getAffiliations } from '../../services/affiliations.services';
import type { AffiliationCandidates } from '#shared/types/affiliation.types';

export default defineEventHandler(
  (event): Promise<AffiliationCandidates> => getAffiliations(event),
);
