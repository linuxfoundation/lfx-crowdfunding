// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, getCookie, createError, getRequestURL } from 'h3';
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
  { match: (p) => p === '/api/presigned-url', methods: ['POST'] },
  { match: (p) => p.startsWith('/api/payment/') },
  { match: (p) => RE_DONATIONS.test(p), methods: ['POST'] },
  { match: (p) => RE_PROCESS_APPROVAL.test(p), methods: ['POST'] },
  { match: (p) => RE_EXPENSE_ACTION.test(p), methods: ['POST'] },
  { match: (p) => p === '/api/fundraise', methods: ['POST'] },
];

export default defineEventHandler((event) => {
  const path = getRequestURL(event).pathname;
  const method = event.method.toUpperCase();

  const isProtected = PROTECTED.some(
    (route) => route.match(path) && (!route.methods || route.methods.includes(method)),
  );

  if (!isProtected) return;

  const token = getCookie(event, 'auth_oidc_token');
  if (!token) {
    throw createError({ statusCode: 401, statusMessage: 'Authentication required' });
  }
  // Token format/expiry validation is intentionally delegated to the Go backend,
  // which performs full RS256 JWT validation on every request. This layer only
  // ensures a token cookie is present so the backend sees an Authorization header.
});
