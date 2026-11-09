// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { H3Error } from 'h3';
import { ResponseBodyError } from 'openid-client';

// Auth failures that happen in normal traffic and are already handled (cookies cleared, user
// sent back to login): Auth0 rejecting an expired/revoked refresh token or a reused code, and
// the callback rejecting a request without a valid login cookie (bots, back button, stale tabs).
export function isExpectedAuthError(error: unknown): boolean {
  if (error instanceof ResponseBodyError) return error.error === 'invalid_grant';
  if (error instanceof H3Error) return error.statusCode >= 400 && error.statusCode < 500;
  return false;
}

// Logs expected failures as one line: Datadog counts every stderr line toward the error-rate
// monitor, and a full error object is ~18 lines. Anything else keeps its stack.
export function logAuthError(label: string, error: unknown): void {
  if (!isExpectedAuthError(error)) {
    console.error(`${label}:`, error);
    return;
  }
  let detail: string;
  if (error instanceof ResponseBodyError) {
    detail = `${error.error}: ${error.error_description ?? error.message}`;
  } else {
    const h3Error = error as H3Error;
    detail = `${h3Error.statusCode}: ${h3Error.statusMessage || h3Error.message}`;
  }
  // The callback's message can carry query-string text; keep it on one line.
  console.warn(`${label} (expected) ${detail.replace(/\s+/g, ' ')}`);
}
