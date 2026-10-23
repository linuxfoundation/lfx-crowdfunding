// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

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
  /^(https?:\/\/)?([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z0-9][a-z0-9-]*[a-z0-9](\/\S*)?$/i;
