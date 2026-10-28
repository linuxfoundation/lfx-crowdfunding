// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { OrgClaim } from '#shared/types/organization.types';

// Org search (Clearbit via query-service) in the "I'm donating as" step (#409).

/** Shortest query sent to the org search; the BFF ignores anything shorter. */
export const ORG_SEARCH_MIN_CHARS = 2;

/** Pause after the last keystroke before searching. */
export const ORG_SEARCH_DEBOUNCE_MS = 300;

/**
 * Loose check on a typed website, so Continue stays disabled for obvious typos.
 * The backend normalizes and validates the domain for real; like it, this rejects
 * ports and labels that start or end with a hyphen.
 */
export const ORG_WEBSITE_PATTERN =
  /^(https?:\/\/)?([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z0-9][a-z0-9-]*[a-z0-9]\.?([/?#]\S*)?$/i;

/**
 * The claim for a typed-in org, or null while the name is blank or the website
 * doesn't look like one (which keeps Continue disabled). The domain is the bare
 * host, e.g. "https://www.Acme.com/about" -> "acme.com"; the backend re-normalizes.
 */
export const manualOrgClaim = (name: string, website: string): OrgClaim | null => {
  const n = name.trim();
  const w = website.trim();
  if (!n || !ORG_WEBSITE_PATTERN.test(w)) return null;
  const domain = w
    .replace(/^https?:\/\//i, '')
    .replace(/[/?#].*$/, '')
    .replace(/\.$/, '')
    .replace(/^www\./i, '')
    .toLowerCase();
  return { name: n, domain };
};
