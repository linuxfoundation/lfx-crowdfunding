// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, getRequestURL, sendRedirect } from 'h3';

// The legacy host (crowdfunding.lfx.linuxfoundation.org) redirects here keeping the
// path, so old bookmarks, project websites and emails still arrive with the legacy
// Angular app's URL shapes. Projects, entities, events and initiatives all became
// initiatives with their legacy id kept, and the API resolves an id or a slug, so
// every legacy detail page maps onto /initiatives/{idOrSlug}.
// Compiled once at module load — not inside the handler — to avoid allocating a
// new RegExp on every request.
const RE_LEGACY_DETAIL = /^\/(?:projects|details|events|initiative)\/([^/]+)(?:\/.*)?$/;
// Authenticated flows (project creation, applications, email approvals) have no
// equivalent on this site, and their tokens are legacy-only, so the query is dropped.
const RE_LEGACY_FLOW = /^\/(?:projects\/create|email|apply)(?:\/.*)?$/;

export default defineEventHandler((event) => {
  const method = event.method.toUpperCase();
  if (method !== 'GET' && method !== 'HEAD') return;

  const url = getRequestURL(event);
  // Checked first: /projects/create would otherwise read as a project slug.
  if (RE_LEGACY_FLOW.test(url.pathname)) return sendRedirect(event, '/', 301);

  const detail = RE_LEGACY_DETAIL.exec(url.pathname);
  if (detail) return sendRedirect(event, `/initiatives/${detail[1]}${url.search}`, 301);
});
