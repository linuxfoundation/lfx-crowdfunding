// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { ref } from 'vue';
import type { DonationRequest, DonationResult } from '#shared/types/payment.types';

// Module-level — loading/error reflect the current donation in progress.
// Disable the donate button while loading.value is true to prevent double-charges.
const loading = ref(false);
const error = ref<string | null>(null);

export const useDonate = () => {
  const { getStripe } = useStripe();

  const donate = async (initiativeId: string, input: DonationRequest): Promise<DonationResult> => {
    loading.value = true;
    error.value = null;
    try {
      const result = await $fetch<DonationResult>(`/api/initiatives/${initiativeId}/donations`, {
        method: 'POST',
        body: input,
        headers: { 'Idempotency-Key': crypto.randomUUID() },
      });

      if (result.client_secret) {
        const stripe = await getStripe();
        if (!stripe) throw new Error('Stripe.js failed to load.');
        const { error: stripeError } = await stripe.confirmCardPayment(result.client_secret);
        if (stripeError) throw new Error(stripeError.message);
      }

      return result;
    } catch (e: unknown) {
      const err = e as { data?: { message?: string }; message?: string };
      error.value = err?.data?.message ?? err?.message ?? 'Donation failed. Please try again.';
      throw e;
    } finally {
      loading.value = false;
    }
  };

  return { loading, error, donate };
};
