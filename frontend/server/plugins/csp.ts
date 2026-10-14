// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { randomBytes } from 'node:crypto';
import { setResponseHeader } from 'h3';
import { defineNitroPlugin, useRuntimeConfig } from 'nitropack/runtime';
import { addScriptNonce, buildCspPolicy, datadogCspReportUri } from '../utils/csp';

// Per-request nonce for the full CSP (lfx-self-serve-ops#491), sent report-only while
// it soaks. A nonce per response means rendered HTML must never be cached or
// prerendered — switch to hashes first if that's ever needed.
export default defineNitroPlugin((nitroApp) => {
  // Vite's dev client relies on inline scripts and an HMR websocket.
  if (import.meta.dev) return;

  nitroApp.hooks.hook('render:html', (html, { event }) => {
    const nonce = randomBytes(16).toString('base64');
    for (const key of ['head', 'bodyPrepend', 'body', 'bodyAppend'] as const) {
      // Typed string[], but Nuxt can pass an undefined entry (e.g. empty bodyTags).
      html[key] = html[key].map((chunk) =>
        typeof chunk === 'string' ? addScriptNonce(chunk, nonce) : chunk,
      );
    }

    const { appEnv, datadogRumClientToken, datadogRumVersion } = useRuntimeConfig(event).public;
    const reportUri = datadogRumClientToken
      ? datadogCspReportUri(
          datadogRumClientToken as string,
          appEnv as string,
          datadogRumVersion as string,
        )
      : undefined;
    setResponseHeader(
      event,
      'Content-Security-Policy-Report-Only',
      buildCspPolicy(nonce, reportUri),
    );
  });
});
