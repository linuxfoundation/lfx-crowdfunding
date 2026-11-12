// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Initialises Datadog Real User Monitoring (client-only).
// Skips init gracefully when credentials are not configured (e.g. local dev).
import { datadogRum } from '@datadog/browser-rum';
import { useRuntimeConfig } from 'nuxt/app';
import { hydrationWarningMessage, isInjectedEvalCspReport, isTranslatedPage } from '~/utils/rum';

export default defineNuxtPlugin(() => {
  const {
    public: { appEnv, appUrl, datadogRumAppId, datadogRumClientToken, appVersion },
  } = useRuntimeConfig();

  const applicationId = datadogRumAppId as string;
  const clientToken = datadogRumClientToken as string;

  if (!applicationId || !clientToken) {
    // RUM credentials not configured — skip init (e.g. local dev without DD creds).
    return;
  }

  datadogRum.init({
    applicationId,
    clientToken,
    site: 'datadoghq.com',
    service: 'crowdfunding',
    env: appEnv as string,
    version: (appVersion as string) || undefined,
    sessionSampleRate: 100,
    sessionReplaySampleRate: 100,
    trackResources: true,
    trackUserInteractions: true,
    trackLongTasks: true,
    defaultPrivacyLevel: 'mask-user-input',
    traceSampleRate: 100,
    // Trace requests to the BFF (/api/* calls only).
    allowedTracingUrls: [`${appUrl as string}/api`],
    beforeSend: (event) => {
      // The CSP report-uri still sends these to Datadog Logs.
      if (isInjectedEvalCspReport(event)) return false;
      // Browser translation rewrites the DOM, which shows up as hydration mismatches; tag
      // these errors so queries can exclude them (-@context.translated:true).
      if (event.type === 'error' && isTranslatedPage(document.documentElement)) {
        event.context = { ...event.context, translated: true };
      }
      return true;
    },
  });

  // Plugins run before the app hydrates, so this sees every mismatch warning. Only
  // hydration warnings are reported; everything still reaches the console as before.
  const warn = console.warn.bind(console);
  console.warn = (...args: unknown[]) => {
    const message = hydrationWarningMessage(args);
    if (message) datadogRum.addError(new Error(message), { source: 'vue-hydration' });
    warn(...args);
  };
});
