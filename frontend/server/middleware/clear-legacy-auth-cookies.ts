// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, getCookie, setCookie } from 'h3';
import { AUTH_COOKIE_NAMES, authCookieOptions } from '../utils/auth-cookies';

// Auth cookies used to be issued with Domain=NUXT_AUTH0_COOKIE_DOMAIN, which exposed them to
// every sibling host. Browsers still holding those copies would send them (ahead of the new
// host-only cookies), so expire them on the next request.
// ponytail: remove with NUXT_AUTH0_COOKIE_DOMAIN after 30 days (max legacy refresh-token life)
export default defineEventHandler((event) => {
  const legacyDomain = useRuntimeConfig().auth0CookieDomain as string | undefined;
  if (!legacyDomain) return;

  const opts = { ...authCookieOptions(0), domain: legacyDomain };
  for (const name of AUTH_COOKIE_NAMES) {
    // A Domain-scoped delete never touches the host-only cookie of the same name.
    if (getCookie(event, name) !== undefined) setCookie(event, name, '', opts);
  }
});
