// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineNuxtPlugin, useRoute } from 'nuxt/app';
import { authState, login } from '~/composables/useAuth';

/**
 * Returns true when the request targets an auth-system route (/api/auth/*).
 * Used to prevent the refresh interceptor from looping on auth endpoints.
 */
function isAuthRoute(request: string | Request): boolean {
  const url = typeof request === 'string' ? request : request.url;
  try {
    return new URL(url, 'http://n').pathname.startsWith('/api/auth/');
  } catch {
    return false;
  }
}

/**
 * Replaces the global $fetch with a wrapper that silently refreshes the Auth0
 * access token on the first 401 and retries the original request once.
 *
 * Guards:
 *   - Only fires when the user had an active session (authState.isAuthenticated).
 *     Anonymous users that hit a stray 401 are not force-redirected to login.
 *   - Skips /api/auth/* routes to prevent refresh loops.
 *
 * Deduplication: concurrent 401s share a single in-flight refresh promise so
 * exactly one POST /api/auth/refresh is made per expiry event.
 *
 * The wrapped function delegates to the original $fetch (raw) for all requests,
 * so there is no recursive loop on the retry call.
 */
export default defineNuxtPlugin(() => {
  const route = useRoute();
  // Capture the original $fetch before replacing it.
  const raw = globalThis.$fetch;

  let refreshInFlight: Promise<boolean> | null = null;

  /** Calls POST /api/auth/refresh exactly once regardless of concurrent callers. */
  function refreshOnce(): Promise<boolean> {
    if (!refreshInFlight) {
      refreshInFlight = raw('/api/auth/refresh', { method: 'POST' })
        .then(() => true)
        .catch(() => false)
        .finally(() => {
          refreshInFlight = null;
        });
    }
    return refreshInFlight;
  }

  // Wrap $fetch with try/catch so we can intercept 401s and retry after refresh.
  // $fetch.create's onResponseError cannot return a new response value, so we
  // wrap at the function level instead.
  //
  // We omit the <T> generic on the inner function so TypeScript infers the return
  // type directly from `raw(...)`, which matches Nuxt's TypedInternalResponse
  // signature. The outer `as typeof raw` cast restores the full $Fetch interface.
  // Note: Nuxt's $Fetch type does not expose `.native`, so it is omitted here.
  const wrapped = Object.assign(
    async function (request: Parameters<typeof raw>[0], options?: Parameters<typeof raw>[1]) {
      try {
        return await raw(request, options);
      } catch (err) {
        const status =
          (err as { status?: number })?.status ?? (err as { statusCode?: number })?.statusCode;

        if (status === 401 && authState.value.isAuthenticated && !isAuthRoute(request)) {
          const ok = await refreshOnce();

          if (ok) {
            // Token refreshed — retry with the same options (new cookie is now set).
            return raw(request, options);
          }

          // Refresh failed (revoked / expired refresh token) — redirect to login.
          await login(route.fullPath);
        }

        throw err;
      }
    },
    {
      // Preserve the $fetch interface so call sites using .create / .raw still work.
      create: raw.create.bind(raw),
      raw: raw.raw.bind(raw),
    },
  ) as typeof raw;

  globalThis.$fetch = wrapped;
});
