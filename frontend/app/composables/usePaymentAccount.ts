// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { ref } from 'vue';
import type { StripeCardNumberElement } from '@stripe/stripe-js';
import type { CardDetails, SetupIntentResult } from '#shared/types/payment.types';

// Module-level singleton — card state is shared across all components that call usePaymentAccount().
const card = ref<CardDetails | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);

export const usePaymentAccount = () => {
  const { getStripe } = useStripe();

  const fetchCard = async () => {
    loading.value = true;
    error.value = null;
    try {
      card.value = await $fetch<CardDetails>('/api/payment/account');
    } catch (e: unknown) {
      const err = e as { statusCode?: number; data?: { message?: string } };
      if (err?.statusCode === 404) {
        card.value = null;
      } else {
        error.value = err?.data?.message ?? 'Could not load your payment account.';
      }
    } finally {
      loading.value = false;
    }
  };

  const saveCard = async (cardElement: StripeCardNumberElement) => {
    loading.value = true;
    error.value = null;
    try {
      const { client_secret } = await $fetch<SetupIntentResult>('/api/payment/setup-intent', {
        method: 'POST',
      });

      const stripe = await getStripe();
      if (!stripe)
        throw new Error('Stripe.js failed to load — check NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY.');

      const { setupIntent, error: stripeError } = await stripe.confirmCardSetup(client_secret, {
        payment_method: { card: cardElement },
      });

      if (stripeError) throw new Error(stripeError.message);
      if (!setupIntent?.payment_method)
        throw new Error('Stripe did not return a payment method. Please try again.');

      const paymentMethodId =
        typeof setupIntent.payment_method === 'string'
          ? setupIntent.payment_method
          : setupIntent.payment_method.id;

      card.value = await $fetch<CardDetails>('/api/payment/method', {
        method: 'POST',
        body: { payment_method_id: paymentMethodId },
      });
    } catch (e: unknown) {
      const err = e as { message?: string };
      error.value = err?.message ?? 'Failed to save your card.';
      throw e;
    } finally {
      loading.value = false;
    }
  };

  return { card, loading, error, fetchCard, saveCard };
};
