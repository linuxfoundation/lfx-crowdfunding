// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Initialises the OpenFeature/LaunchDarkly client (client-only).
// Skips init gracefully when no client ID is configured (e.g. local dev).
import { useRuntimeConfig } from 'nuxt/app';
import { initFeatureFlags } from '~/composables/useFeatureFlags';

export default defineNuxtPlugin(() => {
  const {
    public: { ldClientId },
  } = useRuntimeConfig();

  const clientId = ldClientId as string;
  if (!clientId) return;

  initFeatureFlags(clientId).catch((err) => {
    // Crawlers/headless browsers routinely can't reach LaunchDarkly; warn (not captured
    // by RUM) for them so console.error stays a real-user signal. Flags fall back to defaults.
    // ponytail: UA + webdriver heuristic; stealth bots spoofing both still log as errors.
    const isBot =
      navigator.webdriver || /bot\/|crawler|spider|slurp|headless/i.test(navigator.userAgent);
    (isBot ? console.warn : console.error)(
      '[LaunchDarkly] Failed to initialize feature flags:',
      err,
    );
  });
});
