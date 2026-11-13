// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { PaymentErrorLike } from '~/types/payment-error.types';

export const PROFILE_SYNC_FAILED_MESSAGE =
  'We could not update your profile. Please try again later.';

// The backend refuses the sync (403) when a new account's first token carries
// its Auth0 subject instead of the username (#485). Trying again later doesn't
// help there; a fresh sign-in does.
export const SIGN_IN_AGAIN_MESSAGE =
  "Your sign-in didn't finish setting up your account. Please sign out and sign in again.";

const SIGN_IN_AGAIN_RE = /sign out and sign in again/i;

// profileSyncErrorMessage returns the toast text for a failed PATCH /api/me.
export const profileSyncErrorMessage = (e: unknown): string => {
  const err = (e ?? {}) as PaymentErrorLike;
  const serverMessages = [err.data?.error, err.data?.message, err.data?.statusMessage];
  return serverMessages.some((v) => typeof v === 'string' && SIGN_IN_AGAIN_RE.test(v))
    ? SIGN_IN_AGAIN_MESSAGE
    : PROFILE_SYNC_FAILED_MESSAGE;
};
