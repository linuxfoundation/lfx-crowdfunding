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
  (isProduction ? 'https://app.lfx.dev' : 'https://app.dev.lfx.dev');
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
    intercomAppId:
      process.env.NUXT_PUBLIC_INTERCOM_APP_ID || (isProduction ? 'w29sqomy' : 'mxl90k6y'),
    // Datadog RUM — leave empty locally; set via NUXT_PUBLIC_DATADOG_RUM_* in k8s secrets.
    datadogRumAppId: process.env.NUXT_PUBLIC_DATADOG_RUM_APP_ID || '',
    datadogRumClientToken: process.env.NUXT_PUBLIC_DATADOG_RUM_CLIENT_TOKEN || '',
    // Version is injected from the git tag at deploy time (e.g. "0.1.12").
    datadogRumVersion: process.env.NUXT_PUBLIC_APP_VERSION || '',
    // LaunchDarkly client-side ID — leave empty locally to disable feature flags.
    ldClientId: process.env.NUXT_PUBLIC_LD_CLIENT_ID || '',
  },
};
