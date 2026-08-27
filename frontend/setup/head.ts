// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

const defaultDescription = 'Fund open source projects and mentorships through LFX.';

export default {
  titleTemplate: '%s | LFX Crowdfunding',
  htmlAttrs: { lang: 'en' },
  meta: [
    { charset: 'utf-8' },
    { name: 'viewport', content: 'width=device-width, initial-scale=1' },
    { name: 'description', content: defaultDescription },
    // Open Graph defaults — overridden per-page via useSeoMeta.
    // og:url/og:image/twitter:image are set at runtime in app/plugins/canonical.ts
    // (they depend on runtimeConfig.public.appUrl, which is only resolved correctly
    // per-environment at server start, not here at build time).
    { hid: 'og:type', property: 'og:type', content: 'website' },
    { hid: 'og:site_name', property: 'og:site_name', content: 'LFX Crowdfunding' },
    { hid: 'og:title', property: 'og:title', content: 'LFX Crowdfunding' },
    { hid: 'og:description', property: 'og:description', content: defaultDescription },
    { hid: 'og:image:width', property: 'og:image:width', content: '1200' },
    { hid: 'og:image:height', property: 'og:image:height', content: '630' },
    // Twitter Card defaults
    { hid: 'twitter:card', name: 'twitter:card', content: 'summary_large_image' },
    { hid: 'twitter:title', name: 'twitter:title', content: 'LFX Crowdfunding' },
    { hid: 'twitter:description', name: 'twitter:description', content: defaultDescription },
  ],
  link: [
    {
      rel: 'icon',
      type: 'image/x-icon',
      href: 'https://cdn.platform.linuxfoundation.org/assets/lf-favicon.png',
    },
    { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
    { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: 'anonymous' },
    { rel: 'dns-prefetch', href: 'https://kit.fontawesome.com' },
    {
      rel: 'preload',
      as: 'style',
      href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=Roboto+Slab:wght@300;400;600&display=swap',
      onload: "this.onload=null;this.rel='stylesheet'",
    },
    {
      rel: 'stylesheet',
      href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=Roboto+Slab:wght@300;400;600&display=swap',
      media: 'print',
      onload: "this.media='all'",
    },
  ],
  script: [
    { src: 'https://kit.fontawesome.com/0c49a28643.js', crossorigin: 'anonymous', async: true },
    // Privacy-friendly analytics by Plausible
    { src: 'https://plausible.io/js/pa-Z7youDetgVMZFqKWkN7xd.js', async: true },
    {
      innerHTML:
        'window.plausible=window.plausible||function(){(plausible.q=plausible.q||[]).push(arguments)},plausible.init=plausible.init||function(i){plausible.o=i||{}};plausible.init()',
    },
  ],
};
