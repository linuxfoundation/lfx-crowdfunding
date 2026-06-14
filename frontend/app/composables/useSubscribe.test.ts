// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';

const mockGetStripe = vi.fn();
const mockShowError = vi.fn();
const mockFetch = vi.fn();

globalThis.$fetch = mockFetch as typeof $fetch;
// @ts-expect-error — Nuxt auto-import stub
globalThis.useStripe = () => ({ getStripe: mockGetStripe });
// @ts-expect-error — Nuxt auto-import stub
globalThis.useErrorToast = () => ({ showError: mockShowError });

import { useSubscribe } from './useSubscribe';

describe('useSubscribe', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const { loading, error } = useSubscribe();
    loading.value = false;
    error.value = null;
  });

  describe('subscribe — happy path (no 3D Secure)', () => {
    it('calls the subscriptions endpoint with the correct body and idempotency header', async () => {
      mockFetch.mockResolvedValueOnce({ clientSecret: null });

      const { subscribe } = useSubscribe();
      await subscribe('init-1', { amountCents: 1000, frequency: 'monthly' });

      expect(mockFetch).toHaveBeenCalledOnce();
      const [url, opts] = mockFetch.mock.calls[0] as [
        string,
        RequestInit & { headers: Record<string, string> },
      ];
      expect(url).toBe('/api/initiatives/init-1/subscriptions');
      expect(opts.method).toBe('POST');
      expect(opts.body).toMatchObject({ amountCents: 1000, frequency: 'monthly' });
      expect(opts.headers['Idempotency-Key']).toMatch(
        /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i,
      );
    });

    it('returns the result from the endpoint', async () => {
      const serverResult = { clientSecret: null, subscriptionId: 'sub-1' };
      mockFetch.mockResolvedValueOnce(serverResult);

      const { subscribe } = useSubscribe();
      const result = await subscribe('init-1', { amountCents: 500, frequency: 'monthly' });

      expect(result).toEqual(serverResult);
    });

    it('sets loading=true during the call and resets it afterwards', async () => {
      const { subscribe, loading } = useSubscribe();

      let loadingDuringCall = false;
      mockFetch.mockImplementationOnce(async () => {
        loadingDuringCall = loading.value;
        return { clientSecret: null };
      });

      await subscribe('init-1', { amountCents: 100, frequency: 'monthly' });

      expect(loadingDuringCall).toBe(true);
      expect(loading.value).toBe(false);
    });
  });

  describe('subscribe — 3D Secure / Stripe confirmation', () => {
    it('calls stripe.confirmCardPayment when a clientSecret is returned', async () => {
      mockFetch.mockResolvedValueOnce({ clientSecret: 'seti_secret_xyz' });
      const mockStripe = { confirmCardPayment: vi.fn().mockResolvedValueOnce({ error: null }) };
      mockGetStripe.mockResolvedValueOnce(mockStripe);

      const { subscribe } = useSubscribe();
      await subscribe('init-1', { amountCents: 2000, frequency: 'monthly' });

      expect(mockGetStripe).toHaveBeenCalledOnce();
      expect(mockStripe.confirmCardPayment).toHaveBeenCalledWith('seti_secret_xyz');
    });

    it('throws when Stripe.js fails to load (getStripe returns null)', async () => {
      mockFetch.mockResolvedValueOnce({ clientSecret: 'seti_secret_xyz' });
      mockGetStripe.mockResolvedValueOnce(null);

      const { subscribe, error } = useSubscribe();
      await expect(subscribe('init-1', { amountCents: 500, frequency: 'monthly' })).rejects.toThrow(
        'Stripe.js failed to load.',
      );
      expect(error.value).toBe('Stripe.js failed to load.');
      expect(mockShowError).toHaveBeenCalledWith('Stripe.js failed to load.');
    });

    it('throws and sets error when Stripe confirmCardPayment fails', async () => {
      mockFetch.mockResolvedValueOnce({ clientSecret: 'seti_secret_xyz' });
      const mockStripe = {
        confirmCardPayment: vi
          .fn()
          .mockResolvedValueOnce({ error: { message: 'Insufficient funds' } }),
      };
      mockGetStripe.mockResolvedValueOnce(mockStripe);

      const { subscribe, error } = useSubscribe();
      await expect(
        subscribe('init-1', { amountCents: 500, frequency: 'monthly' }),
      ).rejects.toThrow();
      expect(error.value).toBe('Insufficient funds');
      expect(mockShowError).toHaveBeenCalledWith('Insufficient funds');
    });
  });

  describe('subscribe — error handling', () => {
    it('returns the already-subscribed message on 409', async () => {
      mockFetch.mockRejectedValueOnce({ statusCode: 409 });

      const { subscribe, error } = useSubscribe();
      await expect(
        subscribe('init-1', { amountCents: 500, frequency: 'monthly' }),
      ).rejects.toBeTruthy();
      expect(error.value).toBe('You are already subscribed to this initiative.');
      expect(mockShowError).toHaveBeenCalledWith('You are already subscribed to this initiative.');
    });

    it('also handles 409 via the status field (not statusCode)', async () => {
      mockFetch.mockRejectedValueOnce({ status: 409 });

      const { subscribe, error } = useSubscribe();
      await expect(
        subscribe('init-1', { amountCents: 500, frequency: 'monthly' }),
      ).rejects.toBeTruthy();
      expect(error.value).toBe('You are already subscribed to this initiative.');
    });

    it('uses data.error on non-409 API errors', async () => {
      mockFetch.mockRejectedValueOnce({ statusCode: 422, data: { error: 'Invalid amount' } });

      const { subscribe, error } = useSubscribe();
      await expect(
        subscribe('init-1', { amountCents: 0, frequency: 'monthly' }),
      ).rejects.toBeTruthy();
      expect(error.value).toBe('Invalid amount');
    });

    it('falls back to data.message when data.error is absent', async () => {
      mockFetch.mockRejectedValueOnce({ statusCode: 500, data: { message: 'Server error' } });

      const { subscribe, error } = useSubscribe();
      await expect(
        subscribe('init-1', { amountCents: 500, frequency: 'monthly' }),
      ).rejects.toBeTruthy();
      expect(error.value).toBe('Server error');
    });

    it('uses default message when error has no structured data', async () => {
      mockFetch.mockRejectedValueOnce({});

      const { subscribe, error } = useSubscribe();
      await expect(
        subscribe('init-1', { amountCents: 500, frequency: 'monthly' }),
      ).rejects.toBeTruthy();
      expect(error.value).toBe('Subscription failed. Please try again.');
    });

    it('resets loading=false even when the call throws', async () => {
      mockFetch.mockRejectedValueOnce(new Error('fail'));

      const { subscribe, loading } = useSubscribe();
      await expect(
        subscribe('init-1', { amountCents: 100, frequency: 'monthly' }),
      ).rejects.toBeTruthy();
      expect(loading.value).toBe(false);
    });

    it('re-throws the original error so callers can handle it', async () => {
      const originalError = { statusCode: 409 };
      mockFetch.mockRejectedValueOnce(originalError);

      const { subscribe } = useSubscribe();
      await expect(
        subscribe('init-1', { amountCents: 100, frequency: 'monthly' }),
      ).rejects.toMatchObject({ statusCode: 409 });
    });
  });
});
