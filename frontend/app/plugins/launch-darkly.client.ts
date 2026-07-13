// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Initialises the OpenFeature/LaunchDarkly client (client-only).
// Skips init gracefully when no client ID is configured (e.g. local dev).
import { useRuntimeConfig } from 'nuxt/app';
import { initFeatureFlags } from '~/composables/useFeatureFlags';

export default defineNuxtPlugin(() => {
  const {
    public: { launchDarklyClientId },
  } = useRuntimeConfig();

  const clientId = launchDarklyClientId as string;
  if (!clientId) return;

  initFeatureFlags(clientId).catch((err) => {
    console.error('[LaunchDarkly] Failed to initialize feature flags:', err);
  });
});
