// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { H3Event } from 'h3';
import { getCookie, createError } from 'h3';

export const useBackendFetch = async <T = unknown>(
  event: H3Event,
  path: string,
  options: {
    method?: 'GET' | 'POST' | 'DELETE' | 'PATCH';
    body?: unknown;
    headers?: Record<string, string>;
  } = {},
): Promise<T> => {
  const config = useRuntimeConfig();
  const baseURL = config.backendBaseUrl as string;
  const token = getCookie(event, 'auth_oidc_token') ?? '';

  try {
    return await $fetch<T>(path, {
      baseURL,
      method: options.method ?? 'GET',
      body: options.body,
      headers: {
        ...(token ? { authorization: `Bearer ${token}` } : {}),
        ...options.headers,
      },
    });
  } catch (err: any) {
    const status = err?.status ?? err?.statusCode ?? 500;
    const message = err?.data?.message ?? err?.statusMessage ?? 'Upstream error';
    throw createError({ statusCode: status, statusMessage: message });
  }
};
