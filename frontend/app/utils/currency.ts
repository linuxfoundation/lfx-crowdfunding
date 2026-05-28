// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

/**
 * Parses a user-entered dollar string (e.g. "$50,000" or "50000") into cents.
 * Returns undefined if the value is empty, zero, or non-numeric.
 */
export function parseDollarsToCents(value: string | undefined): number | undefined {
  if (!value) return undefined;
  const num = parseFloat(value.replace(/[^0-9.]/g, ''));
  return isNaN(num) || num <= 0 ? undefined : Math.round(num * 100);
}
