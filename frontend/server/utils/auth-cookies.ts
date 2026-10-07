// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { createCipheriv, createDecipheriv, hkdfSync, randomBytes } from 'node:crypto';
import type { H3Event } from 'h3';
import { getCookie, setCookie } from 'h3';

const VERSION = 'v1.';
const IV_LEN = 12;
const TAG_LEN = 16;

export const AUTH_COOKIE_NAMES = [
  'auth_oidc_token',
  'auth_user_profile',
  'auth_refresh_token',
  'auth_pkce',
  'auth_redirect_to',
] as const;

const isLocal = () => !process.env.NUXT_PUBLIC_APP_ENV;

let cachedKey: Buffer | undefined;

// Key is derived from the Auth0 client secret (already required in production) so no new
// secret has to be provisioned. Rotating the client secret logs everyone out.
function getKey(): Buffer {
  if (cachedKey) return cachedKey;
  // ponytail: fixed dev key only when no client secret is configured locally / in e2e
  const secret =
    useRuntimeConfig().auth0ClientSecret || (isLocal() ? 'local-dev-auth-cookie-secret' : '');
  if (!secret) throw new Error('NUXT_AUTH0_CLIENT_SECRET is required to protect auth cookies');
  cachedKey = Buffer.from(hkdfSync('sha256', secret, '', 'crowdfunding-auth-cookie-v1', 32));
  return cachedKey;
}

export function encryptValue(plain: string): string {
  const iv = randomBytes(IV_LEN);
  const cipher = createCipheriv('aes-256-gcm', getKey(), iv);
  const enc = Buffer.concat([cipher.update(plain, 'utf8'), cipher.final()]);
  return VERSION + Buffer.concat([iv, enc, cipher.getAuthTag()]).toString('base64url');
}

// Returns undefined for anything not produced by encryptValue (missing, legacy plaintext, tampered).
export function decryptValue(value: string | undefined): string | undefined {
  if (!value?.startsWith(VERSION)) return undefined;
  try {
    const raw = Buffer.from(value.slice(VERSION.length), 'base64url');
    if (raw.length < IV_LEN + TAG_LEN) return undefined;
    const decipher = createDecipheriv('aes-256-gcm', getKey(), raw.subarray(0, IV_LEN));
    decipher.setAuthTag(raw.subarray(raw.length - TAG_LEN));
    return Buffer.concat([
      decipher.update(raw.subarray(IV_LEN, raw.length - TAG_LEN)),
      decipher.final(),
    ]).toString('utf8');
  } catch {
    return undefined;
  }
}

// Host-only on purpose: no Domain attribute, so the browser never sends these cookies to
// sibling hosts under a shared parent domain.
export function authCookieOptions(maxAge: number) {
  return {
    httpOnly: true,
    secure: !isLocal(),
    sameSite: 'lax' as const,
    path: '/',
    maxAge,
  };
}

export function setAuthCookie(event: H3Event, name: string, value: string, maxAge: number) {
  setCookie(event, name, encryptValue(value), authCookieOptions(maxAge));
}

export function getAuthCookie(event: H3Event, name: string): string | undefined {
  return decryptValue(getCookie(event, name));
}

// Clears host-only cookies and, if configured, the legacy parent-domain copies.
export function clearAuthCookies(event: H3Event) {
  const opts = authCookieOptions(0);
  const legacyDomain = useRuntimeConfig().auth0CookieDomain as string | undefined;
  for (const name of AUTH_COOKIE_NAMES) {
    setCookie(event, name, '', opts);
    if (legacyDomain) setCookie(event, name, '', { ...opts, domain: legacyDomain });
  }
}
