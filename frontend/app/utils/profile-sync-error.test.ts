// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from 'vitest';
import {
  profileSyncErrorMessage,
  PROFILE_SYNC_FAILED_MESSAGE,
  SIGN_IN_AGAIN_MESSAGE,
} from './profile-sync-error';

describe('profileSyncErrorMessage', () => {
  it('asks the user to sign in again when the backend refused an Auth0-subject principal', () => {
    const e = {
      message: '[PATCH] "/api/me": 403',
      data: {
        error: true,
        statusMessage:
          "Your sign-in didn't finish setting up your account. Please sign out and sign in again.",
      },
    };
    expect(profileSyncErrorMessage(e)).toBe(SIGN_IN_AGAIN_MESSAGE);
  });

  it('keeps the generic text for other failures', () => {
    expect(profileSyncErrorMessage({ data: { error: true, statusMessage: 'forbidden' } })).toBe(
      PROFILE_SYNC_FAILED_MESSAGE,
    );
    expect(profileSyncErrorMessage(new Error('network'))).toBe(PROFILE_SYNC_FAILED_MESSAGE);
    expect(profileSyncErrorMessage(undefined)).toBe(PROFILE_SYNC_FAILED_MESSAGE);
  });
});
