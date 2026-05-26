// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import { VueQueryPlugin, QueryClient, QueryCache, dehydrate, hydrate } from '@tanstack/vue-query';

export default defineNuxtPlugin((nuxtApp) => {
  const queryClient = new QueryClient({
    queryCache: new QueryCache({
      onError: (error) => {
        if (import.meta.client) {
          nuxtApp.runWithContext(() => {
            const { showError } = useErrorToast();
            const err = error as {
              data?: { message?: string; statusMessage?: string };
              message?: string;
            };
            showError(
              err?.data?.message ??
                err?.data?.statusMessage ??
                err?.message ??
                'Failed to load data. Please try again.',
            );
          });
        }
      },
    }),
    defaultOptions: {
      queries: {
        staleTime: 5 * 60 * 1000,
        retry: 1,
        // Prevent orphaned query instances from leaking memory on the server
        ...(import.meta.server ? { gcTime: 0 } : {}),
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
