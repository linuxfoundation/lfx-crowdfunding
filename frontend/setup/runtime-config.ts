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
    'NUXT_JWT_SECRET',
    'NUXT_PUBLIC_AUTH0_AUDIENCE',
  ];
  for (const key of required) {
    if (!process.env[key]) throw new Error(`Missing required env var: ${key}`);
  }
}

const appUrl = process.env.NUXT_APP_URL || 'http://localhost:3000';
const auth0Domain =
  process.env.NUXT_PUBLIC_AUTH0_DOMAIN || 'https://linuxfoundation-staging.auth0.com';
const auth0CookieDomain = process.env.NUXT_AUTH0_COOKIE_DOMAIN;

export default {
  // Server-only
  auth0ClientSecret: process.env.NUXT_AUTH0_CLIENT_SECRET || '',
  auth0CookieDomain,
  jwtSecret: process.env.NUXT_JWT_SECRET || '',
  githubOauthClientSecret: process.env.NUXT_GITHUB_OAUTH_CLIENT_SECRET || '',
  backendBaseUrl: process.env.NUXT_API_BASE_URL ?? 'http://localhost:8080',

  public: {
    apiBase: '/api',
    appEnv,
    appUrl,
    auth0Domain,
    auth0ClientId: process.env.NUXT_PUBLIC_AUTH0_CLIENT_ID || '',
    auth0RedirectUri: `${appUrl}/auth/callback`,
    auth0Audience: process.env.NUXT_PUBLIC_AUTH0_AUDIENCE || '',
    stripePublishableKey: '', // populated from NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY
    githubOauthClientId: process.env.NUXT_PUBLIC_GITHUB_OAUTH_CLIENT_ID || '',
    githubOauthRedirectUri: `${appUrl}/api/github/callback`,
  },
};
