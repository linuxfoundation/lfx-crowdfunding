// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import { randomBytes } from 'node:crypto';

// Per-request CSP nonce (lfx-self-serve-ops#491). Every script Nuxt renders (entry
// modules, payload, app.head and useHead scripts) goes through unhead, so the nonce is
// set on the tag before serialization and properly escaped. Never regex-rewrite the
// rendered HTML instead: owner text in meta attributes and v-html keeps a raw `<`, and
// inserting ` nonce="…"` there would break out of the attribute (XSS).
// server/plugins/csp.ts reads event.context.cspNonce to build the header.
export default defineNuxtPlugin((nuxtApp) => {
  // Vite's dev client relies on inline scripts, so no CSP is sent in dev.
  if (import.meta.dev || !nuxtApp.ssrContext) return;

  const nonce = randomBytes(16).toString('base64');
  nuxtApp.ssrContext.event.context.cspNonce = nonce;
  nuxtApp.ssrContext.head.hooks.hook('tags:resolve', ({ tags }) => {
    for (const tag of tags) {
      if (tag.tag === 'script') tag.props.nonce = nonce;
    }
  });
});
