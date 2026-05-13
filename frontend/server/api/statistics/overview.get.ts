// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { StatisticsOverview } from '#shared/types/statistics.types';
import { MOCK_OVERVIEW } from '#server/mock-data/statistics';

export default defineEventHandler(async (): Promise<StatisticsOverview> => {
  // TODO: replace with a proxy call to the Go backend API once wired up.
  // In our architecture Nuxt's server layer owns auth only — business data comes
  // from the Go microservice. This handler returns fixture data for scaffolding.
  return MOCK_OVERVIEW;
});
