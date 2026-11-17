// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface BackendFetchOptions {
  method?: 'GET' | 'POST' | 'DELETE' | 'PATCH';
  body?: unknown;
  headers?: Record<string, string>;
  // Platform v2 API (e.g. /query/resources) rather than the CF backend.
  platform?: boolean;
}

export interface BackendFetchResult<T> {
  status: number;
  data: T;
}
