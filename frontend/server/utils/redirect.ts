// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

const isLocal = process.env.NUXT_APP_ENV !== 'staging' && process.env.NUXT_APP_ENV !== 'production';

const ALLOWED_REDIRECT_DOMAINS = isLocal
  ? ['linuxfoundation.org', 'auth0.com', 'localhost']
  : ['linuxfoundation.org', 'auth0.com'];

export const DEFAULT_REDIRECT = '/';

/**
 * Validates a redirect URL to prevent open redirect vulnerabilities.
 */
export function isValidRedirectUrl(url: string | undefined | null): boolean {
  if (!url || typeof url !== 'string') return false;

  const trimmed = url.trim();
  if (!trimmed) return false;

  // Reject protocol-relative URLs (//example.com)
  if (trimmed.startsWith('//')) return false;

  // Reject dangerous protocols
  const dangerous = ['javascript:', 'data:', 'vbscript:', 'file:'];
  const lower = trimmed.toLowerCase();
  if (dangerous.some((p) => lower.startsWith(p))) return false;

  // Allow relative URLs starting with /
  if (trimmed.startsWith('/') && !trimmed.startsWith('//')) {
    try {
      if (decodeURIComponent(trimmed).startsWith('//')) return false;
    } catch {
      return false;
    }
    return true;
  }

  // Validate absolute URLs against allowed domains
  try {
    const { protocol, hostname } = new URL(trimmed);
    if (protocol !== 'http:' && protocol !== 'https:') return false;
    const host = hostname.toLowerCase();
    return ALLOWED_REDIRECT_DOMAINS.some((d) => host === d || host.endsWith(`.${d}`));
  } catch {
    return false;
  }
}

/**
 * Returns the URL if valid, otherwise returns the fallback.
 */
export function getSafeRedirectUrl(
  url: string | undefined | null,
  fallback: string = DEFAULT_REDIRECT,
): string {
  return isValidRedirectUrl(url) ? url!.trim() : fallback;
}
