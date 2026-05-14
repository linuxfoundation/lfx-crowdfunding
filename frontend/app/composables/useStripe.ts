// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { loadStripe } from '@stripe/stripe-js';
import type { Stripe } from '@stripe/stripe-js';

// Module-level singleton — Stripe.js is only loaded once per page.
// Keyed by publishable key so a config change (e.g. test → live) gets a fresh instance.
let stripePromise: Promise<Stripe | null> | null = null;
let loadedKey: string | null = null;

export const useStripe = () => {
  const {
    public: { stripePublishableKey },
  } = useRuntimeConfig();

  const getStripe = (): Promise<Stripe | null> => {
    const key = stripePublishableKey as string | undefined;

    if (!key) {
      if (import.meta.dev) {
        console.warn(
          '[useStripe] NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY is not set — did you restart the dev server after adding it to .env?',
        );
      }
      return Promise.resolve(null);
    }

    if (!stripePromise || loadedKey !== key) {
      loadedKey = key;
      stripePromise = loadStripe(key);
    }

    return stripePromise;
  };

  return { getStripe };
};
