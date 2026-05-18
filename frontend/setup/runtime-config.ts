// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

const appEnv = process.env.NUXT_APP_ENV || 'development';
const isProduction = appEnv === 'production';
const isStaging = appEnv === 'staging';

// Fail fast in production when required secrets are absent — a blank JWT secret
// or Auth0 client secret silently breaks security without throwing an obvious error.
if (isProduction) {
  if (!process.env.NUXT_AUTH0_CLIENT_SECRET)
    throw new Error('Missing required env var: NUXT_AUTH0_CLIENT_SECRET');
  if (!process.env.NUXT_JWT_SECRET) throw new Error('Missing required env var: NUXT_JWT_SECRET');
}

const appUrl = isProduction
  ? 'https://crowdfunding.lfx.linuxfoundation.org'
  : isStaging
    ? process.env.NUXT_APP_URL || 'https://crowdfunding-staging.lfx.linuxfoundation.org'
    : 'http://localhost:3000';

const auth0Domain = isProduction
  ? 'https://sso.linuxfoundation.org'
  : isStaging
    ? 'https://linuxfoundation-staging.auth0.com'
    : 'https://linuxfoundation-dev.auth0.com';

const auth0CookieDomain = isProduction
  ? 'crowdfunding.lfx.linuxfoundation.org'
  : isStaging
    ? 'crowdfunding-staging.lfx.linuxfoundation.org'
    : undefined;

export default {
  // Server-only secrets
  auth0ClientSecret: process.env.NUXT_AUTH0_CLIENT_SECRET || '',
  auth0CookieDomain,
  jwtSecret: process.env.NUXT_JWT_SECRET || '',

  public: {
    apiBase: '/api',
    appEnv,
    appUrl,
    auth0Domain,
    auth0ClientId: process.env.NUXT_PUBLIC_AUTH0_CLIENT_ID || '',
    auth0RedirectUri: `${appUrl}/auth/callback`,
    auth0Audience: `${appUrl}/api/`,
    stripePublishableKey: '', // populated from NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY
  },
};
