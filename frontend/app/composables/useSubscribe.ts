// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { ref } from 'vue';
import type { SubscriptionRequest, SubscriptionResult } from '#shared/types/payment.types';

const loading = ref(false);
const error = ref<string | null>(null);

export const useSubscribe = () => {
  const { getStripe } = useStripe();
  const { showError } = useErrorToast();

  const subscribe = async (
    initiativeId: string,
    input: SubscriptionRequest,
  ): Promise<SubscriptionResult> => {
    loading.value = true;
    error.value = null;
    try {
      const result = await $fetch<SubscriptionResult>(
        `/api/initiatives/${initiativeId}/subscriptions`,
        {
          method: 'POST',
          body: input,
          headers: { 'Idempotency-Key': crypto.randomUUID() },
        },
      );

      if (result.clientSecret) {
        const stripe = await getStripe();
        if (!stripe) throw new Error('Stripe.js failed to load.');
        const { error: stripeError } = await stripe.confirmCardPayment(result.clientSecret);
        if (stripeError) throw new Error(stripeError.message);
      }

      return result;
    } catch (e: unknown) {
      const err = e as { data?: { error?: string; message?: string }; message?: string };
      const message =
        err?.data?.error ??
        err?.data?.message ??
        err?.message ??
        'Subscription failed. Please try again.';
      error.value = message;
      showError(message);
      throw e;
    } finally {
      loading.value = false;
    }
  };

  return { loading, error, subscribe };
};
