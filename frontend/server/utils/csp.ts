// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Full Content-Security-Policy for the crowdfunding frontend (lfx-self-serve-ops#491).
// Sent report-only for now; the enforced baseline lives in nuxt.config routeRules.
// Adding a third-party script, API or iframe host? Add it here too, or the browser
// reports (and once enforced, blocks) it.
const SOURCES: Record<string, string[]> = {
  'default-src': ["'self'"],
  // Fallback host list for browsers without 'strict-dynamic'; CSP3 browsers ignore it
  // and trust only nonced scripts plus whatever those scripts load.
  'script-src': [
    "'self'",
    "'strict-dynamic'",
    'https://kit.fontawesome.com',
    'https://ka-f.fontawesome.com',
    'https://ka-p.fontawesome.com',
    'https://plausible.io',
    'https://widget.intercom.io',
    'https://js.intercomcdn.com',
    'https://js.stripe.com',
    'https://*.js.stripe.com',
    // Transcend cookie consent (setup/head.ts).
    'https://transcend-cdn.com',
  ],
  // ponytail: 'unsafe-inline' kept on purpose — SSR style="" attributes, Stripe Elements,
  // Intercom and the Font Awesome kit inject inline styles that can't carry a nonce.
  'style-src': [
    "'self'",
    "'unsafe-inline'",
    'https://fonts.googleapis.com',
    'https://ka-f.fontawesome.com',
    'https://ka-p.fontawesome.com',
  ],
  'font-src': [
    "'self'",
    'data:',
    'https://fonts.gstatic.com',
    'https://ka-f.fontawesome.com',
    'https://ka-p.fontawesome.com',
    'https://js.intercomcdn.com',
    'https://fonts.intercomcdn.com',
  ],
  // Initiative logos and avatars can be any URL.
  'img-src': ["'self'", 'data:', 'blob:', 'https:'],
  'media-src': ["'self'", 'https://js.intercomcdn.com', 'https://downloads.intercomcdn.com'],
  'connect-src': [
    "'self'",
    'https://plausible.io',
    'https://ka-f.fontawesome.com',
    'https://ka-p.fontawesome.com',
    'https://api.stripe.com',
    'https://browser-intake-datadoghq.com',
    'https://*.browser-intake-datadoghq.com',
    'https://clientsdk.launchdarkly.com',
    'https://clientstream.launchdarkly.com',
    'https://app.launchdarkly.com',
    'https://events.launchdarkly.com',
    'https://via.intercom.io',
    'https://api.intercom.io',
    'https://api-iam.intercom.io',
    'https://api-ping.intercom.io',
    'https://*.intercom-messenger.com',
    'wss://*.intercom-messenger.com',
    'https://nexus-websocket-a.intercom.io',
    'wss://nexus-websocket-a.intercom.io',
    'https://nexus-websocket-b.intercom.io',
    'wss://nexus-websocket-b.intercom.io',
    'https://uploads.intercomcdn.com',
    'https://uploads.intercomusercontent.com',
    'https://transcend-cdn.com',
  ],
  'frame-src': [
    'https://js.stripe.com',
    'https://*.js.stripe.com',
    'https://hooks.stripe.com',
    'https://intercom-sheets.com',
    'https://www.intercom-reporting.com',
  ],
  // Datadog session replay runs its compression worker from a blob: URL.
  'worker-src': ["'self'", 'blob:'],
  'form-action': ["'self'"],
  'frame-ancestors': ["'self'"],
  'object-src': ["'none'"],
  'base-uri': ["'self'"],
};

export const buildCspPolicy = (nonce: string, reportUri?: string): string => {
  const directives = Object.entries(SOURCES).map(([name, sources]) => {
    const list = name === 'script-src' ? [`'nonce-${nonce}'`, ...sources] : sources;
    return `${name} ${list.join(' ')}`;
  });
  if (reportUri) directives.push(`report-uri ${reportUri}`);
  return directives.join('; ');
};

// Datadog's log intake accepts browser CSP violation reports; the RUM client token is
// already public (it ships in the page), so it's safe in the header.
export const datadogCspReportUri = (clientToken: string, env: string, version?: string) => {
  const tags = ['service:crowdfunding', `env:${env}`, ...(version ? [`version:${version}`] : [])];
  const params = new URLSearchParams({
    'dd-api-key': clientToken,
    'dd-evp-origin': 'content-security-policy',
    ddsource: 'csp-report',
    ddtags: tags.join(','),
  });
  return `https://browser-intake-datadoghq.com/api/v2/logs?${params}`;
};
