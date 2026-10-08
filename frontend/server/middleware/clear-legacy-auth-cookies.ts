// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, getCookie, setCookie } from 'h3';
import { AUTH_COOKIE_NAMES, authCookieOptions } from '../utils/auth-cookies';

// Auth cookies used to be issued with Domain=NUXT_AUTH0_COOKIE_DOMAIN, which exposed them to
// every sibling host. Browsers still holding those copies would send them (ahead of the new
// host-only cookies), so expire them on the next request.
// getCookie cannot tell a legacy Domain cookie from the new host-only one, so a host-only
// marker makes the cleanup run once per browser instead of on every authenticated request.
// ponytail: remove with NUXT_AUTH0_COOKIE_DOMAIN after 30 days (max legacy refresh-token life)
export const LEGACY_CLEARED_COOKIE = 'auth_legacy_cleared';
const MARKER_MAX_AGE = 30 * 24 * 60 * 60;

export default defineEventHandler((event) => {
  const legacyDomain = useRuntimeConfig().auth0CookieDomain as string | undefined;
  if (!legacyDomain || getCookie(event, LEGACY_CLEARED_COOKIE)) return;

  const opts = { ...authCookieOptions(0), domain: legacyDomain };
  let cleared = false;
  for (const name of AUTH_COOKIE_NAMES) {
    // A Domain-scoped delete never touches the host-only cookie of the same name.
    if (getCookie(event, name) === undefined) continue;
    setCookie(event, name, '', opts);
    cleared = true;
  }
  if (cleared) setCookie(event, LEGACY_CLEARED_COOKIE, '1', authCookieOptions(MARKER_MAX_AGE));
});
