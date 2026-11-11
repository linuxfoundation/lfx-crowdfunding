// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { PaymentErrorLike } from '~/types/payment-error.types';

// The backend's ErrProfileNotSynced (400) messages all tell the caller to
// "call PATCH /crowdfunding/me to sync your profile". That is developer text,
// so donors see this instead. Signing in again re-runs the profile sync.
export const PROFILE_NOT_SYNCED_MESSAGE =
  "We couldn't finish setting up your account for payments. Please sign out, sign back in, and try again.";

const PROFILE_NOT_SYNCED_RE = /sync your profile/i;

// paymentErrorMessage returns the text to show a donor for a failed payment
// call. It prefers the server's message over FetchError.message, which is the
// raw '[POST] "/api/...": 400' line. Only non-empty strings count: Nitro's
// error body sets `error: true`.
export const paymentErrorMessage = (e: unknown, fallback: string): string => {
  const err = (e ?? {}) as PaymentErrorLike;
  const message =
    [err.data?.error, err.data?.message, err.data?.statusMessage, err.message].find(
      (v): v is string => typeof v === 'string' && v.trim() !== '',
    ) ?? fallback;
  return PROFILE_NOT_SYNCED_RE.test(message) ? PROFILE_NOT_SYNCED_MESSAGE : message;
};
