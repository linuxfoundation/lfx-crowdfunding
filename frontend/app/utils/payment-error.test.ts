// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from 'vitest';
import { paymentErrorMessage, PROFILE_NOT_SYNCED_MESSAGE } from './payment-error';

const fallback = 'Something failed.';

describe('paymentErrorMessage', () => {
  it('prefers the server message over the raw FetchError line', () => {
    const e = {
      message: '[POST] "/api/payment/setup-intent": 400',
      data: { message: 'Your card was declined.' },
    };
    expect(paymentErrorMessage(e, fallback)).toBe('Your card was declined.');
  });

  it('replaces the backend profile-not-synced text with friendly copy', () => {
    // h3 strips the em dash from statusMessage, so match on the stable part.
    const e = {
      message: '[POST] "/api/payment/setup-intent": 400',
      data: {
        error: true,
        statusMessage:
          'account id not set  call PATCH /crowdfunding/me to sync your profile before using payment features',
        message:
          'account id not set  call PATCH /crowdfunding/me to sync your profile before using payment features',
      },
    };
    expect(paymentErrorMessage(e, fallback)).toBe(PROFILE_NOT_SYNCED_MESSAGE);
  });

  it('ignores a non-string data.error (Nitro sets error: true)', () => {
    expect(paymentErrorMessage({ data: { error: true, message: 'Bad input' } }, fallback)).toBe(
      'Bad input',
    );
  });

  it('uses a plain Error message', () => {
    expect(paymentErrorMessage(new Error('Stripe.js failed to load.'), fallback)).toBe(
      'Stripe.js failed to load.',
    );
  });

  it('falls back when there is no message', () => {
    expect(paymentErrorMessage({}, fallback)).toBe(fallback);
    expect(paymentErrorMessage(undefined, fallback)).toBe(fallback);
    expect(paymentErrorMessage({ message: '  ' }, fallback)).toBe(fallback);
  });
});
