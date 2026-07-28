// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Globally injects a reactive canonical <link> tag for every route.
// Trailing slashes are stripped; base URL comes from runtimeConfig.public.appUrl.
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
});
