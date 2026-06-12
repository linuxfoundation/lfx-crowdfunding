// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { getRequestURL, getRequestProtocol } from 'h3';
import type { H3Event } from 'h3';

/**
 * Builds the GitHub OAuth callback URL from the incoming request's origin
 * (honoring X-Forwarded-Host / X-Forwarded-Proto set by the ingress), so it
 * matches the host the app is actually served on without relying on NUXT_APP_URL.
 * Must return the same value for the authorize and token-exchange steps.
 *
 * getRequestURL({ xForwardedHost: true }) gives us the correct host, but the
 * protocol reflects the pod-internal connection (always HTTP behind K8s ingress).
 * getRequestProtocol() checks X-Forwarded-Proto to get the external protocol.
 */
export function getGithubCallbackUrl(event: H3Event): string {
  const { host } = getRequestURL(event, { xForwardedHost: true });
  const proto = getRequestProtocol(event);
  return `${proto}://${host}/api/github/callback`;
}
