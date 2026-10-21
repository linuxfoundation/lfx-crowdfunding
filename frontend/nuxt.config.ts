// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
// https://nuxt.com/docs/api/configuration/nuxt-config

import head from './setup/head';
import modules from './setup/modules';
import primevue from './setup/primevue';
import robots from './setup/robots';
import runtimeConfig from './setup/runtime-config';
import site from './setup/site';
import sitemap from './setup/sitemap';
import tailwindcss from './setup/tailwind';
import vite from './setup/vite';

export default defineNuxtConfig({
  app: { head },
  compatibilityDate: '2025-01-01',
  // Lets the e2e csp build use its own dir while pnpm dev holds .nuxt (playwright.config.ts).
  buildDir: process.env.NUXT_BUILD_DIR,
  devtools: { enabled: true },
  experimental: { typedPages: true },
  modules,
  plugins: [
    '~/plugins/canonical.ts',
    '~/plugins/vue-query.ts',
    '~/plugins/lfx-ui-core.client.ts',
    '~/plugins/api.client.ts',
    '~/plugins/auth.client.ts',
    '~/plugins/datadog-rum.client.ts',
    '~/plugins/launch-darkly.client.ts',
  ],
  css: ['~/assets/styles/main.scss'],
  primevue,
  robots,
  routeRules: {
    // Baseline CSP (lfx-self-serve-ops#491): stops owner HTML that slips past
    // the sanitizer from posting forms off-site, and blocks framing.
    // Rendered pages get the full nonce-based policy from server/plugins/csp.ts instead.
    '/**': {
      headers: {
        'Content-Security-Policy':
          "form-action 'self'; frame-ancestors 'self'; object-src 'none'; base-uri 'self'",
      },
    },
    // Auth-gated / transactional pages — exclude from search engines
    '/initiatives/*/process-approval/**': { robots: false },
    '/expense-email/**': { robots: false },
  },
  runtimeConfig,
  site,
  ...sitemap,
  tailwindcss,
  vite,
});
