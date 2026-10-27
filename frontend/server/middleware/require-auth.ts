// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, createError, getRequestURL } from 'h3';
import { getAuthCookie } from '../utils/auth-cookies';
import { refreshSession } from '../utils/auth-refresh';
import type { ProtectedRoute } from '../types/auth.types';

// Compiled once at module load — not inside the match functions — to avoid
// allocating a new RegExp on every request.
const RE_DONATIONS = /^\/api\/initiatives\/[^/]+\/donations$/;
const RE_PROCESS_APPROVAL = /^\/api\/initiatives\/[^/]+\/process-approval\/[^/]+$/;
const RE_EXPENSE_ACTION = /^\/api\/expense-email\/[^/]+\/[^/]+$/;

// Add entries here to require authentication on additional routes.
// Omit `methods` to protect all HTTP methods on matching paths.
const PROTECTED: ProtectedRoute[] = [
  { match: (p) => p === '/api/me', methods: ['PATCH'] },
  { match: (p) => p.startsWith('/api/me/') },
  { match: (p) => p === '/api/presigned-url', methods: ['POST'] },
  { match: (p) => p.startsWith('/api/payment/') },
  { match: (p) => RE_DONATIONS.test(p), methods: ['POST'] },
  { match: (p) => RE_PROCESS_APPROVAL.test(p), methods: ['POST'] },
  { match: (p) => RE_EXPENSE_ACTION.test(p), methods: ['POST'] },
  { match: (p) => p === '/api/fundraise', methods: ['POST'] },
  { match: (p) => p === '/api/orgs/suggest' },
];

export default defineEventHandler((event) => {
  const path = getRequestURL(event).pathname;
  const method = event.method.toUpperCase();

  const isProtected = PROTECTED.some(
    (route) => route.match(path) && (!route.methods || route.methods.includes(method)),
  );

  if (!isProtected) return;

  // Token format/expiry validation is intentionally delegated to the Go backend,
  // which performs full RS256 JWT validation on every request. This layer only
  // ensures a token cookie is present so the backend sees an Authorization header.
  if (getAuthCookie(event, 'auth_oidc_token')) return;

  const unauthorized = () =>
    createError({ statusCode: 401, statusMessage: 'Authentication required' });
  if (!getAuthCookie(event, 'auth_refresh_token')) throw unauthorized();

  // The access token cookie expires after ~1h while the refresh token lives 30 days. Refresh
  // here so a submit from a long-open page (donate, start a fundraiser) doesn't 401; the new
  // token is visible to useBackendFetch later in this same request.
  return refreshSession(event).then((profile) => {
    if (!profile) throw unauthorized();
  });
});
