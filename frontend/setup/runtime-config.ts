// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

const appEnv = process.env.NUXT_APP_ENV || 'development';
const isProduction = appEnv === 'production';

if (isProduction) {
  const required = [
    'NUXT_APP_URL',
    'NUXT_PUBLIC_AUTH0_DOMAIN',
    'NUXT_PUBLIC_AUTH0_CLIENT_ID',
    'NUXT_AUTH0_CLIENT_SECRET',
    'NUXT_AUTH0_COOKIE_DOMAIN',
    'NUXT_PUBLIC_AUTH0_AUDIENCE',
    'NUXT_API_BASE_URL',
  ];
  for (const key of required) {
    if (!process.env[key]) throw new Error(`Missing required env var: ${key}`);
  }
}

const appUrl = process.env.NUXT_APP_URL || 'http://localhost:3000';
const selfServeUrl =
  process.env.NUXT_PUBLIC_SELF_SERVE_URL ||
  (isProduction
    ? 'https://app.lfx.dev/crowdfunding/initiative'
    : 'https://ui-pr-749.dev.v2.cluster.linuxfound.info/crowdfunding/initiatives');
const auth0Domain =
  process.env.NUXT_PUBLIC_AUTH0_DOMAIN || 'https://linuxfoundation-staging.auth0.com';
const auth0CookieDomain = process.env.NUXT_AUTH0_COOKIE_DOMAIN;

export default {
  // Server-only
  apiBaseUrl: process.env.NUXT_API_BASE_URL || 'http://localhost:8080',
  auth0ClientSecret: process.env.NUXT_AUTH0_CLIENT_SECRET || '',
  auth0CookieDomain,
  githubOauthClientSecret: process.env.NUXT_GITHUB_OAUTH_CLIENT_SECRET || '',

  public: {
    apiBase: '/api',
    appEnv,
    appUrl,
    auth0Domain,
    auth0ClientId: process.env.NUXT_PUBLIC_AUTH0_CLIENT_ID || '',
    auth0RedirectUri: process.env.NUXT_PUBLIC_AUTH0_REDIRECT_URI || `${appUrl}/auth/callback`,
    auth0Audience: process.env.NUXT_PUBLIC_AUTH0_AUDIENCE || '',
    selfServeUrl,
    stripePublishableKey: '', // populated from NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY
    githubOauthClientId: process.env.NUXT_PUBLIC_GITHUB_OAUTH_CLIENT_ID || '',
    githubOauthRedirectUri: `${appUrl}/api/github/callback`,
  },
};
