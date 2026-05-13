// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { MOCK_FEATURED_INITIATIVES } from '#server/mock-data/featured-initiatives';
import type { FeaturedInitiativesResponse } from '#shared/types/static-pages.types';

// TODO: replace with a proxy call to the Go backend API
export default defineEventHandler((): FeaturedInitiativesResponse => {
  return MOCK_FEATURED_INITIATIVES;
});
