// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Globally injects a reactive canonical <link> tag and the default Open Graph /
// Twitter image and URL tags for every route. Base URL comes from
// runtimeConfig.public.appUrl, which is re-read from NUXT_PUBLIC_APP_URL at server
// startup — unlike setup/head.ts (baked into the build at build time), this
// resolves correctly per-environment for a single shared Docker image.
export default defineNuxtPlugin(() => {
  const config = useRuntimeConfig();
  const route = useRoute();
  const baseUrl = (config.public.appUrl as string).replace(/\/$/, '');

  useHead({
    link: [
      {
        rel: 'canonical',
        href: () => {
          const path = route.path.replace(/\/$/, '') || '';
          return `${baseUrl}${path}`;
        },
      },
    ],
  });

  useSeoMeta({
    ogUrl: () => {
      const path = route.path.replace(/\/$/, '') || '';
      return `${baseUrl}${path}`;
    },
    ogImage: `${baseUrl}/og-image.png`,
    twitterImage: `${baseUrl}/og-image.png`,
  });
});
