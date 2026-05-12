// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
export default {
  titleTemplate: '%s | LFX Crowdfunding',
  htmlAttrs: { lang: 'en' },
  meta: [
    { charset: 'utf-8' },
    { name: 'viewport', content: 'width=device-width, initial-scale=1' },
    { name: 'description', content: 'Fund open source projects and mentorships through LFX.' },
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
    { src: 'https://kit.fontawesome.com/d65f54d9ea.js', crossorigin: 'anonymous', async: true },
  ],
};
