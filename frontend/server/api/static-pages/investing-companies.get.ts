// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { MOCK_INVESTING_COMPANIES } from '#server/mock-data/investing-companies';
import type { InvestingCompaniesResponse } from '#shared/types/static-pages.types';

// TODO: replace with a proxy call to the Go backend API
export default defineEventHandler((): InvestingCompaniesResponse => {
  return MOCK_INVESTING_COMPANIES;
});
