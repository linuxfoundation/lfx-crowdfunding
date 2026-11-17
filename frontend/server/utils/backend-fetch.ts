// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { H3Event } from 'h3';
import { createError } from 'h3';
import type { BackendFetchOptions, BackendFetchResult } from '../types/backend-fetch.types';
import { getAuthCookie } from './auth-cookies';

/** Calls the backend and returns its reply with the status code, e.g. to tell 201 from 200. */
export const useBackendFetchWithStatus = async <T = unknown>(
  event: H3Event,
  path: string,
  options: BackendFetchOptions = {},
): Promise<BackendFetchResult<T>> => {
  const config = useRuntimeConfig();
  const baseURL = ((options.platform && config.lfxApiBaseUrl) || config.apiBaseUrl) as string;
  const token = getAuthCookie(event, 'auth_oidc_token') ?? '';

  try {
    const res = await $fetch.raw<T>(path, {
      baseURL,
      method: options.method ?? 'GET',
      body: options.body,
      headers: {
        ...(token ? { authorization: `Bearer ${token}` } : {}),
        ...options.headers,
      },
    });
    return { status: res.status, data: res._data as T };
  } catch (err: unknown) {
    const e = err as {
      status?: number;
      statusCode?: number;
      data?: { message?: string };
      statusMessage?: string;
    };
    const status = e?.status ?? e?.statusCode ?? 500;
    const message = e?.data?.error ?? e?.data?.message ?? e?.statusMessage ?? 'Upstream error';
    throw createError({ statusCode: status, statusMessage: message });
  }
};

export const useBackendFetch = async <T = unknown>(
  event: H3Event,
  path: string,
  options: BackendFetchOptions = {},
): Promise<T> => (await useBackendFetchWithStatus<T>(event, path, options)).data;
