// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, getCookie, createError, getRequestURL } from 'h3';
import type { ProtectedRoute } from '../types/auth.types';

// Add entries here to require authentication on additional routes.
// Omit `methods` to protect all HTTP methods on matching paths.
const PROTECTED: ProtectedRoute[] = [
  { match: (p) => p.startsWith('/api/payment/') },
  {
    match: (p) => /^\/api\/initiatives\/[^/]+\/donations$/.test(p),
    methods: ['POST'],
  },
  {
    match: (p) => /^\/api\/initiatives\/[^/]+\/process-approval\/[^/]+$/.test(p),
    methods: ['POST'],
  },
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
});
