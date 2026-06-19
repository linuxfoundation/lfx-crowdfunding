// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendResponse } from '../../../types/initiatives.types';

export default defineSitemapEventHandler(async () => {
  const { apiBaseUrl } = useRuntimeConfig();

  // Fetch all published initiatives in one request. The backend defaults to
  // status=published; we request a large limit to avoid pagination loops.
  const res = await $fetch<BackendResponse>(`${apiBaseUrl}/v1/initiatives?limit=10000&offset=0`);

  return (res.data ?? []).map((initiative) => ({
    loc: `/initiatives/${initiative.slug}`,
    lastmod: initiative.updated_on ?? undefined,
  }));
});
