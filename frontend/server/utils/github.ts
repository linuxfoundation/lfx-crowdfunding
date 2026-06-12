// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { H3Event } from 'h3';

/**
 * Builds the GitHub OAuth callback URL from the incoming request's origin
 * (honoring X-Forwarded-Host / X-Forwarded-Proto set by the ingress), so it
 * matches the host the app is actually served on without relying on NUXT_APP_URL.
 * Must return the same value for the authorize and token-exchange steps.
 */
export function getGithubCallbackUrl(event: H3Event): string {
  const { origin } = getRequestURL(event, { xForwardedHost: true });
  return `${origin}/api/github/callback`;
}
