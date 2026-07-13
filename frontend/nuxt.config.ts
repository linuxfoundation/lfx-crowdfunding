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
