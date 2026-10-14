// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, createError, getRequestHeader, getRequestURL } from 'h3';

// SameSite=Lax does not stop same-site (other *.linuxfoundation.org) forms from riding the
// session cookie, so state-changing /api requests must prove they come from this origin.
const SAFE_METHODS = new Set(['GET', 'HEAD', 'OPTIONS']);
const ALLOWED_FETCH_SITES = new Set(['same-origin', 'none']);

function isSameHost(origin: string, host: string): boolean {
  try {
    return new URL(origin).host === host;
  } catch {
    return false; // e.g. the opaque "null" origin
  }
}

export default defineEventHandler((event) => {
  if (SAFE_METHODS.has(event.method.toUpperCase())) return;

  // Host only, not protocol: behind the ingress the pod sees http while the browser sends https.
  const url = getRequestURL(event, { xForwardedHost: true });
  if (!url.pathname.startsWith('/api/')) return;

  // Browsers always send Origin on cross-origin POSTs; Sec-Fetch-Site covers the rest.
  // ponytail: neither header = non-browser client (SSR, Playwright, curl), which cannot carry a victim's cookies.
  const origin = getRequestHeader(event, 'origin');
  const fetchSite = getRequestHeader(event, 'sec-fetch-site');
  const crossOrigin = origin
    ? !isSameHost(origin, url.host)
    : !!fetchSite && !ALLOWED_FETCH_SITES.has(fetchSite);
  if (crossOrigin) {
    throw createError({ statusCode: 403, statusMessage: 'Cross-origin request rejected' });
  }

  // A browser form can only send urlencoded, multipart or text/plain bodies, so JSON-only
  // keeps a form from producing a usable payload even if the origin check is bypassed.
  const contentType = getRequestHeader(event, 'content-type');
  if (contentType && !contentType.toLowerCase().startsWith('application/json')) {
    throw createError({ statusCode: 415, statusMessage: 'Content-Type must be application/json' });
  }
});
