// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import { VueQueryPlugin, QueryClient, dehydrate, hydrate } from '@tanstack/vue-query';

export default defineNuxtPlugin((nuxtApp) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 5 * 60 * 1000,
        retry: 1,
      },
    },
  });

  nuxtApp.vueApp.use(VueQueryPlugin, { queryClient });

  // SSR: dehydrate state on server, hydrate on client
  if (import.meta.server) {
    nuxtApp.hooks.hook('app:rendered', () => {
      nuxtApp.payload.vueQueryState = dehydrate(queryClient);
    });
  }

  if (import.meta.client) {
    nuxtApp.hooks.hook('app:created', () => {
      hydrate(queryClient, nuxtApp.payload.vueQueryState);
    });
  }
});
