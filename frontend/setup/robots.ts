// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

const isProduction = process.env.NUXT_PUBLIC_APP_ENV === 'production';
const isDevelopment = process.env.NODE_ENV === 'development';

export default {
  // Allow crawling on production and local dev; block all other environments (staging/preview).
  disallow: isProduction || isDevelopment ? [] : ['/'],
};
