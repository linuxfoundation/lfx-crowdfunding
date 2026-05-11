// Copyright (c) 2025 The Linux Foundation and each contributor.
// SPDX-License-Identifier: MIT

const isProduction = process.env.NUXT_APP_ENV === 'production';

export default {
  // Server-only secrets
  auth0ClientSecret: '',
  auth0CookieDomain: isProduction ? 'crowdfunding.lfx.linuxfoundation.org' : 'localhost',
  jwtSecret: '',

  public: {
    apiBase: '/api',
    appEnv: process.env.NUXT_APP_ENV || 'development',
    appUrl: isProduction ? 'https://crowdfunding.lfx.linuxfoundation.org' : 'http://localhost:3000',
    auth0Domain: isProduction
      ? 'https://sso.linuxfoundation.org'
      : 'https://linuxfoundation-staging.auth0.com',
    auth0ClientId: process.env.NUXT_PUBLIC_AUTH0_CLIENT_ID || '',
    auth0RedirectUri: isProduction
      ? 'https://crowdfunding.lfx.linuxfoundation.org/auth/callback'
      : 'http://localhost:3000/auth/callback',
    auth0Audience: isProduction
      ? 'https://crowdfunding.lfx.linuxfoundation.org/api/'
      : 'http://localhost:3000/api/',
  },
};
