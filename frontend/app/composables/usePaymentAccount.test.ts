// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import type { CardDetails } from '#shared/types/payment.types';

const mockGetStripe = vi.fn();
const mockShowError = vi.fn();
const mockFetch = vi.fn();

globalThis.$fetch = mockFetch as typeof $fetch;
// @ts-expect-error — Nuxt auto-import stub
globalThis.useStripe = () => ({ getStripe: mockGetStripe });
// @ts-expect-error — Nuxt auto-import stub
globalThis.useErrorToast = () => ({ showError: mockShowError });

import { usePaymentAccount } from './usePaymentAccount';

const fakeCard: CardDetails = {
  paymentMethodId: 'pm_abc123',
  lastFour: '4242',
  brand: 'visa',
  expiryMonth: 12,
  expiryYear: 2026,
};

// Reset module-level singletons between tests.
// usePaymentAccount uses module-level refs and fetchInFlight — we reimport the
// module fresh each test by resetting via the exported refs.
const resetState = () => {
  const { card, loading, error } = usePaymentAccount();
  card.value = null;
  loading.value = false;
  error.value = null;
};

describe('usePaymentAccount', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetState();
  });

  describe('fetchCard', () => {
    it('calls GET /api/payment/account and stores the result', async () => {
      mockFetch.mockResolvedValueOnce(fakeCard);

      const { fetchCard, card } = usePaymentAccount();
      await fetchCard();

      expect(mockFetch).toHaveBeenCalledWith('/api/payment/account');
      expect(card.value).toEqual(fakeCard);
    });

    it('sets loading=true during the call and false afterwards', async () => {
      const { fetchCard, loading } = usePaymentAccount();

      let loadingDuringCall = false;
      mockFetch.mockImplementationOnce(async () => {
        loadingDuringCall = loading.value;
        return fakeCard;
      });

      await fetchCard();

      expect(loadingDuringCall).toBe(true);
      expect(loading.value).toBe(false);
    });

    it('deduplicates concurrent calls — $fetch called only once', async () => {
      let resolveFirst!: (v: CardDetails) => void;
      const firstFetch = new Promise<CardDetails>((res) => {
        resolveFirst = res;
      });
      mockFetch.mockReturnValueOnce(firstFetch);

      const { fetchCard } = usePaymentAccount();
      const p1 = fetchCard();
      const p2 = fetchCard(); // second call while first is in-flight

      resolveFirst(fakeCard);
      await Promise.all([p1, p2]);

      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    it('sets card=null and does NOT call showError on 404', async () => {
      mockFetch.mockRejectedValueOnce({ statusCode: 404 });

      const { fetchCard, card } = usePaymentAccount();
      await fetchCard(); // should not throw

      expect(card.value).toBeNull();
      expect(mockShowError).not.toHaveBeenCalled();
    });

    it('sets error and calls showError on non-404 API error', async () => {
      mockFetch.mockRejectedValueOnce({
        statusCode: 500,
        data: { message: 'Internal server error' },
      });

      const { fetchCard, error } = usePaymentAccount();
      await fetchCard();

      expect(error.value).toBe('Internal server error');
      expect(mockShowError).toHaveBeenCalledWith('Internal server error');
    });

    it('falls back to data.statusMessage when data.message is absent', async () => {
      mockFetch.mockRejectedValueOnce({
        statusCode: 502,
        data: { statusMessage: 'Bad Gateway' },
      });

      const { fetchCard, error } = usePaymentAccount();
      await fetchCard();

      expect(error.value).toBe('Bad Gateway');
    });

    it('falls back to error.message when data is absent', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      const { fetchCard, error } = usePaymentAccount();
      await fetchCard();

      expect(error.value).toBe('Network error');
    });

    it('uses default message when error has no message at all', async () => {
      mockFetch.mockRejectedValueOnce({});

      const { fetchCard, error } = usePaymentAccount();
      await fetchCard();

      expect(error.value).toBe('Could not load your payment account.');
    });

    it('resets loading=false even when the call throws', async () => {
      mockFetch.mockRejectedValueOnce(new Error('fail'));

      const { fetchCard, loading } = usePaymentAccount();
      await fetchCard();

      expect(loading.value).toBe(false);
    });

    it('allows a second fetch after the first completes', async () => {
      mockFetch
        .mockResolvedValueOnce(fakeCard)
        .mockResolvedValueOnce({ ...fakeCard, lastFour: '9999' });

      const { fetchCard, card } = usePaymentAccount();
      await fetchCard();
      await fetchCard();

      expect(mockFetch).toHaveBeenCalledTimes(2);
      expect(card.value?.lastFour).toBe('9999');
    });
  });

  describe('saveCard', () => {
    const fakeCardElement = {} as import('@stripe/stripe-js').StripeCardNumberElement;

    it('calls setup-intent, then confirmCardSetup, then saves the payment method', async () => {
      mockFetch
        .mockResolvedValueOnce({ clientSecret: 'seti_secret_abc' }) // setup-intent
        .mockResolvedValueOnce(fakeCard); // save payment method

      const mockStripe = {
        confirmCardSetup: vi.fn().mockResolvedValueOnce({
          setupIntent: { payment_method: 'pm_abc123' },
          error: null,
        }),
      };
      mockGetStripe.mockResolvedValueOnce(mockStripe);

      const { saveCard, card } = usePaymentAccount();
      await saveCard(fakeCardElement);

      expect(mockFetch).toHaveBeenCalledTimes(2);
      expect(mockFetch.mock.calls[0]).toEqual(['/api/payment/setup-intent', { method: 'POST' }]);
      expect(mockStripe.confirmCardSetup).toHaveBeenCalledWith('seti_secret_abc', {
        payment_method: { card: fakeCardElement },
      });
      expect(mockFetch.mock.calls[1]).toEqual([
        '/api/payment/method',
        { method: 'POST', body: { paymentMethodId: 'pm_abc123' } },
      ]);
      expect(card.value).toEqual(fakeCard);
    });

    it('extracts paymentMethodId from payment_method object (not string)', async () => {
      mockFetch
        .mockResolvedValueOnce({ clientSecret: 'seti_secret_abc' })
        .mockResolvedValueOnce(fakeCard);

      const mockStripe = {
        confirmCardSetup: vi.fn().mockResolvedValueOnce({
          setupIntent: { payment_method: { id: 'pm_from_object' } },
          error: null,
        }),
      };
      mockGetStripe.mockResolvedValueOnce(mockStripe);

      const { saveCard } = usePaymentAccount();
      await saveCard(fakeCardElement);

      expect(mockFetch.mock.calls[1][1]).toMatchObject({
        body: { paymentMethodId: 'pm_from_object' },
      });
    });

    it('sets loading=true during the call and false afterwards', async () => {
      const { saveCard, loading } = usePaymentAccount();

      let loadingDuringCall = false;
      mockFetch.mockImplementationOnce(async () => {
        loadingDuringCall = loading.value;
        return { clientSecret: 'seti_secret_abc' };
      });
      mockFetch.mockResolvedValueOnce(fakeCard);
      mockGetStripe.mockResolvedValueOnce({
        confirmCardSetup: vi.fn().mockResolvedValueOnce({
          setupIntent: { payment_method: 'pm_abc123' },
          error: null,
        }),
      });

      await saveCard(fakeCardElement);

      expect(loadingDuringCall).toBe(true);
      expect(loading.value).toBe(false);
    });

    it('throws and sets error when Stripe.js fails to load', async () => {
      mockFetch.mockResolvedValueOnce({ clientSecret: 'seti_secret_abc' });
      mockGetStripe.mockResolvedValueOnce(null);

      const { saveCard, error } = usePaymentAccount();
      await expect(saveCard(fakeCardElement)).rejects.toThrow('Stripe.js failed to load');
      expect(error.value).toContain('Stripe.js failed to load');
      expect(mockShowError).toHaveBeenCalled();
    });

    it('throws and sets error when confirmCardSetup returns a stripeError', async () => {
      mockFetch.mockResolvedValueOnce({ clientSecret: 'seti_secret_abc' });
      mockGetStripe.mockResolvedValueOnce({
        confirmCardSetup: vi.fn().mockResolvedValueOnce({
          setupIntent: null,
          error: { message: 'Your card was declined.' },
        }),
      });

      const { saveCard, error } = usePaymentAccount();
      await expect(saveCard(fakeCardElement)).rejects.toThrow();
      expect(error.value).toBe('Your card was declined.');
      expect(mockShowError).toHaveBeenCalledWith('Your card was declined.');
    });

    it('throws and sets error when setupIntent returns no payment_method', async () => {
      mockFetch.mockResolvedValueOnce({ clientSecret: 'seti_secret_abc' });
      mockGetStripe.mockResolvedValueOnce({
        confirmCardSetup: vi.fn().mockResolvedValueOnce({
          setupIntent: { payment_method: null },
          error: null,
        }),
      });

      const { saveCard, error } = usePaymentAccount();
      await expect(saveCard(fakeCardElement)).rejects.toThrow();
      expect(error.value).toContain('Stripe did not return a payment method');
    });

    it('throws and sets error when the setup-intent $fetch call fails', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Gateway timeout'));

      const { saveCard, error } = usePaymentAccount();
      await expect(saveCard(fakeCardElement)).rejects.toThrow('Gateway timeout');
      expect(error.value).toBe('Gateway timeout');
      expect(mockShowError).toHaveBeenCalledWith('Gateway timeout');
    });

    it('throws and sets error when saving the payment method fails', async () => {
      mockFetch
        .mockResolvedValueOnce({ clientSecret: 'seti_secret_abc' })
        .mockRejectedValueOnce(new Error('Method save failed'));

      mockGetStripe.mockResolvedValueOnce({
        confirmCardSetup: vi.fn().mockResolvedValueOnce({
          setupIntent: { payment_method: 'pm_abc123' },
          error: null,
        }),
      });

      const { saveCard, error } = usePaymentAccount();
      await expect(saveCard(fakeCardElement)).rejects.toThrow('Method save failed');
      expect(error.value).toBe('Method save failed');
    });

    it('uses default message when saveCard error has no message', async () => {
      mockFetch.mockRejectedValueOnce({});

      const { saveCard, error } = usePaymentAccount();
      await expect(saveCard(fakeCardElement)).rejects.toBeTruthy();
      expect(error.value).toBe('Failed to save your card.');
    });

    it('resets loading=false even when saveCard throws', async () => {
      mockFetch.mockRejectedValueOnce(new Error('fail'));

      const { saveCard, loading } = usePaymentAccount();
      await expect(saveCard(fakeCardElement)).rejects.toBeTruthy();
      expect(loading.value).toBe(false);
    });

    it('re-throws the original error so callers can handle it', async () => {
      const originalError = { message: 'Card declined', statusCode: 402 };
      mockFetch.mockRejectedValueOnce(originalError);

      const { saveCard } = usePaymentAccount();
      await expect(saveCard(fakeCardElement)).rejects.toMatchObject({ statusCode: 402 });
    });
  });
});
