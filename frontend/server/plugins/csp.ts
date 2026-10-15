// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { setResponseHeader, type H3Event } from 'h3';
import { defineNitroPlugin, useRuntimeConfig } from 'nitropack/runtime';
import { buildCspPolicy, datadogCspReportUri } from '../utils/csp';

// Sends the full CSP (lfx-self-serve-ops#491) report-only while it soaks. The nonce is
// generated and stamped on scripts by app/plugins/csp-nonce.server.ts. A nonce per
// response means rendered HTML must never be cached or prerendered — switch to hashes
// first if that's ever needed.
export const setCspHeader = (event: H3Event) => {
  const nonce = event.context.cspNonce as string | undefined;
  // No nonce means the page wasn't rendered by the app (e.g. dev, or an early error).
  if (!nonce) return;

  const { appEnv, datadogRumClientToken, datadogRumVersion } = useRuntimeConfig(event).public;
  const reportUri = datadogRumClientToken
    ? datadogCspReportUri(
        datadogRumClientToken as string,
        appEnv as string,
        datadogRumVersion as string,
      )
    : undefined;
  setResponseHeader(event, 'Content-Security-Policy-Report-Only', buildCspPolicy(nonce, reportUri));
};

export default defineNitroPlugin((nitroApp) => {
  nitroApp.hooks.hook('render:html', (_html, { event }) => setCspHeader(event));
});
