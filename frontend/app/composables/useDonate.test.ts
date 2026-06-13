// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';

// --- Stub Nuxt auto-imports as globals before the module under test is loaded ---
// useDonate, useStripe, useErrorToast are Nuxt auto-imports (not explicit imports),
// so they must be stubbed on globalThis rather than via vi.mock module paths.

const mockGetStripe = vi.fn();
const mockShowError = vi.fn();
const mockFetch = vi.fn();

globalThis.$fetch = mockFetch as typeof $fetch;
// @ts-expect-error — Nuxt auto-import stub
globalThis.useStripe = () => ({ getStripe: mockGetStripe });
// @ts-expect-error — Nuxt auto-import stub
globalThis.useErrorToast = () => ({ showError: mockShowError });

// --- Import module under test after mocks are in place ---

import { useDonate } from './useDonate';

describe('useDonate', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset module-level loading/error refs between tests.
    const { loading, error } = useDonate();
    loading.value = false;
    error.value = null;
  });

  describe('donate — happy path (no 3D Secure)', () => {
    it('calls the donations endpoint with the correct body and idempotency header', async () => {
      mockFetch.mockResolvedValueOnce({ clientSecret: null });

      const { donate } = useDonate();
      await donate('init-1', { amountCents: 1000, frequency: 'one_time' });

      expect(mockFetch).toHaveBeenCalledOnce();
      const [url, opts] = mockFetch.mock.calls[0] as [
        string,
        RequestInit & { headers: Record<string, string> },
      ];
      expect(url).toBe('/api/initiatives/init-1/donations');
      expect(opts.method).toBe('POST');
      expect(opts.body).toMatchObject({ amountCents: 1000, frequency: 'one_time' });
      expect(opts.headers['Idempotency-Key']).toMatch(
        /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i,
      );
    });

    it('returns the result from the endpoint', async () => {
      const serverResult = { clientSecret: null, donationId: 'don-1' };
      mockFetch.mockResolvedValueOnce(serverResult);

      const { donate } = useDonate();
      const result = await donate('init-1', { amountCents: 500, frequency: 'one_time' });

      expect(result).toEqual(serverResult);
    });

    it('sets loading=true during the call and resets it afterwards', async () => {
      const { donate, loading } = useDonate();

      let loadingDuringCall = false;
      mockFetch.mockImplementationOnce(async () => {
        loadingDuringCall = loading.value;
        return { clientSecret: null };
      });

      await donate('init-1', { amountCents: 100, frequency: 'one_time' });

      expect(loadingDuringCall).toBe(true);
      expect(loading.value).toBe(false);
    });
  });

  describe('donate — 3D Secure / Stripe confirmation', () => {
    it('calls stripe.confirmCardPayment when a clientSecret is returned', async () => {
      mockFetch.mockResolvedValueOnce({ clientSecret: 'pi_secret_abc' });
      const mockStripe = { confirmCardPayment: vi.fn().mockResolvedValueOnce({ error: null }) };
      mockGetStripe.mockResolvedValueOnce(mockStripe);

      const { donate } = useDonate();
      await donate('init-1', { amountCents: 2000, frequency: 'one_time' });

      expect(mockGetStripe).toHaveBeenCalledOnce();
      expect(mockStripe.confirmCardPayment).toHaveBeenCalledWith('pi_secret_abc');
    });

    it('throws when Stripe.js fails to load (getStripe returns null)', async () => {
      mockFetch.mockResolvedValueOnce({ clientSecret: 'pi_secret_abc' });
      mockGetStripe.mockResolvedValueOnce(null);

      const { donate, error } = useDonate();
      await expect(donate('init-1', { amountCents: 500, frequency: 'one_time' })).rejects.toThrow(
        'Stripe.js failed to load.',
      );
      expect(error.value).toBe('Stripe.js failed to load.');
      expect(mockShowError).toHaveBeenCalledWith('Stripe.js failed to load.');
    });

    it('throws and sets error when Stripe confirmCardPayment fails', async () => {
      mockFetch.mockResolvedValueOnce({ clientSecret: 'pi_secret_abc' });
      const mockStripe = {
        confirmCardPayment: vi.fn().mockResolvedValueOnce({ error: { message: 'Card declined' } }),
      };
      mockGetStripe.mockResolvedValueOnce(mockStripe);

      const { donate, error } = useDonate();
      await expect(donate('init-1', { amountCents: 500, frequency: 'one_time' })).rejects.toThrow();
      expect(error.value).toBe('Card declined');
      expect(mockShowError).toHaveBeenCalledWith('Card declined');
    });
  });

  describe('donate — error handling', () => {
    it('sets error and calls showError on $fetch failure', async () => {
      mockFetch.mockRejectedValueOnce({ data: { message: 'Payment failed' } });

      const { donate, error } = useDonate();
      await expect(
        donate('init-1', { amountCents: 500, frequency: 'one_time' }),
      ).rejects.toBeTruthy();
      expect(error.value).toBe('Payment failed');
      expect(mockShowError).toHaveBeenCalledWith('Payment failed');
    });

    it('uses message fallback when error has no data.message', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      const { donate, error } = useDonate();
      await expect(
        donate('init-1', { amountCents: 500, frequency: 'one_time' }),
      ).rejects.toBeTruthy();
      expect(error.value).toBe('Network error');
    });

    it('uses default message when error has no message at all', async () => {
      mockFetch.mockRejectedValueOnce({});

      const { donate, error } = useDonate();
      await expect(
        donate('init-1', { amountCents: 500, frequency: 'one_time' }),
      ).rejects.toBeTruthy();
      expect(error.value).toBe('Donation failed. Please try again.');
    });

    it('resets loading=false even when the call throws', async () => {
      mockFetch.mockRejectedValueOnce(new Error('fail'));

      const { donate, loading } = useDonate();
      await expect(
        donate('init-1', { amountCents: 100, frequency: 'one_time' }),
      ).rejects.toBeTruthy();
      expect(loading.value).toBe(false);
    });

    it('re-throws the original error so callers can handle it', async () => {
      const originalError = { data: { message: 'Payment failed' }, statusCode: 422 };
      mockFetch.mockRejectedValueOnce(originalError);

      const { donate } = useDonate();
      await expect(
        donate('init-1', { amountCents: 100, frequency: 'one_time' }),
      ).rejects.toMatchObject({ statusCode: 422 });
    });
  });
});
